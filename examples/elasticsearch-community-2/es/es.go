// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package es is an Elasticsearch backend whose catalog is a hybrid: columns and types come from the
// index's native _mapping, while primary and foreign keys come from a declared system index (see README).
package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/esapi"

	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

// CatalogIndex is the reserved system index holding the declared keys. It never appears as a SQL table.
// (ES index names may not start with "_", so this is haystak_catalog, not Mongo's __haystak_catalog.)
const CatalogIndex = "haystak_catalog"

// scanSize caps rows returned per index in this demo (Elasticsearch's default max_result_window).
const scanSize = 10000

// Backend serves Elasticsearch indices as SQL tables. Columns/types are read from each index's native
// _mapping; primary/foreign keys are declared in CatalogIndex. Thin (Scanner), writable, DDL-capable.
type Backend struct {
	es *elasticsearch.Client
}

func New(client *elasticsearch.Client) *Backend { return &Backend{es: client} }

func (b *Backend) Capabilities() tds.Caps {
	return tds.Caps{Pushdown: true, Writable: true, DDL: true}
}

// catalogDoc is one declared table's keys; columns are not stored here — ES already knows them.
type catalogDoc struct {
	Table       string      `json:"table"`
	PrimaryKey  []string    `json:"primary_key"`
	ForeignKeys []catalogFK `json:"foreign_keys"`
}

type catalogFK struct {
	Columns    []string `json:"columns"`
	RefTable   string   `json:"ref_table"`
	RefColumns []string `json:"ref_columns"`
}

func (d catalogDoc) toFKs() []catalog.ForeignKey {
	fks := make([]catalog.ForeignKey, len(d.ForeignKeys))
	for i, fk := range d.ForeignKeys {
		fks[i] = catalog.ForeignKey{Columns: fk.Columns, RefTable: fk.RefTable, RefColumns: fk.RefColumns}
	}
	return fks
}

// Describe assembles each declared table: columns from its _mapping, keys from CatalogIndex. A declared
// table whose index is unreadable (e.g. dropped out of band) is skipped so one stale entry can't break all queries.
func (b *Backend) Describe(ctx context.Context) (catalog.Schema, error) {
	docs, err := b.allCatalogDocs(ctx)
	if err != nil {
		return catalog.Schema{}, err
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].Table < docs[j].Table })
	var s catalog.Schema
	for _, d := range docs {
		cols, err := b.mappingColumns(ctx, d.Table)
		if err != nil || len(cols) == 0 {
			continue
		}
		s.Tables = append(s.Tables, catalog.Table{Name: d.Table, Columns: cols, PrimaryKey: d.PrimaryKey, ForeignKeys: d.toFKs()})
	}
	return s, nil
}

// Scan projects the declared columns (from _mapping) of a declared table; an undeclared index is not a SQL table.
func (b *Backend) Scan(ctx context.Context, q *tds.Query) (tds.Rows, error) {
	if _, err := b.catalogDocFor(ctx, q.Table); err != nil {
		return nil, err
	}
	cols, err := b.mappingColumns(ctx, q.Table)
	if err != nil {
		return nil, err
	}
	hits, err := b.search(ctx, q.Table, scanSize)
	if err != nil {
		return nil, err
	}
	data := make([][]any, 0, len(hits))
	for _, h := range hits {
		row := make([]any, len(cols))
		for i, c := range cols {
			row[i] = jsonValue(h.Source[c.Name])
		}
		data = append(data, row)
	}
	return &rows{cols: cols, data: data}, nil
}

func (b *Backend) Insert(ctx context.Context, in *tds.Insert) (tds.Result, error) {
	n := int64(0)
	for _, vals := range in.Rows {
		doc := map[string]any{}
		id := ""
		for i, col := range in.Columns {
			if i >= len(vals) {
				continue
			}
			if col == "_id" {
				id = fmt.Sprintf("%v", vals[i])
				continue
			}
			doc[col] = vals[i]
		}
		body, err := json.Marshal(doc)
		if err != nil {
			return tds.Result{}, err
		}
		opts := []func(*esapi.IndexRequest){
			b.es.Index.WithContext(ctx),
			b.es.Index.WithRefresh("true"),
		}
		if id != "" {
			opts = append(opts, b.es.Index.WithDocumentID(id))
		}
		res, err := b.es.Index(in.Table, bytes.NewReader(body), opts...)
		if err != nil {
			return tds.Result{}, err
		}
		res.Body.Close()
		if res.IsError() {
			return tds.Result{}, fmt.Errorf("es index %s: %s", in.Table, res.String())
		}
		n++
	}
	return tds.Result{RowsAffected: n}, nil
}

func (b *Backend) Update(ctx context.Context, up *tds.Update) (tds.Result, error) {
	srcs := make([]string, len(up.Assignments))
	params := map[string]any{}
	for i, a := range up.Assignments {
		srcs[i] = fmt.Sprintf("ctx._source[params.k%d] = params.v%d", i, i)
		params[fmt.Sprintf("k%d", i)] = a.Column
		params[fmt.Sprintf("v%d", i)] = a.Value
	}
	body, err := json.Marshal(map[string]any{
		"query":  predsToQuery(up.Where),
		"script": map[string]any{"source": strings.Join(srcs, "; "), "params": params},
	})
	if err != nil {
		return tds.Result{}, err
	}
	res, err := b.es.UpdateByQuery(
		[]string{up.Table},
		b.es.UpdateByQuery.WithContext(ctx),
		b.es.UpdateByQuery.WithBody(bytes.NewReader(body)),
		b.es.UpdateByQuery.WithRefresh(true),
	)
	if err != nil {
		return tds.Result{}, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return tds.Result{}, fmt.Errorf("es update_by_query %s: %s", up.Table, res.String())
	}
	var r struct {
		Updated int64 `json:"updated"`
	}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return tds.Result{}, err
	}
	return tds.Result{RowsAffected: r.Updated}, nil
}

func (b *Backend) Delete(ctx context.Context, del *tds.Delete) (tds.Result, error) {
	body, err := json.Marshal(map[string]any{"query": predsToQuery(del.Where)})
	if err != nil {
		return tds.Result{}, err
	}
	res, err := b.es.DeleteByQuery(
		[]string{del.Table},
		bytes.NewReader(body),
		b.es.DeleteByQuery.WithContext(ctx),
		b.es.DeleteByQuery.WithRefresh(true),
	)
	if err != nil {
		return tds.Result{}, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return tds.Result{}, fmt.Errorf("es delete_by_query %s: %s", del.Table, res.String())
	}
	var r struct {
		Deleted int64 `json:"deleted"`
	}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return tds.Result{}, err
	}
	return tds.Result{RowsAffected: r.Deleted}, nil
}

// CreateTable creates the index with an explicit mapping from the statement's columns, then declares its keys.
func (b *Backend) CreateTable(ctx context.Context, t *catalog.Table) error {
	props := map[string]any{}
	for _, c := range t.Columns {
		props[c.Name] = map[string]any{"type": kindToES(c.Type.Kind)}
	}
	body, err := json.Marshal(map[string]any{"mappings": map[string]any{"properties": props}})
	if err != nil {
		return err
	}
	res, err := b.es.Indices.Create(t.Name, b.es.Indices.Create.WithContext(ctx), b.es.Indices.Create.WithBody(bytes.NewReader(body)))
	if err != nil {
		return err
	}
	res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("es create index %s: %s", t.Name, res.String())
	}
	fks := make([]catalogFK, len(t.ForeignKeys))
	for i, fk := range t.ForeignKeys {
		fks[i] = catalogFK{Columns: fk.Columns, RefTable: fk.RefTable, RefColumns: fk.RefColumns}
	}
	return b.writeCatalog(ctx, catalogDoc{Table: t.Name, PrimaryKey: t.PrimaryKey, ForeignKeys: fks})
}

// AlterTable adds columns to the index's mapping (the catalog only holds keys, so it is untouched).
func (b *Backend) AlterTable(ctx context.Context, a *tds.AlterTable) error {
	if len(a.AddColumns) == 0 {
		return nil
	}
	props := map[string]any{}
	for _, c := range a.AddColumns {
		props[c.Name] = map[string]any{"type": kindToES(c.Type.Kind)}
	}
	body, err := json.Marshal(map[string]any{"properties": props})
	if err != nil {
		return err
	}
	res, err := b.es.Indices.PutMapping([]string{a.Table}, bytes.NewReader(body), b.es.Indices.PutMapping.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("es put mapping %s: %s", a.Table, res.String())
	}
	return nil
}

// DropTable deletes the index and removes its catalog declaration.
func (b *Backend) DropTable(ctx context.Context, table string) error {
	res, err := b.es.Indices.Delete([]string{table}, b.es.Indices.Delete.WithContext(ctx))
	if err != nil {
		return err
	}
	res.Body.Close()
	d, err := b.es.Delete(CatalogIndex, table, b.es.Delete.WithContext(ctx), b.es.Delete.WithRefresh("true"))
	if err != nil {
		return err
	}
	d.Body.Close()
	return nil
}

// EnsureCatalog bootstraps the keys (PK="id" when present, FKs empty) when CatalogIndex is missing/empty; columns come from mappings. Never overwrites an existing catalog.
func (b *Backend) EnsureCatalog(ctx context.Context, pattern string) (int, error) {
	docs, err := b.allCatalogDocs(ctx)
	if err != nil {
		return 0, err
	}
	if len(docs) > 0 {
		return 0, nil
	}
	names, err := b.indexNames(ctx, pattern)
	if err != nil {
		return 0, err
	}
	sort.Strings(names)
	written := 0
	for _, name := range names {
		cols, err := b.mappingColumns(ctx, name)
		if err != nil {
			return written, err
		}
		pk := []string{}
		for _, c := range cols {
			if c.Name == "id" {
				pk = []string{"id"}
				break
			}
		}
		if err := b.writeCatalog(ctx, catalogDoc{Table: name, PrimaryKey: pk, ForeignKeys: []catalogFK{}}); err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}

// mappingColumns reads an index's native _mapping and translates ES field types to SQL column types.
func (b *Backend) mappingColumns(ctx context.Context, index string) ([]catalog.Column, error) {
	res, err := b.es.Indices.GetMapping(
		b.es.Indices.GetMapping.WithContext(ctx),
		b.es.Indices.GetMapping.WithIndex(index),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("es get mapping %s: %s", index, res.String())
	}
	var m map[string]struct {
		Mappings struct {
			Properties map[string]struct {
				Type string `json:"type"`
			} `json:"properties"`
		} `json:"mappings"`
	}
	if err := json.NewDecoder(res.Body).Decode(&m); err != nil {
		return nil, err
	}
	var props map[string]struct {
		Type string `json:"type"`
	}
	for _, v := range m {
		props = v.Mappings.Properties
		break
	}
	names := make([]string, 0, len(props))
	for k := range props {
		names = append(names, k)
	}
	sort.Strings(names)
	cols := make([]catalog.Column, 0, len(names))
	for _, name := range names {
		cols = append(cols, catalog.Column{Name: name, Type: esTypeToSQL(props[name].Type)})
	}
	return cols, nil
}

func (b *Backend) catalogDocFor(ctx context.Context, table string) (catalogDoc, error) {
	res, err := b.es.Get(CatalogIndex, table, b.es.Get.WithContext(ctx))
	if err != nil {
		return catalogDoc{}, err
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		return catalogDoc{}, fmt.Errorf("es: table %q is not declared in %s", table, CatalogIndex)
	}
	if res.IsError() {
		return catalogDoc{}, fmt.Errorf("es get %s/%s: %s", CatalogIndex, table, res.String())
	}
	var g struct {
		Source catalogDoc `json:"_source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&g); err != nil {
		return catalogDoc{}, err
	}
	return g.Source, nil
}

func (b *Backend) allCatalogDocs(ctx context.Context) ([]catalogDoc, error) {
	res, err := b.es.Search(
		b.es.Search.WithContext(ctx),
		b.es.Search.WithIndex(CatalogIndex),
		b.es.Search.WithBody(strings.NewReader(`{"query":{"match_all":{}}}`)),
		b.es.Search.WithSize(scanSize),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		return nil, nil
	}
	if res.IsError() {
		return nil, fmt.Errorf("es search %s: %s", CatalogIndex, res.String())
	}
	var sr struct {
		Hits struct {
			Hits []struct {
				Source catalogDoc `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&sr); err != nil {
		return nil, err
	}
	docs := make([]catalogDoc, 0, len(sr.Hits.Hits))
	for _, h := range sr.Hits.Hits {
		docs = append(docs, h.Source)
	}
	return docs, nil
}

func (b *Backend) writeCatalog(ctx context.Context, doc catalogDoc) error {
	if err := b.ensureCatalogIndex(ctx); err != nil {
		return err
	}
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	res, err := b.es.Index(CatalogIndex, bytes.NewReader(body),
		b.es.Index.WithContext(ctx),
		b.es.Index.WithDocumentID(doc.Table),
		b.es.Index.WithRefresh("true"))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("es write catalog %s: %s", doc.Table, res.String())
	}
	return nil
}

func (b *Backend) ensureCatalogIndex(ctx context.Context) error {
	if b.indexExists(ctx, CatalogIndex) {
		return nil
	}
	body := strings.NewReader(`{"mappings":{"properties":{"table":{"type":"keyword"},"primary_key":{"type":"keyword"},"foreign_keys":{"type":"object","enabled":false}}}}`)
	res, err := b.es.Indices.Create(CatalogIndex, b.es.Indices.Create.WithContext(ctx), b.es.Indices.Create.WithBody(body))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() && !strings.Contains(res.String(), "resource_already_exists") {
		return fmt.Errorf("es create %s: %s", CatalogIndex, res.String())
	}
	return nil
}

func (b *Backend) indexExists(ctx context.Context, index string) bool {
	res, err := b.es.Indices.Exists([]string{index}, b.es.Indices.Exists.WithContext(ctx))
	if err != nil {
		return false
	}
	res.Body.Close()
	return res.StatusCode == 200
}

// indexNames resolves a pattern to concrete data-index names, skipping ES system (dot) indices and CatalogIndex.
func (b *Backend) indexNames(ctx context.Context, pattern string) ([]string, error) {
	res, err := b.es.Cat.Indices(
		b.es.Cat.Indices.WithContext(ctx),
		b.es.Cat.Indices.WithIndex(pattern),
		b.es.Cat.Indices.WithFormat("json"),
		b.es.Cat.Indices.WithH("index"),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("es cat indices %q: %s", pattern, res.String())
	}
	var arr []struct {
		Index string `json:"index"`
	}
	if err := json.NewDecoder(res.Body).Decode(&arr); err != nil {
		return nil, err
	}
	var names []string
	for _, r := range arr {
		if strings.HasPrefix(r.Index, ".") || r.Index == CatalogIndex {
			continue
		}
		names = append(names, r.Index)
	}
	return names, nil
}

type hit struct {
	ID     string         `json:"_id"`
	Source map[string]any `json:"_source"`
}

func (b *Backend) search(ctx context.Context, index string, size int) ([]hit, error) {
	res, err := b.es.Search(
		b.es.Search.WithContext(ctx),
		b.es.Search.WithIndex(index),
		b.es.Search.WithBody(strings.NewReader(`{"query":{"match_all":{}}}`)),
		b.es.Search.WithSize(size),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("es search %s: %s", index, res.String())
	}
	dec := json.NewDecoder(res.Body)
	dec.UseNumber()
	var sr struct {
		Hits struct {
			Hits []hit `json:"hits"`
		} `json:"hits"`
	}
	if err := dec.Decode(&sr); err != nil {
		return nil, err
	}
	return sr.Hits.Hits, nil
}

func predsToQuery(preds []tds.Predicate) map[string]any {
	if len(preds) == 0 {
		return map[string]any{"match_all": map[string]any{}}
	}
	must := make([]any, 0, len(preds))
	for _, p := range preds {
		switch p.Op {
		case tds.OpNe:
			must = append(must, map[string]any{"bool": map[string]any{"must_not": map[string]any{"match": map[string]any{p.Column: p.Value}}}})
		case tds.OpLt:
			must = append(must, rangeClause(p.Column, "lt", p.Value))
		case tds.OpLe:
			must = append(must, rangeClause(p.Column, "lte", p.Value))
		case tds.OpGt:
			must = append(must, rangeClause(p.Column, "gt", p.Value))
		case tds.OpGe:
			must = append(must, rangeClause(p.Column, "gte", p.Value))
		default:
			must = append(must, map[string]any{"match": map[string]any{p.Column: p.Value}})
		}
	}
	return map[string]any{"bool": map[string]any{"must": must}}
}

func rangeClause(col, op string, v any) map[string]any {
	return map[string]any{"range": map[string]any{col: map[string]any{op: v}}}
}

// esTypeToSQL maps an Elasticsearch field type to the SPI's column type.
func esTypeToSQL(esType string) types.Type {
	switch esType {
	case "long", "unsigned_long":
		return types.Type{Kind: types.Int64}
	case "integer", "short", "byte":
		return types.Type{Kind: types.Int32}
	case "float", "double", "half_float", "scaled_float":
		return types.Type{Kind: types.Float64}
	case "boolean":
		return types.Type{Kind: types.Bool}
	case "date", "date_nanos":
		return types.Type{Kind: types.Time}
	case "binary":
		return types.Type{Kind: types.Bytes}
	}
	return types.Type{Kind: types.String, MaxLen: 255}
}

func kindToES(k types.Kind) string {
	switch k {
	case types.Int64:
		return "long"
	case types.Int32:
		return "integer"
	case types.Bool:
		return "boolean"
	case types.Float64, types.Decimal:
		return "double"
	case types.Time:
		return "date"
	case types.Bytes:
		return "binary"
	}
	return "keyword"
}

func jsonValue(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case bool:
		return x
	case string:
		return x
	case json.Number:
		if strings.ContainsAny(x.String(), ".eE") {
			f, _ := x.Float64()
			return f
		}
		if n, err := x.Int64(); err == nil {
			return n
		}
		f, _ := x.Float64()
		return f
	case map[string]any, []any:
		bs, _ := json.Marshal(x)
		return string(bs)
	}
	return fmt.Sprintf("%v", v)
}

type rows struct {
	cols []catalog.Column
	data [][]any
	pos  int
}

func (r *rows) Columns() []catalog.Column { return r.cols }
func (r *rows) Next() bool {
	if r.pos >= len(r.data) {
		return false
	}
	r.pos++
	return true
}
func (r *rows) Values() ([]any, error) { return r.data[r.pos-1], nil }
func (r *rows) Err() error             { return nil }
func (r *rows) Close() error           { return nil }

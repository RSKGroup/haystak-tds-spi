// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package es is a haystak-tds-spi backend over Elasticsearch: indices map to SQL tables,
// document fields to columns (inferred by sampling _source), and writes/DDL to native ES APIs.
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

// scanSize caps rows returned per index in this demo (Elasticsearch's default max_result_window).
const scanSize = 10000

// Backend serves a set of Elasticsearch indices as a SQL catalog. It is a thin (Scanner) backend, so
// the gateway engine applies WHERE/JOIN/GROUP BY/etc, and it is writable and DDL-capable.
type Backend struct {
	es      *elasticsearch.Client
	pattern string // index expression exposed as tables ("users,orders" or "sales-*")
	sample  int
}

func New(client *elasticsearch.Client, pattern string) *Backend {
	return &Backend{es: client, pattern: pattern, sample: 100}
}

func (b *Backend) Capabilities() tds.Caps {
	return tds.Caps{Pushdown: true, Writable: true, DDL: true}
}

// Describe lists the matching indices and samples each to infer its columns (the inferred-catalog model).
func (b *Backend) Describe(ctx context.Context) (catalog.Schema, error) {
	names, err := b.indexNames(ctx)
	if err != nil {
		return catalog.Schema{}, err
	}
	sort.Strings(names)
	var s catalog.Schema
	for _, name := range names {
		hits, err := b.search(ctx, name, b.sample)
		if err != nil {
			return catalog.Schema{}, err
		}
		s.Tables = append(s.Tables, catalog.Table{Name: name, Columns: columnsFromHits(hits), PrimaryKey: []string{"_id"}})
	}
	return s, nil
}

// indexNames resolves the configured pattern to concrete index names, skipping ES system (dot) indices.
func (b *Backend) indexNames(ctx context.Context) ([]string, error) {
	res, err := b.es.Cat.Indices(
		b.es.Cat.Indices.WithContext(ctx),
		b.es.Cat.Indices.WithIndex(b.pattern),
		b.es.Cat.Indices.WithFormat("json"),
		b.es.Cat.Indices.WithH("index"),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("es cat indices %q: %s", b.pattern, res.String())
	}
	var rows []struct {
		Index string `json:"index"`
	}
	if err := json.NewDecoder(res.Body).Decode(&rows); err != nil {
		return nil, err
	}
	var names []string
	for _, r := range rows {
		if strings.HasPrefix(r.Index, ".") {
			continue
		}
		names = append(names, r.Index)
	}
	return names, nil
}

// Scan reads the index (capped at scanSize) and flattens hits into rows in the inferred column order.
func (b *Backend) Scan(ctx context.Context, q *tds.Query) (tds.Rows, error) {
	hits, err := b.search(ctx, q.Table, scanSize)
	if err != nil {
		return nil, err
	}
	cols := columnsFromHits(hits)
	data := make([][]any, 0, len(hits))
	for _, h := range hits {
		row := make([]any, len(cols))
		for i, c := range cols {
			if c.Name == "_id" {
				row[i] = h.ID
				continue
			}
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

// Update maps SET … WHERE to _update_by_query with a Painless script (field names passed as params).
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

func (b *Backend) CreateTable(ctx context.Context, t *catalog.Table) error {
	res, err := b.es.Indices.Create(t.Name, b.es.Indices.Create.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("es create index %s: %s", t.Name, res.String())
	}
	return nil
}

func (b *Backend) AlterTable(ctx context.Context, a *tds.AlterTable) error { return nil } // dynamic mapping

func (b *Backend) DropTable(ctx context.Context, table string) error {
	res, err := b.es.Indices.Delete([]string{table}, b.es.Indices.Delete.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("es delete index %s: %s", table, res.String())
	}
	return nil
}

// hit is one search hit: the document id and its _source (numbers decoded as json.Number).
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

// columnsFromHits infers a stable column set from sampled hits: _id first, then fields sorted by name.
func columnsFromHits(hits []hit) []catalog.Column {
	kind := map[string]types.Kind{}
	for _, h := range hits {
		for k, v := range h.Source {
			if _, seen := kind[k]; !seen {
				kind[k] = jsonKind(v)
			}
		}
	}
	names := make([]string, 0, len(kind))
	for k := range kind {
		names = append(names, k)
	}
	sort.Strings(names)
	cols := make([]catalog.Column, 0, len(names)+1)
	cols = append(cols, catalog.Column{Name: "_id", Type: types.Type{Kind: types.String, MaxLen: 64}})
	for _, name := range names {
		cols = append(cols, catalog.Column{Name: name, Type: types.Type{Kind: kind[name], MaxLen: 255}})
	}
	return cols
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

func jsonKind(v any) types.Kind {
	switch x := v.(type) {
	case bool:
		return types.Bool
	case json.Number:
		if strings.ContainsAny(x.String(), ".eE") {
			return types.Float64
		}
		return types.Int64
	}
	return types.String
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

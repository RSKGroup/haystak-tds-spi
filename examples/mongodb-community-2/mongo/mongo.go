// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package mongo is a MongoDB backend whose catalog is declared in a system collection (see README).
package mongo

import (
	"context"
	"fmt"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

// CatalogCollection is the reserved system collection holding the declared catalog. It never appears
// as a SQL table (no catalog document describes itself).
const CatalogCollection = "__haystak_catalog"

// Backend serves one Mongo database as a SQL catalog declared in CatalogCollection. It is a thin
// (Scanner) backend, so the gateway engine applies WHERE/JOIN/GROUP BY/etc, and it is writable and DDL-capable.
type Backend struct {
	client *mongo.Client
	db     string
}

func New(client *mongo.Client, db string) *Backend { return &Backend{client: client, db: db} }

func (b *Backend) database() *mongo.Database { return b.client.Database(b.db) }

func (b *Backend) Capabilities() tds.Caps {
	return tds.Caps{Pushdown: true, Writable: true, DDL: true}
}

// catalogDoc is one row of the declared catalog: a table's columns, PK, and FK edges.
type catalogDoc struct {
	Table       string       `bson:"table"`
	Columns     []catalogCol `bson:"columns"`
	PrimaryKey  []string     `bson:"primary_key"`
	ForeignKeys []catalogFK  `bson:"foreign_keys"`
}

type catalogCol struct {
	Name string `bson:"name"`
	Type string `bson:"type"`
}

type catalogFK struct {
	Columns    []string `bson:"columns"`
	RefTable   string   `bson:"ref_table"`
	RefColumns []string `bson:"ref_columns"`
}

// Describe reads the declared catalog: one cheap, indexed read of CatalogCollection, no sampling.
func (b *Backend) Describe(ctx context.Context) (catalog.Schema, error) {
	cur, err := b.database().Collection(CatalogCollection).Find(ctx, bson.D{})
	if err != nil {
		return catalog.Schema{}, err
	}
	defer cur.Close(ctx)
	var docs []catalogDoc
	if err := cur.All(ctx, &docs); err != nil {
		return catalog.Schema{}, err
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].Table < docs[j].Table })
	var s catalog.Schema
	for _, d := range docs {
		s.Tables = append(s.Tables, d.toTable())
	}
	return s, nil
}

func (d catalogDoc) toTable() catalog.Table {
	cols := make([]catalog.Column, len(d.Columns))
	for i, c := range d.Columns {
		cols[i] = catalog.Column{Name: c.Name, Type: sqlType(c.Type)}
	}
	fks := make([]catalog.ForeignKey, len(d.ForeignKeys))
	for i, fk := range d.ForeignKeys {
		fks[i] = catalog.ForeignKey{Columns: fk.Columns, RefTable: fk.RefTable, RefColumns: fk.RefColumns}
	}
	return catalog.Table{Name: d.Table, Columns: cols, PrimaryKey: d.PrimaryKey, ForeignKeys: fks}
}

// tableDef fetches a single table's declared definition; an undeclared collection is not a SQL table.
func (b *Backend) tableDef(ctx context.Context, name string) (catalog.Table, error) {
	var d catalogDoc
	err := b.database().Collection(CatalogCollection).FindOne(ctx, bson.D{{Key: "table", Value: name}}).Decode(&d)
	if err != nil {
		return catalog.Table{}, fmt.Errorf("mongo: table %q is not declared in %s: %w", name, CatalogCollection, err)
	}
	return d.toTable(), nil
}

// Scan reads the named collection and projects the declared columns in declared order.
func (b *Backend) Scan(ctx context.Context, q *tds.Query) (tds.Rows, error) {
	t, err := b.tableDef(ctx, q.Table)
	if err != nil {
		return nil, err
	}
	cur, err := b.database().Collection(q.Table).Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var data [][]any
	for cur.Next(ctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		row := make([]any, len(t.Columns))
		for i, c := range t.Columns {
			row[i] = bsonValue(doc[c.Name])
		}
		data = append(data, row)
	}
	return &rows{cols: t.Columns, data: data}, cur.Err()
}

func (b *Backend) Insert(ctx context.Context, in *tds.Insert) (tds.Result, error) {
	docs := make([]any, 0, len(in.Rows))
	for _, vals := range in.Rows {
		doc := bson.D{}
		for i, col := range in.Columns {
			if i < len(vals) {
				doc = append(doc, bson.E{Key: col, Value: vals[i]})
			}
		}
		docs = append(docs, doc)
	}
	res, err := b.database().Collection(in.Table).InsertMany(ctx, docs)
	if err != nil {
		return tds.Result{}, err
	}
	return tds.Result{RowsAffected: int64(len(res.InsertedIDs))}, nil
}

func (b *Backend) Update(ctx context.Context, up *tds.Update) (tds.Result, error) {
	set := bson.D{}
	for _, a := range up.Assignments {
		set = append(set, bson.E{Key: a.Column, Value: a.Value})
	}
	res, err := b.database().Collection(up.Table).UpdateMany(ctx, predsToFilter(up.Where),
		bson.D{{Key: "$set", Value: set}})
	if err != nil {
		return tds.Result{}, err
	}
	return tds.Result{RowsAffected: res.ModifiedCount}, nil
}

func (b *Backend) Delete(ctx context.Context, del *tds.Delete) (tds.Result, error) {
	res, err := b.database().Collection(del.Table).DeleteMany(ctx, predsToFilter(del.Where))
	if err != nil {
		return tds.Result{}, err
	}
	return tds.Result{RowsAffected: res.DeletedCount}, nil
}

// CreateTable also writes the catalog doc so Describe reflects the new table (PK/FK seeded separately).
func (b *Backend) CreateTable(ctx context.Context, t *catalog.Table) error {
	if err := b.database().CreateCollection(ctx, t.Name); err != nil {
		return err
	}
	return b.writeCatalog(ctx, *t)
}

func (b *Backend) writeCatalog(ctx context.Context, t catalog.Table) error {
	cols := make([]catalogCol, len(t.Columns))
	for i, c := range t.Columns {
		cols[i] = catalogCol{Name: c.Name, Type: kindToSQL(c.Type)}
	}
	fks := make([]catalogFK, len(t.ForeignKeys))
	for i, fk := range t.ForeignKeys {
		fks[i] = catalogFK{Columns: fk.Columns, RefTable: fk.RefTable, RefColumns: fk.RefColumns}
	}
	doc := catalogDoc{Table: t.Name, Columns: cols, PrimaryKey: t.PrimaryKey, ForeignKeys: fks}
	_, err := b.database().Collection(CatalogCollection).ReplaceOne(ctx,
		bson.D{{Key: "table", Value: t.Name}}, doc, options.Replace().SetUpsert(true))
	return err
}

// AlterTable appends columns to the table's declared catalog entry (the data store is schemaless).
func (b *Backend) AlterTable(ctx context.Context, a *tds.AlterTable) error {
	add := make([]catalogCol, len(a.AddColumns))
	for i, c := range a.AddColumns {
		add[i] = catalogCol{Name: c.Name, Type: kindToSQL(c.Type)}
	}
	if len(add) == 0 {
		return nil
	}
	_, err := b.database().Collection(CatalogCollection).UpdateOne(ctx,
		bson.D{{Key: "table", Value: a.Table}},
		bson.D{{Key: "$push", Value: bson.D{{Key: "columns", Value: bson.D{{Key: "$each", Value: add}}}}}})
	return err
}

// DropTable drops the collection and removes its catalog declaration.
func (b *Backend) DropTable(ctx context.Context, table string) error {
	if err := b.database().Collection(table).Drop(ctx); err != nil {
		return err
	}
	_, err := b.database().Collection(CatalogCollection).DeleteOne(ctx, bson.D{{Key: "table", Value: table}})
	return err
}

func (b *Backend) CreateDatabase(ctx context.Context, name string) error {
	return b.client.Database(name).CreateCollection(ctx, CatalogCollection)
}

func (b *Backend) DropDatabase(ctx context.Context, name string) error {
	return b.client.Database(name).Drop(ctx)
}

func (b *Backend) Databases(ctx context.Context) ([]string, error) {
	return b.client.ListDatabaseNames(ctx, bson.D{})
}

func (b *Backend) DescribeDatabase(ctx context.Context, db string) (catalog.Schema, error) {
	return (&Backend{client: b.client, db: db}).Describe(ctx)
}

func predsToFilter(preds []tds.Predicate) bson.D {
	f := bson.D{}
	for _, p := range preds {
		op := "$eq"
		switch p.Op {
		case tds.OpNe:
			op = "$ne"
		case tds.OpLt:
			op = "$lt"
		case tds.OpLe:
			op = "$lte"
		case tds.OpGt:
			op = "$gt"
		case tds.OpGe:
			op = "$gte"
		}
		f = append(f, bson.E{Key: p.Column, Value: bson.D{{Key: op, Value: p.Value}}})
	}
	return f
}

// sqlType maps a declared SQL-ish type name to the SPI's type kind.
func sqlType(name string) types.Type {
	switch name {
	case "bigint":
		return types.Type{Kind: types.Int64}
	case "int", "integer", "smallint", "tinyint":
		return types.Type{Kind: types.Int32}
	case "bit", "bool", "boolean":
		return types.Type{Kind: types.Bool}
	case "float", "real", "double":
		return types.Type{Kind: types.Float64}
	case "decimal", "numeric", "money":
		return types.Type{Kind: types.Decimal, Precision: 18, Scale: 2}
	case "date", "datetime", "datetime2", "time":
		return types.Type{Kind: types.Time}
	case "uniqueidentifier", "uuid":
		return types.Type{Kind: types.UUID}
	case "varbinary", "binary":
		return types.Type{Kind: types.Bytes}
	}
	return types.Type{Kind: types.String, MaxLen: 255}
}

func kindToSQL(t types.Type) string {
	switch t.Kind {
	case types.Int64:
		return "bigint"
	case types.Int32:
		return "int"
	case types.Bool:
		return "bit"
	case types.Float64:
		return "float"
	case types.Decimal:
		return "decimal"
	case types.Time:
		return "datetime"
	case types.UUID:
		return "uniqueidentifier"
	case types.Bytes:
		return "varbinary"
	}
	return "varchar"
}

func bsonValue(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case int32:
		return int64(x)
	case int64:
		return x
	case float64:
		return x
	case bool:
		return x
	case string:
		return x
	case primitive.ObjectID:
		return x.Hex()
	case primitive.DateTime:
		return x.Time().UTC()
	case time.Time:
		return x.UTC()
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

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package mongo is a MongoDB backend whose catalog is a hardcoded Go literal, never read from the store (see README).
package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

// staticSchema is the entire catalog, fixed in code. It matches the customers/products/orders data
// the demo serves, but is authoritative on its own: Describe returns this regardless of the store.
var staticSchema = catalog.Schema{Tables: []catalog.Table{
	{
		Name: "customers",
		Columns: []catalog.Column{
			{Name: "id", Type: types.Type{Kind: types.Int64}},
			{Name: "name", Type: types.Type{Kind: types.String, MaxLen: 255}},
			{Name: "age", Type: types.Type{Kind: types.Int64}},
		},
		PrimaryKey: []string{"id"},
	},
	{
		Name: "products",
		Columns: []catalog.Column{
			{Name: "id", Type: types.Type{Kind: types.Int64}},
			{Name: "name", Type: types.Type{Kind: types.String, MaxLen: 255}},
			{Name: "price", Type: types.Type{Kind: types.Float64}},
		},
		PrimaryKey: []string{"id"},
	},
	{
		Name: "orders",
		Columns: []catalog.Column{
			{Name: "id", Type: types.Type{Kind: types.Int64}},
			{Name: "customer_id", Type: types.Type{Kind: types.Int64}},
			{Name: "product_id", Type: types.Type{Kind: types.Int64}},
			{Name: "qty", Type: types.Type{Kind: types.Int64}},
		},
		PrimaryKey: []string{"id"},
		ForeignKeys: []catalog.ForeignKey{
			{Columns: []string{"customer_id"}, RefTable: "customers", RefColumns: []string{"id"}},
			{Columns: []string{"product_id"}, RefTable: "products", RefColumns: []string{"id"}},
		},
	},
}}

// Backend serves the hardcoded staticSchema over one Mongo database. The catalog is fixed in code; the
// data is read live from Mongo. It is thin (Scanner) and writable, but intentionally NOT DDL-capable.
type Backend struct {
	client *mongo.Client
	db     string
}

func New(client *mongo.Client, db string) *Backend { return &Backend{client: client, db: db} }

func (b *Backend) database() *mongo.Database { return b.client.Database(b.db) }

func (b *Backend) Capabilities() tds.Caps {
	return tds.Caps{Pushdown: true, Writable: true}
}

// Describe returns the hardcoded catalog without touching Mongo: zero discovery cost, no bootstrap.
func (b *Backend) Describe(ctx context.Context) (catalog.Schema, error) {
	return staticSchema, nil
}

func tableColumns(name string) ([]catalog.Column, bool) {
	for _, t := range staticSchema.Tables {
		if t.Name == name {
			return t.Columns, true
		}
	}
	return nil, false
}

// Scan reads the named collection and projects the hardcoded columns in declared order.
func (b *Backend) Scan(ctx context.Context, q *tds.Query) (tds.Rows, error) {
	cols, ok := tableColumns(q.Table)
	if !ok {
		return nil, fmt.Errorf("mongo: table %q is not in the hardcoded catalog", q.Table)
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
		row := make([]any, len(cols))
		for i, c := range cols {
			row[i] = bsonValue(doc[c.Name])
		}
		data = append(data, row)
	}
	return &rows{cols: cols, data: data}, cur.Err()
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

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package mongo is a haystak-tds-spi backend over MongoDB: collections map to SQL tables,
// document fields to columns (inferred by sampling), and writes/DDL to native Mongo operations.
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

// Backend serves one Mongo database as a SQL catalog. It is a thin (Scanner) backend, so the
// gateway engine applies WHERE/JOIN/GROUP BY/etc, and it is writable, DDL-capable, and multi-database.
type Backend struct {
	client *mongo.Client
	db     string
	sample int64
}

func New(client *mongo.Client, db string) *Backend {
	return &Backend{client: client, db: db, sample: 100}
}

func (b *Backend) database() *mongo.Database { return b.client.Database(b.db) }

func (b *Backend) Capabilities() tds.Caps {
	return tds.Caps{Pushdown: true, Writable: true, DDL: true}
}

// Describe lists collections and samples each to infer its columns (the inferred-catalog model).
func (b *Backend) Describe(ctx context.Context) (catalog.Schema, error) {
	names, err := b.database().ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return catalog.Schema{}, err
	}
	sort.Strings(names)
	var s catalog.Schema
	for _, name := range names {
		cols, err := b.inferColumns(ctx, name)
		if err != nil {
			return catalog.Schema{}, err
		}
		s.Tables = append(s.Tables, catalog.Table{Name: name, Columns: cols, PrimaryKey: []string{"_id"}})
	}
	return s, nil
}

func (b *Backend) inferColumns(ctx context.Context, coll string) ([]catalog.Column, error) {
	cur, err := b.database().Collection(coll).Find(ctx, bson.D{}, options.Find().SetLimit(b.sample))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var order []string
	kind := map[string]types.Kind{}
	for cur.Next(ctx) {
		var doc bson.D
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		for _, e := range doc {
			if _, seen := kind[e.Key]; !seen {
				order = append(order, e.Key)
				kind[e.Key] = bsonKind(e.Value)
			}
		}
	}
	cols := make([]catalog.Column, 0, len(order)+1)
	if k, ok := kind["_id"]; ok {
		cols = append(cols, catalog.Column{Name: "_id", Type: types.Type{Kind: k, MaxLen: 64}})
	}
	for _, name := range order {
		if name == "_id" {
			continue
		}
		cols = append(cols, catalog.Column{Name: name, Type: types.Type{Kind: kind[name], MaxLen: 255}})
	}
	if len(cols) == 0 {
		cols = []catalog.Column{{Name: "_id", Type: types.Type{Kind: types.String, MaxLen: 64}}}
	}
	return cols, nil
}

// Scan reads the whole collection and flattens documents into rows in the inferred column order.
func (b *Backend) Scan(ctx context.Context, q *tds.Query) (tds.Rows, error) {
	cols, err := b.inferColumns(ctx, q.Table)
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

func (b *Backend) CreateTable(ctx context.Context, t *catalog.Table) error {
	return b.database().CreateCollection(ctx, t.Name)
}

func (b *Backend) AlterTable(ctx context.Context, a *tds.AlterTable) error { return nil } // schemaless

func (b *Backend) DropTable(ctx context.Context, table string) error {
	return b.database().Collection(table).Drop(ctx)
}

func (b *Backend) CreateDatabase(ctx context.Context, name string) error {
	return b.client.Database(name).CreateCollection(ctx, "_haystak_init")
}

func (b *Backend) DropDatabase(ctx context.Context, name string) error {
	return b.client.Database(name).Drop(ctx)
}

func (b *Backend) Databases(ctx context.Context) ([]string, error) {
	return b.client.ListDatabaseNames(ctx, bson.D{})
}

func (b *Backend) DescribeDatabase(ctx context.Context, db string) (catalog.Schema, error) {
	return (&Backend{client: b.client, db: db, sample: b.sample}).Describe(ctx)
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

func bsonKind(v any) types.Kind {
	switch v.(type) {
	case int32, int64:
		return types.Int64
	case float64:
		return types.Float64
	case bool:
		return types.Bool
	case primitive.DateTime, time.Time:
		return types.Time
	}
	return types.String
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

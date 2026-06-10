// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package mongo_test

import (
	"context"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	mongobk "github.com/RSKGroup/haystak-tds-spi/examples/mongodb-community-2/mongo"
	"github.com/RSKGroup/haystak-tds-spi/tds/tdstest"
)

// TestConformance drives the engine against a real MongoDB with a declared catalog; skips if mongod is down.
func TestConformance(t *testing.T) {
	client, dbName := dial(t)
	db := client.Database(dbName)
	_ = db.Drop(context.Background())
	seedDeclared(t, db)
	defer db.Drop(context.Background())

	tdstest.RunConformance(t, mongobk.New(client, dbName))
}

// TestDeclaredRelationships asserts the backend surfaces the PK/FK declared in the system collection,
// the property that distinguishes this example from the inferred mongodb-community one.
func TestDeclaredRelationships(t *testing.T) {
	client, dbName := dial(t)
	db := client.Database(dbName)
	_ = db.Drop(context.Background())
	seedDeclared(t, db)
	defer db.Drop(context.Background())

	schema, err := mongobk.New(client, dbName).Describe(context.Background())
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if len(schema.Tables) != 3 {
		t.Fatalf("want 3 declared tables, got %d", len(schema.Tables))
	}
	var orders *struct {
		pk  []string
		fks int
	}
	for _, tbl := range schema.Tables {
		if tbl.Name == "__haystak_catalog" {
			t.Fatalf("system collection must not appear as a SQL table")
		}
		if tbl.Name == "orders" {
			orders = &struct {
				pk  []string
				fks int
			}{tbl.PrimaryKey, len(tbl.ForeignKeys)}
			for _, fk := range tbl.ForeignKeys {
				if fk.RefTable != "customers" && fk.RefTable != "products" {
					t.Fatalf("orders FK references unexpected table %q", fk.RefTable)
				}
			}
		}
	}
	if orders == nil {
		t.Fatal("orders table not declared")
	}
	if len(orders.pk) != 1 || orders.pk[0] != "id" {
		t.Fatalf("orders PK = %v, want [id]", orders.pk)
	}
	if orders.fks != 2 {
		t.Fatalf("orders declared %d FKs, want 2", orders.fks)
	}
}

func dial(t *testing.T) (*mongo.Client, string) {
	t.Helper()
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Skipf("mongo unavailable: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		t.Skipf("mongo ping failed (is mongod running on %s?): %v", uri, err)
	}
	t.Cleanup(func() { client.Disconnect(context.Background()) })
	return client, "haystak_catalog_conformance_test"
}

func seedDeclared(t *testing.T, db *mongo.Database) {
	t.Helper()
	ctx := context.Background()
	insert := func(coll string, docs []any) {
		if _, err := db.Collection(coll).InsertMany(ctx, docs); err != nil {
			t.Fatalf("seed %s: %v", coll, err)
		}
	}
	insert("customers", []any{
		bson.D{{Key: "id", Value: int64(1)}, {Key: "name", Value: "ada"}, {Key: "age", Value: int64(36)}},
		bson.D{{Key: "id", Value: int64(2)}, {Key: "name", Value: "alan"}, {Key: "age", Value: int64(41)}},
	})
	insert("products", []any{
		bson.D{{Key: "id", Value: int64(100)}, {Key: "name", Value: "widget"}, {Key: "price", Value: 9.99}},
	})
	insert("orders", []any{
		bson.D{{Key: "id", Value: int64(10)}, {Key: "customer_id", Value: int64(1)}, {Key: "product_id", Value: int64(100)}, {Key: "qty", Value: int64(2)}},
	})
	insert("__haystak_catalog", []any{
		bson.D{
			{Key: "table", Value: "customers"},
			{Key: "columns", Value: bson.A{
				bson.D{{Key: "name", Value: "id"}, {Key: "type", Value: "bigint"}},
				bson.D{{Key: "name", Value: "name"}, {Key: "type", Value: "varchar"}},
				bson.D{{Key: "name", Value: "age"}, {Key: "type", Value: "bigint"}},
			}},
			{Key: "primary_key", Value: bson.A{"id"}},
			{Key: "foreign_keys", Value: bson.A{}},
		},
		bson.D{
			{Key: "table", Value: "products"},
			{Key: "columns", Value: bson.A{
				bson.D{{Key: "name", Value: "id"}, {Key: "type", Value: "bigint"}},
				bson.D{{Key: "name", Value: "name"}, {Key: "type", Value: "varchar"}},
				bson.D{{Key: "name", Value: "price"}, {Key: "type", Value: "float"}},
			}},
			{Key: "primary_key", Value: bson.A{"id"}},
			{Key: "foreign_keys", Value: bson.A{}},
		},
		bson.D{
			{Key: "table", Value: "orders"},
			{Key: "columns", Value: bson.A{
				bson.D{{Key: "name", Value: "id"}, {Key: "type", Value: "bigint"}},
				bson.D{{Key: "name", Value: "customer_id"}, {Key: "type", Value: "bigint"}},
				bson.D{{Key: "name", Value: "product_id"}, {Key: "type", Value: "bigint"}},
				bson.D{{Key: "name", Value: "qty"}, {Key: "type", Value: "bigint"}},
			}},
			{Key: "primary_key", Value: bson.A{"id"}},
			{Key: "foreign_keys", Value: bson.A{
				bson.D{{Key: "columns", Value: bson.A{"customer_id"}}, {Key: "ref_table", Value: "customers"}, {Key: "ref_columns", Value: bson.A{"id"}}},
				bson.D{{Key: "columns", Value: bson.A{"product_id"}}, {Key: "ref_table", Value: "products"}, {Key: "ref_columns", Value: bson.A{"id"}}},
			}},
		},
	})
}

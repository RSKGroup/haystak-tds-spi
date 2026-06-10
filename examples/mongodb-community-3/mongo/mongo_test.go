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

	mongobk "github.com/RSKGroup/haystak-tds-spi/examples/mongodb-community-3/mongo"
	"github.com/RSKGroup/haystak-tds-spi/tds/tdstest"
)

// TestConformance drives the engine against a hardcoded catalog (no system collection); skips if mongod is down.
func TestConformance(t *testing.T) {
	client, dbName := dial(t)
	db := client.Database(dbName)
	_ = db.Drop(context.Background())
	seedData(t, db)
	defer db.Drop(context.Background())

	tdstest.RunConformance(t, mongobk.New(client, dbName))
}

// TestHardcodedRelationships proves the catalog, including FK edges, comes entirely from code: the
// relationships are present even though NO catalog/system collection was ever written to Mongo.
func TestHardcodedRelationships(t *testing.T) {
	client, dbName := dial(t)
	db := client.Database(dbName)
	_ = db.Drop(context.Background())
	seedData(t, db)
	defer db.Drop(context.Background())

	// Confirm there is no system collection: the catalog cannot be coming from the store.
	names, err := db.ListCollectionNames(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("ListCollectionNames: %v", err)
	}
	for _, n := range names {
		if n == "__haystak_catalog" {
			t.Fatalf("a system collection exists; this test must prove the catalog is code-only")
		}
	}

	schema, err := mongobk.New(client, dbName).Describe(context.Background())
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if len(schema.Tables) != 3 {
		t.Fatalf("want 3 hardcoded tables, got %d", len(schema.Tables))
	}
	var fks int
	var pk []string
	for _, tbl := range schema.Tables {
		if tbl.Name == "orders" {
			pk = tbl.PrimaryKey
			fks = len(tbl.ForeignKeys)
			for _, fk := range tbl.ForeignKeys {
				if fk.RefTable != "customers" && fk.RefTable != "products" {
					t.Fatalf("orders FK references unexpected table %q", fk.RefTable)
				}
			}
		}
	}
	if len(pk) != 1 || pk[0] != "id" {
		t.Fatalf("orders PK = %v, want [id]", pk)
	}
	if fks != 2 {
		t.Fatalf("orders has %d FKs, want 2", fks)
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
	return client, "haystak_hardcoded_conformance_test"
}

func seedData(t *testing.T, db *mongo.Database) {
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
}

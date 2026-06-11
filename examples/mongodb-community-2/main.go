// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Command mongodb-community-2 serves MongoDB over TDS using a declared-catalog backend (see README).
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	mongobk "github.com/RSKGroup/haystak-tds-spi/examples/mongodb-community-2/mongo"
	"github.com/RSKGroup/haystak-tds-spi/server"
)

func main() {
	host := flag.String("host", "", "MongoDB host (url:port or mongodb:// URI); default mongodb://localhost:27017")
	byoDB := flag.String("db", "", "serve an existing MongoDB database; a missing catalog is bootstrapped by inference. blank seeds the demo database")
	flag.Parse()

	uri := mongoURI(*host)
	addr := envOr("ADDR", "127.0.0.1:1433")
	dbName := *byoDB
	if dbName == "" {
		dbName = envOr("MONGO_DB", "haystakcatalog")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, clientOpts(uri))
	cancel()
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	if err := client.Ping(context.Background(), nil); err != nil {
		log.Fatalf("mongo ping %s: %v", uri, err)
	}
	defer client.Disconnect(context.Background())

	bk := mongobk.New(client, dbName)
	// Blank --db seeds our data + hand-authored catalog. A named --db bootstraps a draft catalog by inference when one is absent, then leaves it for the operator to edit.
	if *byoDB == "" {
		if err := seed(context.Background(), client.Database(dbName)); err != nil {
			log.Fatalf("seed: %v", err)
		}
	} else if n, err := bk.EnsureCatalog(context.Background()); err != nil {
		log.Fatalf("bootstrap catalog: %v", err)
	} else if n > 0 {
		log.Printf("bootstrapped %s with %d inferred table(s) — edit it to declare PK/FK relationships", mongobk.CatalogCollection, n)
	}

	gw := &server.Server{Backend: bk, Database: dbName, Logf: log.Printf}
	log.Printf("mongodb-community-2 gateway → mongo %s db=%q (declared catalog), listening on %s", uri, dbName, addr)
	log.Fatal(gw.ListenAndServe(addr))
}

// seed materializes database X: three data collections plus the system collection that DECLARES their
// columns, primary keys, and foreign keys. Idempotent (only fills empty collections).
func seed(ctx context.Context, db *mongo.Database) error {
	if err := seedIfEmpty(ctx, db.Collection("customers"), []any{
		bson.D{{Key: "id", Value: int64(1)}, {Key: "name", Value: "ada"}, {Key: "age", Value: int64(36)}},
		bson.D{{Key: "id", Value: int64(2)}, {Key: "name", Value: "alan"}, {Key: "age", Value: int64(41)}},
		bson.D{{Key: "id", Value: int64(3)}, {Key: "name", Value: "grace"}, {Key: "age", Value: int64(50)}},
	}); err != nil {
		return err
	}
	if err := seedIfEmpty(ctx, db.Collection("products"), []any{
		bson.D{{Key: "id", Value: int64(100)}, {Key: "name", Value: "widget"}, {Key: "price", Value: 9.99}},
		bson.D{{Key: "id", Value: int64(101)}, {Key: "name", Value: "gadget"}, {Key: "price", Value: 19.99}},
	}); err != nil {
		return err
	}
	if err := seedIfEmpty(ctx, db.Collection("orders"), []any{
		bson.D{{Key: "id", Value: int64(10)}, {Key: "customer_id", Value: int64(1)}, {Key: "product_id", Value: int64(100)}, {Key: "qty", Value: int64(2)}},
		bson.D{{Key: "id", Value: int64(11)}, {Key: "customer_id", Value: int64(2)}, {Key: "product_id", Value: int64(101)}, {Key: "qty", Value: int64(1)}},
		bson.D{{Key: "id", Value: int64(12)}, {Key: "customer_id", Value: int64(2)}, {Key: "product_id", Value: int64(100)}, {Key: "qty", Value: int64(5)}},
	}); err != nil {
		return err
	}
	return seedCatalog(ctx, db)
}

// seedCatalog populates the reserved __haystak_catalog collection: one document per table declaring
// its columns, primary key, and foreign-key edges. This is the "system stuff" the backend reads.
func seedCatalog(ctx context.Context, db *mongo.Database) error {
	return seedIfEmpty(ctx, db.Collection("__haystak_catalog"), []any{
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

func seedIfEmpty(ctx context.Context, coll *mongo.Collection, docs []any) error {
	n, err := coll.CountDocuments(ctx, bson.D{})
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err = coll.InsertMany(ctx, docs)
	return err
}

// clientOpts builds Mongo client options from MONGO_URI, plus optional credential env vars
// (MONGO_USER/MONGO_PASS/MONGO_AUTHDB) for installs that require authentication.
func clientOpts(uri string) *options.ClientOptions {
	o := options.Client().ApplyURI(uri)
	if user := os.Getenv("MONGO_USER"); user != "" {
		o.SetAuth(options.Credential{
			Username:   user,
			Password:   os.Getenv("MONGO_PASS"),
			AuthSource: envOr("MONGO_AUTHDB", "admin"),
		})
	}
	return o
}

// mongoURI resolves the effective connection URI: --host, else MONGO_URI, else the localhost default.
// A bare host:port gets the mongodb:// scheme; a full mongodb:// or mongodb+srv:// URI is used as-is.
func mongoURI(host string) string {
	if host == "" {
		host = os.Getenv("MONGO_URI")
	}
	if host == "" {
		return "mongodb://localhost:27017"
	}
	if strings.Contains(host, "://") {
		return host
	}
	return "mongodb://" + host
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

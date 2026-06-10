// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Command mongodb-community-3 serves MongoDB over TDS using a hardcoded-catalog backend (see README).
package main

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	mongobk "github.com/RSKGroup/haystak-tds-spi/examples/mongodb-community-3/mongo"
	"github.com/RSKGroup/haystak-tds-spi/server"
)

func main() {
	uri := envOr("MONGO_URI", "mongodb://localhost:27017")
	dbName := envOr("MONGO_DB", "haystakcatalog")
	addr := envOr("ADDR", "127.0.0.1:1433")

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

	if err := seed(context.Background(), client.Database(dbName)); err != nil {
		log.Fatalf("seed: %v", err)
	}

	gw := &server.Server{Backend: mongobk.New(client, dbName), Database: dbName, Logf: log.Printf}
	log.Printf("mongodb-community-3 gateway → mongo %s db=%q (hardcoded catalog), listening on %s", uri, dbName, addr)
	log.Fatal(gw.ListenAndServe(addr))
}

// seed materializes only the data collections. There is no system collection; the catalog is in code. Idempotent.
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
	return seedIfEmpty(ctx, db.Collection("orders"), []any{
		bson.D{{Key: "id", Value: int64(10)}, {Key: "customer_id", Value: int64(1)}, {Key: "product_id", Value: int64(100)}, {Key: "qty", Value: int64(2)}},
		bson.D{{Key: "id", Value: int64(11)}, {Key: "customer_id", Value: int64(2)}, {Key: "product_id", Value: int64(101)}, {Key: "qty", Value: int64(1)}},
		bson.D{{Key: "id", Value: int64(12)}, {Key: "customer_id", Value: int64(2)}, {Key: "product_id", Value: int64(100)}, {Key: "qty", Value: int64(5)}},
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

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Command mongodb-community runs a haystak-tds-spi gateway over a local MongoDB. It seeds a demo
// database (using Mongo's dynamic create-db/collection/insert), then serves it on the TDS wire.
package main

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	mongobk "github.com/RSKGroup/haystak-tds-spi/examples/mongodb-community/mongo"
	"github.com/RSKGroup/haystak-tds-spi/server"
)

func main() {
	uri := envOr("MONGO_URI", "mongodb://localhost:27017")
	dbName := envOr("MONGO_DB", "haystakdemo")
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
	log.Printf("mongodb-community gateway → mongo %s db=%q, listening on %s", uri, dbName, addr)
	log.Fatal(gw.ListenAndServe(addr))
}

// seed materializes a demo database with two collections if they are empty, using Mongo's
// dynamic create-collection + insert so there is something to query out of the box.
func seed(ctx context.Context, db *mongo.Database) error {
	if err := seedIfEmpty(ctx, db.Collection("users"), []any{
		bson.D{{Key: "id", Value: int64(1)}, {Key: "name", Value: "ada"}, {Key: "age", Value: int64(36)}},
		bson.D{{Key: "id", Value: int64(2)}, {Key: "name", Value: "alan"}, {Key: "age", Value: int64(41)}},
		bson.D{{Key: "id", Value: int64(3)}, {Key: "name", Value: "grace"}, {Key: "age", Value: int64(50)}},
	}); err != nil {
		return err
	}
	return seedIfEmpty(ctx, db.Collection("orders"), []any{
		bson.D{{Key: "id", Value: int64(10)}, {Key: "user_id", Value: int64(1)}, {Key: "amount", Value: int64(100)}},
		bson.D{{Key: "id", Value: int64(11)}, {Key: "user_id", Value: int64(2)}, {Key: "amount", Value: int64(200)}},
		bson.D{{Key: "id", Value: int64(12)}, {Key: "user_id", Value: int64(2)}, {Key: "amount", Value: int64(50)}},
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
// (MONGO_USER/MONGO_PASS/MONGO_AUTHDB) for installs that require authentication. Credentials may
// also be embedded directly in MONGO_URI; ApplyURI parses standard connection-string auth.
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

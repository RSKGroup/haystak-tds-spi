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

	mongobk "github.com/RSKGroup/haystak-tds-spi/examples/mongodb-community/mongo"
	"github.com/RSKGroup/haystak-tds-spi/tds/tdstest"
)

// TestConformance runs the SPI conformance harness against a real MongoDB. It seeds a temporary
// database, exercises the backend through the engine, and drops it. Skips if mongod is unreachable.
func TestConformance(t *testing.T) {
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
	defer client.Disconnect(context.Background())

	dbName := "haystak_conformance_test"
	db := client.Database(dbName)
	_ = db.Drop(ctx)
	if _, err := db.Collection("widgets").InsertMany(ctx, []any{
		bson.D{{Key: "id", Value: int64(1)}, {Key: "name", Value: "a"}},
		bson.D{{Key: "id", Value: int64(2)}, {Key: "name", Value: "b"}},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	defer db.Drop(context.Background())

	tdstest.RunConformance(t, mongobk.New(client, dbName))
}

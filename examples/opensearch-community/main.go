// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Command opensearch-community runs a haystak-tds-spi gateway over OpenSearch. It seeds a pair of demo
// indices (unless pointed at your own with --db), then serves them on the TDS wire.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	osgo "github.com/opensearch-project/opensearch-go/v2"

	osbk "github.com/RSKGroup/haystak-tds-spi/examples/opensearch-community/opensearch"
	"github.com/RSKGroup/haystak-tds-spi/server"
)

var demoIndices = []string{"users", "orders"}

func main() {
	host := flag.String("host", "", "OpenSearch host (url:port); default http://localhost:9201")
	db := flag.String("db", "", "index pattern to serve (BYO data, no seeding), e.g. 'sales-*'; blank seeds + serves the demo indices")
	flag.Parse()

	addr := envOr("ADDR", "127.0.0.1:1433")
	url := osURL(*host)
	client, err := osgo.NewClient(osConfig(url))
	if err != nil {
		log.Fatalf("opensearch client: %v", err)
	}
	info, err := client.Info()
	if err != nil {
		log.Fatalf("opensearch ping %s: %v", url, err)
	}
	info.Body.Close()

	serve := *db
	if serve == "" {
		if err := seed(context.Background(), client); err != nil {
			log.Fatalf("seed: %v", err)
		}
		serve = strings.Join(demoIndices, ",")
	}

	gw := &server.Server{Backend: osbk.New(client, serve), Database: "opensearch", Logf: log.Printf}
	log.Printf("opensearch-community gateway → opensearch %s indices=%q (inferred catalog), listening on %s", url, serve, addr)
	log.Fatal(gw.ListenAndServe(addr))
}

// seed materializes two demo indices if they are empty, creating each explicitly (so the demo also works
// on clusters with action.auto_create_index disabled) and indexing the demo documents.
func seed(ctx context.Context, client *osgo.Client) error {
	if err := seedIfEmpty(ctx, client, "users", []map[string]any{
		{"id": 1, "name": "ada", "age": 36},
		{"id": 2, "name": "alan", "age": 41},
		{"id": 3, "name": "grace", "age": 50},
	}); err != nil {
		return err
	}
	return seedIfEmpty(ctx, client, "orders", []map[string]any{
		{"id": 10, "user_id": 1, "amount": 100},
		{"id": 11, "user_id": 2, "amount": 200},
		{"id": 12, "user_id": 2, "amount": 50},
	})
}

func seedIfEmpty(ctx context.Context, client *osgo.Client, index string, docs []map[string]any) error {
	if count(ctx, client, index) > 0 {
		return nil
	}
	if err := ensureIndex(ctx, client, index); err != nil {
		return err
	}
	for _, d := range docs {
		body, _ := json.Marshal(d)
		res, err := client.Index(index, bytes.NewReader(body), client.Index.WithContext(ctx), client.Index.WithRefresh("true"))
		if err != nil {
			return err
		}
		res.Body.Close()
		if res.IsError() {
			return fmt.Errorf("seed %s: %s", index, res.String())
		}
	}
	return nil
}

func count(ctx context.Context, client *osgo.Client, index string) int {
	cnt, err := client.Count(client.Count.WithContext(ctx), client.Count.WithIndex(index))
	if err != nil {
		return 0
	}
	defer cnt.Body.Close()
	if cnt.IsError() {
		return 0
	}
	var r struct {
		Count int `json:"count"`
	}
	json.NewDecoder(cnt.Body).Decode(&r)
	return r.Count
}

// ensureIndex creates the index if it does not exist; clusters with action.auto_create_index disabled
// reject indexing into a missing index, so the demo creates it explicitly.
func ensureIndex(ctx context.Context, client *osgo.Client, index string) error {
	r, err := client.Indices.Exists([]string{index}, client.Indices.Exists.WithContext(ctx))
	if err != nil {
		return err
	}
	r.Body.Close()
	if r.StatusCode == 200 {
		return nil
	}
	c, err := client.Indices.Create(index, client.Indices.Create.WithContext(ctx))
	if err != nil {
		return err
	}
	defer c.Body.Close()
	if c.IsError() {
		return fmt.Errorf("create index %s: %s", index, c.String())
	}
	return nil
}

// osConfig builds the OpenSearch client config for url, plus optional OS_USER/OS_PASS for secured clusters.
func osConfig(url string) osgo.Config {
	cfg := osgo.Config{Addresses: []string{url}}
	if u := os.Getenv("OS_USER"); u != "" {
		cfg.Username = u
		cfg.Password = os.Getenv("OS_PASS")
	}
	return cfg
}

// osURL resolves the effective OpenSearch URL: --host, else OS_URL, else the localhost default (:9201).
func osURL(host string) string {
	if host == "" {
		host = os.Getenv("OS_URL")
	}
	if host == "" {
		return "http://localhost:9201"
	}
	if strings.Contains(host, "://") {
		return host
	}
	return "http://" + host
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

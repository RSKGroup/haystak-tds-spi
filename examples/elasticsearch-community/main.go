// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Command elasticsearch-community runs a haystak-tds-spi gateway over Elasticsearch. It seeds a pair
// of demo indices (unless pointed at your own with --index), then serves them on the TDS wire.
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

	"github.com/elastic/go-elasticsearch/v9"

	esbk "github.com/RSKGroup/haystak-tds-spi/examples/elasticsearch-community/es"
	"github.com/RSKGroup/haystak-tds-spi/server"
)

var demoIndices = []string{"users", "orders"}

func main() {
	host := flag.String("host", "", "Elasticsearch host (url:port); default http://localhost:9200")
	db := flag.String("db", "", "index pattern to serve (BYO data, no seeding), e.g. 'sales-*'; blank seeds + serves the demo indices")
	flag.Parse()

	addr := envOr("ADDR", "127.0.0.1:1433")
	url := esURL(*host)
	es, err := elasticsearch.NewClient(esConfig(url))
	if err != nil {
		log.Fatalf("es client: %v", err)
	}
	info, err := es.Info()
	if err != nil {
		log.Fatalf("es ping %s: %v", url, err)
	}
	info.Body.Close()

	serve := *db
	if serve == "" {
		if err := seed(context.Background(), es); err != nil {
			log.Fatalf("seed: %v", err)
		}
		serve = strings.Join(demoIndices, ",")
	}

	gw := &server.Server{Backend: esbk.New(es, serve), Database: "elasticsearch", Logf: log.Printf}
	log.Printf("elasticsearch-community gateway → es %s indices=%q (inferred catalog), listening on %s", url, serve, addr)
	log.Fatal(gw.ListenAndServe(addr))
}

// seed materializes two demo indices if they are empty, using ES's dynamic create-index + index so
// there is something to query out of the box.
func seed(ctx context.Context, es *elasticsearch.Client) error {
	if err := seedIfEmpty(ctx, es, "users", []map[string]any{
		{"id": 1, "name": "ada", "age": 36},
		{"id": 2, "name": "alan", "age": 41},
		{"id": 3, "name": "grace", "age": 50},
	}); err != nil {
		return err
	}
	return seedIfEmpty(ctx, es, "orders", []map[string]any{
		{"id": 10, "user_id": 1, "amount": 100},
		{"id": 11, "user_id": 2, "amount": 200},
		{"id": 12, "user_id": 2, "amount": 50},
	})
}

func seedIfEmpty(ctx context.Context, es *elasticsearch.Client, index string, docs []map[string]any) error {
	if count(ctx, es, index) > 0 {
		return nil
	}
	if err := ensureIndex(ctx, es, index); err != nil {
		return err
	}
	for _, d := range docs {
		body, _ := json.Marshal(d)
		res, err := es.Index(index, bytes.NewReader(body), es.Index.WithContext(ctx), es.Index.WithRefresh("true"))
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

func count(ctx context.Context, es *elasticsearch.Client, index string) int {
	cnt, err := es.Count(es.Count.WithContext(ctx), es.Count.WithIndex(index))
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
// (a common production guard) reject indexing into a missing index, so the demo creates it explicitly.
func ensureIndex(ctx context.Context, es *elasticsearch.Client, index string) error {
	r, err := es.Indices.Exists([]string{index}, es.Indices.Exists.WithContext(ctx))
	if err != nil {
		return err
	}
	r.Body.Close()
	if r.StatusCode == 200 {
		return nil
	}
	c, err := es.Indices.Create(index, es.Indices.Create.WithContext(ctx))
	if err != nil {
		return err
	}
	defer c.Body.Close()
	if c.IsError() {
		return fmt.Errorf("create index %s: %s", index, c.String())
	}
	return nil
}

// esConfig builds the ES client config for url, plus optional ES_USER/ES_PASS for secured clusters.
func esConfig(url string) elasticsearch.Config {
	cfg := elasticsearch.Config{Addresses: []string{url}}
	if u := os.Getenv("ES_USER"); u != "" {
		cfg.Username = u
		cfg.Password = os.Getenv("ES_PASS")
	}
	return cfg
}

// esURL resolves the effective Elasticsearch URL: --host, else ES_URL, else the localhost default.
func esURL(host string) string {
	if host == "" {
		host = os.Getenv("ES_URL")
	}
	if host == "" {
		return "http://localhost:9200"
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

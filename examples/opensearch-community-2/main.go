// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Command opensearch-community-2 serves OpenSearch over TDS using a hybrid declared catalog: columns/types
// come from each index's native _mapping, primary/foreign keys from haystak_catalog (see README).
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

	osbk "github.com/RSKGroup/haystak-tds-spi/examples/opensearch-community-2/opensearch"
	"github.com/RSKGroup/haystak-tds-spi/server"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

func main() {
	host := flag.String("host", "", "OpenSearch host (url:port); default http://localhost:9201")
	db := flag.String("db", "", "index pattern to serve; a missing catalog is bootstrapped from mappings. blank seeds the demo indices")
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

	bk := osbk.New(client)
	// Blank --db seeds demo tables (mappings + a hand-authored catalog). A named --db bootstraps the
	// keys from mappings when no catalog exists, then leaves it for the operator to declare FKs.
	if *db == "" {
		if err := seed(context.Background(), client, bk); err != nil {
			log.Fatalf("seed: %v", err)
		}
	} else if n, err := bk.EnsureCatalog(context.Background(), *db); err != nil {
		log.Fatalf("bootstrap catalog: %v", err)
	} else if n > 0 {
		log.Printf("bootstrapped %s with %d table(s) — edit it to declare PK/FK relationships", osbk.CatalogIndex, n)
	}

	gw := &server.Server{Backend: bk, Database: "opensearch", Logf: log.Printf}
	log.Printf("opensearch-community-2 gateway → opensearch %s (declared catalog: mappings + %s), listening on %s", url, osbk.CatalogIndex, addr)
	log.Fatal(gw.ListenAndServe(addr))
}

// seed creates three demo indices with explicit mappings plus the catalog declaring their PKs and the
// two FK edges from orders, then indexes the data. Idempotent (skips an index that already exists).
func seed(ctx context.Context, client *osgo.Client, bk *osbk.Backend) error {
	tables := []catalog.Table{
		{Name: "customers", PrimaryKey: []string{"id"}, Columns: []catalog.Column{
			{Name: "id", Type: types.Type{Kind: types.Int64}},
			{Name: "name", Type: types.Type{Kind: types.String}},
			{Name: "age", Type: types.Type{Kind: types.Int64}},
		}},
		{Name: "products", PrimaryKey: []string{"id"}, Columns: []catalog.Column{
			{Name: "id", Type: types.Type{Kind: types.Int64}},
			{Name: "name", Type: types.Type{Kind: types.String}},
			{Name: "price", Type: types.Type{Kind: types.Float64}},
		}},
		{Name: "orders", PrimaryKey: []string{"id"}, ForeignKeys: []catalog.ForeignKey{
			{Columns: []string{"customer_id"}, RefTable: "customers", RefColumns: []string{"id"}},
			{Columns: []string{"product_id"}, RefTable: "products", RefColumns: []string{"id"}},
		}, Columns: []catalog.Column{
			{Name: "id", Type: types.Type{Kind: types.Int64}},
			{Name: "customer_id", Type: types.Type{Kind: types.Int64}},
			{Name: "product_id", Type: types.Type{Kind: types.Int64}},
			{Name: "qty", Type: types.Type{Kind: types.Int64}},
		}},
	}
	data := map[string][]map[string]any{
		"customers": {{"id": 1, "name": "ada", "age": 36}, {"id": 2, "name": "alan", "age": 41}, {"id": 3, "name": "grace", "age": 50}},
		"products":  {{"id": 100, "name": "widget", "price": 9.99}, {"id": 101, "name": "gadget", "price": 19.99}},
		"orders":    {{"id": 10, "customer_id": 1, "product_id": 100, "qty": 2}, {"id": 11, "customer_id": 2, "product_id": 101, "qty": 1}, {"id": 12, "customer_id": 2, "product_id": 100, "qty": 5}},
	}
	for _, t := range tables {
		if indexExists(ctx, client, t.Name) {
			continue
		}
		if err := bk.CreateTable(ctx, &t); err != nil {
			return err
		}
		for _, d := range data[t.Name] {
			if err := indexDoc(ctx, client, t.Name, d); err != nil {
				return err
			}
		}
	}
	return nil
}

func indexExists(ctx context.Context, client *osgo.Client, index string) bool {
	res, err := client.Indices.Exists([]string{index}, client.Indices.Exists.WithContext(ctx))
	if err != nil {
		return false
	}
	res.Body.Close()
	return res.StatusCode == 200
}

func indexDoc(ctx context.Context, client *osgo.Client, index string, d map[string]any) error {
	body, _ := json.Marshal(d)
	res, err := client.Index(index, bytes.NewReader(body), client.Index.WithContext(ctx), client.Index.WithRefresh("true"))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("index %s: %s", index, res.String())
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

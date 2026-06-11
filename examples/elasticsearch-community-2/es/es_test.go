// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package es_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/elastic/go-elasticsearch/v9"

	esbk "github.com/RSKGroup/haystak-tds-spi/examples/elasticsearch-community-2/es"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/tdstest"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

// TestConformance drives the conformance harness against a real ES (declared model); skips if ES is down.
func TestConformance(t *testing.T) {
	url := os.Getenv("ES_URL")
	if url == "" {
		url = "http://localhost:9200"
	}
	es, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{url}})
	if err != nil {
		t.Skipf("es client: %v", err)
	}
	info, err := es.Info()
	if err != nil {
		t.Skipf("es unavailable (is it running on %s?): %v", url, err)
	}
	info.Body.Close()

	bk := esbk.New(es)
	index := "haystak_conformance_test_2"
	bk.DropTable(t.Context(), index)
	defer bk.DropTable(t.Context(), index)
	tbl := catalog.Table{Name: index, PrimaryKey: []string{"id"}, Columns: []catalog.Column{
		{Name: "id", Type: types.Type{Kind: types.Int64}},
		{Name: "name", Type: types.Type{Kind: types.String}},
	}}
	if err := bk.CreateTable(t.Context(), &tbl); err != nil {
		t.Fatalf("create table: %v", err)
	}
	for _, d := range []map[string]any{{"id": 1, "name": "a"}, {"id": 2, "name": "b"}} {
		body, _ := json.Marshal(d)
		res, err := es.Index(index, bytes.NewReader(body), es.Index.WithRefresh("true"))
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		res.Body.Close()
	}

	tdstest.RunConformance(t, bk)
}

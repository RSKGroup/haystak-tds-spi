// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package es_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/elastic/go-elasticsearch/v9"

	esbk "github.com/RSKGroup/haystak-tds-spi/examples/elasticsearch-community/es"
	"github.com/RSKGroup/haystak-tds-spi/tds/tdstest"
)

// TestConformance runs the SPI conformance harness against a real Elasticsearch. It seeds a temporary
// index, exercises the backend through the engine, and drops it. Skips if ES is unreachable.
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

	index := "haystak_conformance_test"
	es.Indices.Delete([]string{index})
	defer es.Indices.Delete([]string{index})
	if c, err := es.Indices.Create(index); err != nil {
		t.Fatalf("create index: %v", err)
	} else {
		c.Body.Close()
	}
	for _, d := range []map[string]any{
		{"id": 1, "name": "a"},
		{"id": 2, "name": "b"},
	} {
		body, _ := json.Marshal(d)
		res, err := es.Index(index, bytes.NewReader(body), es.Index.WithRefresh("true"))
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		res.Body.Close()
	}

	tdstest.RunConformance(t, esbk.New(es, index))
}

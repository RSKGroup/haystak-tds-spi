// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package opensearch_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	osgo "github.com/opensearch-project/opensearch-go/v2"

	osbk "github.com/RSKGroup/haystak-tds-spi/examples/opensearch-community/opensearch"
	"github.com/RSKGroup/haystak-tds-spi/tds/tdstest"
)

// TestConformance runs the SPI conformance harness against a real OpenSearch. It seeds a temporary
// index, exercises the backend through the engine, and drops it. Skips if OpenSearch is unreachable.
func TestConformance(t *testing.T) {
	url := os.Getenv("OS_URL")
	if url == "" {
		url = "http://localhost:9201"
	}
	client, err := osgo.NewClient(osgo.Config{Addresses: []string{url}})
	if err != nil {
		t.Skipf("opensearch client: %v", err)
	}
	info, err := client.Info()
	if err != nil {
		t.Skipf("opensearch unavailable (is it running on %s?): %v", url, err)
	}
	info.Body.Close()

	index := "haystak_conformance_test"
	client.Indices.Delete([]string{index})
	defer client.Indices.Delete([]string{index})
	if c, err := client.Indices.Create(index); err != nil {
		t.Fatalf("create index: %v", err)
	} else {
		c.Body.Close()
	}
	for _, d := range []map[string]any{
		{"id": 1, "name": "a"},
		{"id": 2, "name": "b"},
	} {
		body, _ := json.Marshal(d)
		res, err := client.Index(index, bytes.NewReader(body), client.Index.WithRefresh("true"))
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		res.Body.Close()
	}

	tdstest.RunConformance(t, osbk.New(client, index))
}

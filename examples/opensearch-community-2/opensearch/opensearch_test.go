// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package opensearch_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	osgo "github.com/opensearch-project/opensearch-go/v2"

	osbk "github.com/RSKGroup/haystak-tds-spi/examples/opensearch-community-2/opensearch"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/tdstest"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

// TestConformance drives the conformance harness against a real OpenSearch (declared model); skips if it is down.
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

	bk := osbk.New(client)
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
		res, err := client.Index(index, bytes.NewReader(body), client.Index.WithRefresh("true"))
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		res.Body.Close()
	}

	tdstest.RunConformance(t, bk)
}

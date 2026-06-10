// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package inmem_test

import (
	"context"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/tdstest"
)

func TestConformance(t *testing.T) {
	tdstest.RunConformance(t, inmem.New())
}

func TestScanReturnsTable(t *testing.T) {
	rs, err := inmem.New().Scan(context.Background(), &tds.Query{Table: "users"})
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
	n := 0
	for rs.Next() {
		if _, err := rs.Values(); err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 2 {
		t.Fatalf("rows = %d, want 2", n)
	}
}

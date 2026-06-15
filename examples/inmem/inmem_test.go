// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package inmem_test

import (
	"context"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/tdstest"
)

func TestConformance(t *testing.T) {
	tdstest.RunConformance(t, inmem.New())
}

func TestSurface(t *testing.T) {
	tdstest.RunSurface(t, inmem.New())
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

// TestPopulatedSeams proves the populated side of the seam-first catalog views: a seeded table type
// surfaces through sys.table_types, and an authenticated Principal is reflected in sys.database_principals.
func TestPopulatedSeams(t *testing.T) {
	b := inmem.New()
	if !rowMatch(t, context.Background(), b, "SELECT name FROM sys.table_types", "OrderLineType") {
		t.Errorf("sys.table_types did not surface the seeded OrderLineType")
	}
	ctx := tds.WithPrincipal(context.Background(), tds.Principal{Username: "alice", Roles: []string{"sales"}})
	if !rowMatch(t, ctx, b, "SELECT name FROM sys.database_principals", "alice") {
		t.Errorf("sys.database_principals did not reflect the live principal alice")
	}
	if !rowMatch(t, ctx, b, "SELECT name FROM sys.database_principals", "sales") {
		t.Errorf("sys.database_principals did not reflect the principal role sales")
	}
}

// TestEmptyDegrade proves the empty-degrade side: a backend with no tables still returns the correct
// catalog columns and zero rows, rather than erroring.
func TestEmptyDegrade(t *testing.T) {
	b := emptyBackend{}
	for _, sql := range []string{
		"SELECT object_id, name, column_id FROM sys.columns",
		"SELECT TABLE_NAME, TABLE_TYPE FROM INFORMATION_SCHEMA.TABLES",
	} {
		rs, err := engine.Query(context.Background(), b, sql)
		if err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		if len(rs.Columns()) == 0 {
			t.Errorf("%s: no columns on empty backend", sql)
		}
		n := 0
		for rs.Next() {
			n++
		}
		rs.Close()
		if n != 0 {
			t.Errorf("%s: got %d rows on empty backend, want 0", sql, n)
		}
	}
}

func rowMatch(t *testing.T, ctx context.Context, b tds.Backend, sql, want string) bool {
	t.Helper()
	rs, err := engine.Query(ctx, b, sql)
	if err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	defer rs.Close()
	for rs.Next() {
		v, err := rs.Values()
		if err != nil {
			t.Fatal(err)
		}
		if len(v) > 0 && v[0] == want {
			return true
		}
	}
	return false
}

// emptyBackend is a queryable backend with no tables, for the empty-degrade path.
type emptyBackend struct{}

func (emptyBackend) Describe(context.Context) (catalog.Schema, error) { return catalog.Schema{}, nil }
func (emptyBackend) Capabilities() tds.Caps                           { return tds.Caps{Pushdown: true} }
func (emptyBackend) Scan(context.Context, *tds.Query) (tds.Rows, error) {
	return nil, tds.ErrUnsupported
}

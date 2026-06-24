// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
)

// countingBackend wraps the inmem backend and counts catalog-introspection calls so a test can assert
// the per-statement memo collapses repeated derivation.
type countingBackend struct {
	*inmem.Backend
	describe int
	routines int
}

func (c *countingBackend) Describe(ctx context.Context) (catalog.Schema, error) {
	c.describe++
	return c.Backend.Describe(ctx)
}

func (c *countingBackend) ListRoutines(ctx context.Context, database string) ([]*tds.Routine, error) {
	c.routines++
	return c.Backend.ListRoutines(ctx, database)
}

// A correlated APPLY whose right side reads the system catalog once per outer row must NOT re-derive the
// whole schema per row -- the per-statement memo collapses it to a single introspection. This is exactly
// the pattern SSMS Object Explorer issues; without the memo it is O(outer rows) catalog derivations.
func TestApplyCatalogIntrospectsOnce(t *testing.T) {
	b := &countingBackend{Backend: inmem.New()}
	const q = "SELECT t.name, x.cols FROM sys.tables t CROSS APPLY " +
		"(SELECT COUNT(*) AS cols FROM sys.columns c WHERE c.object_id = t.object_id) x"
	rs, err := engine.Query(context.Background(), b, q)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	got := collect(t, rs)
	if len(got) == 0 {
		t.Fatalf("expected one row per table, got none")
	}
	if b.describe > 1 {
		t.Errorf("Describe called %d times; want 1 -- memo must collapse per-row catalog derivation, not scale with %d outer rows", b.describe, len(got))
	}
	if b.routines > 1 {
		t.Errorf("ListRoutines called %d times; want <=1 (memoized per statement)", b.routines)
	}
}

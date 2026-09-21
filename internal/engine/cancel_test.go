// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"strings"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
)

// A cancelled caller must stop the statement. Before this, exec, engine and tsql held ZERO
// ctx.Done/ctx.Err checks between them: one abandoned join ran 25m24s with the client long gone,
// because every table in the plan was scanned to completion and nothing ever asked.
func TestJoinStopsWhenTheCallerCancels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := engine.Query(ctx, inmem.New(),
		"SELECT u.name FROM users u JOIN users v ON u.id = v.id")
	if err == nil {
		t.Fatal("a cancelled context still ran the join to completion")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("error %q does not say the query was cancelled", err)
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("error %q loses the underlying cause", err)
	}
}

// The check must not fire on a live context, or every join breaks.
func TestJoinRunsWithALiveContext(t *testing.T) {
	rs, err := engine.Query(context.Background(), inmem.New(),
		"SELECT u.name FROM users u JOIN users v ON u.id = v.id")
	if err != nil {
		t.Fatalf("a live context was refused: %v", err)
	}
	if got := collect(t, rs); len(got) == 0 {
		t.Fatal("join returned no rows on a live context")
	}
}

// APPLY re-runs its right side per OUTER row, so it is the other loop an abandoned query burns in.
func TestApplyStopsWhenTheCallerCancels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := engine.Query(ctx, inmem.New(),
		"SELECT t.name, x.cols FROM sys.tables t CROSS APPLY "+
			"(SELECT COUNT(*) AS cols FROM sys.columns c WHERE c.object_id = t.object_id) x")
	if err == nil {
		t.Fatal("a cancelled context still ran the apply to completion")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("error %q does not say the query was cancelled", err)
	}
}

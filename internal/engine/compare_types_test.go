// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
)

// A predicate across types used to fall through to a text comparison and return a CONFIDENT wrong
// answer. `WHERE name > 1` matched 2 of the 2 named rows, because "alan" sorts after "1" as text.
// It is now UNKNOWN - the same handling NULL gets - so the rows are dropped rather than invented.
func TestTextColumnComparedToANumberMatchesNothing(t *testing.T) {
	rs, err := engine.Query(context.Background(), inmem.New(), "SELECT name FROM users WHERE name > 1")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if got := collect(t, rs); len(got) != 0 {
		t.Errorf("got %d rows, want 0: a text column compared to a number cannot be ordered", len(got))
	}
}

// The conversion must still HAPPEN where it is meaningful, or the fix is just a way to return
// nothing: a numeric-looking string compares as a number, not as text.
func TestNumericTextComparesNumerically(t *testing.T) {
	rs, err := engine.Query(context.Background(), inmem.New(), "SELECT name FROM users WHERE id > '1'")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if got := collect(t, rs); len(got) == 0 {
		t.Error("id > '1' matched nothing; the numeric string was not converted")
	}
}

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
)

// orders.amount per user_id: user 1 -> {100}, user 2 -> {200, 50}.
func TestStringAggGrouped(t *testing.T) {
	rows := qry(t, "SELECT user_id, STRING_AGG(amount, '-') AS s FROM orders GROUP BY user_id ORDER BY user_id")
	if len(rows) != 2 {
		t.Fatalf("got %d groups, want 2", len(rows))
	}
	if cell(rows[0][1]) != "100" {
		t.Errorf("user 1 STRING_AGG = %v, want 100", rows[0][1])
	}
	if cell(rows[1][1]) != "200-50" {
		t.Errorf("user 2 STRING_AGG = %v, want 200-50", rows[1][1])
	}
}

func TestStringAggSeparatorRequired(t *testing.T) {
	if _, err := engine.Query(context.Background(), inmem.New(), "SELECT STRING_AGG(amount) FROM orders"); err == nil {
		t.Fatal("STRING_AGG without a separator should be a parse error")
	}
}

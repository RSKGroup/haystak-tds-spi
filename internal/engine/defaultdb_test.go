// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/internal/tsql"
)

func TestApplyDefaultDBRecurses(t *testing.T) {
	q, err := tsql.Parse("SELECT a FROM t WHERE id IN (SELECT id FROM u WHERE x IN (SELECT id FROM v))")
	if err != nil {
		t.Fatal(err)
	}
	applyDefaultDB(q, "db1")
	if q.Database != "db1" {
		t.Fatalf("outer Database = %q, want db1", q.Database)
	}
	sub := q.Where.Pred.Sub
	if sub == nil || sub.Database != "db1" {
		t.Fatalf("subquery Database not qualified: %+v", sub)
	}
	if sub2 := sub.Where.Pred.Sub; sub2 == nil || sub2.Database != "db1" {
		t.Fatal("nested subquery Database not qualified")
	}
}

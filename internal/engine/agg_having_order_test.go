// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
)

// orders: user_id 1 has 1 row, user_id 2 has 2 rows.

func TestHavingCountStar(t *testing.T) {
	rs, err := engine.Query(context.Background(), inmem.New(),
		"SELECT user_id, COUNT(*) AS n FROM orders GROUP BY user_id HAVING COUNT(*) > 1")
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, rs)
	if len(got) != 1 || got[0][0] != int64(2) || got[0][1] != int64(2) {
		t.Fatalf("HAVING COUNT(*) > 1 got %v, want [[2 2]]", got)
	}
}

func TestHavingCountColumn(t *testing.T) {
	rs, err := engine.Query(context.Background(), inmem.New(),
		"SELECT user_id, COUNT(*) AS n FROM orders GROUP BY user_id HAVING COUNT(user_id) > 0")
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, rs)
	if len(got) != 2 {
		t.Fatalf("HAVING COUNT(user_id) > 0 got %v, want 2 rows", got)
	}
}

func TestOrderByCountStar(t *testing.T) {
	rs, err := engine.Query(context.Background(), inmem.New(),
		"SELECT user_id, COUNT(*) AS n FROM orders GROUP BY user_id ORDER BY COUNT(*) DESC")
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, rs)
	if len(got) != 2 || got[0][0] != int64(2) || got[1][0] != int64(1) {
		t.Fatalf("ORDER BY COUNT(*) DESC got %v, want user_id 2 then 1", got)
	}
}

func TestOrderByAggValidatesAgainstAlias(t *testing.T) {
	rs, err := engine.Query(context.Background(), inmem.New(),
		"SELECT user_id, COUNT(*) AS n FROM orders GROUP BY user_id ORDER BY n ASC")
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, rs)
	if len(got) != 2 || got[0][0] != int64(1) || got[1][0] != int64(2) {
		t.Fatalf("ORDER BY n ASC got %v, want user_id 1 then 2", got)
	}
}

func TestHavingSum(t *testing.T) {
	// user_id 1 → amount 100; user_id 2 → 200+50 = 250.
	rs, err := engine.Query(context.Background(), inmem.New(),
		"SELECT user_id, SUM(amount) AS total FROM orders GROUP BY user_id HAVING SUM(amount) > 150")
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, rs)
	if len(got) != 1 || got[0][0] != int64(2) {
		t.Fatalf("HAVING SUM(amount) > 150 got %v, want only user_id 2", got)
	}
}

func TestHavingAndOrderByAggregate(t *testing.T) {
	rs, err := engine.Query(context.Background(), inmem.New(),
		"SELECT user_id, COUNT(*) AS n FROM orders GROUP BY user_id HAVING COUNT(*) >= 1 ORDER BY COUNT(*) DESC")
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, rs)
	if len(got) != 2 || got[0][0] != int64(2) || got[1][0] != int64(1) {
		t.Fatalf("HAVING + ORDER BY aggregate got %v, want user_id 2 then 1", got)
	}
}

func TestOrderByScalarExprNonAggregate(t *testing.T) {
	// users: ada (len 3), alan (len 4). ORDER BY LEN(name) DESC → alan, ada.
	rs, err := engine.Query(context.Background(), inmem.New(),
		"SELECT name FROM users ORDER BY LEN(name) DESC")
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, rs)
	if len(got) != 2 || got[0][0] != "alan" || got[1][0] != "ada" {
		t.Fatalf("ORDER BY LEN(name) DESC got %v, want alan then ada", got)
	}
}

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

// Regression: FROM (derived) JOIN real once dropped the join and returned the 2 un-joined derived rows.
func TestDerivedJoinSilentDrop(t *testing.T) {
	const realReal = "SELECT u.name, o.amount FROM users u JOIN orders o ON o.user_id = u.id ORDER BY o.id"
	if n := len(qry(t, realReal)); n != 3 {
		t.Fatalf("real JOIN real = %d rows, want 3 (baseline)", n)
	}
	const derivedReal = "SELECT u.name, o.amount FROM (SELECT id, name FROM users) u JOIN orders o ON o.user_id = u.id ORDER BY o.id"
	got := qry(t, derivedReal)
	if len(got) != 3 {
		t.Fatalf("derived JOIN real = %d rows, want 3 (pre-fix dropped the join and returned 2)", len(got))
	}
	want := [][]any{{"ada", int64(100)}, {"alan", int64(200)}, {"alan", int64(50)}}
	for i, w := range want {
		if got[i][0] != w[0] || got[i][1] != w[1] {
			t.Errorf("row %d = %v, want %v", i, got[i], w)
		}
	}
}

func TestDerivedJoinRightSide(t *testing.T) {
	const q = "SELECT u.name, o.amount FROM users u JOIN (SELECT id, user_id, amount FROM orders) o ON o.user_id = u.id ORDER BY o.amount"
	got := qry(t, q)
	want := [][]any{{"alan", int64(50)}, {"ada", int64(100)}, {"alan", int64(200)}}
	if len(got) != 3 {
		t.Fatalf("real JOIN (derived) = %d rows, want 3", len(got))
	}
	for i, w := range want {
		if got[i][0] != w[0] || got[i][1] != w[1] {
			t.Errorf("row %d = %v, want %v", i, got[i], w)
		}
	}
}

func TestDerivedJoinBothSides(t *testing.T) {
	const q = "SELECT u.name, o.amount FROM (SELECT id, name FROM users) u JOIN (SELECT user_id, amount FROM orders) o ON o.user_id = u.id ORDER BY o.amount DESC"
	got := qry(t, q)
	want := [][]any{{"alan", int64(200)}, {"ada", int64(100)}, {"alan", int64(50)}}
	if len(got) != 3 {
		t.Fatalf("(derived) JOIN (derived) = %d rows, want 3", len(got))
	}
	for i, w := range want {
		if got[i][0] != w[0] || got[i][1] != w[1] {
			t.Errorf("row %d = %v, want %v", i, got[i], w)
		}
	}
}

func TestDerivedJoinThreeTableChain(t *testing.T) {
	const q = "SELECT u.name, o.amount, u2.name FROM (SELECT id, name FROM users) u " +
		"JOIN orders o ON o.user_id = u.id " +
		"JOIN (SELECT id, name FROM users) u2 ON u2.id = u.id ORDER BY o.id"
	got := qry(t, q)
	want := [][]any{{"ada", int64(100), "ada"}, {"alan", int64(200), "alan"}, {"alan", int64(50), "alan"}}
	if len(got) != 3 {
		t.Fatalf("derived-real-derived chain = %d rows, want 3", len(got))
	}
	for i, w := range want {
		if got[i][0] != w[0] || got[i][1] != w[1] || got[i][2] != w[2] {
			t.Errorf("row %d = %v, want %v", i, got[i], w)
		}
	}
}

func TestDerivedLeftJoinOuter(t *testing.T) {
	const q = "SELECT u.name, o.amount FROM users u LEFT JOIN (SELECT user_id, amount FROM orders WHERE amount > 100) o ON o.user_id = u.id ORDER BY u.id"
	got := qry(t, q)
	if len(got) != 2 {
		t.Fatalf("LEFT JOIN (derived) = %d rows, want 2", len(got))
	}
	if got[0][0] != "ada" || got[0][1] != nil {
		t.Errorf("row 0 = %v, want [ada <nil>]", got[0])
	}
	if got[1][0] != "alan" || got[1][1] != int64(200) {
		t.Errorf("row 1 = %v, want [alan 200]", got[1])
	}
}

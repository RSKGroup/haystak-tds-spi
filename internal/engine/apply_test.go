// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

// users: ada(1), alan(2). orders: u1=[100], u2=[200,50]. APPLY runs the right side once per left row.

func TestCrossApplyCorrelated(t *testing.T) {
	const q = "SELECT u.name, x.amount FROM users u CROSS APPLY (SELECT amount FROM orders WHERE user_id = u.id) x ORDER BY u.name, x.amount"
	got := qry(t, q)
	want := [][]any{{"ada", int64(100)}, {"alan", int64(50)}, {"alan", int64(200)}}
	if len(got) != 3 {
		t.Fatalf("CROSS APPLY = %d rows, want 3", len(got))
	}
	for i, w := range want {
		if got[i][0] != w[0] || got[i][1] != w[1] {
			t.Errorf("row %d = %v, want %v", i, got[i], w)
		}
	}
}

// CROSS APPLY drops a left row whose right side is empty.
func TestCrossApplyDropsEmpty(t *testing.T) {
	const q = "SELECT u.name, x.amount FROM users u CROSS APPLY (SELECT amount FROM orders WHERE user_id = u.id AND amount > 150) x ORDER BY u.id"
	got := qry(t, q)
	if len(got) != 1 || got[0][0] != "alan" || got[0][1] != int64(200) {
		t.Fatalf("CROSS APPLY (amount>150) = %v, want [[alan 200]]", got)
	}
}

// OUTER APPLY keeps the left row with right NULLs when the right side is empty.
func TestOuterApplyKeepsEmpty(t *testing.T) {
	const q = "SELECT u.name, x.amount FROM users u OUTER APPLY (SELECT amount FROM orders WHERE user_id = u.id AND amount > 150) x ORDER BY u.id"
	got := qry(t, q)
	if len(got) != 2 {
		t.Fatalf("OUTER APPLY = %d rows, want 2", len(got))
	}
	if got[0][0] != "ada" || got[0][1] != nil {
		t.Errorf("row 0 = %v, want [ada <nil>]", got[0])
	}
	if got[1][0] != "alan" || got[1][1] != int64(200) {
		t.Errorf("row 1 = %v, want [alan 200]", got[1])
	}
}

// CROSS APPLY a table-valued function whose argument references the left row.
func TestCrossApplyTableFunc(t *testing.T) {
	const q = "SELECT s.value FROM (SELECT 'a,b,c' AS tags) t CROSS APPLY STRING_SPLIT(t.tags, ',') s ORDER BY s.value"
	got := qry(t, q)
	want := []any{"a", "b", "c"}
	if len(got) != 3 {
		t.Fatalf("CROSS APPLY STRING_SPLIT = %d rows, want 3", len(got))
	}
	for i, w := range want {
		if got[i][0] != w {
			t.Errorf("row %d = %v, want %v", i, got[i][0], w)
		}
	}
}

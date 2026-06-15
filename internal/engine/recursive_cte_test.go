// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

// Self-contained recursion (the recursive arm's FROM is the CTE itself).
func TestRecursiveCTENumbers(t *testing.T) {
	rows := qry(t, "WITH nums AS (SELECT 1 AS n UNION ALL SELECT n+1 FROM nums WHERE n < 5) SELECT n FROM nums")
	if len(rows) != 5 {
		t.Fatalf("got %d rows, want 5", len(rows))
	}
	for i, r := range rows {
		if cell(r[0]) != cellInt(i+1) {
			t.Errorf("row %d = %v, want %d", i, r[0], i+1)
		}
	}
}

// Hierarchy recursion (the recursive arm JOINs a real table against the CTE) -- the case that was broken.
func TestRecursiveCTEHierarchy(t *testing.T) {
	const q = "WITH tree AS (" +
		"SELECT category_id, name, parent_id FROM categories WHERE parent_id IS NULL " +
		"UNION ALL " +
		"SELECT c.category_id, c.name, c.parent_id FROM categories c JOIN tree t ON c.parent_id = t.category_id" +
		") SELECT name FROM tree"
	rows := qry(t, q)
	got := make([]string, len(rows))
	for i, r := range rows {
		got[i] = cell(r[0])
	}
	want := []string{"All", "Electronics", "Phones"}
	if len(got) != len(want) {
		t.Fatalf("hierarchy = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("hierarchy = %v, want %v", got, want)
		}
	}
}

func cellInt(n int) string { return cell(int64(n)) }

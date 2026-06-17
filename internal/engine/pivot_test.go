// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

// sales: E/A=10, E/B=20, W/A=30, W/A=5. PIVOT rotates product values into columns.
func TestPivotSum(t *testing.T) {
	const q = "SELECT region, [A], [B] FROM sales PIVOT (SUM(amount) FOR product IN ([A], [B])) AS p ORDER BY region"
	got := qry(t, q)
	if len(got) != 2 {
		t.Fatalf("PIVOT = %d rows, want 2", len(got))
	}
	if cell(got[0][0]) != "E" || num(got[0][1]) != 10 || num(got[0][2]) != 20 {
		t.Errorf("row E = %v, want [E 10 20]", got[0])
	}
	if cell(got[1][0]) != "W" || num(got[1][1]) != 35 || got[1][2] != nil {
		t.Errorf("row W = %v, want [W 35 <nil>]", got[1])
	}
}

// UNPIVOT rotates the q1/q2 columns of one wide row into (quarter, amount) rows.
func TestUnpivot(t *testing.T) {
	const q = "SELECT region, quarter, amount FROM (SELECT 'E' AS region, 10 AS q1, 20 AS q2) t UNPIVOT (amount FOR quarter IN (q1, q2)) AS u ORDER BY quarter"
	got := qry(t, q)
	if len(got) != 2 {
		t.Fatalf("UNPIVOT = %d rows, want 2", len(got))
	}
	if cell(got[0][0]) != "E" || cell(got[0][1]) != "q1" || num(got[0][2]) != 10 {
		t.Errorf("row 0 = %v, want [E q1 10]", got[0])
	}
	if cell(got[1][1]) != "q2" || num(got[1][2]) != 20 {
		t.Errorf("row 1 = %v, want [E q2 20]", got[1])
	}
}

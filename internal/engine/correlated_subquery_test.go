// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

// A scalar correlated subquery in the SELECT list must bind the outer row per-row, not collapse to a
// single db-wide value. inmem: users ada(id 1)->1 order, alan(id 2)->2 orders; 3 orders total.
func TestScalarCorrelatedSubqueryPerRow(t *testing.T) {
	rows := qry(t, "SELECT u.name, (SELECT COUNT(*) FROM orders o WHERE o.user_id = u.id) AS n FROM users u ORDER BY u.name")
	want := map[string]string{"ada": "1", "alan": "2"}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	for _, r := range rows {
		name := cell(r[0])
		if got, w := cell(r[1]), want[name]; got != w {
			t.Errorf("%s = %q, want %q (per-row; the bug returns the db-wide total 3 for every row)", name, got, w)
		}
	}
}

// The scalar form must agree with the equivalent OUTER APPLY form row-for-row.
func TestScalarSubqueryMatchesOuterApply(t *testing.T) {
	scalar := qry(t, "SELECT u.name, (SELECT COUNT(*) FROM orders o WHERE o.user_id = u.id) FROM users u ORDER BY u.name")
	apply := qry(t, "SELECT u.name, x.n FROM users u OUTER APPLY (SELECT COUNT(*) n FROM orders o WHERE o.user_id = u.id) x ORDER BY u.name")
	if len(scalar) != len(apply) {
		t.Fatalf("row counts differ: scalar=%d apply=%d", len(scalar), len(apply))
	}
	for i := range scalar {
		if cell(scalar[i][1]) != cell(apply[i][1]) {
			t.Errorf("row %d (%s): scalar=%v apply=%v", i, cell(scalar[i][0]), scalar[i][1], apply[i][1])
		}
	}
}

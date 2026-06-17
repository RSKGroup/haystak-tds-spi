// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

// orders has 3 rows. SET ROWCOUNT caps later statements in the batch; 0 resets to unlimited.
func TestSetRowcount(t *testing.T) {
	if n := len(qry(t, "SET ROWCOUNT 2; SELECT id FROM orders")); n != 2 {
		t.Errorf("SET ROWCOUNT 2 = %d rows, want 2", n)
	}
	if n := len(qry(t, "SET ROWCOUNT 1; SELECT id FROM orders ORDER BY id")); n != 1 {
		t.Errorf("SET ROWCOUNT 1 = %d rows, want 1", n)
	}
	if n := len(qry(t, "SET ROWCOUNT 0; SELECT id FROM orders")); n != 3 {
		t.Errorf("SET ROWCOUNT 0 = %d rows, want 3 (unlimited)", n)
	}
	if n := len(qry(t, "SELECT id FROM orders")); n != 3 {
		t.Errorf("no ROWCOUNT = %d rows, want 3", n)
	}
}

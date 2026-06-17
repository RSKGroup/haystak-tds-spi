// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

// Table/query hints are parsed and ignored; the query still runs and returns the same rows.
func TestHintsExecute(t *testing.T) {
	if got := qry(t, "SELECT name FROM users WITH (NOLOCK) WHERE id = 2"); len(got) != 1 || cell(got[0][0]) != "alan" {
		t.Errorf("WITH (NOLOCK) = %v, want [[alan]]", got)
	}
	if n := len(qry(t, "SELECT id FROM orders TABLESAMPLE (50 PERCENT)")); n != 3 {
		t.Errorf("TABLESAMPLE = %d rows, want 3 (ignored, all rows returned)", n)
	}
	if n := len(qry(t, "SELECT id FROM orders ORDER BY id OPTION (MAXDOP 1)")); n != 3 {
		t.Errorf("OPTION = %d rows, want 3", n)
	}
	if n := len(qry(t, "SELECT u.id FROM users u WITH (NOLOCK) JOIN orders o WITH (NOLOCK) ON o.user_id = u.id")); n != 3 {
		t.Errorf("JOIN with hints = %d rows, want 3", n)
	}
}

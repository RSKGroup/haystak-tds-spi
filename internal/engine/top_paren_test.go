// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

func TestTopParenLimits(t *testing.T) {
	if n := len(qry(t, "SELECT TOP (2) id FROM orders ORDER BY id")); n != 2 {
		t.Errorf("TOP (2) = %d rows, want 2", n)
	}
	if n := len(qry(t, "SELECT TOP (50) PERCENT id FROM orders ORDER BY id")); n != 2 {
		t.Errorf("TOP (50) PERCENT of 3 rows = %d, want 2 (ceil)", n)
	}
}

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

// orders.user_id = {1, 2, 2}; amounts = {100, 200, 50}.
func TestCountDistinct(t *testing.T) {
	if g := qry(t, "SELECT COUNT(DISTINCT user_id) FROM orders"); g[0][0] != int64(2) {
		t.Errorf("COUNT(DISTINCT user_id) = %v, want 2", g[0][0])
	}
	if g := qry(t, "SELECT COUNT(user_id) FROM orders"); g[0][0] != int64(3) {
		t.Errorf("COUNT(user_id) = %v, want 3", g[0][0])
	}
	if g := qry(t, "SELECT COUNT(DISTINCT amount) FROM orders"); g[0][0] != int64(3) {
		t.Errorf("COUNT(DISTINCT amount) = %v, want 3", g[0][0])
	}
}

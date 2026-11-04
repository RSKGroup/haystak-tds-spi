// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"fmt"
	"testing"
)

func TestDistinctAggregateInExpression(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"SELECT COUNT(DISTINCT user_id) + 1 AS n FROM orders", "[[3]]"},
		{"SELECT COUNT(DISTINCT user_id) * 10 - COUNT(*) AS n FROM orders", "[[17]]"},
		{"SELECT SUM(DISTINCT v.a) + 0 AS s FROM (VALUES (5), (5), (7)) v(a)", "[[12]]"},
		{"SELECT user_id FROM orders GROUP BY user_id HAVING COUNT(DISTINCT amount) > 1", "[[2]]"},
		{"SELECT user_id, COUNT(DISTINCT amount) - 1 AS extra FROM orders GROUP BY user_id ORDER BY user_id", "[[1 0] [2 1]]"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(qry(t, c.sql)); got != c.want {
			t.Errorf("%s\n  = %s, want %s", c.sql, got, c.want)
		}
	}
}

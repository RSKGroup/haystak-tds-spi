// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"fmt"
	"testing"
)

func TestSetOpInSubqueries(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"SELECT b.id FROM (SELECT id FROM users WHERE id = 1 UNION ALL SELECT id FROM orders WHERE id = 10) b ORDER BY b.id", "[[1] [10]]"},
		{"SELECT COUNT(*) AS n FROM (SELECT user_id FROM orders UNION SELECT id FROM users) u", "[[2]]"},
		{"SELECT u.name FROM users u JOIN (SELECT 1 AS k UNION ALL SELECT 2) d ON d.k = u.id ORDER BY u.name", "[[ada] [alan]]"},
		{"SELECT u.name, c.n FROM users u CROSS APPLY (SELECT COUNT(*) AS n FROM (SELECT id FROM orders WHERE user_id = u.id UNION ALL SELECT 0) x) c ORDER BY u.name", "[[ada 2] [alan 3]]"},
		{"SELECT name FROM users WHERE id IN (SELECT 1 UNION SELECT 3)", "[[ada]]"},
		{"SELECT name FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE amount > 1000 UNION ALL SELECT 1 FROM orders WHERE amount = 50) ORDER BY name", "[[ada] [alan]]"},
		{"SELECT (SELECT MAX(v) FROM (SELECT 1 AS v UNION ALL SELECT 7) m) AS hi", "[[7]]"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(qry(t, c.sql)); got != c.want {
			t.Errorf("%s\n  = %s, want %s", c.sql, got, c.want)
		}
	}
}

func TestAggregateOperandAndValues(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"SELECT COUNT(*) + 1 AS n FROM users", "[[3]]"},
		{"SELECT 1 + COUNT(*) AS n FROM users", "[[3]]"},
		{"SELECT MIN(name) + 'x' AS n FROM users", "[[adax]]"},
		{"SELECT MAX(amount) - MIN(amount) AS spread FROM orders", "[[150]]"},
		{"SELECT MIN(v.x) AS m FROM (VALUES ('b'),('A')) v(x)", "[[A]]"},
		{"SELECT v.k, v.s FROM (VALUES (1, 'one'), (2, 'two')) AS v(k, s) WHERE v.k = 2", "[[2 two]]"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(qry(t, c.sql)); got != c.want {
			t.Errorf("%s\n  = %s, want %s", c.sql, got, c.want)
		}
	}
}

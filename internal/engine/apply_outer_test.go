// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"fmt"
	"testing"
)

func TestApplyReadsOuterRow(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"SELECT u.name, c.x FROM users u CROSS APPLY (SELECT u.id + 1 AS x) c ORDER BY u.name", "[[ada 2] [alan 3]]"},
		{"SELECT u.name, c.id FROM users u CROSS APPLY (SELECT u.id) c ORDER BY u.id", "[[ada 1] [alan 2]]"},
		{"SELECT u.name, s.k FROM users u CROSS APPLY (SELECT CASE u.id WHEN 1 THEN 'one' ELSE CAST(u.id AS varchar(5)) END AS k) s ORDER BY u.id", "[[ada one] [alan 2]]"},
		{"SELECT u.name, t.n FROM users u CROSS APPLY (SELECT (SELECT COUNT(*) FROM orders o WHERE o.user_id = u.id) AS n) t ORDER BY u.id", "[[ada 1] [alan 2]]"},
		{"SELECT u.name, t.n FROM users u CROSS APPLY (SELECT COUNT(*) AS n FROM STRING_SPLIT(u.name, 'l') s) t ORDER BY u.id", "[[ada 1] [alan 2]]"},
		{"SELECT u.name, t.n FROM users u OUTER APPLY (SELECT COUNT(*) AS n FROM orders u WHERE u.amount > 60) t ORDER BY u.name", "[[ada 2] [alan 2]]"},
		{"SELECT u.name, c.n FROM users u CROSS APPLY (SELECT COUNT(*) AS n FROM (SELECT id FROM orders WHERE user_id = u.id UNION ALL SELECT 0) x) c ORDER BY u.name", "[[ada 2] [alan 3]]"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(qry(t, c.sql)); got != c.want {
			t.Errorf("%s\n  = %s, want %s", c.sql, got, c.want)
		}
	}
}

func TestCorrelatedToJoinedSource(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"SELECT e.name FROM depts d JOIN emps e ON e.dept_id = d.id WHERE EXISTS (SELECT 1 FROM orders o WHERE o.id = e.id AND o.amount > 150)", "[[bob]]"},
		{"SELECT d.name, (SELECT COUNT(*) FROM emps x WHERE x.dept_id = d.id) AS n FROM emps e JOIN depts d ON d.id = e.dept_id ORDER BY d.name", "[[eng 1] [sales 1]]"},
		{"SELECT t.k, (SELECT COUNT(*) FROM orders o WHERE o.user_id = t.k) AS n FROM (SELECT id AS k FROM users) t ORDER BY t.k", "[[1 1] [2 2]]"},
		{"SELECT s.value, u.id FROM STRING_SPLIT('ada,alan', ',') s JOIN users u ON u.name = s.value ORDER BY u.id", "[[ada 1] [alan 2]]"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(qry(t, c.sql)); got != c.want {
			t.Errorf("%s\n  = %s, want %s", c.sql, got, c.want)
		}
	}
}

func TestAliasStarOnSingleSource(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"SELECT u.* FROM users u WHERE u.id = 1", "[[1 ada]]"},
		{"SELECT q.*, q.a + 1 AS b FROM (SELECT 1 AS a) q", "[[1 2]]"},
		{"SELECT z.a, z.b FROM (SELECT q.*, q.a + 1 AS b FROM (SELECT 1 AS a) q) z", "[[1 2]]"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(qry(t, c.sql)); got != c.want {
			t.Errorf("%s\n  = %s, want %s", c.sql, got, c.want)
		}
	}
}

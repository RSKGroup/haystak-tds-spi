// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
)

// 376: a CTE named inside a derived table or subquery was "not found"; its name is in scope for the whole statement.
func TestCTEVisibleInNestedQueries(t *testing.T) {
	b := collationBackend(t)
	cases := []struct{ name, sql, want string }{
		{"derived-table", "WITH v AS (SELECT city c FROM people) SELECT COUNT(*) FROM (SELECT c FROM v GROUP BY c) g", "3"},
		{"derived-over-union-cte", "WITH v AS (SELECT name m FROM people WHERE id = 1 UNION ALL SELECT name FROM people WHERE id = 2) SELECT COUNT(*) FROM (SELECT m FROM v GROUP BY m) g", "1"},
		{"in-subquery", "WITH v AS (SELECT id FROM people WHERE city = 'nyc') SELECT name FROM people WHERE id IN (SELECT id FROM v) ORDER BY id", "jones;Jones"},
		{"exists", "WITH v AS (SELECT code FROM cities) SELECT COUNT(*) FROM people WHERE EXISTS (SELECT 1 FROM v)", "6"},
		{"scalar-subquery", "WITH v AS (SELECT id FROM people WHERE id > 4) SELECT (SELECT COUNT(*) FROM v)", "2"},
		{"joined-derived", "WITH v AS (SELECT code, label FROM cities) SELECT p.id, d.label FROM people p JOIN (SELECT code, label FROM v) d ON p.city = d.code WHERE p.id < 3 ORDER BY p.id", "1|MA;2|MA"},
		{"cte-over-cte-in-derived", "WITH a AS (SELECT id FROM people WHERE id < 4), b AS (SELECT id FROM (SELECT id FROM a) x) SELECT COUNT(*) FROM b", "3"},
		{"table-without-cte-unchanged", "SELECT COUNT(*) FROM (SELECT id FROM people) x", "6"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rs, err := engine.Query(context.Background(), b, c.sql)
			if err != nil {
				t.Fatalf("%s: %v", c.sql, err)
			}
			if got := render(collect(t, rs)); got != c.want {
				t.Errorf("%s\n got  %s\n want %s", c.sql, got, c.want)
			}
		})
	}
}

func TestRecursiveCTEWithScopedNames(t *testing.T) {
	b := collationBackend(t)
	for sql, want := range map[string]string{
		"WITH n AS (SELECT 1 i UNION ALL SELECT i + 1 FROM n WHERE i < 4) SELECT COUNT(*) FROM n":                                        "4",
		"WITH lim AS (SELECT 3 m), n AS (SELECT 1 i UNION ALL SELECT i + 1 FROM n WHERE i < (SELECT m FROM lim)) SELECT COUNT(*) FROM n": "3",
		"WITH n AS (SELECT 1 i UNION ALL SELECT i + 1 FROM n WHERE i < 3) SELECT COUNT(*) FROM (SELECT i FROM n) x":                      "3",
	} {
		rs, err := engine.Query(context.Background(), b, sql)
		if err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		if got := render(collect(t, rs)); got != want {
			t.Errorf("%s\n got  %s\n want %s", sql, got, want)
		}
	}
}

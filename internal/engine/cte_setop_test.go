// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
)

// 369: a set operation inside a CTE body kept only its first arm, silently.
func TestSetOperationInsideCTE(t *testing.T) {
	b := collationBackend(t)
	cases := []struct{ name, sql, want string }{
		{"union-all-literals", "WITH v AS (SELECT 'b' x UNION ALL SELECT 'A') SELECT COUNT(*) FROM v", "2"},
		{"three-arms", "WITH v AS (SELECT 'b' x UNION ALL SELECT 'A' UNION ALL SELECT 'C') SELECT x FROM v", "b;A;C"},
		{"table-arms", "WITH v AS (SELECT name FROM people WHERE id = 1 UNION ALL SELECT name FROM people WHERE id = 3) SELECT COUNT(*) FROM v", "2"},
		{"union-dedups-case-insensitively", "WITH v AS (SELECT name FROM people WHERE id = 1 UNION SELECT name FROM people WHERE id = 2) SELECT COUNT(*) FROM v", "1"},
		{"intersect", "WITH v AS (SELECT city FROM people WHERE id < 3 INTERSECT SELECT code FROM cities) SELECT COUNT(*) FROM v", "1"},
		{"except", "WITH v AS (SELECT city FROM people EXCEPT SELECT code FROM cities) SELECT city FROM v", "Paris"},
		{"cte-over-cte", "WITH a AS (SELECT id FROM people WHERE id < 3), v AS (SELECT id FROM a UNION ALL SELECT id FROM a) SELECT COUNT(*) FROM v", "4"},
		{"later-arm-reads-cte", "WITH v AS (SELECT name FROM people WHERE id = 1) SELECT name FROM v UNION ALL SELECT name FROM v", "Smith;Smith"},
		{"grouped-over-union", "WITH v AS (SELECT city c FROM people WHERE id = 1 UNION ALL SELECT city FROM people WHERE id = 3) SELECT COUNT(*) FROM v GROUP BY c ORDER BY c", "1;1"},
		{"recursive-unchanged", "WITH n AS (SELECT 1 i UNION ALL SELECT i + 1 FROM n WHERE i < 4) SELECT COUNT(*) FROM n", "4"},
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

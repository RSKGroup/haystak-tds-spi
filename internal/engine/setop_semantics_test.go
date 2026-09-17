// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
)

// 374: a TOP limits its own SELECT; only a trailing OFFSET ... FETCH pages the whole set operation.
// 375: INTERSECT binds before UNION and EXCEPT, and each operator removes duplicates from its own step only.
func TestSetOperationSemantics(t *testing.T) {
	b := collationBackend(t)
	cases := []struct{ name, sql, want string }{
		{"top-per-arm", "SELECT TOP 1 'a' x UNION ALL SELECT TOP 1 'b'", "a;b"},
		{"top-per-arm-table", "SELECT TOP 2 id FROM people WHERE id < 3 UNION ALL SELECT TOP 1 id FROM people WHERE id = 5", "1;2;5"},
		{"top-last-arm-with-order", "SELECT id FROM people WHERE id = 6 UNION ALL SELECT TOP 2 id FROM people WHERE id < 3 ORDER BY id DESC", "6;2;1"},
		{"fetch-pages-whole", "SELECT id FROM people WHERE id < 3 UNION ALL SELECT id FROM people WHERE id > 4 ORDER BY id OFFSET 1 ROWS FETCH NEXT 2 ROWS ONLY", "2;5"},
		{"top-arm-then-fetch", "SELECT TOP 1 id FROM people WHERE id < 3 UNION ALL SELECT id FROM people WHERE id > 3 ORDER BY id DESC OFFSET 0 ROWS FETCH NEXT 2 ROWS ONLY", "6;5"},
		{"union-then-union-all-keeps-dupes", "SELECT 1 x UNION SELECT 1 UNION ALL SELECT 1", "1;1"},
		{"union-all-then-union-dedups", "SELECT 1 x UNION ALL SELECT 1 UNION SELECT 1", "1"},
		{"intersect-binds-first", "SELECT 1 x UNION ALL SELECT 2 INTERSECT SELECT 2", "1;2"},
		{"intersect-chain", "SELECT 1 x UNION SELECT 2 INTERSECT SELECT 3 INTERSECT SELECT 2", "1"},
		{"except-left-to-right", "SELECT 1 x UNION SELECT 2 EXCEPT SELECT 2 UNION SELECT 3", "1;3"},
		{"except-then-union-all", "SELECT 1 x EXCEPT SELECT 2 UNION ALL SELECT 1", "1;1"},
		{"case-insensitive-union", "SELECT name FROM people WHERE id = 1 UNION SELECT name FROM people WHERE id = 2", "Smith"},
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

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
)

// 370: a scalar function around an aggregate was evaluated per row, one blank row each; expressions in
// a grouped select list were refused.
func TestAggregateInsideScalarExpression(t *testing.T) {
	b := collationBackend(t)
	cases := []struct{ name, sql, want string }{
		{"upper-of-min", "SELECT UPPER(MIN(name)) FROM people WHERE id < 5", "JONES"},
		{"len-of-max", "SELECT LEN(MAX(city)) FROM people WHERE id < 5", "3"},
		{"cast-of-count", "SELECT CAST(COUNT(*) AS VARCHAR(10)) FROM people", "6"},
		{"per-group", "SELECT city, UPPER(MIN(name)) FROM people WHERE id < 5 GROUP BY city ORDER BY city", "Boston|SMITH;NYC|JONES"},
		{"grouped-column-in-function", "SELECT UPPER(city), COUNT(*) FROM people GROUP BY city ORDER BY city", "BOSTON|2;NYC|2;PARIS|2"},
		{"case-over-aggregate", "SELECT city, CASE WHEN COUNT(*) > 1 THEN 'many' ELSE 'one' END FROM people WHERE id <> 2 GROUP BY city ORDER BY city", "Boston|one;NYC|many;Paris|many"},
		{"aggregate-of-expression", "SELECT UPPER(MIN(LOWER(name))) FROM people WHERE id < 5", "JONES"},
		{"coalesce-empty-group", "SELECT COALESCE(MAX(name), 'none') FROM people WHERE id > 99", "none"},
		{"having-over-nested", "SELECT city FROM people GROUP BY city HAVING UPPER(MIN(name)) = 'JONES'", "NYC"},
		{"alias-kept", "SELECT UPPER(MIN(name)) AS first_name FROM people WHERE id = 1", "SMITH"},
		{"window-untouched", "SELECT id, MIN(name) OVER (PARTITION BY city) FROM people WHERE id < 3 ORDER BY id", "1|Smith;2|Smith"},
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
	rs, err := engine.Query(context.Background(), b, "SELECT UPPER(MIN(name)) AS first_name FROM people")
	if err != nil {
		t.Fatal(err)
	}
	if cols := rs.Columns(); len(cols) != 1 || cols[0].Name != "first_name" {
		t.Errorf("columns %+v, want one named first_name", cols)
	}
	rs.Close()
}

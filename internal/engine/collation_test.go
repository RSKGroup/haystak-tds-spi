// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"strings"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
	"github.com/RSKGroup/haystak-tds-spi/tds"
)

func collationBackend(t *testing.T) tds.Backend {
	t.Helper()
	b := inmem.New()
	for _, sql := range []string{
		"CREATE TABLE people (id INT, name VARCHAR(50), city VARCHAR(50))",
		"INSERT INTO people (id, name, city) VALUES (1, 'Smith', 'Boston')",
		"INSERT INTO people (id, name, city) VALUES (2, 'SMITH', 'boston')",
		"INSERT INTO people (id, name, city) VALUES (3, 'jones', 'NYC')",
		"INSERT INTO people (id, name, city) VALUES (4, 'Jones', 'nyc')",
		"INSERT INTO people (id, name, city) VALUES (5, N'Émile', 'Paris')",
		"INSERT INTO people (id, name, city) VALUES (6, N'émile', 'paris')",
		"CREATE TABLE cities (code VARCHAR(10), label VARCHAR(50))",
		"INSERT INTO cities (code, label) VALUES ('BOSTON', 'MA')",
		"INSERT INTO cities (code, label) VALUES ('Nyc', 'NY')",
	} {
		if _, _, err := engine.Exec(context.Background(), b, sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	return b
}

func render(rows [][]any) string {
	out := make([]string, len(rows))
	for i, r := range rows {
		cells := make([]string, len(r))
		for j, v := range r {
			cells[j] = cell(v)
		}
		out[i] = strings.Join(cells, "|")
	}
	return strings.Join(out, ";")
}

func TestCaseInsensitiveDefault(t *testing.T) {
	b := collationBackend(t)
	cases := []struct{ name, sql, want string }{
		{"eq", "SELECT id FROM people WHERE name = 'smith' ORDER BY id", "1;2"},
		{"ne", "SELECT id FROM people WHERE name <> 'SMITH' AND id < 5 ORDER BY id", "3;4"},
		{"lt", "SELECT id FROM people WHERE name < 'SMITH' AND id < 5 ORDER BY id", "3;4"},
		{"in", "SELECT id FROM people WHERE name IN ('JONES', 'x') ORDER BY id", "3;4"},
		{"not-in", "SELECT id FROM people WHERE name NOT IN ('smith', 'JONES') ORDER BY id", "5;6"},
		{"in-subquery", "SELECT id FROM people WHERE name IN (SELECT UPPER(name) FROM people WHERE id = 3) ORDER BY id", "3;4"},
		{"between", "SELECT id FROM people WHERE name BETWEEN 'JONES' AND 'JONES' ORDER BY id", "3;4"},
		{"like", "SELECT id FROM people WHERE name LIKE 'sM%' ORDER BY id", "1;2"},
		{"like-underscore", "SELECT id FROM people WHERE name LIKE '_ONES' ORDER BY id", "3;4"},
		{"non-ascii-eq", "SELECT id FROM people WHERE name = N'ÉMILE'", "5"},
		{"non-ascii-like", "SELECT id FROM people WHERE name LIKE N'é%'", "6"},
		{"case-simple", "SELECT CASE name WHEN 'SMITH' THEN 'y' ELSE 'n' END FROM people WHERE id = 1", "y"},
		{"case-searched", "SELECT CASE WHEN city = 'BOSTON' THEN 'y' ELSE 'n' END FROM people WHERE id = 1", "y"},
		{"nullif", "SELECT NULLIF(name, 'smith') FROM people WHERE id = 1", "<nil>"},
		{"join-on", "SELECT p.id, c.label FROM people p JOIN cities c ON p.city = c.code ORDER BY p.id", "1|MA;2|MA;3|NY;4|NY"},
		{"group-by-first-spelling", "SELECT city, COUNT(*) FROM people GROUP BY city ORDER BY city", "Boston|2;NYC|2;Paris|2"},
		{"having", "SELECT city, COUNT(*) FROM people GROUP BY city HAVING city = 'BOSTON'", "Boston|2"},
		{"having-in", "SELECT city FROM people GROUP BY city HAVING city IN ('nyc')", "NYC"},
		{"having-like", "SELECT city FROM people GROUP BY city HAVING city LIKE 'PAR%'", "Paris"},
		{"order-by", "SELECT id FROM people WHERE id < 5 ORDER BY name, id", "3;4;1;2"},
		{"order-by-desc-stable", "SELECT id FROM people WHERE id < 5 ORDER BY name DESC", "1;2;3;4"},
		{"agg-order-by", "SELECT name, COUNT(*) FROM people WHERE id < 5 GROUP BY name ORDER BY name DESC", "Smith|2;jones|2"},
		{"distinct", "SELECT DISTINCT city FROM people", "Boston;NYC;Paris"},
		{"union", "SELECT name FROM people WHERE id = 1 UNION SELECT name FROM people WHERE id = 2", "Smith"},
		{"intersect", "SELECT name FROM people WHERE id = 1 INTERSECT SELECT name FROM people WHERE id = 2", "Smith"},
		{"except", "SELECT name FROM people WHERE id IN (1, 3) EXCEPT SELECT name FROM people WHERE id = 2", "jones"},
		{"min-max", "SELECT MIN(name), MAX(name) FROM people WHERE id < 5", "jones|Smith"},
		{"count-distinct", "SELECT COUNT(DISTINCT city) FROM people", "3"},
		{"approx-count-distinct", "SELECT APPROX_COUNT_DISTINCT(name) FROM people", "4"},
		{"partition-by", "SELECT id, COUNT(*) OVER (PARTITION BY city) FROM people ORDER BY id", "1|2;2|2;3|2;4|2;5|2;6|2"},
		{"window-rank-peers", "SELECT id, RANK() OVER (ORDER BY name) FROM people WHERE id < 5 ORDER BY id", "1|3;2|3;3|1;4|1"},
		{"window-min-max", "SELECT id, MIN(name) OVER (), MAX(name) OVER () FROM people WHERE id < 5 ORDER BY id", "1|jones|Smith;2|jones|Smith;3|jones|Smith;4|jones|Smith"},
		{"pivot-group-key", "SELECT * FROM (SELECT name, city, id FROM people) s PIVOT (COUNT(id) FOR city IN (boston, nyc)) p", "Smith|2|0;jones|0|2;Émile|0|0;émile|0|0"},
		{"pivot", "SELECT * FROM (SELECT city, id FROM people) s PIVOT (COUNT(id) FOR city IN (boston, NYC)) p", "2|2"},
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

func TestCollateCaseSensitive(t *testing.T) {
	b := collationBackend(t)
	cases := []struct{ name, sql, want string }{
		{"eq-left", "SELECT id FROM people WHERE name COLLATE Latin1_General_CS_AS = 'Smith'", "1"},
		{"eq-right", "SELECT id FROM people WHERE name = 'smith' COLLATE SQL_Latin1_General_CP1_CS_AS", ""},
		{"ci-name-keeps-default", "SELECT id FROM people WHERE name = 'smith' COLLATE SQL_Latin1_General_CP1_CI_AS ORDER BY id", "1;2"},
		{"ne", "SELECT id FROM people WHERE name COLLATE Latin1_General_CS_AS <> 'Smith' AND id < 5 ORDER BY id", "2;3;4"},
		{"in", "SELECT id FROM people WHERE name COLLATE Latin1_General_CS_AS IN ('JONES', 'jones')", "3"},
		{"between", "SELECT id FROM people WHERE name BETWEEN 'jones' COLLATE Latin1_General_CS_AS AND 'jones' ORDER BY id", "3"},
		{"like", "SELECT id FROM people WHERE name LIKE 'SM%' COLLATE Latin1_General_CS_AS", "2"},
		{"join-on", "SELECT p.id FROM people p JOIN cities c ON p.city COLLATE Latin1_General_CS_AS = c.code", ""},
		{"having", "SELECT city FROM people GROUP BY city HAVING city COLLATE Latin1_General_CS_AS = 'boston'", ""},
		{"order-by", "SELECT id FROM people WHERE id < 5 ORDER BY name COLLATE Latin1_General_CS_AS", "4;2;1;3"},
		{"order-by-expr", "SELECT id FROM people WHERE id < 5 ORDER BY LEFT(name, 5) COLLATE Latin1_General_CS_AS", "4;2;1;3"},
		{"agg-order-by", "SELECT name FROM people WHERE id < 5 GROUP BY name COLLATE Latin1_General_CS_AS ORDER BY name COLLATE Latin1_General_CS_AS", "Jones;SMITH;Smith;jones"},
		{"group-by", "SELECT city, COUNT(*) FROM people GROUP BY city COLLATE Latin1_General_CS_AS", "Boston|1;boston|1;NYC|1;nyc|1;Paris|1;paris|1"},
		{"window-order-by", "SELECT id, RANK() OVER (ORDER BY name COLLATE Latin1_General_CS_AS) FROM people WHERE id < 5 ORDER BY id", "1|3;2|2;3|4;4|1"},
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

func TestCollateWritePredicate(t *testing.T) {
	b := collationBackend(t)
	if _, n, err := engine.Exec(context.Background(), b, "DELETE FROM people WHERE name = 'smith' COLLATE Latin1_General_CS_AS"); err != nil || n != 0 {
		t.Fatalf("case-sensitive DELETE affected %d (err %v), want 0", n, err)
	}
	if _, n, err := engine.Exec(context.Background(), b, "DELETE FROM people WHERE name = 'smith'"); err != nil || n != 2 {
		t.Fatalf("default DELETE affected %d (err %v), want 2", n, err)
	}
}

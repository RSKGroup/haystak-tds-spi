package tsql

import "testing"

func TestParseStringLiteralAlias(t *testing.T) {
	for _, sql := range []string{
		`SELECT 1 AS 'DatabaseEngineType'`,
		`SELECT case when 1=1 then 2 else 1 end as 'x'`,
	} {
		q, err := Parse(sql)
		if err != nil {
			t.Fatalf("Parse(%q) = %v", sql, err)
		}
		if got := q.Select[0].Alias; got != "DatabaseEngineType" && got != "x" {
			t.Fatalf("Parse(%q) alias = %q", sql, got)
		}
	}
}

func TestParseNonReservedAlias(t *testing.T) {
	for _, c := range []struct{ sql, col, tbl string }{
		{`SELECT u.name AS first FROM users u`, "first", "u"},
		{`SELECT u.name first FROM users u`, "first", "u"},
		{`SELECT first = u.name FROM users u`, "first", "u"},
		{`SELECT u.name AS next FROM users AS rows`, "next", "rows"},
		{`SELECT x.name AS last FROM users first`, "last", "first"},
	} {
		q, err := Parse(c.sql)
		if err != nil {
			t.Fatalf("Parse(%q) = %v", c.sql, err)
		}
		if got := q.Select[0].Alias; got != c.col {
			t.Errorf("Parse(%q) column alias = %q, want %q", c.sql, got, c.col)
		}
		if q.FromAlias != c.tbl {
			t.Errorf("Parse(%q) table alias = %q, want %q", c.sql, q.FromAlias, c.tbl)
		}
	}
	q, err := Parse(`SELECT v.first FROM (VALUES (1)) AS v(first)`)
	if err != nil {
		t.Fatalf("VALUES column list: %v", err)
	}
	if q.FromAlias != "v" {
		t.Errorf("VALUES alias = %q, want v", q.FromAlias)
	}
}

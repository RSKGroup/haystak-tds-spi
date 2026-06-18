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

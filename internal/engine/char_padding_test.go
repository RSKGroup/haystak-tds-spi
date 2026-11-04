// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"fmt"
	"testing"
)

func TestCharPaddingAndTrailingSpaces(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"SELECT CAST('ab' AS char(5)) + '|'", "ab   |"},
		{"SELECT CAST(N'ab' AS nchar(4)) + '|'", "ab  |"},
		{"SELECT CAST(1 AS char(3)) + '|'", "1  |"},
		{"SELECT CAST('abcdef' AS char(3)) + '|'", "abc|"},
		{"SELECT CONVERT(char(12), DATEFROMPARTS(1980, 5, 1), 23) + '|'", "1980-05-01  |"},
		{"SELECT CAST('ab' AS varchar(5)) + '|'", "ab|"},
		{"SELECT LEN(CAST('ab' AS char(5)))", "2"},
		{"SELECT CASE WHEN CAST('ab' AS char(5)) = 'ab' THEN 1 ELSE 0 END", "1"},
		{"SELECT CASE WHEN 'ab' = 'ab   ' AND 'ab' COLLATE Latin1_General_CS_AS = 'ab  ' THEN 1 ELSE 0 END", "1"},
		{"SELECT CASE WHEN 'ab' < 'ab ' OR 'ab ' < 'ab' THEN 1 ELSE 0 END", "0"},
		{"SELECT CASE WHEN CAST('ab' AS char(5)) LIKE 'ab' THEN 1 ELSE 0 END", "1"},
		{"SELECT CASE WHEN 'ab' LIKE 'ab ' THEN 1 ELSE 0 END", "0"},
		{"SELECT CASE WHEN 'ab ' LIKE 'ab_' THEN 1 ELSE 0 END", "1"},
		{"SELECT COUNT(*) FROM (SELECT 'a' AS v UNION SELECT 'a  ') u", "1"},
		{"SELECT COUNT(DISTINCT v.x) FROM (VALUES ('a'), ('a ')) v(x)", "1"},
		{"SELECT COUNT(*) FROM (VALUES ('a'), ('a ')) v(x) GROUP BY v.x", "2"},
		{"SELECT NULLIF('a', 'a  ')", "<nil>"},
	}
	for _, c := range cases {
		rows := qry(t, c.sql)
		if len(rows) != 1 || fmt.Sprint(rows[0][0]) != c.want {
			t.Errorf("%s = %v, want %q", c.sql, rows, c.want)
		}
	}
}

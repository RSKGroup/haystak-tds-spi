// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"fmt"
	"strings"
	"testing"
)

// A string function given a date reads it as CAST with no style writes it: 'Nov  4 2026  2:05PM'.
func TestStringFunctionsOnDateTime(t *testing.T) {
	const d = "DATETIMEFROMPARTS(2026, 11, 4, 14, 5, 0, 0)"
	cases := []struct{ sql, want string }{
		{"SELECT LEFT(@d, 3)", "Nov"},
		{"SELECT RIGHT(@d, 6)", "2:05PM"},
		{"SELECT LEN(@d)", "19"},
		{"SELECT SUBSTRING(@d, 8, 4)", "2026"},
		{"SELECT REPLACE(@d, '2026', 'yyyy')", "Nov  4 yyyy  2:05PM"},
		{"SELECT UPPER(@d)", "NOV  4 2026  2:05PM"},
		{"SELECT LOWER(@d)", "nov  4 2026  2:05pm"},
		{"SELECT CHARINDEX('2026', @d)", "8"},
		{"SELECT PATINDEX('%2026%', @d)", "8"},
		{"SELECT STUFF(@d, 1, 3, 'Dec')", "Dec  4 2026  2:05PM"},
		{"SELECT REVERSE(@d)", "MP50:2  6202 4  voN"},
		{"SELECT LTRIM(RIGHT(@d, 7))", "2:05PM"},
		{"SELECT ASCII(@d)", "78"},
		{"SELECT REPLICATE(DATEFROMPARTS(1980, 5, 1), 2)", "1980-05-011980-05-01"},
		{"SELECT QUOTENAME(DATEFROMPARTS(1980, 5, 1))", "[1980-05-01]"},
		{"SELECT LEFT(born, 4) FROM typed", "2026"},
		{"SELECT FORMAT(@d, 'yyyy')", "2026"},
		{"SELECT DATEPART(year, @d)", "2026"},
	}
	for _, c := range cases {
		sql := strings.ReplaceAll(c.sql, "@d", d)
		rows := qry(t, sql)
		if len(rows) != 1 || fmt.Sprint(rows[0][0]) != c.want {
			t.Errorf("%s = %v, want %q", sql, rows, c.want)
		}
	}
}

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"fmt"
	"testing"
	"time"
)

func TestConvertDateWithStyle(t *testing.T) {
	day := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }
	cases := []struct {
		sql  string
		want any
	}{
		{"SELECT TRY_CONVERT(date, '05/01/1980', 101)", day(1980, 5, 1)},
		{"SELECT TRY_CONVERT(date, '1980-05-01', 23)", day(1980, 5, 1)},
		{"SELECT CONVERT(date, '05/01/1980', 101)", day(1980, 5, 1)},
		{"SELECT CONVERT(date, '01/05/1980', 103)", day(1980, 5, 1)},
		{"SELECT CONVERT(date, '1980-05-01', 101)", day(1980, 5, 1)},
		{"SELECT CAST('1980-05-01 10:30' AS date)", day(1980, 5, 1)},
		{"SELECT CONVERT(datetime, '01/01/2025')", day(2025, 1, 1)},
		{"SELECT TRY_CONVERT(date, '', 23)", day(1900, 1, 1)},
		{"SELECT DATEADD(day, 1, TRY_CONVERT(date, '05/01/1980', 101))", day(1980, 5, 2)},
		{"SELECT TRY_CONVERT(date, '02/30/1980', 101)", nil},
		{"SELECT TRY_CONVERT(date, 'not a date', 101)", nil},
	}
	for _, c := range cases {
		rows := qry(t, c.sql)
		if len(rows) != 1 {
			t.Fatalf("%s: %d rows", c.sql, len(rows))
		}
		got := rows[0][0]
		if want, ok := c.want.(time.Time); ok {
			if g, ok := got.(time.Time); !ok || !g.Equal(want) {
				t.Errorf("%s = %#v, want the date %s", c.sql, got, want.Format("2006-01-02"))
			}
			continue
		}
		if got != nil {
			t.Errorf("%s = %#v, want NULL", c.sql, got)
		}
	}
}

func TestConvertDateComparesAsDate(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"SELECT CASE WHEN TRY_CONVERT(date, '05/01/1980', 101) > TRY_CONVERT(date, '1979-12-31', 23) THEN 1 ELSE 0 END", "[[1]]"},
		{"SELECT COUNT(*) FROM products WHERE created >= '06/02/2026'", "[[1]]"},
		{"SELECT COUNT(*) FROM products WHERE created >= CONVERT(datetime, '06/02/2026') AND created < CONVERT(datetime, '06/03/2026')", "[[1]]"},
		{"SELECT YEAR('1980-05-01'), MONTH(CONVERT(date, '05/01/1980', 101))", "[[1980 5]]"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(qry(t, c.sql)); got != c.want {
			t.Errorf("%s\n  = %s, want %s", c.sql, got, c.want)
		}
	}
}

func TestConvertToCharWithStyle(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"SELECT CONVERT(char(10), made, 23) FROM items", "2026-06-10"},
		{"SELECT CONVERT(char(16), DATETIMEFROMPARTS(2027, 1, 1, 9, 30, 0, 0), 120)", "2027-01-01 09:30"},
		{"SELECT CONVERT(varchar(10), DATEFROMPARTS(1980, 5, 1), 101)", "05/01/1980"},
		{"SELECT CONVERT(varchar(10), DATEFROMPARTS(1980, 5, 1), 103)", "01/05/1980"},
		{"SELECT CONVERT(varchar(8), DATEFROMPARTS(1980, 5, 1), 112)", "19800501"},
		{"SELECT CONVERT(varchar(30), DATETIMEFROMPARTS(2026, 11, 4, 14, 5, 0, 0), 100)", "Nov  4 2026  2:05PM"},
		{"SELECT CONVERT(varchar(30), DATETIMEFROMPARTS(2026, 11, 4, 14, 5, 6, 7), 121)", "2026-11-04 14:05:06.007"},
		{"SELECT CONVERT(varchar(30), DATETIMEFROMPARTS(2026, 11, 4, 14, 5, 6, 0), 126)", "2026-11-04T14:05:06"},
		{"SELECT CAST('abcdef' AS varchar(3))", "abc"},
		{"SELECT CAST(12345 AS varchar(3))", "*"},
	}
	for _, c := range cases {
		rows := qry(t, c.sql)
		if len(rows) != 1 || fmt.Sprint(rows[0][0]) != c.want {
			t.Errorf("%s = %v, want %q", c.sql, rows, c.want)
		}
	}
}

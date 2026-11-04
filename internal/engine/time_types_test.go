// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"fmt"
	"regexp"
	"testing"
	"time"
)

func TestTimeAndOffsetConversions(t *testing.T) {
	clock := func(h, m, s int) time.Time { return time.Date(1900, 1, 1, h, m, s, 0, time.UTC) }
	cases := []struct {
		sql  string
		want time.Time
		off  int
	}{
		{"SELECT CAST('14:30' AS time)", clock(14, 30, 0), 0},
		{"SELECT CONVERT(time, '2:30 PM')", clock(14, 30, 0), 0},
		{"SELECT CAST(DATETIMEFROMPARTS(2026, 11, 4, 14, 5, 6, 0) AS time)", clock(14, 5, 6), 0},
		{"SELECT CAST('2026-11-04 10:00:00 +05:30' AS datetimeoffset)", time.Date(2026, 11, 4, 10, 0, 0, 0, time.FixedZone("", 19800)), 19800},
		{"SELECT CAST(DATETIMEFROMPARTS(2026, 11, 4, 10, 0, 0, 0) AS datetimeoffset)", time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC), 0},
	}
	for _, c := range cases {
		rows := qry(t, c.sql)
		got, ok := rows[0][0].(time.Time)
		if !ok || !got.Equal(c.want) {
			t.Errorf("%s = %#v, want %s", c.sql, rows[0][0], c.want)
			continue
		}
		if _, off := got.Zone(); off != c.off {
			t.Errorf("%s offset = %d, want %d", c.sql, off, c.off)
		}
	}
	more := []struct{ sql, want string }{
		{"SELECT DATEPART(hour, CAST('14:30' AS time))", "[[14]]"},
		{"SELECT CASE WHEN CAST('14:30' AS time) > CAST('9:00' AS time) THEN 1 ELSE 0 END", "[[1]]"},
		{"SELECT DATEPART(hour, CAST('2026-11-04 10:00:00 -03:00' AS datetimeoffset))", "[[10]]"},
		{"SELECT TRY_CONVERT(time, 'nope')", "[[<nil>]]"},
	}
	for _, c := range more {
		if got := fmt.Sprint(qry(t, c.sql)); got != c.want {
			t.Errorf("%s = %s, want %s", c.sql, got, c.want)
		}
	}
}

func TestDateTimeToTextWithoutStyle(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"SELECT CAST(DATETIMEFROMPARTS(2026, 11, 4, 14, 5, 0, 0) AS varchar(30))", "Nov  4 2026  2:05PM"},
		{"SELECT CONVERT(varchar(30), DATETIMEFROMPARTS(2026, 1, 15, 9, 7, 0, 0))", "Jan 15 2026  9:07AM"},
		{"SELECT CAST(made AS varchar(30)) FROM items", "Jun 10 2026 12:00AM"},
		{"SELECT CAST(born AS varchar(30)) FROM typed", "2026-01-01"},
		{"SELECT CAST(DATEFROMPARTS(1980, 5, 1) AS varchar(10))", "1980-05-01"},
		{"SELECT CAST(CAST('1980-05-01' AS date) AS varchar(30))", "1980-05-01"},
		{"SELECT CAST(CAST('14:30' AS time) AS varchar(30))", "14:30:00.0000000"},
		{"SELECT CAST(CAST('14:30:15.1234567' AS time(0)) AS varchar(30))", "14:30:15"},
		{"SELECT CAST(CAST('2026-11-04 14:30:15.5' AS datetime2) AS varchar(30))", "2026-11-04 14:30:15.5000000"},
		{"SELECT CAST(CAST('2026-11-04 14:30' AS datetime2(2)) AS varchar(30))", "2026-11-04 14:30:00.00"},
		{"SELECT CAST(CAST('2026-11-04 14:30' AS smalldatetime) AS varchar(30))", "Nov  4 2026  2:30PM"},
		{"SELECT CAST(CAST('2026-11-04 10:00:00 +05:30' AS datetimeoffset) AS varchar(40))", "2026-11-04 10:00:00.0000000 +05:30"},
		{"SELECT CONVERT(varchar(8), CAST('14:30' AS time), 108)", "14:30:00"},
		{"SELECT CONCAT('on ', DATEFROMPARTS(1980, 5, 1), ' at ', DATETIMEFROMPARTS(2026, 11, 4, 14, 5, 0, 0))", "on 1980-05-01 at Nov  4 2026  2:05PM"},
		{"SELECT CONCAT_WS('|', CAST('14:30' AS time), born) FROM typed", "14:30:00.0000000|2026-01-01"},
	}
	for _, c := range cases {
		rows := qry(t, c.sql)
		if len(rows) != 1 || fmt.Sprint(rows[0][0]) != c.want {
			t.Errorf("%s = %v, want %q", c.sql, rows, c.want)
		}
	}
	now := fmt.Sprint(qry(t, "SELECT CONVERT(varchar(30), GETDATE())")[0][0])
	if !regexp.MustCompile(`^[A-Z][a-z]{2} [ 1-3][0-9] \d{4} [ 1][0-9]:\d{2}[AP]M$`).MatchString(now) {
		t.Errorf("CONVERT(varchar, GETDATE()) = %q, want style 0 (mon dd yyyy hh:miAM)", now)
	}
}

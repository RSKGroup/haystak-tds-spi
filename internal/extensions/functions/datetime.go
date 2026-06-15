// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package functions

import (
	"strconv"
	"strings"
	"time"
)

func init() {
	register("YEAR", func(a []any) any { return datePart(a, func(t time.Time) int64 { return int64(t.Year()) }) })
	register("MONTH", func(a []any) any { return datePart(a, func(t time.Time) int64 { return int64(t.Month()) }) })
	register("DAY", func(a []any) any { return datePart(a, func(t time.Time) int64 { return int64(t.Day()) }) })
	register("GETDATE", utcNow)
	register("GETUTCDATE", utcNow)
	register("SYSDATETIME", utcNow)
	register("SYSUTCDATETIME", utcNow)
	register("SYSDATETIMEOFFSET", utcNow)
	register("CURRENT_TIMESTAMP", utcNow)
	register("DATEADD", dateAdd)
	register("DATEDIFF", dateDiff)
	register("DATEPART", func(a []any) any { return datePartNum(a) })
	register("DATENAME", dateName)
	register("EOMONTH", eomonth)
	register("ISDATE", isDate)
	register("DATETRUNC", dateTrunc)
	register("DATEDIFF_BIG", dateDiff) // same boundary count; bigint return
	register("DATEFROMPARTS", func(a []any) any {
		y, ok1 := argInt(a, 0)
		m, ok2 := argInt(a, 1)
		d, ok3 := argInt(a, 2)
		if !ok1 || !ok2 || !ok3 {
			return nil
		}
		return time.Date(int(y), time.Month(m), int(d), 0, 0, 0, 0, time.UTC)
	})
	register("DATETIMEFROMPARTS", func(a []any) any {
		if len(a) < 7 {
			return nil
		}
		y, _ := argInt(a, 0)
		m, _ := argInt(a, 1)
		d, _ := argInt(a, 2)
		h, _ := argInt(a, 3)
		mi, _ := argInt(a, 4)
		s, _ := argInt(a, 5)
		ms, _ := argInt(a, 6)
		return time.Date(int(y), time.Month(m), int(d), int(h), int(mi), int(s), int(ms)*1e6, time.UTC)
	})
	register("SWITCHOFFSET", func(a []any) any { // same instant, new displayed offset
		t, ok := a[0].(time.Time)
		off, ok2 := parseTZOffset(argStr(a, 1))
		if !ok || !ok2 {
			return nil
		}
		return t.In(time.FixedZone("", off))
	})
	register("TODATETIMEOFFSET", func(a []any) any { // attach an offset to a wall-clock datetime
		t, ok := a[0].(time.Time)
		off, ok2 := parseTZOffset(argStr(a, 1))
		if !ok || !ok2 {
			return nil
		}
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.FixedZone("", off))
	})
}

// parseTZOffset reads a timezone offset as signed minutes or "+HH:MM"/"-HH:MM", returning seconds.
func parseTZOffset(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if n, err := strconv.Atoi(s); err == nil {
		return n * 60, true
	}
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimLeft(s, "+-")
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, false
	}
	h, e1 := strconv.Atoi(parts[0])
	m, e2 := strconv.Atoi(parts[1])
	if e1 != nil || e2 != nil {
		return 0, false
	}
	off := (h*60 + m) * 60
	if neg {
		off = -off
	}
	return off, true
}

// dateTrunc is DATETRUNC: the start of the datepart containing the date.
func dateTrunc(a []any) any {
	if len(a) < 2 {
		return nil
	}
	t, ok := argTime(a, 1)
	if !ok {
		return nil
	}
	switch normDatePart(argStr(a, 0)) {
	case "year":
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	case "quarter":
		return time.Date(t.Year(), time.Month((quarter(t)-1)*3+1), 1, 0, 0, 0, 0, t.Location())
	case "month":
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	case "week":
		return dayFloor(t).AddDate(0, 0, -int(dayFloor(t).Weekday())) // back to Sunday
	case "day", "dayofyear", "weekday":
		return dayFloor(t)
	case "hour":
		return t.Truncate(time.Hour)
	case "minute":
		return t.Truncate(time.Minute)
	case "second":
		return t.Truncate(time.Second)
	case "millisecond":
		return t.Truncate(time.Millisecond)
	}
	return nil
}

func utcNow([]any) any { return time.Now().UTC() }

func datePart(a []any, f func(time.Time) int64) any {
	if len(a) >= 1 {
		if t, ok := a[0].(time.Time); ok {
			return f(t)
		}
	}
	return nil
}

func argTime(a []any, i int) (time.Time, bool) {
	if i >= len(a) {
		return time.Time{}, false
	}
	switch v := a[i].(type) {
	case time.Time:
		return v, true
	case string:
		return parseTime(v)
	}
	return time.Time{}, false
}

func parseTime(s string) (time.Time, bool) {
	for _, layout := range []string{
		"2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05", "2006-01-02T15:04:05",
		"2006-01-02", "15:04:05", time.RFC3339,
	} {
		if t, err := time.Parse(layout, strings.TrimSpace(s)); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// normDatePart folds datepart abbreviations (yy, mm, dd) to a canonical token.
func normDatePart(s string) string {
	switch strings.ToLower(s) {
	case "year", "yy", "yyyy":
		return "year"
	case "quarter", "qq", "q":
		return "quarter"
	case "month", "mm", "m":
		return "month"
	case "dayofyear", "dy", "y":
		return "dayofyear"
	case "day", "dd", "d":
		return "day"
	case "week", "wk", "ww":
		return "week"
	case "weekday", "dw", "w":
		return "weekday"
	case "hour", "hh":
		return "hour"
	case "minute", "mi", "n":
		return "minute"
	case "second", "ss", "s":
		return "second"
	case "millisecond", "ms":
		return "millisecond"
	}
	return strings.ToLower(s)
}

func dayFloor(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func quarter(t time.Time) int { return (int(t.Month())-1)/3 + 1 }

func daysInMonth(y int, m time.Month) int { return time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Day() }

// addMonths clamps the day to the target month end (T-SQL: month+1 of Jan 31 -> Feb 29).
func addMonths(t time.Time, n int) time.Time {
	total := int(t.Month()) - 1 + n
	ny := t.Year() + total/12
	nm := total % 12
	if nm < 0 {
		nm += 12
		ny--
	}
	month := time.Month(nm + 1)
	day := t.Day()
	if last := daysInMonth(ny, month); day > last {
		day = last
	}
	return time.Date(ny, month, day, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

func dateAdd(a []any) any {
	if len(a) < 3 {
		return nil
	}
	n, ok := argInt(a, 1)
	t, ok2 := argTime(a, 2)
	if !ok || !ok2 {
		return nil
	}
	switch normDatePart(argStr(a, 0)) {
	case "year":
		return addMonths(t, int(n)*12)
	case "quarter":
		return addMonths(t, int(n)*3)
	case "month":
		return addMonths(t, int(n))
	case "week":
		return t.AddDate(0, 0, int(n)*7)
	case "day", "dayofyear", "weekday":
		return t.AddDate(0, 0, int(n))
	case "hour":
		return t.Add(time.Duration(n) * time.Hour)
	case "minute":
		return t.Add(time.Duration(n) * time.Minute)
	case "second":
		return t.Add(time.Duration(n) * time.Second)
	case "millisecond":
		return t.Add(time.Duration(n) * time.Millisecond)
	}
	return nil
}

// dateDiff counts datepart boundaries crossed (T-SQL), not elapsed time.
func dateDiff(a []any) any {
	if len(a) < 3 {
		return nil
	}
	start, ok := argTime(a, 1)
	end, ok2 := argTime(a, 2)
	if !ok || !ok2 {
		return nil
	}
	switch normDatePart(argStr(a, 0)) {
	case "year":
		return int64(end.Year() - start.Year())
	case "quarter":
		return int64((end.Year()-start.Year())*4 + quarter(end) - quarter(start))
	case "month":
		return int64((end.Year()-start.Year())*12 + int(end.Month()) - int(start.Month()))
	case "week":
		return int64(dayFloor(end).Sub(dayFloor(start)) / (7 * 24 * time.Hour))
	case "day", "dayofyear", "weekday":
		return int64(dayFloor(end).Sub(dayFloor(start)) / (24 * time.Hour))
	case "hour":
		return int64(end.Truncate(time.Hour).Sub(start.Truncate(time.Hour)) / time.Hour)
	case "minute":
		return int64(end.Truncate(time.Minute).Sub(start.Truncate(time.Minute)) / time.Minute)
	case "second":
		return int64(end.Truncate(time.Second).Sub(start.Truncate(time.Second)) / time.Second)
	case "millisecond":
		return int64(end.Truncate(time.Millisecond).Sub(start.Truncate(time.Millisecond)) / time.Millisecond)
	}
	return nil
}

func datePartNum(a []any) any {
	if len(a) < 2 {
		return nil
	}
	t, ok := argTime(a, 1)
	if !ok {
		return nil
	}
	switch normDatePart(argStr(a, 0)) {
	case "year":
		return int64(t.Year())
	case "quarter":
		return int64(quarter(t))
	case "month":
		return int64(t.Month())
	case "day":
		return int64(t.Day())
	case "dayofyear":
		return int64(t.YearDay())
	case "weekday":
		return int64(t.Weekday()) + 1 // T-SQL default @@DATEFIRST=7: Sunday=1..Saturday=7
	case "week":
		_, wk := t.ISOWeek()
		return int64(wk)
	case "hour":
		return int64(t.Hour())
	case "minute":
		return int64(t.Minute())
	case "second":
		return int64(t.Second())
	case "millisecond":
		return int64(t.Nanosecond() / 1e6)
	}
	return nil
}

func dateName(a []any) any {
	if len(a) < 2 {
		return nil
	}
	t, ok := argTime(a, 1)
	if !ok {
		return nil
	}
	switch normDatePart(argStr(a, 0)) {
	case "month":
		return t.Month().String()
	case "weekday":
		return t.Weekday().String()
	}
	if n, ok := datePartNum(a).(int64); ok {
		return strconv.FormatInt(n, 10)
	}
	return nil
}

func eomonth(a []any) any {
	t, ok := argTime(a, 0)
	if !ok {
		return nil
	}
	n := 0
	if v, ok := argInt(a, 1); ok {
		n = int(v)
	}
	first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	t = addMonths(first, n)
	return time.Date(t.Year(), t.Month(), daysInMonth(t.Year(), t.Month()), 0, 0, 0, 0, t.Location())
}

func isDate(a []any) any {
	if len(a) == 0 || a[0] == nil {
		return int64(0)
	}
	switch v := a[0].(type) {
	case time.Time:
		return int64(1)
	case string:
		if _, ok := parseTime(v); ok {
			return int64(1)
		}
	}
	return int64(0)
}

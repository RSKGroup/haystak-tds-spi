// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package functions

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// NoStyle is the style of a conversion written without one (CAST, or CONVERT with two arguments).
const NoStyle = -1

var (
	reYearFirst = regexp.MustCompile(`^(\d{4})[-/.](\d{1,2})[-/.](\d{1,2})(?:[ T]+(.+))?$`)
	reNumeric   = regexp.MustCompile(`^(\d{1,2})[-/.](\d{1,2})[-/.](\d{4}|\d{2})(?:\s+(.+))?$`)
	reUnsep     = regexp.MustCompile(`^(\d{8}|\d{6}|\d{4})(?:[ T]+(.+))?$`)
	reMonFirst  = regexp.MustCompile(`^([A-Za-z]{3,9})\.?\s+(\d{1,2}),?\s+(\d{4}|\d{2})(?:\s+(.+))?$`)
	reDayFirst  = regexp.MustCompile(`^(\d{1,2})\s+([A-Za-z]{3,9})\.?,?\s+(\d{4}|\d{2})(?:\s+(.+))?$`)
	reClock     = regexp.MustCompile(`^(\d{1,2})(?::(\d{1,2}))?(?::(\d{1,2}))?(?:[.:](\d{1,9}))?\s*([AaPp][Mm])?Z?$`)
)

var monthAbbr = [...]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

// styleOrder is the field order a style reads m/d/y text in; year-first text reads the same under every style.
func styleOrder(style int) string {
	switch style {
	case 3, 103, 4, 104, 5, 105, 6, 106, 13, 113, 130, 131:
		return "dmy"
	case 2, 102, 11, 111, 12, 112, 20, 120, 21, 121, 23, 25, 126, 127:
		return "ymd"
	}
	return "mdy" // us_english's DATEFORMAT, which is also what an unstyled conversion uses
}

// ParseDate reads text as CONVERT does with style (NoStyle: the mdy default); empty text is 1900-01-01, as in SQL Server.
func ParseDate(s string, style int) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), true
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, true
	}
	if m := reYearFirst.FindStringSubmatch(s); m != nil {
		return dateAt(atoi(m[1]), atoi(m[2]), atoi(m[3]), m[4])
	}
	if m := reNumeric.FindStringSubmatch(s); m != nil {
		a, b, y := atoi(m[1]), atoi(m[2]), fullYear(m[3])
		switch styleOrder(style) {
		case "mdy":
			return dateAt(y, a, b, m[4])
		case "dmy":
			return dateAt(y, b, a, m[4])
		}
		if len(m[3]) == 2 { // yy/mm/dd under a year-first style
			return dateAt(fullYear(m[1]), b, atoi(m[3]), m[4])
		}
		return time.Time{}, false
	}
	if m := reUnsep.FindStringSubmatch(s); m != nil {
		d := m[1]
		switch len(d) {
		case 8:
			return dateAt(atoi(d[:4]), atoi(d[4:6]), atoi(d[6:]), m[2])
		case 6:
			return dateAt(fullYear(d[:2]), atoi(d[2:4]), atoi(d[4:]), m[2])
		}
		return dateAt(atoi(d), 1, 1, m[2])
	}
	if m := reMonFirst.FindStringSubmatch(s); m != nil {
		if mon, ok := monthNumber(m[1]); ok {
			return dateAt(fullYear(m[3]), mon, atoi(m[2]), m[4])
		}
	}
	if m := reDayFirst.FindStringSubmatch(s); m != nil {
		if mon, ok := monthNumber(m[2]); ok {
			return dateAt(fullYear(m[3]), mon, atoi(m[1]), m[4])
		}
	}
	if strings.Contains(s, ":") || strings.HasSuffix(strings.ToUpper(s), "M") {
		return dateAt(1900, 1, 1, s) // a time alone falls on SQL Server's base date
	}
	return time.Time{}, false
}

func dateAt(y, m, d int, clock string) (time.Time, bool) {
	if y < 1 || y > 9999 || m < 1 || m > 12 || d < 1 || d > daysInMonth(y, time.Month(m)) {
		return time.Time{}, false
	}
	var h, mi, sec, ns int
	if clock = strings.TrimSpace(clock); clock != "" {
		c := reClock.FindStringSubmatch(clock)
		if c == nil || (c[2] == "" && c[5] == "") {
			return time.Time{}, false
		}
		h, mi, sec = atoi(c[1]), atoi(c[2]), atoi(c[3])
		if c[4] != "" {
			frac := (c[4] + "000000000")[:9]
			ns = atoi(frac)
		}
		switch strings.ToUpper(c[5]) {
		case "AM":
			if h == 12 {
				h = 0
			}
		case "PM":
			if h < 12 {
				h += 12
			}
		}
		if h > 23 || mi > 59 || sec > 59 {
			return time.Time{}, false
		}
	}
	return time.Date(y, time.Month(m), d, h, mi, sec, ns, time.UTC), true
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// fullYear widens a two-digit year by SQL Server's default cutoff: 00-49 are 20xx, 50-99 are 19xx.
func fullYear(s string) int {
	y := atoi(s)
	if len(s) == 2 {
		if y < 50 {
			return 2000 + y
		}
		return 1900 + y
	}
	return y
}

func monthNumber(s string) (int, bool) {
	if len(s) < 3 {
		return 0, false
	}
	for i, m := range monthAbbr {
		if strings.EqualFold(s[:3], m) {
			return i + 1, true
		}
	}
	return 0, false
}

// FormatDate writes t as CONVERT(varchar, t, style) does; ok is false for a style it does not know.
func FormatDate(t time.Time, style int) (string, bool) {
	y, mo, d := t.Date()
	h, mi, s := t.Clock()
	ms := t.Nanosecond() / 1e6
	yy, mon := y%100, monthAbbr[mo-1]
	h12, ampm := h%12, "AM"
	if h12 == 0 {
		h12 = 12
	}
	if h >= 12 {
		ampm = "PM"
	}
	switch style {
	case 0, 100:
		return fmt.Sprintf("%s %2d %d %2d:%02d%s", mon, d, y, h12, mi, ampm), true
	case 1:
		return fmt.Sprintf("%02d/%02d/%02d", mo, d, yy), true
	case 101:
		return fmt.Sprintf("%02d/%02d/%04d", mo, d, y), true
	case 2:
		return fmt.Sprintf("%02d.%02d.%02d", yy, mo, d), true
	case 102:
		return fmt.Sprintf("%04d.%02d.%02d", y, mo, d), true
	case 3:
		return fmt.Sprintf("%02d/%02d/%02d", d, mo, yy), true
	case 103:
		return fmt.Sprintf("%02d/%02d/%04d", d, mo, y), true
	case 4:
		return fmt.Sprintf("%02d.%02d.%02d", d, mo, yy), true
	case 104:
		return fmt.Sprintf("%02d.%02d.%04d", d, mo, y), true
	case 5:
		return fmt.Sprintf("%02d-%02d-%02d", d, mo, yy), true
	case 105:
		return fmt.Sprintf("%02d-%02d-%04d", d, mo, y), true
	case 6:
		return fmt.Sprintf("%02d %s %02d", d, mon, yy), true
	case 106:
		return fmt.Sprintf("%02d %s %04d", d, mon, y), true
	case 7:
		return fmt.Sprintf("%s %02d, %02d", mon, d, yy), true
	case 107:
		return fmt.Sprintf("%s %02d, %04d", mon, d, y), true
	case 8, 24, 108:
		return fmt.Sprintf("%02d:%02d:%02d", h, mi, s), true
	case 9, 109:
		return fmt.Sprintf("%s %2d %d %2d:%02d:%02d:%03d%s", mon, d, y, h12, mi, s, ms, ampm), true
	case 10:
		return fmt.Sprintf("%02d-%02d-%02d", mo, d, yy), true
	case 110:
		return fmt.Sprintf("%02d-%02d-%04d", mo, d, y), true
	case 11:
		return fmt.Sprintf("%02d/%02d/%02d", yy, mo, d), true
	case 111:
		return fmt.Sprintf("%04d/%02d/%02d", y, mo, d), true
	case 12:
		return fmt.Sprintf("%02d%02d%02d", yy, mo, d), true
	case 112:
		return fmt.Sprintf("%04d%02d%02d", y, mo, d), true
	case 13, 113:
		return fmt.Sprintf("%02d %s %04d %02d:%02d:%02d:%03d", d, mon, y, h, mi, s, ms), true
	case 14, 114:
		return fmt.Sprintf("%02d:%02d:%02d:%03d", h, mi, s, ms), true
	case 20, 120:
		return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", y, mo, d, h, mi, s), true
	case 21, 25, 121:
		return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d.%03d", y, mo, d, h, mi, s, ms), true
	case 22:
		return fmt.Sprintf("%02d/%02d/%02d %2d:%02d:%02d %s", mo, d, yy, h12, mi, s, ampm), true
	case 23:
		return fmt.Sprintf("%04d-%02d-%02d", y, mo, d), true
	case 126, 127:
		out := fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d", y, mo, d, h, mi, s)
		if ms != 0 {
			out += fmt.Sprintf(".%03d", ms)
		}
		if style == 127 {
			out += "Z"
		}
		return out, true
	}
	return "", false
}

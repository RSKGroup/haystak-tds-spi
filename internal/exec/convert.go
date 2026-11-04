// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package exec

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/functions"
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

// convertCall evaluates CAST/CONVERT/TRY_CONVERT: the value, then CONVERT's style when one is given.
func convertCall(idx map[string]int, row []any, ve *tds.ValueExpr, env *Env) (any, error) {
	v, err := evalValue(idx, row, ve.Left, env)
	if err != nil {
		return nil, err
	}
	style := functions.NoStyle
	if len(ve.Args) > 0 {
		sv, err := evalValue(idx, row, ve.Args[0], env)
		if err != nil {
			return nil, err
		}
		if n, ok := toInt(sv); ok {
			style = int(n)
		}
	}
	if t, ok := v.(time.Time); ok && style == functions.NoStyle && textType(ve.Cast) {
		src, scale := temporalType(ve.Left, env)
		return convertValue(defaultText(t, src, scale), ve.Cast, ve.CastLen, style), nil
	}
	return convertValue(v, ve.Cast, ve.CastLen, style), nil
}

func textType(typ string) bool {
	switch typ {
	case "VARCHAR", "NVARCHAR", "CHAR", "NCHAR", "TEXT", "NTEXT":
		return true
	}
	return false
}

func convertValue(v any, typ string, length, style int) any {
	if v == nil {
		return nil
	}
	switch typ {
	case "DATE", "DATETIME", "DATETIME2", "SMALLDATETIME", "TIME", "DATETIMEOFFSET":
		return toDateValue(v, typ, length, style)
	}
	if textType(typ) {
		s := toStr(v)
		if t, ok := v.(time.Time); ok && style != functions.NoStyle {
			if f, ok := functions.FormatDate(t, style); ok {
				s = f
			}
		}
		s = fitLength(s, v, length)
		if (typ == "CHAR" || typ == "NCHAR") && length > 0 {
			if n := utf8.RuneCountInString(s); n < length {
				s += strings.Repeat(" ", length-n)
			}
		}
		return s
	}
	return castValue(v, typ)
}

var reOffset = regexp.MustCompile(`\s*(Z|[+-]\d{1,2}:\d{2})$`)

// toDateValue reads v as a date/time type: DATE drops the clock, TIME keeps only it (on 1900-01-01), DATETIMEOFFSET a zone.
func toDateValue(v any, typ string, scale, style int) any {
	var t time.Time
	switch x := v.(type) {
	case time.Time:
		t = x
	case string:
		off, hasOff := 0, false
		if typ == "DATETIMEOFFSET" {
			if m := reOffset.FindStringSubmatchIndex(x); m != nil {
				off, hasOff = offsetSeconds(x[m[2]:m[3]]), true
				x = x[:m[0]]
			}
		}
		p, ok := functions.ParseDate(x, style)
		if !ok {
			return nil
		}
		t = p
		if hasOff {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.FixedZone("", off))
		}
	case int64:
		if typ != "DATETIME" && typ != "SMALLDATETIME" {
			return nil // SQL Server converts a number to datetime and smalldatetime only
		}
		t = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(x))
	default:
		return nil
	}
	switch typ {
	case "DATE":
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	case "TIME":
		t = roundScale(t, scale)
		return time.Date(1900, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
	case "DATETIME2":
		return roundScale(t, scale)
	case "DATETIMEOFFSET":
		if _, off := t.Zone(); off == 0 {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.FixedZone("", 0))
		}
		return roundScale(t, scale)
	}
	return t
}

func offsetSeconds(s string) int {
	if s == "Z" {
		return 0
	}
	sign := 1
	if s[0] == '-' {
		sign = -1
	}
	h, m, _ := strings.Cut(s[1:], ":")
	hh, _ := strconv.Atoi(h)
	mm, _ := strconv.Atoi(m)
	return sign * (hh*3600 + mm*60)
}

// roundScale rounds t to scale fractional-second digits, as a time(n) or datetime2(n) value holds it.
func roundScale(t time.Time, scale int) time.Time {
	if scale < 0 || scale >= 9 {
		return t
	}
	unit := time.Duration(1)
	for i := scale; i < 9; i++ {
		unit *= 10
	}
	return t.Round(unit)
}

// temporalType is the SQL type a datetime-valued expression has, with its scale, for CAST's default text.
func temporalType(ve *tds.ValueExpr, env *Env) (string, int) {
	if ve == nil {
		return "datetime", 3
	}
	switch ve.Kind {
	case tds.ValCast:
		switch ve.Cast {
		case "DATE", "SMALLDATETIME", "DATETIME":
			return strings.ToLower(ve.Cast), 3
		case "TIME", "DATETIME2", "DATETIMEOFFSET":
			return strings.ToLower(ve.Cast), ve.CastLen
		}
	case tds.ValCol:
		if env != nil && env.colType != nil {
			if t, ok := env.colType(ve.Col); ok && t.Kind == types.Time {
				switch n := strings.ToLower(t.Name); n {
				case "date", "smalldatetime", "time", "datetime2", "datetimeoffset":
					return n, 7
				}
			}
		}
	case tds.ValFunc:
		switch ve.Func {
		case "SYSDATETIME", "SYSUTCDATETIME":
			return "datetime2", 7
		case "SYSDATETIMEOFFSET", "SWITCHOFFSET", "TODATETIMEOFFSET":
			return "datetimeoffset", 7
		case "DATEFROMPARTS", "EOMONTH":
			return "date", 0
		case "DATEADD", "DATETRUNC":
			if len(ve.Args) > 0 {
				return temporalType(ve.Args[len(ve.Args)-1], env)
			}
		case "ISNULL", "COALESCE", "MIN", "MAX":
			if len(ve.Args) > 0 {
				return temporalType(ve.Args[0], env)
			}
		}
	}
	return "datetime", 3
}

// defaultText is CAST's text with no style: style 0 for datetime/smalldatetime, ODBC form with the scale's digits otherwise.
func defaultText(t time.Time, typ string, scale int) string {
	switch typ {
	case "date":
		return t.Format("2006-01-02")
	case "time":
		return t.Format("15:04:05") + fraction(t, scale)
	case "datetime2":
		return t.Format("2006-01-02 15:04:05") + fraction(t, scale)
	case "datetimeoffset":
		return t.Format("2006-01-02 15:04:05") + fraction(t, scale) + t.Format(" -07:00")
	}
	s, _ := functions.FormatDate(t, 0)
	return s
}

// concatText writes CONCAT's date/time arguments as CAST with no style would, before they are joined.
func concatText(exprs []*tds.ValueExpr, args []any, env *Env) {
	for i, a := range args {
		if t, ok := a.(time.Time); ok && i < len(exprs) {
			src, scale := temporalType(exprs[i], env)
			args[i] = defaultText(t, src, scale)
		}
	}
}

func fraction(t time.Time, scale int) string {
	if scale <= 0 {
		return ""
	}
	if scale > 7 {
		scale = 7
	}
	return "." + fmt.Sprintf("%09d", t.Nanosecond())[:scale]
}

// fitLength cuts text to a declared length; a number too wide becomes "*", as SQL Server writes it.
func fitLength(s string, src any, length int) string {
	if length <= 0 || utf8.RuneCountInString(s) <= length {
		return s
	}
	if _, ok := toFloatOk(src); ok {
		return "*"
	}
	return string([]rune(s)[:length])
}

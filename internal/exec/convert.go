// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package exec

import (
	"time"
	"unicode/utf8"

	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/functions"
	"github.com/RSKGroup/haystak-tds-spi/tds"
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
	return convertValue(v, ve.Cast, ve.CastLen, style), nil
}

func convertValue(v any, typ string, length, style int) any {
	if v == nil {
		return nil
	}
	switch typ {
	case "DATE", "DATETIME", "DATETIME2", "SMALLDATETIME":
		return toDateValue(v, typ, style)
	case "VARCHAR", "NVARCHAR", "CHAR", "NCHAR", "TEXT", "NTEXT":
		s := toStr(v)
		if t, ok := v.(time.Time); ok && style != functions.NoStyle {
			if f, ok := functions.FormatDate(t, style); ok {
				s = f
			}
		}
		return fitLength(s, v, length)
	}
	return castValue(v, typ)
}

// toDateValue converts to a date type; text is read with the style, and DATE drops the time of day.
func toDateValue(v any, typ string, style int) any {
	var t time.Time
	switch x := v.(type) {
	case time.Time:
		t = x
	case string:
		p, ok := functions.ParseDate(x, style)
		if !ok {
			return nil
		}
		t = p
	case int64:
		if typ == "DATE" {
			return nil // SQL Server has no int to date conversion
		}
		t = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(x))
	default:
		return nil
	}
	if typ == "DATE" {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}
	return t
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

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package batch

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// DeclaredTextType makes a declared char/varchar/nchar/nvarchar length explicit: none means 1 for a variable, not CAST's 30.
func DeclaredTextType(typ string) (string, bool) {
	t := strings.TrimSpace(typ)
	word := strings.ToUpper(firstWord(strings.ReplaceAll(t, "(", " (")))
	switch word {
	case "CHAR", "VARCHAR", "NCHAR", "NVARCHAR":
	default:
		return "", false
	}
	if strings.Contains(t, "(") {
		return t, true
	}
	return word + "(1)", true
}

// fitDeclared fits a DECLARE/SET value to a declared text type: a literal is cut (char padded), anything else CAST.
func fitDeclared(value, typ string) string {
	cast, ok := DeclaredTextType(typ)
	v := strings.TrimSpace(value)
	if !ok || v == "" || strings.EqualFold(v, "NULL") {
		return value
	}
	n, pad := declaredLength(cast)
	if s, prefix, isLit := unquote(v); isLit {
		return prefix + quote(fitText(s, n, pad, false))
	}
	if _, err := strconv.ParseFloat(v, 64); err == nil {
		return quote(fitText(v, n, pad, true))
	}
	return "CAST((" + v + ") AS " + cast + ")"
}

func declaredLength(cast string) (n int, pad bool) {
	upper := strings.ToUpper(cast)
	pad = strings.HasPrefix(upper, "CHAR") || strings.HasPrefix(upper, "NCHAR")
	open, close := strings.IndexByte(cast, '('), strings.IndexByte(cast, ')')
	if open < 0 || close < open {
		return -1, pad
	}
	inner := strings.TrimSpace(cast[open+1 : close])
	if strings.EqualFold(inner, "MAX") {
		return -1, pad
	}
	n, err := strconv.Atoi(inner)
	if err != nil {
		return -1, pad
	}
	return n, pad
}

// fitText cuts s to n characters (a number too wide becomes "*", as SQL Server writes it) and pads char types.
func fitText(s string, n int, pad, number bool) string {
	if n < 0 {
		return s
	}
	if utf8.RuneCountInString(s) > n {
		if number {
			s = "*"
		} else {
			s = string([]rune(s)[:n])
		}
	}
	if pad {
		if k := utf8.RuneCountInString(s); k < n {
			s += strings.Repeat(" ", n-k)
		}
	}
	return s
}

// unquote reads a '...' or N'...' literal; ok is false for anything else.
func unquote(v string) (s, prefix string, ok bool) {
	if len(v) > 0 && (v[0] == 'N' || v[0] == 'n') {
		prefix, v = v[:1], v[1:]
	}
	if len(v) < 2 || v[0] != '\'' || skipString(v, 0) != len(v) {
		return "", "", false
	}
	return strings.ReplaceAll(v[1:len(v)-1], "''", "'"), prefix, true
}

func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

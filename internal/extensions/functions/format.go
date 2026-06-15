// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package functions

import (
	"strconv"
	"strings"
	"time"
)

func init() {
	// FORMAT(value, format): .NET-style formatting of a number or datetime. Standard numeric specifiers
	// (N/F/D/C/P/X) plus simple custom patterns; datetime via .NET token mapping (yyyy/MM/dd/HH/mm/ss/…).
	register("FORMAT", func(a []any) any {
		if len(a) < 2 || a[0] == nil {
			return nil
		}
		f := argStr(a, 1)
		if t, ok := a[0].(time.Time); ok {
			return formatDate(t, f)
		}
		if n, ok := numOf(a[0]); ok {
			return formatNumber(n, f)
		}
		return nil
	})
}

func numOf(v any) (float64, bool) {
	switch n := v.(type) {
	case int64:
		return float64(n), true
	case int:
		return float64(n), true
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case string:
		f, err := strconv.ParseFloat(n, 64)
		return f, err == nil
	}
	return 0, false
}

func formatNumber(n float64, f string) any {
	if f == "" {
		return strconv.FormatFloat(n, 'g', -1, 64)
	}
	digits := -1
	if len(f) > 1 {
		if d, err := strconv.Atoi(f[1:]); err == nil {
			digits = d
		}
	}
	switch f[0] {
	case 'N', 'n':
		return addThousands(strconv.FormatFloat(n, 'f', orDefault(digits, 2), 64))
	case 'F', 'f':
		return strconv.FormatFloat(n, 'f', orDefault(digits, 2), 64)
	case 'C', 'c':
		return "$" + addThousands(strconv.FormatFloat(n, 'f', orDefault(digits, 2), 64))
	case 'P', 'p':
		return strconv.FormatFloat(n*100, 'f', orDefault(digits, 2), 64) + " %"
	case 'D', 'd':
		return padInt(int64(n), orDefault(digits, 0))
	case 'X':
		return strings.ToUpper(strconv.FormatInt(int64(n), 16))
	case 'x':
		return strconv.FormatInt(int64(n), 16)
	}
	// custom pattern: decimals from chars after '.', thousands if ',' present
	dec := 0
	if dot := strings.IndexByte(f, '.'); dot >= 0 {
		for _, c := range f[dot+1:] {
			if c == '0' || c == '#' {
				dec++
			}
		}
	}
	out := strconv.FormatFloat(n, 'f', dec, 64)
	if strings.Contains(f, ",") {
		out = addThousands(out)
	}
	return out
}

func orDefault(v, def int) int {
	if v < 0 {
		return def
	}
	return v
}

func padInt(n int64, width int) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	for len(s) < width {
		s = "0" + s
	}
	if neg {
		s = "-" + s
	}
	return s
}

func addThousands(s string) string {
	sign := ""
	if strings.HasPrefix(s, "-") {
		sign, s = "-", s[1:]
	}
	intPart, frac := s, ""
	if dot := strings.IndexByte(s, '.'); dot >= 0 {
		intPart, frac = s[:dot], s[dot:]
	}
	var b strings.Builder
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return sign + b.String() + frac
}

// dateTokens maps .NET datetime tokens to Go reference-layout fragments, longest-first per letter.
var dateTokens = []struct{ net, layout string }{
	{"yyyy", "2006"}, {"yy", "06"},
	{"MMMM", "January"}, {"MMM", "Jan"}, {"MM", "01"}, {"M", "1"},
	{"dddd", "Monday"}, {"ddd", "Mon"}, {"dd", "02"}, {"d", "2"},
	{"HH", "15"}, {"hh", "03"}, {"h", "3"},
	{"mm", "04"}, {"ss", "05"}, {"tt", "PM"},
}

func formatDate(t time.Time, f string) string {
	var b strings.Builder
	for i := 0; i < len(f); {
		matched := false
		for _, tok := range dateTokens {
			if strings.HasPrefix(f[i:], tok.net) {
				b.WriteString(t.Format(tok.layout))
				i += len(tok.net)
				matched = true
				break
			}
		}
		if !matched {
			b.WriteByte(f[i])
			i++
		}
	}
	return b.String()
}

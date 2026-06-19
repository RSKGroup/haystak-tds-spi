// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package functions

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func init() {
	register("LEN", func(a []any) any { return int64(len(argStr(a, 0))) })
	register("DATALEN", func(a []any) any { return int64(len(argStr(a, 0))) })
	register("UPPER", func(a []any) any { return strings.ToUpper(argStr(a, 0)) })
	register("LOWER", func(a []any) any { return strings.ToLower(argStr(a, 0)) })
	register("LTRIM", func(a []any) any { return strings.TrimLeft(argStr(a, 0), " ") })
	register("RTRIM", func(a []any) any { return strings.TrimRight(argStr(a, 0), " ") })
	register("TRIM", func(a []any) any { return strings.TrimSpace(argStr(a, 0)) })
	register("CONCAT", func(a []any) any {
		var b strings.Builder
		for _, v := range a {
			if v != nil {
				b.WriteString(toStr(v))
			}
		}
		return b.String()
	})
	register("REPLACE", func(a []any) any {
		if len(a) == 3 {
			return strings.ReplaceAll(toStr(a[0]), toStr(a[1]), toStr(a[2]))
		}
		return nil
	})
	register("SUBSTRING", func(a []any) any {
		if len(a) == 3 {
			start, _ := argInt(a, 1)
			length, _ := argInt(a, 2)
			return substr(argStr(a, 0), int(start), int(length))
		}
		return nil
	})
	register("QUOTENAME", func(a []any) any {
		if len(a) < 1 {
			return nil
		}
		s := toStr(a[0])
		q := "["
		if len(a) >= 2 {
			q = argStr(a, 1)
		}
		switch q {
		case "'":
			return "'" + strings.ReplaceAll(s, "'", "''") + "'"
		case "\"":
			return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
		}
		return "[" + strings.ReplaceAll(s, "]", "]]") + "]"
	})
	register("CHARINDEX", func(a []any) any {
		if len(a) < 2 {
			return nil
		}
		sub, s := argStr(a, 0), argStr(a, 1)
		if sub == "" {
			return int64(0)
		}
		start := 1
		if n, ok := argInt(a, 2); ok && n > 1 {
			start = int(n)
		}
		if start > len(s) {
			return int64(0)
		}
		idx := strings.Index(s[start-1:], sub)
		if idx < 0 {
			return int64(0)
		}
		return int64(start + idx)
	})
	register("LEFT", func(a []any) any { return leftRight(a, true) })
	register("RIGHT", func(a []any) any { return leftRight(a, false) })
	register("REPLICATE", func(a []any) any {
		n, ok := argInt(a, 1)
		if !ok || n <= 0 {
			if !ok {
				return nil
			}
			return ""
		}
		return strings.Repeat(argStr(a, 0), int(n))
	})
	register("STUFF", func(a []any) any {
		if len(a) < 4 {
			return nil
		}
		s := argStr(a, 0)
		start, _ := argInt(a, 1)
		length, _ := argInt(a, 2)
		if start < 1 || int(start) > len(s) || length < 0 {
			return nil
		}
		st, end := int(start)-1, int(start)-1+int(length)
		if end > len(s) {
			end = len(s)
		}
		return s[:st] + argStr(a, 3) + s[end:]
	})
	register("REVERSE", func(a []any) any {
		r := []rune(argStr(a, 0))
		for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
			r[i], r[j] = r[j], r[i]
		}
		return string(r)
	})
	register("SPACE", func(a []any) any {
		n, ok := argInt(a, 0)
		if !ok || n <= 0 {
			if !ok {
				return nil
			}
			return ""
		}
		return strings.Repeat(" ", int(n))
	})
	register("ASCII", func(a []any) any {
		s := argStr(a, 0)
		if s == "" {
			return nil
		}
		return int64(s[0])
	})
	register("CHAR", func(a []any) any {
		n, ok := argInt(a, 0)
		if !ok || n < 0 || n > 255 {
			return nil
		}
		return string([]byte{byte(n)})
	})
	register("UNICODE", func(a []any) any {
		s := argStr(a, 0)
		if s == "" {
			return nil
		}
		return int64([]rune(s)[0])
	})
	register("NCHAR", func(a []any) any {
		n, ok := argInt(a, 0)
		if !ok || n < 0 || n > 0x10FFFF {
			return nil
		}
		return string(rune(n))
	})
	register("PATINDEX", func(a []any) any {
		if len(a) < 2 {
			return nil
		}
		return patIndex(argStr(a, 0), argStr(a, 1))
	})
	register("CONCAT_WS", func(a []any) any {
		if len(a) < 1 {
			return nil
		}
		sep := argText(a[0])
		var parts []string
		for _, v := range a[1:] {
			if v != nil {
				parts = append(parts, argText(v))
			}
		}
		return strings.Join(parts, sep)
	})
	register("TRANSLATE", func(a []any) any {
		if len(a) < 3 {
			return nil
		}
		from, to := []rune(argStr(a, 1)), []rune(argStr(a, 2))
		if len(from) != len(to) {
			return nil
		}
		m := make(map[rune]rune, len(from))
		for i, c := range from {
			m[c] = to[i]
		}
		out := []rune(argStr(a, 0))
		for i, c := range out {
			if r, ok := m[c]; ok {
				out[i] = r
			}
		}
		return string(out)
	})
	register("STR", func(a []any) any {
		f, ok := toFloatOk(arg0(a))
		if !ok {
			return nil
		}
		length, dec := 10, 0
		if n, ok := argInt(a, 1); ok {
			length = int(n)
		}
		if n, ok := argInt(a, 2); ok {
			dec = int(n)
		}
		s := strconv.FormatFloat(f, 'f', dec, 64)
		if len(s) > length {
			return strings.Repeat("*", length) // does not fit -> asterisks, as SQL Server does
		}
		return fmt.Sprintf("%*s", length, s)
	})
	register("STRING_ESCAPE", func(a []any) any {
		if len(a) < 2 || a[0] == nil || !strings.EqualFold(argStr(a, 1), "json") {
			return nil
		}
		b, err := json.Marshal(argStr(a, 0))
		if err != nil {
			return nil
		}
		return string(b[1 : len(b)-1]) // strip the quotes Marshal wraps around the escaped text
	})
	register("SOUNDEX", func(a []any) any { return soundex(argStr(a, 0)) })
	register("DIFFERENCE", func(a []any) any { return difference(argStr(a, 0), argStr(a, 1)) })
	register("FORMATMESSAGE", func(a []any) any {
		if len(a) == 0 {
			return nil
		}
		return formatMessage(argStr(a, 0), a[1:])
	})
}

// soundex is the standard four-character phonetic code: first letter plus three consonant digits.
func soundex(s string) string {
	s = strings.ToUpper(s)
	var first byte
	i := 0
	for ; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			first = s[i]
			i++
			break
		}
	}
	if first == 0 {
		return ""
	}
	out := []byte{first}
	prev := soundexCode(first)
	for ; i < len(s) && len(out) < 4; i++ {
		c := s[i]
		if c < 'A' || c > 'Z' {
			continue
		}
		d := soundexCode(c)
		if d != '0' {
			if d != prev {
				out = append(out, d)
			}
			prev = d
		} else if c != 'H' && c != 'W' {
			prev = '0' // a vowel separates equal codes; H/W do not
		}
	}
	for len(out) < 4 {
		out = append(out, '0')
	}
	return string(out)
}

func soundexCode(c byte) byte {
	switch c {
	case 'B', 'F', 'P', 'V':
		return '1'
	case 'C', 'G', 'J', 'K', 'Q', 'S', 'X', 'Z':
		return '2'
	case 'D', 'T':
		return '3'
	case 'L':
		return '4'
	case 'M', 'N':
		return '5'
	case 'R':
		return '6'
	}
	return '0'
}

func difference(a, b string) int64 {
	x, y := soundex(a), soundex(b)
	for len(x) < 4 {
		x += "0"
	}
	for len(y) < 4 {
		y += "0"
	}
	var n int64
	for i := 0; i < 4; i++ {
		if x[i] == y[i] {
			n++
		}
	}
	return n
}

// formatMessage substitutes printf-style %s/%d/%i/%u/%x placeholders (and %%) with the trailing args.
func formatMessage(msg string, args []any) string {
	var b strings.Builder
	ai := 0
	for i := 0; i < len(msg); i++ {
		if msg[i] != '%' || i+1 >= len(msg) {
			b.WriteByte(msg[i])
			continue
		}
		c := msg[i+1]
		switch {
		case c == '%':
			b.WriteByte('%')
		case strings.IndexByte("sdiux", c) >= 0 && ai < len(args):
			if c == 's' {
				b.WriteString(argText(args[ai]))
			} else if n, ok := toInt64(args[ai]); ok {
				if c == 'x' {
					b.WriteString(strconv.FormatInt(n, 16))
				} else {
					b.WriteString(strconv.FormatInt(n, 10))
				}
			}
			ai++
		default:
			b.WriteByte('%')
			b.WriteByte(c)
		}
		i++
	}
	return b.String()
}

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	}
	return 0, false
}

func argText(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case nil:
		return ""
	}
	return fmt.Sprint(v)
}

func leftRight(a []any, left bool) any {
	n, ok := argInt(a, 1)
	if !ok {
		return nil
	}
	s := argStr(a, 0)
	if n <= 0 {
		return ""
	}
	if int(n) >= len(s) {
		return s
	}
	if left {
		return s[:n]
	}
	return s[len(s)-int(n):]
}

// patIndex is PATINDEX: 1-based start of the first LIKE-pattern match, else 0.
func patIndex(pattern, s string) int64 {
	re, err := regexp.Compile(likeToRegexp(pattern))
	if err != nil {
		return 0
	}
	if loc := re.FindStringIndex(s); loc != nil {
		return int64(loc[0] + 1)
	}
	return 0
}

// likeToRegexp converts a LIKE pattern to regexp; a leading % stays unanchored for PATINDEX.
func likeToRegexp(pattern string) string {
	var b strings.Builder
	if strings.HasPrefix(pattern, "%") {
		pattern = pattern[1:]
	} else {
		b.WriteByte('^')
	}
	for i := 0; i < len(pattern); i++ {
		switch c := pattern[i]; c {
		case '%':
			b.WriteString(".*")
		case '_':
			b.WriteByte('.')
		case '[', ']':
			b.WriteByte(c)
		case '\\', '.', '+', '*', '?', '(', ')', '{', '}', '^', '$', '|':
			b.WriteByte('\\')
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// substr is SQL Server's 1-based SUBSTRING.
func substr(s string, start, length int) string {
	if start < 1 {
		length += start - 1
		start = 1
	}
	if start > len(s) || length <= 0 {
		return ""
	}
	end := start - 1 + length
	if end > len(s) {
		end = len(s)
	}
	return s[start-1 : end]
}

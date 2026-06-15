// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package functions

import (
	"regexp"
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
		if len(a) >= 1 {
			return "[" + strings.ReplaceAll(toStr(a[0]), "]", "]]") + "]"
		}
		return nil
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
}

// leftRight is LEFT (n leftmost bytes) when left, else RIGHT (n rightmost bytes).
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

// patIndex is PATINDEX: the 1-based start of the first substring of s matching the LIKE pattern, else 0.
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

// likeToRegexp translates a T-SQL LIKE pattern to a Go regexp. A leading % stays unanchored (PATINDEX
// reports where the non-% match begins); other %/_ become .*/. and regex metacharacters are escaped.
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

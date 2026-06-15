// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package functions

import "strings"

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

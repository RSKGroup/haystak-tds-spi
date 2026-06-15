// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package funcs

import "time"

func init() {
	register("YEAR", func(a []any) any { return datePart(a, func(t time.Time) int64 { return int64(t.Year()) }) })
	register("MONTH", func(a []any) any { return datePart(a, func(t time.Time) int64 { return int64(t.Month()) }) })
	register("DAY", func(a []any) any { return datePart(a, func(t time.Time) int64 { return int64(t.Day()) }) })
	register("GETDATE", utcNow)
	register("GETUTCDATE", utcNow)
	register("SYSDATETIME", utcNow)
	register("SYSUTCDATETIME", utcNow)
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

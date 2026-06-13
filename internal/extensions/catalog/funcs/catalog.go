// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package funcs

import "strings"

func init() {
	register("DB_ID", func(a []any) any { return dbIDOf(argStr(a, 0)) })
	register("HAS_DBACCESS", func(a []any) any { return int64(1) })
	register("SCHEMA_NAME", func(a []any) any { return "dbo" })
	register("SCHEMA_ID", func(a []any) any { return int64(1) })
	register("OBJECT_ID", func(a []any) any {
		if s := argStr(a, 0); s != "" {
			return dbIDOf(s) + 100000
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

// dbIDOf maps a database name to a stable non-zero id (system dbs keep their canonical ids).
func dbIDOf(name string) int64 {
	switch strings.ToLower(name) {
	case "master":
		return 1
	case "tempdb":
		return 2
	case "model":
		return 3
	case "msdb":
		return 4
	case "":
		return 0
	}
	var h int64
	for _, c := range name {
		h = h*31 + int64(c)
	}
	if h < 0 {
		h = -h
	}
	return h%30000 + 5
}

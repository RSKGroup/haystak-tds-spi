// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package functions

import "strings"

func init() {
	register("DB_ID", func(a []any) any { return dbIDOf(argStr(a, 0)) })
	register("HAS_DBACCESS", func(a []any) any { return int64(1) })
	register("SCHEMA_NAME", func(a []any) any { return "dbo" })
	register("SCHEMA_ID", func(a []any) any { return int64(1) })
	register("OBJECT_ID", func(a []any) any {
		if s := argStr(a, 0); s != "" {
			return ObjectID(s)
		}
		return nil
	})
	register("OBJECT_SCHEMA_NAME", func(a []any) any { return "dbo" })
	register("TYPE_NAME", func(a []any) any {
		if id, ok := argInt(a, 0); ok {
			if n, ok := sysTypeNames[id]; ok {
				return n
			}
		}
		return nil
	})
	register("TYPE_ID", func(a []any) any {
		if id, ok := typeIDByName[strings.ToLower(argStr(a, 0))]; ok {
			return id
		}
		return nil
	})
}

// ObjectID maps an object name (bare, schema-qualified, or bracketed) to the stable id SQL Server
// exposes as object_id — the value sys.* views emit, so OBJECT_ID('x') joins them and OBJECT_NAME reverses it.
func ObjectID(name string) int64 { return dbIDOf(bareName(name)) + 100000 }

// DBID maps a database name to its database_id (the value DB_ID returns and sys.databases reports).
func DBID(name string) int64 { return dbIDOf(name) }

func bareName(s string) string {
	s = strings.Trim(s, "[]\"`")
	if i := strings.LastIndexByte(s, '.'); i >= 0 {
		s = s[i+1:]
	}
	return strings.Trim(s, "[]\"`")
}

func argInt(a []any, i int) (int64, bool) {
	if i >= len(a) {
		return 0, false
	}
	switch v := a[i].(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case float64:
		return int64(v), true
	}
	return 0, false
}

// sysTypeNames reverses system_type_id → type name for TYPE_NAME (SQL Server's fixed system type ids).
var sysTypeNames = map[int64]string{
	34: "image", 35: "text", 36: "uniqueidentifier", 40: "date", 41: "time", 42: "datetime2",
	48: "tinyint", 52: "smallint", 56: "int", 59: "real", 60: "money", 61: "datetime",
	62: "float", 98: "sql_variant", 99: "ntext", 104: "bit", 106: "decimal", 108: "numeric",
	122: "smallmoney", 127: "bigint", 165: "varbinary", 167: "varchar", 173: "binary",
	175: "char", 231: "nvarchar", 239: "nchar", 241: "xml",
}

// typeIDByName reverses sysTypeNames for TYPE_ID (name -> system_type_id).
var typeIDByName = func() map[string]int64 {
	m := make(map[string]int64, len(sysTypeNames))
	for id, name := range sysTypeNames {
		m[name] = id
	}
	return m
}()

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

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tdstest

var jsonCases = []Case{
	{Element: "func:ISJSON", Name: "ISJSON/valid", SQL: `SELECT ISJSON('{"a":1}')`, Want: []any{1}},
	{Element: "func:ISJSON", Name: "ISJSON/invalid", SQL: `SELECT ISJSON('not json')`, Want: []any{0}},
	{Element: "func:JSON_VALUE", SQL: `SELECT JSON_VALUE('{"name":"Bob","age":30}','$.name')`, Want: []any{"Bob"}},
	{Element: "func:JSON_QUERY", SQL: `SELECT JSON_QUERY('{"a":[1,2,3]}','$.a')`, Want: []any{"[1,2,3]"}},
}

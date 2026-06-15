// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tdstest

var stringCases = []Case{
	{Element: "func:LEN", SQL: "SELECT LEN('hello')", Want: []any{5}},
	{Element: "func:DATALEN", SQL: "SELECT DATALEN('hello')", Want: []any{5}},
	{Element: "func:UPPER", SQL: "SELECT UPPER('aBc')", Want: []any{"ABC"}},
	{Element: "func:LOWER", SQL: "SELECT LOWER('aBc')", Want: []any{"abc"}},
	{Element: "func:LTRIM", SQL: "SELECT LTRIM('  x ')", Want: []any{"x "}},
	{Element: "func:RTRIM", SQL: "SELECT RTRIM(' x  ')", Want: []any{" x"}},
	{Element: "func:TRIM", SQL: "SELECT TRIM('  x  ')", Want: []any{"x"}},
	{Element: "func:CONCAT", SQL: "SELECT CONCAT('a','b','c')", Want: []any{"abc"}},
	{Element: "func:REPLACE", SQL: "SELECT REPLACE('a-b-c','-','+')", Want: []any{"a+b+c"}},
	{Element: "func:SUBSTRING", SQL: "SELECT SUBSTRING('abcdef',2,3)", Want: []any{"bcd"}},
	{Element: "func:QUOTENAME", SQL: "SELECT QUOTENAME('x')", Want: []any{"[x]"}},
	{Element: "func:CHARINDEX", Name: "CHARINDEX", SQL: "SELECT CHARINDEX('cd','abcdef')", Want: []any{3}},
	{Element: "func:CHARINDEX", Name: "CHARINDEX/start", SQL: "SELECT CHARINDEX('a','aXaXa',2)", Want: []any{3}},
	{Element: "func:LEFT", SQL: "SELECT LEFT('abcdef',3)", Want: []any{"abc"}},
	{Element: "func:RIGHT", SQL: "SELECT RIGHT('abcdef',2)", Want: []any{"ef"}},
	{Element: "func:REPLICATE", SQL: "SELECT REPLICATE('ab',3)", Want: []any{"ababab"}},
	{Element: "func:STUFF", SQL: "SELECT STUFF('abcdef',2,3,'XY')", Want: []any{"aXYef"}},
	{Element: "func:REVERSE", SQL: "SELECT REVERSE('abc')", Want: []any{"cba"}},
	{Element: "func:SPACE", SQL: "SELECT SPACE(3)", Want: []any{"   "}},
	{Element: "func:ASCII", SQL: "SELECT ASCII('A')", Want: []any{65}},
	{Element: "func:CHAR", SQL: "SELECT CHAR(65)", Want: []any{"A"}},
	{Element: "func:UNICODE", SQL: "SELECT UNICODE('A')", Want: []any{65}},
	{Element: "func:NCHAR", SQL: "SELECT NCHAR(65)", Want: []any{"A"}},
	{Element: "func:PATINDEX", SQL: "SELECT PATINDEX('%cd%','abcdef')", Want: []any{3}},
	{Element: "func:CONCAT_WS", SQL: "SELECT CONCAT_WS('-','a','b','c')", Want: []any{"a-b-c"}},
	{Element: "func:TRANSLATE", SQL: "SELECT TRANSLATE('2*[3]','[]','()')", Want: []any{"2*(3)"}},
	{Element: "func:STR", SQL: "SELECT STR(3.14159,6,2)", Want: []any{"  3.14"}},
	{Element: "func:STRING_ESCAPE", SQL: `SELECT STRING_ESCAPE('a"b','json')`, Want: []any{`a\"b`}},
}

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tdstest

// The parser-special conversion family and the IIF desugar. TRY_* and TRY_PARSE return NULL on failure.
var conversionCases = []Case{
	{Element: "parser:CAST", SQL: "SELECT CAST('42' AS INT)", Want: []any{42}},
	{Element: "parser:CONVERT", SQL: "SELECT CONVERT(INT,'42')", Want: []any{42}},
	{Element: "parser:TRY_CAST", Name: "TRY_CAST/ok", SQL: "SELECT TRY_CAST('42' AS INT)", Want: []any{42}},
	{Element: "parser:TRY_CAST", Name: "TRY_CAST/fail", SQL: "SELECT TRY_CAST('x' AS INT)", Want: []any{P(isNull)}},
	{Element: "parser:TRY_CONVERT", SQL: "SELECT TRY_CONVERT(INT,'42')", Want: []any{42}},
	{Element: "parser:PARSE", SQL: "SELECT PARSE('42' AS INT)", Want: []any{42}},
	{Element: "parser:TRY_PARSE", SQL: "SELECT TRY_PARSE('x' AS INT)", Want: []any{P(isNull)}},
	{Element: "parser:CAST", Name: "CAST/date-drops-time", SQL: "SELECT CAST('1980-05-01 10:30' AS date)", Want: []any{dateIs(1980, 5, 1)}},
	{Element: "parser:CAST", Name: "CAST/varchar-length", SQL: "SELECT CAST('abcdef' AS varchar(3))", Want: []any{"abc"}},
	{Element: "parser:CONVERT", Name: "CONVERT/date-style-101", SQL: "SELECT CONVERT(date, '05/01/1980', 101)", Want: []any{dateIs(1980, 5, 1)}},
	{Element: "parser:CONVERT", Name: "CONVERT/date-style-103", SQL: "SELECT CONVERT(date, '01/05/1980', 103)", Want: []any{dateIs(1980, 5, 1)}},
	{Element: "parser:CONVERT", Name: "CONVERT/char-style-23", SQL: "SELECT CONVERT(char(10), DATEFROMPARTS(1980, 5, 1), 23)", Want: []any{"1980-05-01"}},
	{Element: "parser:CONVERT", Name: "CONVERT/char-style-120", SQL: "SELECT CONVERT(char(16), DATETIMEFROMPARTS(2027, 1, 1, 9, 30, 0, 0), 120)", Want: []any{"2027-01-01 09:30"}},
	{Element: "parser:TRY_CONVERT", Name: "TRY_CONVERT/date-style-23", SQL: "SELECT TRY_CONVERT(date, '1980-05-01', 23)", Want: []any{dateIs(1980, 5, 1)}},
	{Element: "parser:TRY_CONVERT", Name: "TRY_CONVERT/date-fail", SQL: "SELECT TRY_CONVERT(date, '02/30/1980', 101)", Want: []any{P(isNull)}},
	{Name: "lang/converted-date-compares-as-date", SQL: "SELECT CASE WHEN TRY_CONVERT(date, '05/01/1980', 101) > TRY_CONVERT(date, '1979-12-31', 23) THEN 1 ELSE 0 END", Want: []any{1}},
	{Element: "parser:IIF", Name: "IIF/true", SQL: "SELECT IIF(5 > 3,'yes','no')", Want: []any{"yes"}},
	{Element: "parser:IIF", Name: "IIF/false", SQL: "SELECT IIF(1 > 2,'yes','no')", Want: []any{"no"}},
}

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
	{Element: "parser:IIF", Name: "IIF/true", SQL: "SELECT IIF(5 > 3,'yes','no')", Want: []any{"yes"}},
	{Element: "parser:IIF", Name: "IIF/false", SQL: "SELECT IIF(1 > 2,'yes','no')", Want: []any{"no"}},
}

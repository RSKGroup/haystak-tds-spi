// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tdstest

// Expression-language features that are not named wired elements (so Element is empty and they are
// exempt from the completeness gate); they still run on every backend as extra coverage.
var languageCases = []Case{
	{Name: "lang/arith-precedence", SQL: "SELECT 2 + 3 * 4", Want: []any{14}},
	{Name: "lang/paren-grouping", SQL: "SELECT (10 - 2) * 2", Want: []any{16}},
	{Name: "lang/string-concat", SQL: "SELECT 'a' + 'b' + 'c'", Want: []any{"abc"}},
	{Name: "lang/case-searched", SQL: "SELECT CASE WHEN 1 = 1 THEN 'eq' ELSE 'ne' END", Want: []any{"eq"}},
	{Name: "lang/case-simple", SQL: "SELECT CASE 2 WHEN 1 THEN 'one' WHEN 2 THEN 'two' ELSE '?' END", Want: []any{"two"}},
	{Name: "lang/nested-call", SQL: "SELECT UPPER(LEFT('abcdef',3))", Want: []any{"ABC"}},
	{Name: "lang/column-alias", SQL: "SELECT 7 AS n", Want: []any{7}},
}

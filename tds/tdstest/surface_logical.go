// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tdstest

var logicalCases = []Case{
	{Element: "func:ISNULL", SQL: "SELECT ISNULL(NULLIF(1,1),'x')", Want: []any{"x"}},
	{Element: "func:COALESCE", SQL: "SELECT COALESCE('a','b')", Want: []any{"a"}},
	{Element: "func:NULLIF", Name: "NULLIF/distinct", SQL: "SELECT NULLIF(5,6)", Want: []any{5}},
	{Element: "func:NULLIF", Name: "NULLIF/equal", SQL: "SELECT NULLIF(5,5)", Want: []any{P(isNull)}},
	{Element: "func:CHOOSE", SQL: "SELECT CHOOSE(2,'a','b','c')", Want: []any{"b"}},
}

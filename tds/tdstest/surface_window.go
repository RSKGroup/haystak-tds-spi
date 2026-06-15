// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tdstest

// Window functions over sys.types (static 20 rows, schema_id=4 everywhere): all rows tie under
// ORDER BY schema_id, so RANK/DENSE_RANK collapse to 1 under DISTINCT; the others assert shape.
var windowCases = []Case{
	{Element: "win:ROW_NUMBER", SQL: "SELECT ROW_NUMBER() OVER (ORDER BY name) AS rn FROM sys.types", Check: checks(wantCols("rn"), exactRows(20))},
	{Element: "win:RANK", SQL: "SELECT DISTINCT rk FROM (SELECT RANK() OVER (ORDER BY schema_id) AS rk FROM sys.types) t", Want: []any{1}},
	{Element: "win:DENSE_RANK", SQL: "SELECT DISTINCT dr FROM (SELECT DENSE_RANK() OVER (ORDER BY schema_id) AS dr FROM sys.types) t", Want: []any{1}},
	{Element: "win:LAG", SQL: "SELECT LAG(name) OVER (ORDER BY name) AS lg FROM sys.types", Check: checks(wantCols("lg"), exactRows(20))},
	{Element: "win:LEAD", SQL: "SELECT LEAD(name) OVER (ORDER BY name) AS ld FROM sys.types", Check: checks(wantCols("ld"), exactRows(20))},
}

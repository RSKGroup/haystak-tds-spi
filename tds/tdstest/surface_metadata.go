// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tdstest

// Registry metadata scalars plus the env-resolved catalog scalars (OBJECT_NAME etc.), which resolve
// against the live catalog in projection; the cross-backend cases assert their null-handling contract.
var metadataCases = []Case{
	{Element: "func:DB_ID", SQL: "SELECT DB_ID('master')", Want: []any{1}},
	{Element: "func:HAS_DBACCESS", SQL: "SELECT HAS_DBACCESS('master')", Want: []any{1}},
	{Element: "func:SCHEMA_NAME", SQL: "SELECT SCHEMA_NAME(4)", Want: []any{"dbo"}},
	{Element: "func:SCHEMA_ID", SQL: "SELECT SCHEMA_ID('dbo')", Want: []any{1}},
	{Element: "func:OBJECT_ID", SQL: "SELECT OBJECT_ID('products')", Want: []any{102329}},
	{Element: "func:OBJECT_SCHEMA_NAME", SQL: "SELECT OBJECT_SCHEMA_NAME(1)", Want: []any{"dbo"}},
	{Element: "func:TYPE_NAME", SQL: "SELECT TYPE_NAME(56)", Want: []any{"int"}},
	{Element: "func:TYPE_ID", SQL: "SELECT TYPE_ID('int')", Want: []any{56}},
	{Element: "func:STATS_DATE", SQL: "SELECT STATS_DATE(1,1)", Want: []any{P(isNull)}},

	{Element: "func:OBJECT_NAME", SQL: "SELECT OBJECT_NAME(0)", Want: []any{P(isNull)}},
	{Element: "func:DB_NAME", SQL: "SELECT DB_NAME()", Want: []any{"master"}},
	{Element: "func:COL_NAME", SQL: "SELECT COL_NAME(0,1)", Want: []any{P(isNull)}},
	{Element: "func:COL_LENGTH", SQL: "SELECT COL_LENGTH('nope','nope')", Want: []any{P(isNull)}},
	{Element: "func:COLUMNPROPERTY", SQL: "SELECT COLUMNPROPERTY(0,'x','ColumnId')", Want: []any{P(isNull)}},
	{Element: "func:OBJECTPROPERTY", SQL: "SELECT OBJECTPROPERTY(0,'IsUserTable')", Want: []any{0}},
	{Element: "func:OBJECTPROPERTYEX", SQL: "SELECT OBJECTPROPERTYEX(0,'IsView')", Want: []any{0}},
	{Element: "func:OBJECT_DEFINITION", SQL: "SELECT OBJECT_DEFINITION(0)", Want: []any{P(isNull)}},
}

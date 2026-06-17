// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tdstest

// Table-scoped procs target a nonexistent name to assert the empty-degrade column contract on every backend.
var procCases = []Case{
	{Element: "proc:sp_databases", SQL: "EXEC sp_databases", Check: wantCols("DATABASE_NAME", "REMARKS")},
	{Element: "proc:sp_tables", SQL: "EXEC sp_tables", Check: wantCols("TABLE_NAME", "TABLE_TYPE")},
	{Element: "proc:sp_columns", SQL: "EXEC sp_columns 'zzz_surface_none'", Check: checks(wantCols("TABLE_NAME", "COLUMN_NAME", "DATA_TYPE"), exactRows(0))},
	{Element: "proc:sp_columns_90", SQL: "EXEC sp_columns_90 'zzz_surface_none'", Check: checks(wantCols("TABLE_NAME", "COLUMN_NAME", "DATA_TYPE"), exactRows(0))},
	{Element: "proc:sp_helptext", SQL: "EXEC sp_helptext 'zzz_surface_none'", Check: checks(wantCols("Text"), exactRows(0))},
	{Element: "proc:sp_help", SQL: "EXEC sp_help", Check: wantCols("Name", "Owner", "Object_type")},
	{Element: "proc:sp_pkeys", SQL: "EXEC sp_pkeys 'zzz_surface_none'", Check: checks(wantCols("TABLE_NAME", "COLUMN_NAME", "PK_NAME"), exactRows(0))},
	{Element: "proc:sp_fkeys", SQL: "EXEC sp_fkeys @fktable_name = 'zzz_surface_none'", Check: checks(wantCols("FKTABLE_NAME", "PKTABLE_NAME", "FK_NAME"), exactRows(0))},
	{Element: "proc:sp_statistics", SQL: "EXEC sp_statistics 'zzz_surface_none'", Check: checks(wantCols("TABLE_NAME", "INDEX_NAME", "COLUMN_NAME"), exactRows(0))},
	{Element: "proc:sp_special_columns", SQL: "EXEC sp_special_columns 'zzz_surface_none'", Check: checks(wantCols("COLUMN_NAME", "DATA_TYPE"), exactRows(0))},
	{Element: "proc:sp_stored_procedures", SQL: "EXEC sp_stored_procedures", Check: wantCols("PROCEDURE_NAME", "PROCEDURE_TYPE")},
	{Element: "proc:sp_sproc_columns", SQL: "EXEC sp_sproc_columns 'zzz_surface_none'", Check: checks(wantCols("PROCEDURE_NAME", "COLUMN_NAME"), exactRows(0))},
	{Element: "proc:sp_helpindex", SQL: "EXEC sp_helpindex 'zzz_surface_none'", Check: checks(wantCols("index_name", "index_keys"), exactRows(0))},
	{Element: "proc:sp_helpconstraint", SQL: "EXEC sp_helpconstraint 'zzz_surface_none'", Check: checks(wantCols("constraint_type", "constraint_name"), exactRows(0))},
	{Element: "proc:sp_helpdb", SQL: "EXEC sp_helpdb", Check: checks(wantCols("name", "dbid"), atLeastRows(1))},
	{Element: "proc:sp_configure", SQL: "EXEC sp_configure", Check: checks(wantCols("name", "config_value"), atLeastRows(1))},
	{Element: "proc:sp_lock", SQL: "EXEC sp_lock", Check: checks(wantCols("spid", "Type"), exactRows(0))},
	{Element: "proc:sp_server_info", SQL: "EXEC sp_server_info", Check: checks(wantCols("attribute_id", "attribute_name", "attribute_value"), atLeastRows(1))},
	{Element: "proc:sp_datatype_info", SQL: "EXEC sp_datatype_info", Check: checks(wantCols("TYPE_NAME", "DATA_TYPE"), atLeastRows(1))},
	{Element: "proc:sp_datatype_info_100", SQL: "EXEC sp_datatype_info_100", Check: checks(wantCols("TYPE_NAME", "DATA_TYPE"), atLeastRows(1))},
	{Element: "proc:sp_table_privileges", SQL: "EXEC sp_table_privileges 'zzz_surface_none'", Check: checks(wantCols("TABLE_NAME", "PRIVILEGE", "GRANTEE"), exactRows(0))},
	{Element: "proc:sp_column_privileges", SQL: "EXEC sp_column_privileges 'zzz_surface_none'", Check: checks(wantCols("COLUMN_NAME", "PRIVILEGE", "GRANTEE"), exactRows(0))},
	{Element: "proc:sp_helptrigger", SQL: "EXEC sp_helptrigger 'zzz_surface_none'", Check: checks(wantCols("trigger_name", "isupdate", "isafter"), exactRows(0))},
	{Element: "proc:sp_depends", SQL: "EXEC sp_depends 'zzz_surface_none'", Check: checks(wantCols("name", "type"), exactRows(0))},
	{Element: "proc:sp_executesql", SQL: "EXEC sp_executesql N'SELECT 1 AS x'", Want: []any{1}},
}

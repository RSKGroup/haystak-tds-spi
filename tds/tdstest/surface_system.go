// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tdstest

// System scalars and the @@-constant family. The error scalars return NULL outside a CATCH block.
var systemCases = []Case{
	{Element: "func:SERVERPROPERTY", Name: "SERVERPROPERTY/Edition", SQL: "SELECT SERVERPROPERTY('Edition')", Want: []any{"Developer Edition (64-bit)"}},
	{Element: "func:SERVERPROPERTY", Name: "SERVERPROPERTY/ProductVersion", SQL: "SELECT SERVERPROPERTY('ProductVersion')", Want: []any{"16.0.1000.6"}},
	{Element: "func:DATABASEPROPERTYEX", Name: "DATABASEPROPERTYEX/Status", SQL: "SELECT DATABASEPROPERTYEX('master','Status')", Want: []any{"ONLINE"}},
	{Element: "func:DATABASEPROPERTYEX", Name: "DATABASEPROPERTYEX/Updateability", SQL: "SELECT DATABASEPROPERTYEX('master','Updateability')", Want: []any{"READ_WRITE"}},
	{Element: "func:DATABASEPROPERTYEX", Name: "DATABASEPROPERTYEX/Recovery", SQL: "SELECT DATABASEPROPERTYEX('master','Recovery')", Want: []any{"SIMPLE"}},
	{Element: "func:NEWID", SQL: "SELECT NEWID()", Want: []any{P(isGUID)}},
	{Element: "func:SCOPE_IDENTITY", SQL: "SELECT SCOPE_IDENTITY()", Want: []any{P(isNull)}},
	{Element: "func:IDENT_CURRENT", SQL: "SELECT IDENT_CURRENT('products')", Want: []any{P(isNull)}},
	{Element: "func:ERROR_MESSAGE", SQL: "SELECT ERROR_MESSAGE()", Want: []any{P(isNull)}},
	{Element: "func:ERROR_NUMBER", SQL: "SELECT ERROR_NUMBER()", Want: []any{P(isNull)}},
	{Element: "func:ERROR_SEVERITY", SQL: "SELECT ERROR_SEVERITY()", Want: []any{P(isNull)}},
	{Element: "func:ERROR_STATE", SQL: "SELECT ERROR_STATE()", Want: []any{P(isNull)}},
	{Element: "func:ERROR_LINE", SQL: "SELECT ERROR_LINE()", Want: []any{P(isNull)}},
	{Element: "func:ERROR_PROCEDURE", SQL: "SELECT ERROR_PROCEDURE()", Want: []any{P(isNull)}},

	{Element: "func:@@VERSION", SQL: "SELECT @@VERSION", Want: []any{contains("SQL Server")}},
	{Element: "func:@@SPID", SQL: "SELECT @@SPID", Want: []any{1}},
	{Element: "func:@@SERVERNAME", SQL: "SELECT @@SERVERNAME", Want: []any{"haystak-tds-spi"}},
	{Element: "func:@@LANGUAGE", SQL: "SELECT @@LANGUAGE", Want: []any{"us_english"}},
	{Element: "func:@@ROWCOUNT", SQL: "SELECT @@ROWCOUNT", Want: []any{0}},
	{Element: "func:@@ERROR", SQL: "SELECT @@ERROR", Want: []any{0}},
	{Element: "func:@@TRANCOUNT", SQL: "SELECT @@TRANCOUNT", Want: []any{0}},
	{Element: "func:@@FETCH_STATUS", SQL: "SELECT @@FETCH_STATUS", Want: []any{0}},
	{Element: "func:@@IDENTITY", SQL: "SELECT @@IDENTITY", Want: []any{P(isNull)}},
	{Element: "func:@@PROCID", SQL: "SELECT @@PROCID", Want: []any{P(isNull)}},
	{Element: "func:@@SERVICENAME", SQL: "SELECT @@SERVICENAME", Want: []any{"MSSQLSERVER"}},
	{Element: "func:@@NESTLEVEL", SQL: "SELECT @@NESTLEVEL", Want: []any{0}},
	{Element: "func:@@CURSOR_ROWS", SQL: "SELECT @@CURSOR_ROWS", Want: []any{0}},
	{Element: "func:@@MAX_PRECISION", SQL: "SELECT @@MAX_PRECISION", Want: []any{38}},
	{Element: "func:@@DATEFIRST", SQL: "SELECT @@DATEFIRST", Want: []any{7}},
	{Element: "func:@@LOCK_TIMEOUT", SQL: "SELECT @@LOCK_TIMEOUT", Want: []any{-1}},
	{Element: "func:@@OPTIONS", SQL: "SELECT @@OPTIONS", Want: []any{5496}},

	{Element: "func:NEWSEQUENTIALID", SQL: "SELECT NEWSEQUENTIALID()", Want: []any{P(isGUID)}},
	{Element: "func:XACT_STATE", SQL: "SELECT XACT_STATE()", Want: []any{0}},
	{Element: "func:CURSOR_STATUS", SQL: "SELECT CURSOR_STATUS('global','c')", Want: []any{-3}},
	{Element: "func:CONNECTIONPROPERTY", SQL: "SELECT CONNECTIONPROPERTY('net_transport')", Want: []any{"TCP"}},
}

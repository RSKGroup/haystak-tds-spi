// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tdstest

var datetimeCases = []Case{
	{Element: "func:YEAR", SQL: "SELECT YEAR(DATEFROMPARTS(2024,3,15))", Want: []any{2024}},
	{Element: "func:MONTH", SQL: "SELECT MONTH(DATEFROMPARTS(2024,3,15))", Want: []any{3}},
	{Element: "func:DAY", SQL: "SELECT DAY(DATEFROMPARTS(2024,3,15))", Want: []any{15}},
	{Element: "func:GETDATE", SQL: "SELECT GETDATE()", Want: []any{P(notNull)}},
	{Element: "func:GETUTCDATE", SQL: "SELECT GETUTCDATE()", Want: []any{P(notNull)}},
	{Element: "func:SYSDATETIME", SQL: "SELECT SYSDATETIME()", Want: []any{P(notNull)}},
	{Element: "func:SYSUTCDATETIME", SQL: "SELECT SYSUTCDATETIME()", Want: []any{P(notNull)}},
	{Element: "func:SYSDATETIMEOFFSET", SQL: "SELECT SYSDATETIMEOFFSET()", Want: []any{P(notNull)}},
	{Element: "func:CURRENT_TIMESTAMP", SQL: "SELECT CURRENT_TIMESTAMP", Want: []any{P(notNull)}},
	{Element: "func:DATEADD", Name: "DATEADD", SQL: "SELECT DATEADD(day,1,DATEFROMPARTS(2024,1,15))", Want: []any{contains("2024-01-16")}},
	{Element: "func:DATEADD", Name: "DATEADD/month-clamp", SQL: "SELECT DATEADD(month,1,DATEFROMPARTS(2024,1,31))", Want: []any{contains("2024-02-29")}},
	{Element: "func:DATEDIFF", SQL: "SELECT DATEDIFF(day,DATEFROMPARTS(2024,1,1),DATEFROMPARTS(2024,1,15))", Want: []any{14}},
	{Element: "func:DATEDIFF_BIG", SQL: "SELECT DATEDIFF_BIG(day,DATEFROMPARTS(2024,1,1),DATEFROMPARTS(2024,1,15))", Want: []any{14}},
	{Element: "func:DATEPART", SQL: "SELECT DATEPART(month,DATEFROMPARTS(2024,3,15))", Want: []any{3}},
	{Element: "func:DATENAME", SQL: "SELECT DATENAME(month,DATEFROMPARTS(2024,3,15))", Want: []any{"March"}},
	{Element: "func:DATETRUNC", SQL: "SELECT DATETRUNC(month,DATEFROMPARTS(2024,3,15))", Want: []any{contains("2024-03-01")}},
	{Element: "func:EOMONTH", SQL: "SELECT EOMONTH(DATEFROMPARTS(2024,2,10))", Want: []any{contains("2024-02-29")}},
	{Element: "func:ISDATE", SQL: "SELECT ISDATE('2024-01-01')", Want: []any{1}},
	{Element: "func:DATEFROMPARTS", SQL: "SELECT DATEFROMPARTS(2024,2,29)", Want: []any{contains("2024-02-29")}},
	{Element: "func:DATETIMEFROMPARTS", SQL: "SELECT DATETIMEFROMPARTS(2024,1,1,1,1,1,0)", Want: []any{contains("2024-01-01 01:01:01")}},
}

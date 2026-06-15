// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tdstest

// FORMAT numeric specifiers and datetime token mapping; constant inputs, so backend-independent.
var formatCases = []Case{
	{Element: "func:FORMAT", Name: "FORMAT/N2", SQL: "SELECT FORMAT(1234.5, 'N2')", Want: []any{"1,234.50"}},
	{Element: "func:FORMAT", Name: "FORMAT/N0", SQL: "SELECT FORMAT(1234567, 'N0')", Want: []any{"1,234,567"}},
	{Element: "func:FORMAT", Name: "FORMAT/C2", SQL: "SELECT FORMAT(1234.5, 'C2')", Want: []any{"$1,234.50"}},
	{Element: "func:FORMAT", Name: "FORMAT/P2", SQL: "SELECT FORMAT(0.1234, 'P2')", Want: []any{"12.34 %"}},
	{Element: "func:FORMAT", Name: "FORMAT/D5", SQL: "SELECT FORMAT(42, 'D5')", Want: []any{"00042"}},
	{Element: "func:FORMAT", Name: "FORMAT/X", SQL: "SELECT FORMAT(255, 'X')", Want: []any{"FF"}},
	{Element: "func:FORMAT", Name: "FORMAT/custom", SQL: "SELECT FORMAT(1234.567, '#,##0.00')", Want: []any{"1,234.57"}},
	{Element: "func:FORMAT", Name: "FORMAT/date", SQL: "SELECT FORMAT(DATEFROMPARTS(2024,3,15), 'yyyy-MM-dd')", Want: []any{"2024-03-15"}},
	{Element: "func:FORMAT", Name: "FORMAT/datetime", SQL: "SELECT FORMAT(DATETIMEFROMPARTS(2024,3,5,9,7,3,0), 'yyyy-MM-dd HH:mm:ss')", Want: []any{"2024-03-05 09:07:03"}},
	{Element: "func:FORMAT", Name: "FORMAT/month", SQL: "SELECT FORMAT(DATEFROMPARTS(2024,3,15), 'MMMM d, yyyy')", Want: []any{"March 15, 2024"}},
}

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tdstest

var mathCases = []Case{
	{Element: "func:ABS", SQL: "SELECT ABS(-3)", Want: []any{3}},
	{Element: "func:CEILING", SQL: "SELECT CEILING(2.1)", Want: []any{3}},
	{Element: "func:FLOOR", SQL: "SELECT FLOOR(2.9)", Want: []any{2}},
	{Element: "func:SQRT", SQL: "SELECT SQRT(16)", Want: []any{approx(4)}},
	{Element: "func:EXP", SQL: "SELECT EXP(0)", Want: []any{approx(1)}},
	{Element: "func:LOG10", SQL: "SELECT LOG10(100)", Want: []any{approx(2)}},
	{Element: "func:LOG", SQL: "SELECT LOG(1)", Want: []any{approx(0)}},
	{Element: "func:SIN", SQL: "SELECT SIN(0)", Want: []any{approx(0)}},
	{Element: "func:COS", SQL: "SELECT COS(0)", Want: []any{approx(1)}},
	{Element: "func:TAN", SQL: "SELECT TAN(0)", Want: []any{approx(0)}},
	{Element: "func:COT", SQL: "SELECT COT(1)", Want: []any{approx(0.6420926159343308)}},
	{Element: "func:ASIN", SQL: "SELECT ASIN(0)", Want: []any{approx(0)}},
	{Element: "func:ACOS", SQL: "SELECT ACOS(1)", Want: []any{approx(0)}},
	{Element: "func:ATAN", SQL: "SELECT ATAN(0)", Want: []any{approx(0)}},
	{Element: "func:ATN2", SQL: "SELECT ATN2(0,1)", Want: []any{approx(0)}},
	{Element: "func:DEGREES", SQL: "SELECT DEGREES(PI())", Want: []any{approx(180)}},
	{Element: "func:RADIANS", SQL: "SELECT RADIANS(180)", Want: []any{approx(3.141592653589793)}},
	{Element: "func:PI", SQL: "SELECT PI()", Want: []any{approx(3.141592653589793)}},
	{Element: "func:SQUARE", SQL: "SELECT SQUARE(5)", Want: []any{approx(25)}},
	{Element: "func:POWER", SQL: "SELECT POWER(2,10)", Want: []any{approx(1024)}},
	{Element: "func:ROUND", SQL: "SELECT ROUND(123.456,2)", Want: []any{approx(123.46)}},
	{Element: "func:SIGN", SQL: "SELECT SIGN(-7)", Want: []any{-1}},
	{Element: "func:RAND", SQL: "SELECT RAND()", Want: []any{inRange(0, 1)}},
}

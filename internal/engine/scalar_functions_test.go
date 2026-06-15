// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"strings"
	"testing"
)

func TestMathFunctions(t *testing.T) {
	rows := qry(t, "SELECT CEILING(2.1), FLOOR(2.9), ROUND(123.456,2), POWER(2,10), SQRT(16), SQUARE(5), SIGN(-7), ABS(-3)")
	r := rows[0]
	if cell(r[0]) != "3" || cell(r[1]) != "2" || cell(r[2]) != "123.46" || cell(r[3]) != "1024" {
		t.Errorf("CEILING/FLOOR/ROUND/POWER = %v", r[0:4])
	}
	if cell(r[4]) != "4" || cell(r[5]) != "25" || cell(r[6]) != "-1" || cell(r[7]) != "3" {
		t.Errorf("SQRT/SQUARE/SIGN/ABS = %v", r[4:8])
	}
}

func TestLogicalAndConversion(t *testing.T) {
	rows := qry(t, "SELECT IIF(5 > 3, 'yes', 'no'), IIF(1 > 2, 'yes', 'no'), CHOOSE(2, 'a', 'b', 'c'), TRY_CAST('42' AS INT), TRY_CONVERT(INT, 'notint')")
	r := rows[0]
	if cell(r[0]) != "yes" || cell(r[1]) != "no" || cell(r[2]) != "b" {
		t.Errorf("IIF/CHOOSE = %v", r[0:3])
	}
	if cell(r[3]) != "42" {
		t.Errorf("TRY_CAST('42' AS INT) = %v, want 42", r[3])
	}
	if cell(r[4]) != "<nil>" {
		t.Errorf("TRY_CONVERT(INT,'notint') = %v, want NULL", r[4])
	}
}

func TestDateFunctions(t *testing.T) {
	rows := qry(t, "SELECT DATEDIFF(day,'2024-01-01','2024-01-15'), DATEPART(month,'2024-03-15'), DATEPART(year,'2024-03-15'), DATEPART(quarter,'2024-07-01'), DATENAME(month,'2024-03-15'), DATENAME(weekday,'2024-03-15')")
	r := rows[0]
	if cell(r[0]) != "14" || cell(r[1]) != "3" || cell(r[2]) != "2024" || cell(r[3]) != "3" {
		t.Errorf("DATEDIFF/DATEPART = %v", r[0:4])
	}
	if cell(r[4]) != "March" || cell(r[5]) != "Friday" {
		t.Errorf("DATENAME = %v/%v, want March/Friday", r[4], r[5])
	}
}

func TestDateAddEomonthIsdate(t *testing.T) {
	rows := qry(t, "SELECT DATEADD(day,1,'2024-01-15'), DATEADD(month,1,'2024-01-31'), EOMONTH('2024-02-10'), ISDATE('2024-01-01'), ISDATE('nope')")
	r := rows[0]
	if !strings.Contains(cell(r[0]), "2024-01-16") {
		t.Errorf("DATEADD day = %v, want 2024-01-16", r[0])
	}
	if !strings.Contains(cell(r[1]), "2024-02-29") { // month-end clamp
		t.Errorf("DATEADD month = %v, want 2024-02-29", r[1])
	}
	if !strings.Contains(cell(r[2]), "2024-02-29") {
		t.Errorf("EOMONTH = %v, want 2024-02-29", r[2])
	}
	if cell(r[3]) != "1" || cell(r[4]) != "0" {
		t.Errorf("ISDATE = %v/%v, want 1/0", r[3], r[4])
	}
}

func TestCurrentTimestamp(t *testing.T) {
	rows := qry(t, "SELECT CURRENT_TIMESTAMP")
	if len(rows) != 1 || cell(rows[0][0]) == "<nil>" || cell(rows[0][0]) == "" {
		t.Fatalf("CURRENT_TIMESTAMP = %v, want a timestamp", rows)
	}
}

func TestTypeID(t *testing.T) {
	rows := qry(t, "SELECT TYPE_ID('int'), TYPE_ID('nvarchar'), TYPE_ID('uniqueidentifier')")
	if cell(rows[0][0]) != "56" || cell(rows[0][1]) != "231" || cell(rows[0][2]) != "36" {
		t.Errorf("TYPE_ID = %v, want 56/231/36", rows[0])
	}
}

func TestStringFunctions(t *testing.T) {
	rows := qry(t, "SELECT CHARINDEX('cd','abcdef'), CHARINDEX('x','abc'), CHARINDEX('a','aXaXa',2), LEFT('abcdef',3), RIGHT('abcdef',2), REPLICATE('ab',3), STUFF('abcdef',2,3,'XY'), REVERSE('abc')")
	r := rows[0]
	if cell(r[0]) != "3" || cell(r[1]) != "0" || cell(r[2]) != "3" {
		t.Errorf("CHARINDEX = %v/%v/%v, want 3/0/3", r[0], r[1], r[2])
	}
	if cell(r[3]) != "abc" || cell(r[4]) != "ef" || cell(r[5]) != "ababab" || cell(r[6]) != "aXYef" || cell(r[7]) != "cba" {
		t.Errorf("LEFT/RIGHT/REPLICATE/STUFF/REVERSE = %v", r[3:])
	}
}

func TestStringCharFunctions(t *testing.T) {
	rows := qry(t, "SELECT ASCII('A'), CHAR(65), UNICODE('A'), NCHAR(65), LEN(SPACE(3)), PATINDEX('%cd%','abcdef'), PATINDEX('%xy%','abc')")
	r := rows[0]
	if cell(r[0]) != "65" || cell(r[1]) != "A" || cell(r[2]) != "65" || cell(r[3]) != "A" {
		t.Errorf("ASCII/CHAR/UNICODE/NCHAR = %v", r[0:4])
	}
	if cell(r[4]) != "3" || cell(r[5]) != "3" || cell(r[6]) != "0" {
		t.Errorf("SPACE/PATINDEX = %v/%v/%v, want 3/3/0", r[4], r[5], r[6])
	}
}

func TestColNameAndLength(t *testing.T) {
	// COL_NAME(object_id, column_id) -> name; COL_LENGTH('table','column') -> byte width
	rows := qry(t, "SELECT COL_NAME(OBJECT_ID('products'), 1), COL_LENGTH('products', 'sku')")
	if cell(rows[0][0]) != "product_id" {
		t.Errorf("COL_NAME = %v, want product_id", rows[0][0])
	}
	if cell(rows[0][1]) == "<nil>" || cell(rows[0][1]) == "" {
		t.Errorf("COL_LENGTH(products, sku) = %v, want a byte width", rows[0][1])
	}
}

func TestColNameInOrderBy(t *testing.T) {
	// catalog scalars resolve in ORDER BY (env-threaded), not in WHERE/HAVING (nil env)
	rows := qry(t, "SELECT name FROM products ORDER BY COL_NAME(OBJECT_ID('products'), 1), name")
	if len(rows) == 0 {
		t.Fatal("ORDER BY COL_NAME returned no rows")
	}
}

func TestObjectProperty(t *testing.T) {
	rows := qry(t, "SELECT OBJECTPROPERTY(OBJECT_ID('products'), 'IsUserTable'), OBJECTPROPERTY(OBJECT_ID('products'), 'TableHasPrimaryKey'), OBJECTPROPERTY(OBJECT_ID('vActiveProducts'), 'IsView')")
	if cell(rows[0][0]) != "1" || cell(rows[0][1]) != "1" || cell(rows[0][2]) != "1" {
		t.Errorf("OBJECTPROPERTY = %v, want 1/1/1", rows[0])
	}
}

func TestColumnProperty(t *testing.T) {
	rows := qry(t, "SELECT COLUMNPROPERTY(OBJECT_ID('products'), 'product_id', 'ColumnId'), COLUMNPROPERTY(OBJECT_ID('products'), 'product_id', 'IsIdentity')")
	if cell(rows[0][0]) != "1" || cell(rows[0][1]) != "1" {
		t.Errorf("COLUMNPROPERTY = %v, want ColumnId=1, IsIdentity=1", rows[0])
	}
}

func TestConfigConstants(t *testing.T) {
	// @@ constants resolve through the single bare-scalar probe path (one per SELECT).
	want := map[string]string{
		"@@MAX_PRECISION": "38", "@@NESTLEVEL": "0", "@@DATEFIRST": "7",
		"@@LOCK_TIMEOUT": "-1", "@@CURSOR_ROWS": "0", "@@SERVICENAME": "MSSQLSERVER", "@@OPTIONS": "5496",
	}
	for name, exp := range want {
		rows := qry(t, "SELECT "+name)
		if len(rows) != 1 || cell(rows[0][0]) != exp {
			t.Errorf("%s = %v, want %s", name, rows, exp)
		}
	}
}

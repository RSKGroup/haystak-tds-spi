// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

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

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

func TestStringSplit(t *testing.T) {
	rows := qry(t, "SELECT value FROM STRING_SPLIT('a,b,c', ',')")
	if len(rows) != 3 || cell(rows[0][0]) != "a" || cell(rows[1][0]) != "b" || cell(rows[2][0]) != "c" {
		t.Fatalf("STRING_SPLIT = %v, want a/b/c", rows)
	}
	// WHERE applies over the function rowset.
	if n := len(qry(t, "SELECT value FROM STRING_SPLIT('a,b,c', ',') WHERE value <> 'b'")); n != 2 {
		t.Errorf("filtered STRING_SPLIT = %d rows, want 2", n)
	}
}

func TestOpenJSONArray(t *testing.T) {
	rows := qry(t, "SELECT [key], value, type FROM OPENJSON('[10,20,30]')")
	if len(rows) != 3 {
		t.Fatalf("OPENJSON array = %v, want 3 rows", rows)
	}
	if cell(rows[0][0]) != "0" || cell(rows[0][1]) != "10" || cell(rows[0][2]) != "2" {
		t.Errorf("row 0 = %v, want key=0 value=10 type=2", rows[0])
	}
}

func TestOpenJSONObjectAndPath(t *testing.T) {
	rows := qry(t, `SELECT [key], value FROM OPENJSON('{"a":1,"b":"hi"}')`)
	if len(rows) != 2 || cell(rows[0][0]) != "a" || cell(rows[1][0]) != "b" {
		t.Fatalf("OPENJSON object = %v, want keys a,b", rows)
	}
	if n := len(qry(t, `SELECT value FROM OPENJSON('{"items":[7,8,9]}', '$.items')`)); n != 3 {
		t.Errorf("OPENJSON path = %d rows, want 3", n)
	}
}

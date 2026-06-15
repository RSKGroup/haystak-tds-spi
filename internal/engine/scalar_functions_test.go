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

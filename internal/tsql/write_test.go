// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tsql

import (
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

func TestParseAlterAdd(t *testing.T) {
	stmt, ok, err := ParseWrite("ALTER TABLE users ADD age INT, note VARCHAR")
	if err != nil || !ok {
		t.Fatalf("ParseWrite: ok=%v err=%v", ok, err)
	}
	a := stmt.Alter
	if a == nil {
		t.Fatal("Alter is nil")
	}
	if a.Table != "users" {
		t.Errorf("Table = %q, want users", a.Table)
	}
	if len(a.AddColumns) != 2 {
		t.Fatalf("AddColumns = %d, want 2", len(a.AddColumns))
	}
	if a.AddColumns[0].Name != "age" || a.AddColumns[0].Type.Kind != types.Int32 {
		t.Errorf("col0 = %+v, want age int32", a.AddColumns[0])
	}
	if a.AddColumns[1].Name != "note" {
		t.Errorf("col1 = %q, want note", a.AddColumns[1].Name)
	}
}

func TestParseAlterDropColumn(t *testing.T) {
	stmt, ok, err := ParseWrite("ALTER TABLE dbo.users DROP COLUMN age, note")
	if err != nil || !ok {
		t.Fatalf("ParseWrite: ok=%v err=%v", ok, err)
	}
	a := stmt.Alter
	if a == nil {
		t.Fatal("Alter is nil")
	}
	if a.Table != "users" {
		t.Errorf("Table = %q, want users", a.Table)
	}
	want := []string{"age", "note"}
	if len(a.DropColumns) != len(want) {
		t.Fatalf("DropColumns = %v, want %v", a.DropColumns, want)
	}
	for i, c := range want {
		if a.DropColumns[i] != c {
			t.Errorf("DropColumns[%d] = %q, want %q", i, a.DropColumns[i], c)
		}
	}
}

func TestParseAlterBad(t *testing.T) {
	if _, _, err := ParseWrite("ALTER TABLE users RENAME foo"); err == nil {
		t.Error("expected error for unsupported ALTER action")
	}
}

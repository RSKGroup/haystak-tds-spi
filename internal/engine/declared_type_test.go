// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
)

func TestDeclaredTypeInfoSchema(t *testing.T) {
	rows := qry(t, "SELECT COLUMN_NAME, DATA_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = 'typed' ORDER BY ORDINAL_POSITION")
	want := map[string]string{"id": "bigint", "code": "varchar", "grade": "char", "born": "date", "level": "smallint"}
	if len(rows) != 5 {
		t.Fatalf("typed columns = %d, want 5", len(rows))
	}
	for _, r := range rows {
		name := cell(r[0])
		if got := cell(r[1]); got != want[name] {
			t.Errorf("%s DATA_TYPE = %q, want %q", name, got, want[name])
		}
	}
	un := qry(t, "SELECT DATA_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = 'users' AND COLUMN_NAME = 'name'")
	if len(un) != 1 || cell(un[0][0]) != "nvarchar" {
		t.Errorf("users.name DATA_TYPE = %v, want nvarchar (unset Name unchanged)", un)
	}
}

func TestDeclaredTypeSysColumns(t *testing.T) {
	rows := qry(t, "SELECT name, system_type_id, max_length FROM sys.columns WHERE object_id = OBJECT_ID('typed') ORDER BY column_id")
	want := map[string][2]int64{
		"id":    {127, 8},
		"code":  {167, 10}, // varchar is 1 byte/char, not 2
		"grade": {175, 2},
		"born":  {40, 3},
		"level": {52, 2},
	}
	if len(rows) != 5 {
		t.Fatalf("sys.columns(typed) = %d rows, want 5", len(rows))
	}
	for _, r := range rows {
		name := cell(r[0])
		w := want[name]
		if r[1] != w[0] || r[2] != w[1] {
			t.Errorf("%s = system_type_id %v / max_length %v, want %d / %d", name, r[1], r[2], w[0], w[1])
		}
	}
}

func TestDeclaredTypeSpColumns(t *testing.T) {
	// sp_columns is scoped to the connected database; master returns nothing, so scope to inmem's catalog.
	ctx := engine.WithDatabase(context.Background(), "haystak")
	rs, err := engine.Query(ctx, inmem.New(), "EXEC sp_columns 'typed'")
	if err != nil {
		t.Fatal(err)
	}
	rows := collect(t, rs)
	// cols: 3 COLUMN_NAME, 4 DATA_TYPE, 5 TYPE_NAME, 7 LENGTH.
	type tc struct {
		odbc   int64
		name   string
		length int64
	}
	want := map[string]tc{
		"code":  {12, "varchar", 10},
		"grade": {1, "char", 2},
		"level": {5, "smallint", 2},
	}
	seen := map[string]bool{}
	for _, r := range rows {
		name := cell(r[3])
		w, ok := want[name]
		if !ok {
			continue
		}
		seen[name] = true
		if r[4] != w.odbc || cell(r[5]) != w.name || r[7] != w.length {
			t.Errorf("%s = DATA_TYPE %v / TYPE_NAME %v / LENGTH %v, want %d / %s / %d",
				name, r[4], r[5], r[7], w.odbc, w.name, w.length)
		}
	}
	for name := range want {
		if !seen[name] {
			t.Errorf("sp_columns 'typed' missing row for %s", name)
		}
	}
}

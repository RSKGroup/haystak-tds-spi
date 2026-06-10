// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
	"github.com/RSKGroup/haystak-tds-spi/tds"
)

func collect(t *testing.T, rs tds.Rows) [][]any {
	t.Helper()
	defer rs.Close()
	var out [][]any
	for rs.Next() {
		v, err := rs.Values()
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, v)
	}
	return out
}

func TestQueryBackendTable(t *testing.T) {
	rs, err := engine.Query(context.Background(), inmem.New(), "SELECT name FROM users WHERE id = 2")
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, rs)
	if len(got) != 1 || got[0][0] != "alan" {
		t.Fatalf("got %v, want [[alan]]", got)
	}
}

func TestQueryInformationSchema(t *testing.T) {
	rs, err := engine.Query(context.Background(), inmem.New(),
		"SELECT COLUMN_NAME, DATA_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = 'users' ORDER BY ORDINAL_POSITION")
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, rs)
	if len(got) != 2 || got[0][0] != "id" || got[0][1] != "bigint" || got[1][0] != "name" || got[1][1] != "nvarchar" {
		t.Fatalf("got %v, want [[id bigint] [name nvarchar]]", got)
	}
}

func TestExecAlterTable(t *testing.T) {
	ctx := context.Background()
	b := inmem.New()
	const colsQ = "SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = 'users' ORDER BY ORDINAL_POSITION"

	if _, _, err := engine.Exec(ctx, b, "ALTER TABLE users ADD age INT"); err != nil {
		t.Fatalf("ALTER ADD: %v", err)
	}
	rs, err := engine.Query(ctx, b, colsQ)
	if err != nil {
		t.Fatal(err)
	}
	if got := collect(t, rs); len(got) != 3 || got[2][0] != "age" {
		t.Fatalf("after ADD got %v, want id/name/age", got)
	}
	rs, err = engine.Query(ctx, b, "SELECT * FROM users")
	if err != nil {
		t.Fatalf("scan after ADD: %v", err)
	}
	for _, row := range collect(t, rs) {
		if len(row) != 3 {
			t.Fatalf("row width = %d, want 3 (rows aligned with new column)", len(row))
		}
	}

	if _, _, err := engine.Exec(ctx, b, "ALTER TABLE users DROP COLUMN age"); err != nil {
		t.Fatalf("ALTER DROP: %v", err)
	}
	rs, err = engine.Query(ctx, b, colsQ)
	if err != nil {
		t.Fatal(err)
	}
	if got := collect(t, rs); len(got) != 2 {
		t.Fatalf("after DROP got %v, want 2 columns", got)
	}
}

func TestQueryUnknownTable(t *testing.T) {
	if _, err := engine.Query(context.Background(), inmem.New(), "SELECT * FROM nope"); err == nil {
		t.Fatal("expected error for unknown table")
	}
}

func TestQueryParseError(t *testing.T) {
	if _, err := engine.Query(context.Background(), inmem.New(), "NOPE"); err == nil {
		t.Fatal("expected parse error")
	}
}

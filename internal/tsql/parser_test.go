// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tsql

import (
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/tds"
)

func TestParseSelectStar(t *testing.T) {
	q, err := Parse("SELECT * FROM users")
	if err != nil {
		t.Fatal(err)
	}
	if q.Table != "users" {
		t.Errorf("Table = %q, want users", q.Table)
	}
	if len(q.Select) != 0 {
		t.Errorf("Select = %v, want empty (all columns)", q.Select)
	}
}

func TestParseFull(t *testing.T) {
	q, err := Parse("SELECT TOP 5 name, id FROM dbo.users WHERE id >= 2 AND name <> 'ada' ORDER BY name DESC")
	if err != nil {
		t.Fatal(err)
	}
	if q.Limit != 5 {
		t.Errorf("Limit = %d, want 5", q.Limit)
	}
	if len(q.Select) != 2 || q.Select[0].Column != "name" || q.Select[1].Column != "id" {
		t.Errorf("Select = %+v", q.Select)
	}
	if q.Table != "users" {
		t.Errorf("Table = %q, want users (last segment of dbo.users)", q.Table)
	}
	if q.Schema != "dbo" {
		t.Errorf("Schema = %q, want dbo", q.Schema)
	}
	if q.Where == nil || len(q.Where.And) != 2 {
		t.Fatalf("Where = %+v, want And of 2", q.Where)
	}
	p0, p1 := q.Where.And[0].Pred, q.Where.And[1].Pred
	if p0 == nil || p0.Column != "id" || p0.Op != tds.OpGe || litOf(p0.Value) != int64(2) {
		t.Errorf("And[0] = %+v", q.Where.And[0])
	}
	if p1 == nil || p1.Column != "name" || p1.Op != tds.OpNe || litOf(p1.Value) != "ada" {
		t.Errorf("And[1] = %+v", q.Where.And[1])
	}
	if len(q.OrderBy) != 1 || q.OrderBy[0].Column != "name" || !q.OrderBy[0].Desc {
		t.Errorf("OrderBy = %v", q.OrderBy)
	}
}

func litOf(v any) any {
	if ve, ok := v.(*tds.ValueExpr); ok {
		return ve.Lit
	}
	return v
}

func TestParseBracketIdent(t *testing.T) {
	q, err := Parse("SELECT [first name] FROM [my table]")
	if err != nil {
		t.Fatal(err)
	}
	if q.Table != "my table" || len(q.Select) != 1 || q.Select[0].Column != "first name" {
		t.Errorf("got Table=%q Select=%v", q.Table, q.Select)
	}
}

func TestParseErrors(t *testing.T) {
	for _, sql := range []string{"", "SELECT", "SELECT * users", "DELETE FROM users", "SELECT * FROM users WHERE id"} {
		if _, err := Parse(sql); err == nil {
			t.Errorf("expected error for %q", sql)
		}
	}
}

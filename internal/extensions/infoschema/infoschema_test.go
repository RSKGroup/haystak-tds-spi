// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package infoschema

import (
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

func sampleSchema() catalog.Schema {
	return catalog.Schema{Tables: []catalog.Table{{
		Name: "users",
		Columns: []catalog.Column{
			{Name: "id", Type: types.Type{Kind: types.Int64}},
			{Name: "name", Type: types.Type{Kind: types.String, MaxLen: 128, Nullable: true}},
		},
		PrimaryKey: []string{"id"},
	}}}
}

func TestTypeNameDeclared(t *testing.T) {
	cases := []struct {
		t    types.Type
		want string
	}{
		{types.Type{Kind: types.String, MaxLen: 10, Name: "varchar"}, "varchar"},
		{types.Type{Kind: types.String, MaxLen: 2, Name: "char"}, "char"},
		{types.Type{Kind: types.Time, Name: "date"}, "date"},
		{types.Type{Kind: types.Int32, Name: "smallint"}, "smallint"},
		{types.Type{Kind: types.Decimal, Name: "MONEY(19,4)"}, "money"},
		{types.Type{Kind: types.String, MaxLen: 10}, "nvarchar"},
		{types.Type{Kind: types.Time}, "datetime2"},
		{types.Type{Kind: types.Int32}, "int"},
	}
	for _, c := range cases {
		if got := TypeName(c.t); got != c.want {
			t.Errorf("TypeName(%+v) = %q, want %q", c.t, got, c.want)
		}
	}
}

func TestResolveColumns(t *testing.T) {
	q := &tds.Query{
		Schema: "INFORMATION_SCHEMA", Table: "COLUMNS",
		Select: []tds.SelectItem{{Column: "COLUMN_NAME"}, {Column: "DATA_TYPE"}, {Column: "IS_NULLABLE"}, {Column: "CHARACTER_MAXIMUM_LENGTH"}},
		Where:  &tds.Expr{Pred: &tds.Predicate{Column: "TABLE_NAME", Op: tds.OpEq, Value: "users"}},
	}
	rs, handled, err := Resolve(sampleSchema(), nil, q)
	if err != nil {
		t.Fatal(err)
	}
	if !handled {
		t.Fatal("expected handled=true")
	}
	defer rs.Close()

	var got [][]any
	for rs.Next() {
		v, _ := rs.Values()
		got = append(got, v)
	}
	if len(got) != 2 {
		t.Fatalf("rows = %v, want 2", got)
	}
	if got[0][0] != "id" || got[0][1] != "bigint" || got[0][2] != "NO" || got[0][3] != nil {
		t.Errorf("row0 = %v, want [id bigint NO <nil>]", got[0])
	}
	if got[1][0] != "name" || got[1][1] != "nvarchar" || got[1][2] != "YES" || got[1][3] != int64(128) {
		t.Errorf("row1 = %v, want [name nvarchar YES 128]", got[1])
	}
}

func TestResolveNonInfoSchema(t *testing.T) {
	_, handled, err := Resolve(sampleSchema(), nil, &tds.Query{Table: "users"})
	if err != nil {
		t.Fatal(err)
	}
	if handled {
		t.Fatal("expected handled=false for a non-INFORMATION_SCHEMA query")
	}
}

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package exec

import (
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

func cols() []catalog.Column {
	return []catalog.Column{
		{Name: "id", Type: types.Type{Kind: types.Int64}},
		{Name: "name", Type: types.Type{Kind: types.String}},
	}
}

func data() [][]any {
	return [][]any{{int64(2), "alan"}, {int64(1), "ada"}, {int64(3), "grace"}}
}

func pred(col string, op tds.Op, v any) *tds.Expr {
	return &tds.Expr{Pred: &tds.Predicate{Column: col, Op: op, Value: v}}
}

func sel(names ...string) []tds.SelectItem {
	out := make([]tds.SelectItem, len(names))
	for i, n := range names {
		out[i] = tds.SelectItem{Column: n}
	}
	return out
}

func collect(t *testing.T, q *tds.Query) [][]any {
	t.Helper()
	rs, err := Apply(cols(), data(), q)
	if err != nil {
		t.Fatal(err)
	}
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

func TestFilterProject(t *testing.T) {
	got := collect(t, &tds.Query{Select: sel("name"), Where: pred("id", tds.OpGe, int64(2))})
	if len(got) != 2 || len(got[0]) != 1 {
		t.Fatalf("got %v", got)
	}
}

func TestOrderByDescLimit(t *testing.T) {
	got := collect(t, &tds.Query{OrderBy: []tds.OrderItem{{Column: "id", Desc: true}}, Limit: 2})
	if len(got) != 2 || got[0][0] != int64(3) || got[1][0] != int64(2) {
		t.Fatalf("got %v", got)
	}
}

func TestOr(t *testing.T) {
	q := &tds.Query{Where: &tds.Expr{Or: []*tds.Expr{pred("id", tds.OpEq, int64(1)), pred("id", tds.OpEq, int64(3))}}}
	if got := collect(t, q); len(got) != 2 {
		t.Fatalf("OR got %v", got)
	}
}

func TestIn(t *testing.T) {
	if got := collect(t, &tds.Query{Where: pred("id", tds.OpIn, []any{int64(1), int64(3)})}); len(got) != 2 {
		t.Fatalf("IN got %v", got)
	}
}

func TestLike(t *testing.T) {
	if got := collect(t, &tds.Query{Where: pred("name", tds.OpLike, "a%")}); len(got) != 2 {
		t.Fatalf("LIKE got %v", got)
	}
}

func TestNot(t *testing.T) {
	if got := collect(t, &tds.Query{Where: &tds.Expr{Not: pred("id", tds.OpEq, int64(1))}}); len(got) != 2 {
		t.Fatalf("NOT got %v", got)
	}
}

func TestDistinct(t *testing.T) {
	d := [][]any{{int64(1), "x"}, {int64(2), "x"}, {int64(3), "y"}}
	rs, err := Apply(cols(), d, &tds.Query{Distinct: true, Select: sel("name")})
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
	var names []any
	for rs.Next() {
		v, _ := rs.Values()
		names = append(names, v[0])
	}
	if len(names) != 2 {
		t.Fatalf("distinct names = %v, want 2 (x, y)", names)
	}
}

func TestAlias(t *testing.T) {
	rs, err := Apply(cols(), data(), &tds.Query{Select: []tds.SelectItem{{Column: "name", Alias: "who"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
	if c := rs.Columns(); len(c) != 1 || c[0].Name != "who" {
		t.Fatalf("alias column = %+v, want who", c)
	}
}

func TestCount(t *testing.T) {
	rs, err := Apply(cols(), data(), &tds.Query{Select: []tds.SelectItem{{Agg: tds.AggCount, Arg: "*"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
	rs.Next()
	v, _ := rs.Values()
	if v[0] != int64(3) {
		t.Fatalf("count = %v, want 3", v[0])
	}
}

func TestSumMaxMin(t *testing.T) {
	q := &tds.Query{Select: []tds.SelectItem{
		{Agg: tds.AggSum, Arg: "id"},
		{Agg: tds.AggMax, Arg: "id"},
		{Agg: tds.AggMin, Arg: "id"},
	}}
	rs, err := Apply(cols(), data(), q)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
	rs.Next()
	v, _ := rs.Values()
	if v[0] != float64(6) || v[1] != int64(3) || v[2] != int64(1) {
		t.Fatalf("sum/max/min = %v, want [6 3 1]", v)
	}
}

func TestGroupBy(t *testing.T) {
	d := [][]any{{int64(1), "x"}, {int64(2), "x"}, {int64(3), "y"}}
	q := &tds.Query{
		Select:  []tds.SelectItem{{Column: "name"}, {Agg: tds.AggCount, Arg: "*", Alias: "n"}},
		GroupBy: []string{"name"},
	}
	rs, err := Apply(cols(), d, q)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
	counts := map[string]int64{}
	for rs.Next() {
		v, _ := rs.Values()
		counts[v[0].(string)] = v[1].(int64)
	}
	if counts["x"] != 2 || counts["y"] != 1 {
		t.Fatalf("group counts = %v", counts)
	}
}

func TestHaving(t *testing.T) {
	d := [][]any{{int64(1), "x"}, {int64(2), "x"}, {int64(3), "y"}}
	q := &tds.Query{
		Select:  []tds.SelectItem{{Column: "name"}, {Agg: tds.AggCount, Arg: "*", Alias: "n"}},
		GroupBy: []string{"name"},
		Having:  &tds.Expr{Pred: &tds.Predicate{Column: "n", Op: tds.OpGt, Value: int64(1)}},
	}
	rs, err := Apply(cols(), d, q)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
	var names []string
	for rs.Next() {
		v, _ := rs.Values()
		names = append(names, v[0].(string))
	}
	if len(names) != 1 || names[0] != "x" {
		t.Fatalf("having = %v, want [x]", names)
	}
}

func TestJoin(t *testing.T) {
	lcols := []catalog.Column{
		{Name: "u.id", Type: types.Type{Kind: types.Int64}},
		{Name: "u.name", Type: types.Type{Kind: types.String}},
	}
	rcols := []catalog.Column{
		{Name: "o.user_id", Type: types.Type{Kind: types.Int64}},
		{Name: "o.amt", Type: types.Type{Kind: types.Int64}},
	}
	rrows := [][]any{{int64(1), int64(100)}, {int64(2), int64(200)}, {int64(2), int64(50)}}
	on := &tds.Expr{Pred: &tds.Predicate{Column: "u.id", Op: tds.OpEq, Value: tds.ColRef{Name: "o.user_id"}}}

	cols, rows, err := Join(lcols, [][]any{{int64(1), "ada"}, {int64(2), "alan"}}, tds.JoinInner, rcols, rrows, on)
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 4 || len(rows) != 3 {
		t.Fatalf("inner join = %d cols / %d rows, want 4/3", len(cols), len(rows))
	}

	_, lrows, err := Join(lcols, [][]any{{int64(3), "grace"}}, tds.JoinLeft, rcols, rrows, on)
	if err != nil {
		t.Fatal(err)
	}
	if len(lrows) != 1 || lrows[0][2] != nil {
		t.Fatalf("left join = %v, want one row with NULL right", lrows)
	}
}

func TestOffsetFetch(t *testing.T) {
	d := [][]any{{int64(1), "a"}, {int64(2), "b"}, {int64(3), "c"}, {int64(4), "d"}}
	q := &tds.Query{OrderBy: []tds.OrderItem{{Column: "id"}}, Offset: 1, Limit: 2}
	rs, err := Apply(cols(), d, q)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
	var ids []int64
	for rs.Next() {
		v, _ := rs.Values()
		ids = append(ids, v[0].(int64))
	}
	if len(ids) != 2 || ids[0] != 2 || ids[1] != 3 {
		t.Fatalf("offset/fetch = %v, want [2 3]", ids)
	}
}

func TestExpr(t *testing.T) {
	q := &tds.Query{Select: []tds.SelectItem{
		{Expr: &tds.ValueExpr{Kind: tds.ValBinary, Op: "+",
			Left:  &tds.ValueExpr{Kind: tds.ValCol, Col: "id"},
			Right: &tds.ValueExpr{Kind: tds.ValLit, Lit: int64(10)}}, Alias: "x"},
		{Expr: &tds.ValueExpr{Kind: tds.ValFunc, Func: "UPPER",
			Args: []*tds.ValueExpr{{Kind: tds.ValCol, Col: "name"}}}, Alias: "u"},
	}}
	rs, err := Apply(cols(), data(), q)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
	rs.Next()
	v, _ := rs.Values()
	if v[0] != int64(12) || v[1] != "ALAN" {
		t.Fatalf("expr = %v, want [12 ALAN]", v)
	}
}

func TestCaseCast(t *testing.T) {
	caseExpr := &tds.ValueExpr{Kind: tds.ValCase,
		Whens: []tds.CaseWhen{{
			Cond:   &tds.Expr{Pred: &tds.Predicate{Column: "id", Op: tds.OpEq, Value: int64(2)}},
			Result: &tds.ValueExpr{Kind: tds.ValLit, Lit: "match"},
		}},
		Else: &tds.ValueExpr{Kind: tds.ValLit, Lit: "no"},
	}
	castExpr := &tds.ValueExpr{Kind: tds.ValCast, Cast: "VARCHAR",
		Left: &tds.ValueExpr{Kind: tds.ValCol, Col: "id"}}
	q := &tds.Query{Select: []tds.SelectItem{{Expr: caseExpr, Alias: "c"}, {Expr: castExpr, Alias: "s"}}}
	rs, err := Apply(cols(), data(), q)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
	rs.Next()
	v, _ := rs.Values()
	if v[0] != "match" || v[1] != "2" {
		t.Fatalf("case/cast = %v, want [match 2]", v)
	}
}

func TestWhereExpr(t *testing.T) {
	where := &tds.Expr{Pred: &tds.Predicate{
		LeftExpr: &tds.ValueExpr{Kind: tds.ValFunc, Func: "LEN",
			Args: []*tds.ValueExpr{{Kind: tds.ValCol, Col: "name"}}},
		Op:    tds.OpEq,
		Value: &tds.ValueExpr{Kind: tds.ValLit, Lit: int64(4)},
	}}
	rs, err := Apply(cols(), data(), &tds.Query{Where: where, Select: sel("name")})
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
	var names []string
	for rs.Next() {
		v, _ := rs.Values()
		names = append(names, v[0].(string))
	}
	if len(names) != 1 || names[0] != "alan" {
		t.Fatalf("where expr = %v, want [alan]", names)
	}
}

func TestUnknownColumnErrors(t *testing.T) {
	if _, err := Apply(cols(), data(), &tds.Query{Where: pred("nope", tds.OpEq, int64(1))}); err == nil {
		t.Fatal("expected error for unknown WHERE column")
	}
	if _, err := Apply(cols(), data(), &tds.Query{Select: sel("nope")}); err == nil {
		t.Fatal("expected error for unknown SELECT column")
	}
}

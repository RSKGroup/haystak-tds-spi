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

func TestParseTopParen(t *testing.T) {
	cases := []struct {
		sql     string
		limit   int
		percent bool
	}{
		{"SELECT TOP (2) id FROM orders ORDER BY id", 2, false},
		{"SELECT TOP (50) PERCENT id FROM orders ORDER BY id", 50, true},
		{"SELECT TOP 5 id FROM orders", 5, false},
		{"SELECT TOP 10 PERCENT id FROM orders", 10, true},
	}
	for _, c := range cases {
		q, err := Parse(c.sql)
		if err != nil {
			t.Fatalf("%s: %v", c.sql, err)
		}
		if q.Limit != c.limit || q.LimitPercent != c.percent {
			t.Errorf("%s: Limit=%d Percent=%v, want %d %v", c.sql, q.Limit, q.LimitPercent, c.limit, c.percent)
		}
	}
	if _, err := Parse("SELECT TOP (2 id FROM orders"); err == nil {
		t.Error("expected error for unclosed TOP paren")
	}
}

func TestParseDerivedJoin(t *testing.T) {
	q, err := Parse("SELECT u.name FROM users u JOIN (SELECT user_id FROM orders) o ON o.user_id = u.id")
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Joins) != 1 {
		t.Fatalf("Joins = %d, want 1", len(q.Joins))
	}
	j := q.Joins[0]
	if j.FromSub == nil || j.Table != "" || j.Alias != "o" {
		t.Errorf("join = %+v, want FromSub set, Table empty, Alias o", j)
	}
	if j.FromSub.Table != "orders" {
		t.Errorf("derived join inner Table = %q, want orders", j.FromSub.Table)
	}
}

func litOf(v any) any {
	if ve, ok := v.(*tds.ValueExpr); ok {
		return ve.Lit
	}
	return v
}

func TestParseHintsIgnored(t *testing.T) {
	// Table hints, TABLESAMPLE, and OPTION parse and are discarded — the core query is unaffected.
	cases := []string{
		"SELECT name FROM users WITH (NOLOCK) WHERE id = 2",
		"SELECT id FROM orders TABLESAMPLE (50 PERCENT)",
		"SELECT id FROM orders TABLESAMPLE SYSTEM (10 ROWS) REPEATABLE (5)",
		"SELECT u.id FROM users u WITH (NOLOCK) JOIN orders o WITH (NOLOCK, INDEX(1)) ON o.user_id = u.id",
		"SELECT id FROM orders ORDER BY id OPTION (MAXDOP 1, RECOMPILE)",
	}
	for _, sql := range cases {
		if _, err := Parse(sql); err != nil {
			t.Errorf("Parse(%q) = %v", sql, err)
		}
	}
	q, err := Parse("SELECT name FROM users WITH (NOLOCK) WHERE id = 2")
	if err != nil {
		t.Fatal(err)
	}
	if q.Table != "users" || q.Where == nil {
		t.Errorf("hint changed the query: Table=%q Where=%v", q.Table, q.Where)
	}
}

func TestParsePivot(t *testing.T) {
	q, err := Parse("SELECT region, [A] FROM sales PIVOT (SUM(amount) FOR product IN ([A], [B])) AS p")
	if err != nil {
		t.Fatal(err)
	}
	pv := q.Pivot
	if pv == nil || pv.Agg != "SUM" || pv.ValueCol != "amount" || pv.PivotCol != "product" ||
		len(pv.Values) != 2 || pv.Values[0] != "A" || pv.Values[1] != "B" || pv.Alias != "p" {
		t.Errorf("Pivot = %+v", pv)
	}
	u, err := Parse("SELECT * FROM src UNPIVOT (amount FOR quarter IN (q1, q2)) AS u")
	if err != nil {
		t.Fatal(err)
	}
	up := u.Unpivot
	if up == nil || up.ValueCol != "amount" || up.NameCol != "quarter" || len(up.Columns) != 2 || up.Alias != "u" {
		t.Errorf("Unpivot = %+v", up)
	}
}

func TestParseGroupingSets(t *testing.T) {
	r, err := Parse("SELECT a FROM t GROUP BY ROLLUP(a, b)")
	if err != nil {
		t.Fatal(err)
	}
	if got := r.GroupingSets; len(got) != 3 || len(got[0]) != 2 || len(got[1]) != 1 || len(got[2]) != 0 {
		t.Errorf("ROLLUP sets = %v, want [[a b] [a] []]", got)
	}
	c, err := Parse("SELECT a FROM t GROUP BY CUBE(a, b)")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.GroupingSets) != 4 {
		t.Errorf("CUBE sets = %v, want 4 sets", c.GroupingSets)
	}
	g, err := Parse("SELECT a FROM t GROUP BY GROUPING SETS((a, b), (a), ())")
	if err != nil {
		t.Fatal(err)
	}
	if len(g.GroupingSets) != 3 || len(g.GroupBy) != 2 {
		t.Errorf("GROUPING SETS = %v, universe %v", g.GroupingSets, g.GroupBy)
	}
	// A plain GROUP BY leaves GroupingSets nil.
	p, err := Parse("SELECT a FROM t GROUP BY a, b")
	if err != nil {
		t.Fatal(err)
	}
	if p.GroupingSets != nil || len(p.GroupBy) != 2 {
		t.Errorf("plain GROUP BY = sets %v, cols %v", p.GroupingSets, p.GroupBy)
	}
}

func TestParseApply(t *testing.T) {
	q, err := Parse("SELECT u.id FROM users u CROSS APPLY (SELECT amount FROM orders WHERE user_id = u.id) x")
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Joins) != 1 || q.Joins[0].Type != tds.JoinCrossApply || q.Joins[0].FromSub == nil || q.Joins[0].Alias != "x" {
		t.Errorf("CROSS APPLY join = %+v", q.Joins)
	}
	q2, err := Parse("SELECT t.id FROM t OUTER APPLY STRING_SPLIT(t.tags, ',') s")
	if err != nil {
		t.Fatal(err)
	}
	j := q2.Joins[0]
	if j.Type != tds.JoinOuterApply || j.FromFunc == nil || j.FromFunc.Name != "STRING_SPLIT" || j.Alias != "s" {
		t.Errorf("OUTER APPLY TVF join = %+v", j)
	}
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

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tsql

import "testing"

func TestCollatePredicate(t *testing.T) {
	cases := []struct {
		sql  string
		want bool
	}{
		{"SELECT a FROM t WHERE name = 'x'", false},
		{"SELECT a FROM t WHERE name COLLATE Latin1_General_CS_AS = 'x'", true},
		{"SELECT a FROM t WHERE name = 'x' COLLATE SQL_Latin1_General_CP1_CS_AS", true},
		{"SELECT a FROM t WHERE name = 'x' COLLATE SQL_Latin1_General_CP1_CI_AS", false},
		{"SELECT a FROM t WHERE name LIKE 'x%' COLLATE Latin1_General_CS_AS", true},
		{"SELECT a FROM t WHERE name COLLATE Latin1_General_CS_AS IN ('x', 'y')", true},
		{"SELECT a FROM t WHERE name IN ('x' COLLATE Latin1_General_CS_AS, 'y')", true},
		{"SELECT a FROM t WHERE name COLLATE Latin1_General_CS_AS IN (SELECT b FROM u)", true},
	}
	for _, c := range cases {
		q, err := Parse(c.sql)
		if err != nil {
			t.Fatalf("%s: %v", c.sql, err)
		}
		if q.Where.Pred == nil || q.Where.Pred.CaseSensitive != c.want {
			t.Errorf("%s: CaseSensitive = %+v, want %v", c.sql, q.Where.Pred, c.want)
		}
	}
}

func TestCollateBetweenAndNot(t *testing.T) {
	q, err := Parse("SELECT a FROM t WHERE name NOT BETWEEN 'a' AND 'b' COLLATE Latin1_General_CS_AS")
	if err != nil {
		t.Fatal(err)
	}
	and := q.Where.Not.And
	if len(and) != 2 || !and[0].Pred.CaseSensitive || !and[1].Pred.CaseSensitive {
		t.Errorf("BETWEEN bounds not case-sensitive: %+v %+v", and[0].Pred, and[1].Pred)
	}
}

func TestCollateOrderAndGroup(t *testing.T) {
	q, err := Parse("SELECT name, COUNT(*) FROM t GROUP BY name COLLATE Latin1_General_CS_AS, city ORDER BY name COLLATE Latin1_General_CS_AS DESC, city")
	if err != nil {
		t.Fatal(err)
	}
	if len(q.GroupBy) != 2 || len(q.GroupByCaseSensitive) != 1 || q.GroupByCaseSensitive[0] != "name" {
		t.Errorf("GroupBy = %v, GroupByCaseSensitive = %v", q.GroupBy, q.GroupByCaseSensitive)
	}
	if len(q.OrderBy) != 2 || !q.OrderBy[0].CaseSensitive || !q.OrderBy[0].Desc || q.OrderBy[1].CaseSensitive {
		t.Errorf("OrderBy = %+v", q.OrderBy)
	}
	q, err = Parse("SELECT a, b, COUNT(*) FROM t GROUP BY ROLLUP(a COLLATE Latin1_General_CS_AS, b)")
	if err != nil {
		t.Fatal(err)
	}
	if len(q.GroupByCaseSensitive) != 1 || q.GroupByCaseSensitive[0] != "a" {
		t.Errorf("ROLLUP GroupByCaseSensitive = %v", q.GroupByCaseSensitive)
	}
}

func TestCollateWriteWhere(t *testing.T) {
	stmt, _, err := ParseWrite("DELETE FROM t WHERE name = 'x' COLLATE Latin1_General_CS_AS AND id = 1")
	if err != nil {
		t.Fatal(err)
	}
	w := stmt.Delete.Where
	if len(w) != 2 || !w[0].CaseSensitive || w[1].CaseSensitive {
		t.Errorf("Delete.Where = %+v", w)
	}
}

func TestCollateMissingName(t *testing.T) {
	if _, err := Parse("SELECT a FROM t WHERE name COLLATE = 'x'"); err == nil {
		t.Errorf("expected an error for COLLATE without a collation name")
	}
}

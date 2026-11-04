// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tsql

import (
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/tds"
)

func mustParse(t *testing.T, sql string) *tds.Query {
	t.Helper()
	q, err := Parse(sql)
	if err != nil {
		t.Fatalf("Parse(%q) = %v", sql, err)
	}
	return q
}

func TestParseLikeAndBetweenOperands(t *testing.T) {
	if v := mustParse(t, "SELECT a FROM t WHERE a LIKE 'x%'").Where.Pred.Value; v != "x%" {
		t.Errorf("literal LIKE pattern = %#v, want the string x%%", v)
	}
	ve, ok := mustParse(t, "SELECT a FROM t WHERE a LIKE b + '%'").Where.Pred.Value.(*tds.ValueExpr)
	if !ok || ve.Kind != tds.ValBinary {
		t.Errorf("expression LIKE pattern = %#v, want a binary *ValueExpr", ve)
	}
	if w := mustParse(t, "SELECT a FROM t WHERE a NOT LIKE UPPER(b)").Where; w.Not == nil || w.Not.Pred.Op != tds.OpLike {
		t.Errorf("NOT LIKE with a function pattern = %#v", w)
	}
	and := mustParse(t, "SELECT a FROM t WHERE a BETWEEN 1 AND b + 1").Where.And
	if len(and) != 2 || and[0].Pred.Value != int64(1) {
		t.Fatalf("BETWEEN low bound = %#v, want literal 1", and)
	}
	if hi, ok := and[1].Pred.Value.(*tds.ValueExpr); !ok || hi.Kind != tds.ValBinary {
		t.Errorf("BETWEEN high bound = %#v, want an expression", and[1].Pred.Value)
	}
}

func TestParseSetOpSubqueriesAndAggregateOperand(t *testing.T) {
	q := mustParse(t, "SELECT x.id FROM (SELECT id FROM t UNION ALL SELECT id FROM u) x")
	if q.FromSub == nil || q.FromSub.Union == nil || q.FromSub.SetOp != tds.SetUnionAll {
		t.Errorf("derived UNION ALL = %#v", q.FromSub)
	}
	q = mustParse(t, "SELECT a FROM t WHERE EXISTS (SELECT 1 FROM u UNION SELECT 2 FROM v)")
	if sub := q.Where.Pred.Sub; sub == nil || sub.Union == nil {
		t.Errorf("EXISTS over a UNION = %#v", q.Where.Pred)
	}
	q = mustParse(t, "SELECT COUNT(*) + 1 AS n FROM t")
	if it := q.Select[0]; it.Agg != tds.AggNone || it.Expr == nil || it.Expr.Kind != tds.ValBinary || it.Alias != "n" {
		t.Errorf("COUNT(*) + 1 = %#v, want an expression item", it)
	}
	q = mustParse(t, "SELECT v.x FROM (VALUES (1), (2)) v(x)")
	if q.FromSub == nil || q.FromSub.Union == nil || q.FromSub.Select[0].Alias != "x" || q.FromAlias != "v" {
		t.Errorf("FROM (VALUES ...) = %#v", q.FromSub)
	}
}

func TestParseConvertStyleAndLength(t *testing.T) {
	ve := mustParse(t, "SELECT CONVERT(char(10), d, 23) FROM t").Select[0].Expr
	if ve == nil || ve.Kind != tds.ValCast || ve.Cast != "CHAR" || ve.CastLen != 10 || len(ve.Args) != 1 || ve.Args[0].Lit != int64(23) {
		t.Errorf("CONVERT(char(10), d, 23) = %#v", ve)
	}
	ve = mustParse(t, "SELECT CAST(d AS varchar(max)) FROM t").Select[0].Expr
	if ve == nil || ve.Cast != "VARCHAR" || ve.CastLen != 0 || ve.Args != nil {
		t.Errorf("CAST(d AS varchar(max)) = %#v", ve)
	}
	ve = mustParse(t, "SELECT CAST(d AS decimal(10, 2)) FROM t").Select[0].Expr
	if ve == nil || ve.Cast != "DECIMAL" || ve.CastLen != 0 {
		t.Errorf("CAST(d AS decimal(10, 2)) = %#v", ve)
	}
}

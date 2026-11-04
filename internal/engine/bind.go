// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/tds"
)

// binder puts an enclosing row's values into a subquery at any depth; a name the subquery defines shadows it.
type binder struct {
	resolve func(string) (any, bool)
}

func bindOuter(sub *tds.Query, resolve func(string) (any, bool)) *tds.Query {
	return binder{resolve}.query(sub, nil)
}

// isCorrelated reports whether sub reads any of the enclosing query's sources, anywhere inside it.
func isCorrelated(sub *tds.Query, quals map[string]bool) bool {
	found := false
	bindOuter(sub, func(col string) (any, bool) {
		if quals[strings.ToLower(qualifierOf(col))] {
			found = true
		}
		return nil, false
	})
	return found
}

func qualifierOf(col string) string {
	if dot := strings.LastIndex(col, "."); dot >= 0 {
		return col[:dot]
	}
	return ""
}

// sourceQuals names q's FROM and JOIN sources; exposed keeps only the alias where one is given, as SQL Server does.
func sourceQuals(q *tds.Query, exposed bool) map[string]bool {
	m := map[string]bool{}
	add := func(alias, schema, table string) {
		if alias != "" {
			m[strings.ToLower(alias)] = true
			if exposed {
				return
			}
		}
		if table != "" {
			m[strings.ToLower(table)] = true
			if schema != "" {
				m[strings.ToLower(schema+"."+table)] = true
			}
		}
	}
	add(q.FromAlias, q.Schema, q.Table)
	for _, j := range q.Joins {
		add(j.Alias, j.Schema, j.Table)
	}
	if q.Pivot != nil {
		add(q.Pivot.Alias, "", "")
	}
	if q.Unpivot != nil {
		add(q.Unpivot.Alias, "", "")
	}
	return m
}

func withSources(local map[string]bool, q *tds.Query) map[string]bool {
	out := sourceQuals(q, true)
	for k := range local {
		out[k] = true
	}
	return out
}

func (b binder) lookup(col string, local map[string]bool) (any, bool) {
	q := qualifierOf(col)
	if q == "" || local[strings.ToLower(q)] {
		return nil, false
	}
	return b.resolve(col)
}

func litVal(v any) *tds.ValueExpr { return &tds.ValueExpr{Kind: tds.ValLit, Lit: v} }

func (b binder) query(q *tds.Query, local map[string]bool) *tds.Query {
	if q == nil {
		return nil
	}
	c := *q
	inner := withSources(local, q)
	if q.Select != nil {
		c.Select = make([]tds.SelectItem, len(q.Select))
		for i, it := range q.Select {
			if it.Column != "" && !strings.HasSuffix(it.Column, ".*") {
				if v, ok := b.lookup(it.Column, inner); ok {
					if it.Alias == "" {
						it.Alias = it.Column[strings.LastIndex(it.Column, ".")+1:]
					}
					it.Expr, it.Column = litVal(v), ""
				}
			}
			it.Expr = b.val(it.Expr, inner)
			it.ArgExpr = b.val(it.ArgExpr, inner)
			if it.Window != nil {
				w := *it.Window
				w.Args = b.vals(w.Args, inner)
				it.Window = &w
			}
			c.Select[i] = it
		}
	}
	c.Where = b.expr(q.Where, inner)
	c.Having = b.expr(q.Having, inner)
	if q.OrderBy != nil {
		c.OrderBy = make([]tds.OrderItem, len(q.OrderBy))
		for i, o := range q.OrderBy {
			if o.Column != "" {
				if v, ok := b.lookup(o.Column, inner); ok {
					o.Expr, o.Column = litVal(v), ""
				}
			}
			o.Expr = b.val(o.Expr, inner)
			c.OrderBy[i] = o
		}
	}
	// A derived table cannot see its query's other sources, only what encloses the query.
	c.FromSub = b.query(q.FromSub, local)
	if q.FromFunc != nil {
		f := *q.FromFunc
		f.Args = b.vals(f.Args, local)
		c.FromFunc = &f
	}
	if q.Joins != nil {
		c.Joins = make([]tds.Join, len(q.Joins))
		for i, j := range q.Joins {
			j.On = b.expr(j.On, inner)
			j.FromSub = b.query(j.FromSub, inner)
			if j.FromFunc != nil {
				f := *j.FromFunc
				f.Args = b.vals(f.Args, inner)
				j.FromFunc = &f
			}
			if j.FromValues != nil {
				v := *j.FromValues
				v.Rows = make([][]*tds.ValueExpr, len(j.FromValues.Rows))
				for r, row := range j.FromValues.Rows {
					v.Rows[r] = b.vals(row, inner)
				}
				j.FromValues = &v
			}
			c.Joins[i] = j
		}
	}
	c.Union = b.query(q.Union, local)
	return &c
}

func (b binder) vals(in []*tds.ValueExpr, local map[string]bool) []*tds.ValueExpr {
	if in == nil {
		return nil
	}
	out := make([]*tds.ValueExpr, len(in))
	for i, a := range in {
		out[i] = b.val(a, local)
	}
	return out
}

func (b binder) val(ve *tds.ValueExpr, local map[string]bool) *tds.ValueExpr {
	if ve == nil || ve.Kind == tds.ValLit {
		return ve
	}
	if ve.Kind == tds.ValCol {
		if v, ok := b.lookup(ve.Col, local); ok {
			return litVal(v)
		}
		return ve
	}
	out := *ve
	out.Left = b.val(ve.Left, local)
	out.Right = b.val(ve.Right, local)
	out.Operand = b.val(ve.Operand, local)
	out.Else = b.val(ve.Else, local)
	out.Args = b.vals(ve.Args, local)
	if ve.Whens != nil {
		out.Whens = make([]tds.CaseWhen, len(ve.Whens))
		for i, w := range ve.Whens {
			out.Whens[i] = tds.CaseWhen{Cond: b.expr(w.Cond, local), Match: b.val(w.Match, local), Result: b.val(w.Result, local)}
		}
	}
	out.Sub = b.query(ve.Sub, local)
	return &out
}

func (b binder) expr(e *tds.Expr, local map[string]bool) *tds.Expr {
	if e == nil {
		return nil
	}
	out := *e
	if e.Pred != nil {
		p := *e.Pred
		if p.Column != "" {
			if v, ok := b.lookup(p.Column, local); ok {
				p.LeftExpr, p.Column = litVal(v), ""
			}
		}
		p.LeftExpr = b.val(p.LeftExpr, local)
		switch v := p.Value.(type) {
		case tds.ColRef:
			if x, ok := b.lookup(v.Name, local); ok {
				p.Value = litVal(x)
			}
		case *tds.ValueExpr:
			p.Value = b.val(v, local)
		}
		p.Sub = b.query(p.Sub, local)
		out.Pred = &p
	}
	out.And = b.exprs(e.And, local)
	out.Or = b.exprs(e.Or, local)
	out.Not = b.expr(e.Not, local)
	return &out
}

func (b binder) exprs(in []*tds.Expr, local map[string]bool) []*tds.Expr {
	if in == nil {
		return nil
	}
	out := make([]*tds.Expr, len(in))
	for i, e := range in {
		out[i] = b.expr(e, local)
	}
	return out
}

// rowResolver reads a qualified column from a row of q: alias.col in a joined row, the bare name otherwise.
func rowResolver(row []any, idx map[string]int, quals map[string]bool, joined bool) func(string) (any, bool) {
	return func(col string) (any, bool) {
		q := qualifierOf(col)
		if q == "" || !quals[strings.ToLower(q)] {
			return nil, false
		}
		if !joined {
			if i, ok := idx[col[len(q)+1:]]; ok {
				return row[i], true
			}
			return nil, false
		}
		return colValue(row, idx, col)
	}
}

// leftRowResolver resolves an APPLY right side's correlated reference to a left column (always alias-qualified).
// A nil row is a schema probe (no left rows): correlated refs bind to "" so the right side yields its columns.
func leftRowResolver(row []any, idx map[string]int) func(string) (any, bool) {
	return func(col string) (any, bool) {
		if !strings.Contains(col, ".") {
			return nil, false
		}
		i, ok := colIndex(idx, col)
		if !ok {
			return nil, false
		}
		if row == nil {
			return "", true
		}
		return row[i], true
	}
}

func colValue(row []any, idx map[string]int, col string) (any, bool) {
	if i, ok := colIndex(idx, col); ok {
		return row[i], true
	}
	return nil, false
}

// colIndex finds a qualified column of a joined row, matching the name case-insensitively as T-SQL does.
func colIndex(idx map[string]int, col string) (int, bool) {
	if i, ok := idx[col]; ok {
		return i, true
	}
	for name, i := range idx {
		if strings.EqualFold(name, col) {
			return i, true
		}
	}
	return 0, false
}

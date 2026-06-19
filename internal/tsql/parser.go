// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tsql

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/tds"
)

// specialFuncs are the parser-only function-shaped constructs (conversion family + IIF), not in the registry.
var specialFuncs = []string{"CAST", "CONVERT", "TRY_CAST", "TRY_CONVERT", "PARSE", "TRY_PARSE", "IIF"}

// SpecialFuncs returns the parser-only special-cased function names.
func SpecialFuncs() []string { return append([]string(nil), specialFuncs...) }

// Parse turns a read-subset T-SQL SELECT into a tds.Query.
func Parse(sql string) (*tds.Query, error) {
	toks, err := lex(sql)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks}
	var ctes map[string]*tds.Query
	if p.isKeyword("WITH") {
		ctes, err = p.cteList()
		if err != nil {
			return nil, err
		}
	}
	q, err := p.selectStmt()
	if err != nil {
		return nil, err
	}
	q.CTEs = ctes
	if err := p.unionTail(q); err != nil {
		return nil, err
	}
	if err := p.optionClause(); err != nil {
		return nil, err
	}
	if p.peek().kind != tEOF {
		return nil, fmt.Errorf("tsql: unexpected %q after query", p.peek().text)
	}
	return q, nil
}

func (p *parser) unionTail(q *tds.Query) error {
	for p.isKeyword("UNION") || p.isKeyword("INTERSECT") || p.isKeyword("EXCEPT") {
		op := tds.SetUnion
		switch {
		case p.isKeyword("UNION"):
			p.next()
			if p.isKeyword("ALL") {
				p.next()
				op = tds.SetUnionAll
			}
		case p.isKeyword("INTERSECT"):
			p.next()
			op = tds.SetIntersect
		case p.isKeyword("EXCEPT"):
			p.next()
			op = tds.SetExcept
		}
		next, err := p.selectStmt()
		if err != nil {
			return err
		}
		tail := q
		for tail.Union != nil {
			tail = tail.Union
		}
		tail.Union = next
		tail.SetOp = op
	}
	return nil
}

func (p *parser) cteList() (map[string]*tds.Query, error) {
	p.next() // WITH
	ctes := map[string]*tds.Query{}
	for {
		name := p.peek()
		if name.kind != tIdent {
			return nil, fmt.Errorf("tsql: expected CTE name, got %q", name.text)
		}
		p.next()
		if err := p.expectKeyword("AS"); err != nil {
			return nil, err
		}
		if p.peek().kind != tLParen {
			return nil, fmt.Errorf("tsql: expected '(' after CTE AS, got %q", p.peek().text)
		}
		p.next()
		sub, err := p.selectStmt()
		if err != nil {
			return nil, err
		}
		if err := p.unionTail(sub); err != nil {
			return nil, err
		}
		if p.peek().kind != tRParen {
			return nil, fmt.Errorf("tsql: expected ')' after CTE, got %q", p.peek().text)
		}
		p.next()
		ctes[name.text] = sub
		if p.peek().kind == tComma {
			p.next()
			continue
		}
		break
	}
	return ctes, nil
}

type parser struct {
	toks []token
	pos  int
}

func (p *parser) peek() token { return p.toks[p.pos] }

func (p *parser) peekN(n int) token {
	i := p.pos + n
	if i >= len(p.toks) {
		i = len(p.toks) - 1
	}
	return p.toks[i]
}

// identLike: current token usable as an identifier (plain ident or non-reserved keyword).
func (p *parser) identLike() bool {
	t := p.peek()
	return t.kind == tIdent || (t.kind == tKeyword && nonReserved[strings.ToUpper(t.text)])
}

func (p *parser) qualifiedName() (string, bool) {
	if !p.identLike() {
		return "", false
	}
	name := p.peek().text
	p.next()
	for p.peek().kind == tDot {
		p.next()
		if !p.identLike() {
			break
		}
		name += "." + p.peek().text
		p.next()
	}
	return name, true
}

func (p *parser) optTableAlias() string {
	if p.isKeyword("AS") {
		p.next()
	}
	if p.peek().kind == tIdent {
		a := p.peek().text
		p.next()
		return a
	}
	return ""
}

// aliasWithCols parses `[AS] alias [(col, col, …)]` for a derived/VALUES table source.
func (p *parser) aliasWithCols() (string, []string) {
	alias := p.optTableAlias()
	var cols []string
	if p.peek().kind == tLParen {
		p.next()
		for p.peek().kind == tIdent {
			cols = append(cols, p.peek().text)
			p.next()
			if p.peek().kind == tComma {
				p.next()
			}
		}
		if p.peek().kind == tRParen {
			p.next()
		}
	}
	return alias, cols
}

// valuesClause parses `VALUES (e, e, …), (…), …` into rows of expressions.
func (p *parser) valuesClause() (*tds.ValuesClause, error) {
	p.next() // VALUES
	vc := &tds.ValuesClause{}
	for {
		if p.peek().kind != tLParen {
			return nil, fmt.Errorf("tsql: expected '(' in VALUES, got %q", p.peek().text)
		}
		p.next()
		var row []*tds.ValueExpr
		for {
			ve, err := p.valueExpr()
			if err != nil {
				return nil, err
			}
			row = append(row, ve)
			if p.peek().kind == tComma {
				p.next()
				continue
			}
			break
		}
		if p.peek().kind != tRParen {
			return nil, fmt.Errorf("tsql: expected ')' after VALUES row, got %q", p.peek().text)
		}
		p.next()
		vc.Rows = append(vc.Rows, row)
		if p.peek().kind == tComma {
			p.next()
			continue
		}
		break
	}
	return vc, nil
}

func (p *parser) optJoin() (*tds.Join, error) {
	var jt tds.JoinType
	switch {
	case p.isKeyword("JOIN"):
		p.next()
		jt = tds.JoinInner
	case p.isKeyword("INNER"):
		p.next()
		if err := p.expectKeyword("JOIN"); err != nil {
			return nil, err
		}
		jt = tds.JoinInner
	case p.isKeyword("LEFT"):
		p.next()
		if p.isKeyword("OUTER") {
			p.next()
		}
		if err := p.expectKeyword("JOIN"); err != nil {
			return nil, err
		}
		jt = tds.JoinLeft
	case p.isKeyword("RIGHT"):
		p.next()
		if p.isKeyword("OUTER") {
			p.next()
		}
		if err := p.expectKeyword("JOIN"); err != nil {
			return nil, err
		}
		jt = tds.JoinRight
	case p.isKeyword("FULL"):
		p.next()
		if p.isKeyword("OUTER") {
			p.next()
		}
		if err := p.expectKeyword("JOIN"); err != nil {
			return nil, err
		}
		jt = tds.JoinFull
	case p.isKeyword("CROSS"):
		p.next()
		if p.isKeyword("APPLY") {
			p.next()
			jt = tds.JoinCrossApply
		} else {
			if err := p.expectKeyword("JOIN"); err != nil {
				return nil, err
			}
			jt = tds.JoinCross
		}
	case p.isKeyword("OUTER"):
		p.next()
		if err := p.expectKeyword("APPLY"); err != nil {
			return nil, err
		}
		jt = tds.JoinOuterApply
	default:
		return nil, nil
	}
	var j *tds.Join
	switch {
	case p.peek().kind == tLParen && p.peekN(1).kind == tKeyword && strings.EqualFold(p.peekN(1).text, "VALUES"):
		p.next() // (
		vc, err := p.valuesClause()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tRParen {
			return nil, fmt.Errorf("tsql: expected ')' after VALUES, got %q", p.peek().text)
		}
		p.next()
		alias, cols := p.aliasWithCols()
		vc.Columns = cols
		j = &tds.Join{Type: jt, FromValues: vc, Alias: alias}
	case p.peek().kind == tLParen:
		p.next()
		sub, err := p.selectStmt()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tRParen {
			return nil, fmt.Errorf("tsql: expected ')' after derived table, got %q", p.peek().text)
		}
		p.next()
		j = &tds.Join{Type: jt, FromSub: sub, Alias: p.optTableAlias()}
	case p.peek().kind == tIdent && p.peekN(1).kind == tLParen:
		ve, err := p.funcCall(strings.ToUpper(p.peek().text))
		if err != nil {
			return nil, err
		}
		j = &tds.Join{Type: jt, FromFunc: &tds.TableFunc{Name: ve.Func, Args: ve.Args}, Alias: p.optTableAlias()}
	default:
		db, sch, tbl, err := p.tableName()
		if err != nil {
			return nil, err
		}
		j = &tds.Join{Type: jt, Database: db, Schema: sch, Table: tbl, Alias: p.optTableAlias()}
	}
	if err := p.tableHints(); err != nil {
		return nil, err
	}
	if jt != tds.JoinCross && jt != tds.JoinCrossApply && jt != tds.JoinOuterApply {
		if err := p.expectKeyword("ON"); err != nil {
			return nil, err
		}
		on, err := p.orExpr()
		if err != nil {
			return nil, err
		}
		j.On = on
	}
	return j, nil
}

func (p *parser) next() token {
	t := p.toks[p.pos]
	if p.pos < len(p.toks)-1 {
		p.pos++
	}
	return t
}

func (p *parser) isKeyword(kw string) bool {
	t := p.peek()
	return t.kind == tKeyword && strings.EqualFold(t.text, kw)
}

func (p *parser) expectKeyword(kw string) error {
	if !p.isKeyword(kw) {
		return fmt.Errorf("tsql: expected %s, got %q", kw, p.peek().text)
	}
	p.next()
	return nil
}

func (p *parser) selectStmt() (*tds.Query, error) {
	if err := p.expectKeyword("SELECT"); err != nil {
		return nil, err
	}
	q := &tds.Query{}

	if p.isKeyword("DISTINCT") {
		p.next()
		q.Distinct = true
	}

	if p.isKeyword("TOP") {
		p.next()
		paren := p.peek().kind == tLParen
		if paren {
			p.next()
		}
		t := p.peek()
		if t.kind != tNumber {
			return nil, fmt.Errorf("tsql: expected number after TOP, got %q", t.text)
		}
		n, err := strconv.Atoi(t.text)
		if err != nil {
			return nil, fmt.Errorf("tsql: bad TOP value %q", t.text)
		}
		q.Limit = n
		p.next()
		if paren {
			if p.peek().kind != tRParen {
				return nil, fmt.Errorf("tsql: expected ')' after TOP value, got %q", p.peek().text)
			}
			p.next()
		}
		if p.isKeyword("PERCENT") {
			p.next()
			q.LimitPercent = true
		}
	}

	if p.peek().kind == tStar {
		p.next()
	} else {
		items, err := p.selectItems()
		if err != nil {
			return nil, err
		}
		q.Select = items
	}

	if p.isKeyword("FROM") {
		p.next()
		if p.peek().kind == tLParen {
			p.next()
			sub, err := p.selectStmt()
			if err != nil {
				return nil, err
			}
			if p.peek().kind != tRParen {
				return nil, fmt.Errorf("tsql: expected ')' after derived table, got %q", p.peek().text)
			}
			p.next()
			q.FromSub = sub
			q.FromAlias = p.optTableAlias()
		} else if p.peek().kind == tIdent && p.peekN(1).kind == tLParen {
			ve, err := p.funcCall(strings.ToUpper(p.peek().text))
			if err != nil {
				return nil, err
			}
			q.FromFunc = &tds.TableFunc{Name: ve.Func, Args: ve.Args}
			q.FromAlias = p.optTableAlias()
		} else {
			db, sch, tbl, err := p.tableName()
			if err != nil {
				return nil, err
			}
			q.Database = db
			q.Schema = sch
			q.Table = tbl
			q.FromAlias = p.optTableAlias()
		}
		if err := p.tableHints(); err != nil {
			return nil, err
		}
		if err := p.pivotClause(q); err != nil {
			return nil, err
		}
	}
	for {
		j, err := p.optJoin()
		if err != nil {
			return nil, err
		}
		if j == nil {
			break
		}
		q.Joins = append(q.Joins, *j)
	}

	if p.isKeyword("WHERE") {
		p.next()
		where, err := p.orExpr()
		if err != nil {
			return nil, err
		}
		q.Where = where
	}

	if p.isKeyword("GROUP") {
		p.next()
		if err := p.expectKeyword("BY"); err != nil {
			return nil, err
		}
		if err := p.groupByClause(q); err != nil {
			return nil, err
		}
	}

	if p.isKeyword("HAVING") {
		p.next()
		h, err := p.orExpr()
		if err != nil {
			return nil, err
		}
		q.Having = h
	}

	if p.isKeyword("ORDER") {
		p.next()
		if err := p.expectKeyword("BY"); err != nil {
			return nil, err
		}
		items, err := p.orderList()
		if err != nil {
			return nil, err
		}
		for i := range items {
			if items[i].Ordinal > 0 {
				oi := items[i].Ordinal - 1
				if oi >= 0 && oi < len(q.Select) {
					si := q.Select[oi]
					if si.Column != "" {
						items[i].Column = si.Column
					} else if si.Alias != "" {
						items[i].Column = si.Alias
					}
				}
			}
		}
		q.OrderBy = items
	}

	if p.isKeyword("OFFSET") {
		p.next()
		n, err := p.intLit()
		if err != nil {
			return nil, err
		}
		q.Offset = n
		if !p.isKeyword("ROWS") && !p.isKeyword("ROW") {
			return nil, fmt.Errorf("tsql: expected ROWS after OFFSET, got %q", p.peek().text)
		}
		p.next()
		if p.isKeyword("FETCH") {
			p.next()
			if p.isKeyword("NEXT") || p.isKeyword("FIRST") {
				p.next()
			}
			m, err := p.intLit()
			if err != nil {
				return nil, err
			}
			q.Limit = m
			if !p.isKeyword("ROWS") && !p.isKeyword("ROW") {
				return nil, fmt.Errorf("tsql: expected ROWS after FETCH, got %q", p.peek().text)
			}
			p.next()
			if err := p.expectKeyword("ONLY"); err != nil {
				return nil, err
			}
		}
	}
	return q, nil
}

func (p *parser) selectItems() ([]tds.SelectItem, error) {
	var out []tds.SelectItem
	for {
		it, err := p.selectItem()
		if err != nil {
			return nil, err
		}
		out = append(out, it)
		if p.peek().kind == tComma {
			p.next()
			continue
		}
		break
	}
	return out, nil
}

func (p *parser) selectItem() (tds.SelectItem, error) {
	leadAlias := ""
	if p.peek().kind == tIdent && p.peekN(1).kind == tOp && p.peekN(1).text == "=" {
		leadAlias = p.peek().text
		p.next()
		p.next()
	}
	t := p.peek()
	if t.kind == tIdent && p.peekN(1).kind == tDot && p.peekN(2).kind == tStar {
		p.next()
		p.next()
		p.next()
		return tds.SelectItem{Column: t.text + ".*", Alias: leadAlias}, nil
	}
	if t.kind == tIdent && isAggName(t.text) && p.peekN(1).kind == tLParen {
		fn := aggOf(t.text)
		p.next()
		p.next()
		distinct := false
		if p.isKeyword("DISTINCT") {
			p.next()
			distinct = true
		}
		arg := ""
		var argExpr *tds.ValueExpr
		if p.peek().kind == tStar {
			arg = "*"
			p.next()
		} else {
			ve, err := p.valueExpr()
			if err != nil {
				return tds.SelectItem{}, err
			}
			if ve.Kind == tds.ValCol {
				arg = ve.Col // simple column — keep the fast column-fold path
			} else {
				argExpr = ve // expression argument (e.g. CASE) — evaluated per row before the fold
			}
		}
		sep := ""
		if fn == tds.AggStringAgg {
			if p.peek().kind != tComma {
				return tds.SelectItem{}, fmt.Errorf("tsql: STRING_AGG requires a separator, got %q", p.peek().text)
			}
			p.next()
			sve, err := p.valueExpr()
			if err != nil {
				return tds.SelectItem{}, err
			}
			s, ok := sve.Lit.(string)
			if sve.Kind != tds.ValLit || !ok {
				return tds.SelectItem{}, fmt.Errorf("tsql: STRING_AGG separator must be a string literal")
			}
			sep = s
		}
		if p.peek().kind != tRParen {
			return tds.SelectItem{}, fmt.Errorf("tsql: expected ')' after aggregate, got %q", p.peek().text)
		}
		p.next()
		if p.peek().kind == tIdent && strings.EqualFold(p.peek().text, "OVER") {
			var wargs []*tds.ValueExpr
			if argExpr != nil {
				wargs = []*tds.ValueExpr{argExpr}
			} else if arg != "" && arg != "*" {
				wargs = []*tds.ValueExpr{{Kind: tds.ValCol, Col: arg}}
			}
			win, err := p.overClause(&tds.ValueExpr{Kind: tds.ValFunc, Func: t.text, Args: wargs})
			if err != nil {
				return tds.SelectItem{}, err
			}
			return tds.SelectItem{Window: win, Alias: aliasOr(leadAlias, p.optAlias())}, nil
		}
		return tds.SelectItem{Agg: fn, Arg: arg, ArgExpr: argExpr, Sep: sep, AggDist: distinct, Alias: aliasOr(leadAlias, p.optAlias())}, nil
	}
	ve, err := p.valueExpr()
	if err != nil {
		return tds.SelectItem{}, err
	}
	if ve.Kind == tds.ValFunc && p.peekIs("WITHIN") {
		win, err := p.withinGroupWindow(ve)
		if err != nil {
			return tds.SelectItem{}, err
		}
		return tds.SelectItem{Window: win, Alias: aliasOr(leadAlias, p.optAlias())}, nil
	}
	if ve.Kind == tds.ValFunc && p.peek().kind == tIdent && strings.EqualFold(p.peek().text, "OVER") {
		win, err := p.overClause(ve)
		if err != nil {
			return tds.SelectItem{}, err
		}
		return tds.SelectItem{Window: win, Alias: aliasOr(leadAlias, p.optAlias())}, nil
	}
	alias := aliasOr(leadAlias, p.optAlias())
	if ve.Kind == tds.ValCol {
		return tds.SelectItem{Column: ve.Col, Alias: alias}, nil
	}
	return tds.SelectItem{Expr: ve, Alias: alias}, nil
}

// overClause parses OVER (PARTITION BY … ORDER BY …) for a window function call.
func (p *parser) overClause(fn *tds.ValueExpr) (*tds.WindowSpec, error) {
	p.next() // OVER
	if p.peek().kind != tLParen {
		return nil, fmt.Errorf("tsql: expected '(' after OVER, got %q", p.peek().text)
	}
	p.next()
	w := &tds.WindowSpec{Func: strings.ToUpper(fn.Func), Args: fn.Args}
	if p.peek().kind == tIdent && strings.EqualFold(p.peek().text, "PARTITION") {
		p.next()
		if err := p.expectKeyword("BY"); err != nil {
			return nil, err
		}
		cols, err := p.identList()
		if err != nil {
			return nil, err
		}
		w.PartitionBy = cols
	}
	if p.isKeyword("ORDER") {
		p.next()
		if err := p.expectKeyword("BY"); err != nil {
			return nil, err
		}
		items, err := p.orderList()
		if err != nil {
			return nil, err
		}
		w.OrderBy = items
	}
	if p.peek().kind != tRParen {
		return nil, fmt.Errorf("tsql: expected ')' after OVER clause, got %q", p.peek().text)
	}
	p.next()
	return w, nil
}

// withinGroupWindow parses an ordered-set aggregate: FUNC(p) WITHIN GROUP (ORDER BY …) OVER (PARTITION BY …).
func (p *parser) withinGroupWindow(fn *tds.ValueExpr) (*tds.WindowSpec, error) {
	p.next() // WITHIN
	if !p.peekIs("GROUP") {
		return nil, fmt.Errorf("tsql: expected GROUP after WITHIN, got %q", p.peek().text)
	}
	p.next()
	if p.peek().kind != tLParen {
		return nil, fmt.Errorf("tsql: expected '(' after WITHIN GROUP, got %q", p.peek().text)
	}
	p.next()
	if err := p.expectKeyword("ORDER"); err != nil {
		return nil, err
	}
	if err := p.expectKeyword("BY"); err != nil {
		return nil, err
	}
	order, err := p.orderList()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tRParen {
		return nil, fmt.Errorf("tsql: expected ')' after WITHIN GROUP, got %q", p.peek().text)
	}
	p.next()
	if !p.peekIs("OVER") {
		return nil, fmt.Errorf("tsql: expected OVER after WITHIN GROUP, got %q", p.peek().text)
	}
	w, err := p.overClause(fn)
	if err != nil {
		return nil, err
	}
	w.WithinGroup = order
	return w, nil
}

func aliasOr(lead, trail string) string {
	if lead != "" {
		return lead
	}
	return trail
}

func (p *parser) valueExpr() (*tds.ValueExpr, error) {
	left, err := p.term()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tOp && (p.peek().text == "+" || p.peek().text == "-" || p.peek().text == "&" || p.peek().text == "|" || p.peek().text == "^") {
		op := p.peek().text
		p.next()
		right, err := p.term()
		if err != nil {
			return nil, err
		}
		left = &tds.ValueExpr{Kind: tds.ValBinary, Op: op, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) term() (*tds.ValueExpr, error) {
	left, err := p.factor()
	if err != nil {
		return nil, err
	}
	for {
		var op string
		switch {
		case p.peek().kind == tStar:
			op = "*"
		case p.peek().kind == tOp && (p.peek().text == "/" || p.peek().text == "%"):
			op = p.peek().text
		default:
			return left, nil
		}
		p.next()
		right, err := p.factor()
		if err != nil {
			return nil, err
		}
		left = &tds.ValueExpr{Kind: tds.ValBinary, Op: op, Left: left, Right: right}
	}
}

func (p *parser) factor() (*tds.ValueExpr, error) {
	if p.peek().kind == tOp && p.peek().text == "-" {
		p.next()
		f, err := p.factor()
		if err != nil {
			return nil, err
		}
		return &tds.ValueExpr{Kind: tds.ValBinary, Op: "-", Left: &tds.ValueExpr{Kind: tds.ValLit, Lit: int64(0)}, Right: f}, nil
	}
	return p.primaryValue()
}

func (p *parser) primaryValue() (*tds.ValueExpr, error) {
	t := p.peek()
	switch {
	case p.isKeyword("CASE"):
		return p.caseExpr()
	case p.isKeyword("NULL"):
		p.next()
		return &tds.ValueExpr{Kind: tds.ValLit, Lit: nil}, nil
	case t.kind == tKeyword && p.peekN(1).kind == tLParen:
		// a reserved keyword that doubles as a function (LEFT/RIGHT/...), disambiguated by the '('
		return p.funcCall(strings.ToUpper(t.text))
	case t.kind == tLParen:
		p.next()
		if p.isKeyword("SELECT") {
			sub, err := p.selectStmt()
			if err != nil {
				return nil, err
			}
			if p.peek().kind != tRParen {
				return nil, fmt.Errorf("tsql: expected ')' after subquery, got %q", p.peek().text)
			}
			p.next()
			return &tds.ValueExpr{Kind: tds.ValSubquery, Sub: sub}, nil
		}
		e, err := p.valueExpr()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tRParen {
			return nil, fmt.Errorf("tsql: expected ')', got %q", p.peek().text)
		}
		p.next()
		return e, nil
	case t.kind == tString, t.kind == tNumber:
		v, err := p.literal()
		if err != nil {
			return nil, err
		}
		return &tds.ValueExpr{Kind: tds.ValLit, Lit: v}, nil
	case p.identLike():
		if strings.HasPrefix(t.text, "@@") {
			p.next()
			return &tds.ValueExpr{Kind: tds.ValFunc, Func: strings.ToUpper(t.text)}, nil
		}
		if strings.EqualFold(t.text, "CAST") && p.peekN(1).kind == tLParen {
			return p.castExpr()
		}
		if strings.EqualFold(t.text, "CONVERT") && p.peekN(1).kind == tLParen {
			return p.convertExpr()
		}
		if (strings.EqualFold(t.text, "TRY_CAST") || strings.EqualFold(t.text, "TRY_CONVERT")) && p.peekN(1).kind == tLParen {
			if strings.EqualFold(t.text, "TRY_CAST") {
				return p.castExpr() // CAST already yields NULL on failure
			}
			return p.convertExpr()
		}
		if (strings.EqualFold(t.text, "PARSE") || strings.EqualFold(t.text, "TRY_PARSE")) && p.peekN(1).kind == tLParen {
			return p.castExpr() // PARSE(expr AS type) shares CAST's shape
		}
		if strings.EqualFold(t.text, "IIF") && p.peekN(1).kind == tLParen {
			return p.iifExpr()
		}
		switch strings.ToUpper(t.text) {
		case "DATEADD", "DATEDIFF", "DATEDIFF_BIG", "DATEPART", "DATENAME", "DATETRUNC":
			if p.peekN(1).kind == tLParen {
				return p.datePartCall(strings.ToUpper(t.text))
			}
		}
		if p.peekN(1).kind == tLParen {
			return p.funcCall(strings.ToUpper(t.text))
		}
		name, _ := p.qualifiedName()
		return &tds.ValueExpr{Kind: tds.ValCol, Col: name}, nil
	}
	return nil, fmt.Errorf("tsql: unexpected %q in expression", t.text)
}

// iifExpr desugars IIF(cond, yes, no) to a searched CASE so the boolean condition gets full parsing.
func (p *parser) iifExpr() (*tds.ValueExpr, error) {
	p.next() // IIF
	p.next() // (
	cond, err := p.orExpr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tComma {
		return nil, fmt.Errorf("tsql: expected ',' in IIF, got %q", p.peek().text)
	}
	p.next()
	yes, err := p.valueExpr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tComma {
		return nil, fmt.Errorf("tsql: expected ',' in IIF, got %q", p.peek().text)
	}
	p.next()
	no, err := p.valueExpr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tRParen {
		return nil, fmt.Errorf("tsql: expected ')' after IIF, got %q", p.peek().text)
	}
	p.next()
	return &tds.ValueExpr{Kind: tds.ValCase, Whens: []tds.CaseWhen{{Cond: cond, Result: yes}}, Else: no}, nil
}

// datePartCall keeps the datepart keyword (year/day/...) as a literal arg, not a column ref.
func (p *parser) datePartCall(name string) (*tds.ValueExpr, error) {
	p.next() // name
	p.next() // (
	part := p.peek()
	if part.kind != tIdent && part.kind != tKeyword {
		return nil, fmt.Errorf("tsql: expected datepart in %s, got %q", name, part.text)
	}
	p.next()
	args := []*tds.ValueExpr{{Kind: tds.ValLit, Lit: strings.ToLower(part.text)}}
	for p.peek().kind == tComma {
		p.next()
		a, err := p.valueExpr()
		if err != nil {
			return nil, err
		}
		args = append(args, a)
	}
	if p.peek().kind != tRParen {
		return nil, fmt.Errorf("tsql: expected ')' after %s args, got %q", name, p.peek().text)
	}
	p.next()
	return &tds.ValueExpr{Kind: tds.ValFunc, Func: name, Args: args}, nil
}

// funcCall parses NAME(args), including reserved-keyword names like LEFT/RIGHT.
func (p *parser) funcCall(name string) (*tds.ValueExpr, error) {
	p.next() // name
	p.next() // (
	var args []*tds.ValueExpr
	jsonObject := name == "JSON_OBJECT"
	if p.peek().kind != tRParen {
		for {
			if p.peek().kind == tStar {
				args = append(args, &tds.ValueExpr{Kind: tds.ValCol, Col: "*"})
				p.next()
			} else {
				a, err := p.valueExpr()
				if err != nil {
					return nil, err
				}
				args = append(args, a)
				if jsonObject {
					if p.peek().kind != tColon {
						return nil, fmt.Errorf("tsql: expected ':' in JSON_OBJECT, got %q", p.peek().text)
					}
					p.next()
					v, err := p.valueExpr()
					if err != nil {
						return nil, err
					}
					args = append(args, v)
				}
			}
			if p.peek().kind == tComma {
				p.next()
				continue
			}
			break
		}
	}
	if p.peek().kind != tRParen {
		return nil, fmt.Errorf("tsql: expected ')' after function args, got %q", p.peek().text)
	}
	p.next()
	return &tds.ValueExpr{Kind: tds.ValFunc, Func: name, Args: args}, nil
}

func (p *parser) caseExpr() (*tds.ValueExpr, error) {
	p.next() // CASE
	ce := &tds.ValueExpr{Kind: tds.ValCase}
	if !p.isKeyword("WHEN") {
		op, err := p.valueExpr()
		if err != nil {
			return nil, err
		}
		ce.Operand = op
	}
	for p.isKeyword("WHEN") {
		p.next()
		var w tds.CaseWhen
		if ce.Operand != nil {
			mv, err := p.valueExpr()
			if err != nil {
				return nil, err
			}
			w.Match = mv
		} else {
			cond, err := p.orExpr()
			if err != nil {
				return nil, err
			}
			w.Cond = cond
		}
		if err := p.expectKeyword("THEN"); err != nil {
			return nil, err
		}
		res, err := p.valueExpr()
		if err != nil {
			return nil, err
		}
		w.Result = res
		ce.Whens = append(ce.Whens, w)
	}
	if p.isKeyword("ELSE") {
		p.next()
		el, err := p.valueExpr()
		if err != nil {
			return nil, err
		}
		ce.Else = el
	}
	if err := p.expectKeyword("END"); err != nil {
		return nil, err
	}
	return ce, nil
}

func (p *parser) castExpr() (*tds.ValueExpr, error) {
	p.next() // CAST
	p.next() // (
	inner, err := p.valueExpr()
	if err != nil {
		return nil, err
	}
	if err := p.expectKeyword("AS"); err != nil {
		return nil, err
	}
	typ, err := p.typeName()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tRParen {
		return nil, fmt.Errorf("tsql: expected ')' after CAST, got %q", p.peek().text)
	}
	p.next()
	return &tds.ValueExpr{Kind: tds.ValCast, Left: inner, Cast: typ}, nil
}

func (p *parser) convertExpr() (*tds.ValueExpr, error) {
	p.next() // CONVERT
	p.next() // (
	typ, err := p.typeName()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tComma {
		return nil, fmt.Errorf("tsql: expected ',' in CONVERT, got %q", p.peek().text)
	}
	p.next()
	inner, err := p.valueExpr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind == tComma {
		p.next()
		if _, err := p.valueExpr(); err != nil {
			return nil, err
		}
	}
	if p.peek().kind != tRParen {
		return nil, fmt.Errorf("tsql: expected ')' after CONVERT, got %q", p.peek().text)
	}
	p.next()
	return &tds.ValueExpr{Kind: tds.ValCast, Left: inner, Cast: typ}, nil
}

func (p *parser) typeName() (string, error) {
	t := p.peek()
	if t.kind != tIdent && t.kind != tKeyword {
		return "", fmt.Errorf("tsql: expected type name, got %q", t.text)
	}
	name := strings.ToUpper(t.text)
	p.next()
	if p.peek().kind == tLParen {
		p.next()
		for p.peek().kind != tRParen && p.peek().kind != tEOF {
			p.next()
		}
		if p.peek().kind == tRParen {
			p.next()
		}
	}
	return name, nil
}

func (p *parser) optAlias() string {
	if p.isKeyword("AS") {
		p.next()
		if k := p.peek().kind; k == tIdent || k == tString {
			a := p.peek().text
			p.next()
			return a
		}
		return ""
	}
	if p.peek().kind == tIdent { // bare alias: `expr alias` with no AS
		a := p.peek().text
		p.next()
		return a
	}
	return ""
}

func isAggName(s string) bool {
	switch strings.ToUpper(s) {
	case "COUNT", "SUM", "AVG", "MIN", "MAX", "COUNT_BIG", "STDEV", "STDEVP", "VAR", "VARP", "STRING_AGG", "CHECKSUM_AGG", "APPROX_COUNT_DISTINCT":
		return true
	}
	return false
}

func aggOf(s string) tds.AggFunc {
	switch strings.ToUpper(s) {
	case "COUNT":
		return tds.AggCount
	case "SUM":
		return tds.AggSum
	case "AVG":
		return tds.AggAvg
	case "MIN":
		return tds.AggMin
	case "MAX":
		return tds.AggMax
	case "COUNT_BIG":
		return tds.AggCountBig
	case "STDEV":
		return tds.AggStdev
	case "STDEVP":
		return tds.AggStdevp
	case "VAR":
		return tds.AggVar
	case "VARP":
		return tds.AggVarp
	case "STRING_AGG":
		return tds.AggStringAgg
	case "CHECKSUM_AGG":
		return tds.AggChecksumAgg
	case "APPROX_COUNT_DISTINCT":
		return tds.AggApproxCountDistinct
	}
	return tds.AggNone
}

func (p *parser) peekIs(s string) bool {
	t := p.peek()
	return (t.kind == tIdent || t.kind == tKeyword) && strings.EqualFold(t.text, s)
}

// skipParenGroup consumes a balanced ( … ) group at the current '(' (used to discard query/table hints).
func (p *parser) skipParenGroup() error {
	if p.peek().kind != tLParen {
		return fmt.Errorf("tsql: expected '(', got %q", p.peek().text)
	}
	depth := 0
	for {
		switch p.peek().kind {
		case tEOF:
			return fmt.Errorf("tsql: unterminated '('")
		case tLParen:
			depth++
		case tRParen:
			depth--
			if depth == 0 {
				p.next()
				return nil
			}
		}
		p.next()
	}
}

// tableHints parses and discards a table source's TABLESAMPLE and WITH (…) hints (honored as no-ops).
func (p *parser) tableHints() error {
	for {
		switch {
		case p.peekIs("TABLESAMPLE"):
			p.next()
			if p.peekIs("SYSTEM") {
				p.next()
			}
			if err := p.skipParenGroup(); err != nil {
				return err
			}
			if p.peekIs("REPEATABLE") {
				p.next()
				if err := p.skipParenGroup(); err != nil {
					return err
				}
			}
		case p.isKeyword("WITH") && p.peekN(1).kind == tLParen:
			p.next()
			if err := p.skipParenGroup(); err != nil {
				return err
			}
		default:
			return nil
		}
	}
}

// optionClause parses and discards a trailing OPTION (…) query hint.
func (p *parser) optionClause() error {
	if p.peekIs("OPTION") && p.peekN(1).kind == tLParen {
		p.next()
		return p.skipParenGroup()
	}
	return nil
}

func (p *parser) pivotClause(q *tds.Query) error {
	switch {
	case p.peekIs("PIVOT"):
		p.next()
		spec, err := p.parsePivot()
		if err != nil {
			return err
		}
		q.Pivot = spec
	case p.peekIs("UNPIVOT"):
		p.next()
		spec, err := p.parseUnpivot()
		if err != nil {
			return err
		}
		q.Unpivot = spec
	}
	return nil
}

// parsePivot parses ( Agg(valueCol) FOR pivotCol IN (v1, v2, …) ) AS alias.
func (p *parser) parsePivot() (*tds.PivotSpec, error) {
	if p.peek().kind != tLParen {
		return nil, fmt.Errorf("tsql: expected '(' after PIVOT, got %q", p.peek().text)
	}
	p.next()
	if !p.identLike() {
		return nil, fmt.Errorf("tsql: expected aggregate in PIVOT, got %q", p.peek().text)
	}
	agg := strings.ToUpper(p.peek().text)
	p.next()
	if p.peek().kind != tLParen {
		return nil, fmt.Errorf("tsql: expected '(' after %s, got %q", agg, p.peek().text)
	}
	p.next()
	valCol := "*"
	if p.peek().kind == tStar {
		p.next()
	} else {
		name, ok := p.qualifiedName()
		if !ok {
			return nil, fmt.Errorf("tsql: expected column in PIVOT aggregate, got %q", p.peek().text)
		}
		valCol = name
	}
	if p.peek().kind != tRParen {
		return nil, fmt.Errorf("tsql: expected ')' after PIVOT aggregate, got %q", p.peek().text)
	}
	p.next()
	if !p.peekIs("FOR") {
		return nil, fmt.Errorf("tsql: expected FOR in PIVOT, got %q", p.peek().text)
	}
	p.next()
	pivotCol, ok := p.qualifiedName()
	if !ok {
		return nil, fmt.Errorf("tsql: expected pivot column, got %q", p.peek().text)
	}
	values, err := p.inValueList()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tRParen {
		return nil, fmt.Errorf("tsql: expected ')' after PIVOT, got %q", p.peek().text)
	}
	p.next()
	return &tds.PivotSpec{Agg: agg, ValueCol: valCol, PivotCol: pivotCol, Values: values, Alias: p.optTableAlias()}, nil
}

// parseUnpivot parses ( valueCol FOR nameCol IN (c1, c2, …) ) AS alias.
func (p *parser) parseUnpivot() (*tds.UnpivotSpec, error) {
	if p.peek().kind != tLParen {
		return nil, fmt.Errorf("tsql: expected '(' after UNPIVOT, got %q", p.peek().text)
	}
	p.next()
	valCol, ok := p.qualifiedName()
	if !ok {
		return nil, fmt.Errorf("tsql: expected value column in UNPIVOT, got %q", p.peek().text)
	}
	if !p.peekIs("FOR") {
		return nil, fmt.Errorf("tsql: expected FOR in UNPIVOT, got %q", p.peek().text)
	}
	p.next()
	nameCol, ok := p.qualifiedName()
	if !ok {
		return nil, fmt.Errorf("tsql: expected name column in UNPIVOT, got %q", p.peek().text)
	}
	cols, err := p.inValueList()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tRParen {
		return nil, fmt.Errorf("tsql: expected ')' after UNPIVOT, got %q", p.peek().text)
	}
	p.next()
	return &tds.UnpivotSpec{ValueCol: valCol, NameCol: nameCol, Columns: cols, Alias: p.optTableAlias()}, nil
}

// inValueList parses `IN ( a, b, … )`, the column/value list shared by PIVOT and UNPIVOT.
func (p *parser) inValueList() ([]string, error) {
	if err := p.expectKeyword("IN"); err != nil {
		return nil, err
	}
	if p.peek().kind != tLParen {
		return nil, fmt.Errorf("tsql: expected '(' after IN, got %q", p.peek().text)
	}
	p.next()
	list, err := p.identList()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tRParen {
		return nil, fmt.Errorf("tsql: expected ')' after IN list, got %q", p.peek().text)
	}
	p.next()
	return list, nil
}

// groupByClause parses a plain column list or ROLLUP/CUBE/GROUPING SETS into q.GroupingSets + q.GroupBy.
func (p *parser) groupByClause(q *tds.Query) error {
	switch {
	case p.peekIs("ROLLUP") && p.peekN(1).kind == tLParen:
		cols, err := p.parenIdentList()
		if err != nil {
			return err
		}
		q.GroupBy, q.GroupingSets = cols, rollupSets(cols)
	case p.peekIs("CUBE") && p.peekN(1).kind == tLParen:
		cols, err := p.parenIdentList()
		if err != nil {
			return err
		}
		q.GroupBy, q.GroupingSets = cols, cubeSets(cols)
	case p.peekIs("GROUPING") && p.peekN(1).kind == tIdent && strings.EqualFold(p.peekN(1).text, "SETS"):
		sets, universe, err := p.groupingSetsClause()
		if err != nil {
			return err
		}
		q.GroupBy, q.GroupingSets = universe, sets
	default:
		cols, err := p.identList()
		if err != nil {
			return err
		}
		q.GroupBy = cols
	}
	return nil
}

// parenIdentList consumes a ROLLUP/CUBE name then a parenthesized column list.
func (p *parser) parenIdentList() ([]string, error) {
	p.next() // ROLLUP / CUBE
	p.next() // (
	cols, err := p.identList()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tRParen {
		return nil, fmt.Errorf("tsql: expected ')' after grouping columns, got %q", p.peek().text)
	}
	p.next()
	return cols, nil
}

// groupingSetsClause parses GROUPING SETS ( set, … ) where each set is ( cols ), (), or a bare column.
func (p *parser) groupingSetsClause() ([][]string, []string, error) {
	p.next() // GROUPING
	p.next() // SETS
	if p.peek().kind != tLParen {
		return nil, nil, fmt.Errorf("tsql: expected '(' after GROUPING SETS, got %q", p.peek().text)
	}
	p.next()
	var sets [][]string
	var universe []string
	seen := map[string]bool{}
	for {
		var set []string
		if p.peek().kind == tLParen {
			p.next()
			if p.peek().kind != tRParen {
				cols, err := p.identList()
				if err != nil {
					return nil, nil, err
				}
				set = cols
			}
			if p.peek().kind != tRParen {
				return nil, nil, fmt.Errorf("tsql: expected ')' in grouping set, got %q", p.peek().text)
			}
			p.next()
		} else {
			name, ok := p.qualifiedName()
			if !ok {
				return nil, nil, fmt.Errorf("tsql: expected column in GROUPING SETS, got %q", p.peek().text)
			}
			set = []string{name}
		}
		sets = append(sets, set)
		for _, c := range set {
			if !seen[c] {
				seen[c] = true
				universe = append(universe, c)
			}
		}
		if p.peek().kind == tComma {
			p.next()
			continue
		}
		break
	}
	if p.peek().kind != tRParen {
		return nil, nil, fmt.Errorf("tsql: expected ')' after GROUPING SETS, got %q", p.peek().text)
	}
	p.next()
	return sets, universe, nil
}

// rollupSets expands ROLLUP(a,b,c) to the grouping sets (a,b,c),(a,b),(a),().
func rollupSets(cols []string) [][]string {
	sets := make([][]string, 0, len(cols)+1)
	for n := len(cols); n >= 0; n-- {
		sets = append(sets, append([]string{}, cols[:n]...))
	}
	return sets
}

// cubeSets expands CUBE(a,b) to every subset: (a,b),(a),(b),().
func cubeSets(cols []string) [][]string {
	n := len(cols)
	sets := make([][]string, 0, 1<<n)
	for mask := (1 << n) - 1; mask >= 0; mask-- {
		var set []string
		for i := 0; i < n; i++ {
			if mask&(1<<(n-1-i)) != 0 {
				set = append(set, cols[i])
			}
		}
		sets = append(sets, set)
	}
	return sets
}

func (p *parser) identList() ([]string, error) {
	var out []string
	for {
		name, ok := p.qualifiedName()
		if !ok {
			return nil, fmt.Errorf("tsql: expected column name, got %q", p.peek().text)
		}
		out = append(out, name)
		if p.peek().kind == tComma {
			p.next()
			continue
		}
		break
	}
	return out, nil
}

func (p *parser) tableName() (db, schema, table string, err error) {
	if !p.identLike() {
		return "", "", "", fmt.Errorf("tsql: expected table name, got %q", p.peek().text)
	}
	parts := []string{p.peek().text}
	p.next()
	for p.peek().kind == tDot {
		p.next()
		if !p.identLike() {
			return "", "", "", fmt.Errorf("tsql: expected name after '.', got %q", p.peek().text)
		}
		parts = append(parts, p.peek().text)
		p.next()
	}
	table = parts[len(parts)-1]
	if len(parts) >= 2 {
		schema = parts[len(parts)-2]
	}
	if len(parts) >= 3 {
		db = parts[len(parts)-3]
	}
	return db, schema, table, nil
}

func (p *parser) orExpr() (*tds.Expr, error) {
	left, err := p.andExpr()
	if err != nil {
		return nil, err
	}
	if !p.isKeyword("OR") {
		return left, nil
	}
	terms := []*tds.Expr{left}
	for p.isKeyword("OR") {
		p.next()
		t, err := p.andExpr()
		if err != nil {
			return nil, err
		}
		terms = append(terms, t)
	}
	return &tds.Expr{Or: terms}, nil
}

func (p *parser) andExpr() (*tds.Expr, error) {
	left, err := p.notExpr()
	if err != nil {
		return nil, err
	}
	if !p.isKeyword("AND") {
		return left, nil
	}
	terms := []*tds.Expr{left}
	for p.isKeyword("AND") {
		p.next()
		t, err := p.notExpr()
		if err != nil {
			return nil, err
		}
		terms = append(terms, t)
	}
	return &tds.Expr{And: terms}, nil
}

func (p *parser) notExpr() (*tds.Expr, error) {
	if p.isKeyword("NOT") {
		p.next()
		e, err := p.notExpr()
		if err != nil {
			return nil, err
		}
		return &tds.Expr{Not: e}, nil
	}
	return p.primaryExpr()
}

func (p *parser) primaryExpr() (*tds.Expr, error) {
	if p.isKeyword("EXISTS") {
		p.next()
		if p.peek().kind != tLParen {
			return nil, fmt.Errorf("tsql: expected '(' after EXISTS, got %q", p.peek().text)
		}
		p.next()
		sub, err := p.selectStmt()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tRParen {
			return nil, fmt.Errorf("tsql: expected ')' after EXISTS subquery, got %q", p.peek().text)
		}
		p.next()
		return &tds.Expr{Pred: &tds.Predicate{Op: tds.OpExists, Sub: sub}}, nil
	}
	if p.peek().kind == tLParen {
		p.next()
		e, err := p.orExpr()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tRParen {
			return nil, fmt.Errorf("tsql: expected ')', got %q", p.peek().text)
		}
		p.next()
		return e, nil
	}
	return p.predicate()
}

func (p *parser) predicate() (*tds.Expr, error) {
	left, err := p.valueExpr()
	if err != nil {
		return nil, err
	}
	col := ""
	var leftExpr *tds.ValueExpr
	if left.Kind == tds.ValCol {
		col = left.Col
	} else {
		leftExpr = left
	}
	mk := func(op tds.Op, val any) *tds.Expr {
		return &tds.Expr{Pred: &tds.Predicate{Column: col, LeftExpr: leftExpr, Op: op, Value: val}}
	}

	// `col NOT IN/LIKE/BETWEEN …` — consume the leading NOT and negate the membership predicate.
	neg := false
	if p.isKeyword("NOT") {
		p.next()
		neg = true
	}
	wrap := func(e *tds.Expr) *tds.Expr {
		if neg {
			return &tds.Expr{Not: e}
		}
		return e
	}

	switch {
	case p.isKeyword("IS"):
		p.next()
		op := tds.OpIsNull
		if p.isKeyword("NOT") {
			p.next()
			op = tds.OpIsNotNull
		}
		if err := p.expectKeyword("NULL"); err != nil {
			return nil, err
		}
		return mk(op, nil), nil

	case p.isKeyword("IN"):
		p.next()
		if p.peek().kind != tLParen {
			return nil, fmt.Errorf("tsql: expected '(' after IN, got %q", p.peek().text)
		}
		p.next()
		if p.isKeyword("SELECT") {
			sub, err := p.selectStmt()
			if err != nil {
				return nil, err
			}
			if p.peek().kind != tRParen {
				return nil, fmt.Errorf("tsql: expected ')' after subquery, got %q", p.peek().text)
			}
			p.next()
			return wrap(&tds.Expr{Pred: &tds.Predicate{Column: col, LeftExpr: leftExpr, Op: tds.OpIn, Sub: sub}}), nil
		}
		vals, err := p.literalList()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tRParen {
			return nil, fmt.Errorf("tsql: expected ')' after IN list, got %q", p.peek().text)
		}
		p.next()
		return wrap(mk(tds.OpIn, vals)), nil

	case p.isKeyword("LIKE"):
		p.next()
		t := p.peek()
		if t.kind != tString {
			return nil, fmt.Errorf("tsql: expected string after LIKE, got %q", t.text)
		}
		p.next()
		return wrap(mk(tds.OpLike, t.text)), nil

	case p.isKeyword("BETWEEN"):
		p.next()
		lo, err := p.literal()
		if err != nil {
			return nil, err
		}
		if err := p.expectKeyword("AND"); err != nil {
			return nil, err
		}
		hi, err := p.literal()
		if err != nil {
			return nil, err
		}
		return wrap(&tds.Expr{And: []*tds.Expr{mk(tds.OpGe, lo), mk(tds.OpLe, hi)}}), nil
	}

	if neg {
		return nil, fmt.Errorf("tsql: expected IN, LIKE, or BETWEEN after NOT, got %q", p.peek().text)
	}
	opTok := p.peek()
	if opTok.kind != tOp {
		return nil, fmt.Errorf("tsql: expected operator, got %q", opTok.text)
	}
	p.next()
	op, err := mapOp(opTok.text)
	if err != nil {
		return nil, err
	}
	rhs, err := p.valueExpr()
	if err != nil {
		return nil, err
	}
	return mk(op, rhs), nil
}

func (p *parser) literalList() ([]any, error) {
	var out []any
	for {
		v, err := p.literal()
		if err != nil {
			return nil, err
		}
		out = append(out, v)
		if p.peek().kind == tComma {
			p.next()
			continue
		}
		break
	}
	return out, nil
}

func (p *parser) literal() (any, error) {
	if p.isKeyword("NULL") {
		p.next()
		return nil, nil
	}
	t := p.peek()
	switch t.kind {
	case tString:
		p.next()
		return t.text, nil
	case tNumber:
		p.next()
		if strings.Contains(t.text, ".") {
			f, err := strconv.ParseFloat(t.text, 64)
			if err != nil {
				return nil, err
			}
			return f, nil
		}
		if len(t.text) > 2 && (t.text[1] == 'x' || t.text[1] == 'X') {
			n, err := strconv.ParseInt(t.text[2:], 16, 64)
			if err != nil {
				return nil, err
			}
			return n, nil
		}
		n, err := strconv.ParseInt(t.text, 10, 64)
		if err != nil {
			return nil, err
		}
		return n, nil
	default:
		return nil, fmt.Errorf("tsql: expected literal, got %q", t.text)
	}
}

func (p *parser) intLit() (int, error) {
	t := p.peek()
	if t.kind != tNumber {
		return 0, fmt.Errorf("tsql: expected number, got %q", t.text)
	}
	n, err := strconv.Atoi(t.text)
	if err != nil {
		return 0, fmt.Errorf("tsql: bad number %q", t.text)
	}
	p.next()
	return n, nil
}

func (p *parser) orderList() ([]tds.OrderItem, error) {
	var out []tds.OrderItem
	for {
		var item tds.OrderItem
		if p.peek().kind == tNumber {
			n, err := strconv.Atoi(p.peek().text)
			if err != nil {
				return nil, fmt.Errorf("tsql: bad ORDER BY ordinal %q", p.peek().text)
			}
			item.Ordinal = n
			p.next()
		} else {
			ve, err := p.valueExpr()
			if err != nil {
				return nil, err
			}
			if ve.Kind == tds.ValCol {
				item.Column = ve.Col
			} else {
				item.Expr = ve
			}
		}
		if p.isKeyword("ASC") {
			p.next()
		} else if p.isKeyword("DESC") {
			item.Desc = true
			p.next()
		}
		out = append(out, item)
		if p.peek().kind == tComma {
			p.next()
			continue
		}
		break
	}
	return out, nil
}

func mapOp(s string) (tds.Op, error) {
	switch s {
	case "=":
		return tds.OpEq, nil
	case "<>":
		return tds.OpNe, nil
	case "<":
		return tds.OpLt, nil
	case "<=":
		return tds.OpLe, nil
	case ">":
		return tds.OpGt, nil
	case ">=":
		return tds.OpGe, nil
	}
	return 0, fmt.Errorf("tsql: unknown operator %q", s)
}

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tsql

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/tds"
)

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
		if err := p.expectKeyword("JOIN"); err != nil {
			return nil, err
		}
		jt = tds.JoinCross
	default:
		return nil, nil
	}
	db, sch, tbl, err := p.tableName()
	if err != nil {
		return nil, err
	}
	j := &tds.Join{Type: jt, Database: db, Schema: sch, Table: tbl, Alias: p.optTableAlias()}
	if jt != tds.JoinCross {
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
		cols, err := p.identList()
		if err != nil {
			return nil, err
		}
		q.GroupBy = cols
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
	if t.kind == tIdent && isAggName(t.text) && p.peekN(1).kind == tLParen {
		fn := aggOf(t.text)
		p.next()
		p.next()
		arg := ""
		if p.peek().kind == tStar {
			arg = "*"
			p.next()
		} else {
			name, ok := p.qualifiedName()
			if !ok {
				return tds.SelectItem{}, fmt.Errorf("tsql: expected column or * in aggregate, got %q", p.peek().text)
			}
			arg = name
		}
		if p.peek().kind != tRParen {
			return tds.SelectItem{}, fmt.Errorf("tsql: expected ')' after aggregate, got %q", p.peek().text)
		}
		p.next()
		return tds.SelectItem{Agg: fn, Arg: arg, Alias: aliasOr(leadAlias, p.optAlias())}, nil
	}
	ve, err := p.valueExpr()
	if err != nil {
		return tds.SelectItem{}, err
	}
	alias := aliasOr(leadAlias, p.optAlias())
	if ve.Kind == tds.ValCol {
		return tds.SelectItem{Column: ve.Col, Alias: alias}, nil
	}
	return tds.SelectItem{Expr: ve, Alias: alias}, nil
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
	for p.peek().kind == tOp && (p.peek().text == "+" || p.peek().text == "-") {
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
		if strings.EqualFold(t.text, "CAST") && p.peekN(1).kind == tLParen {
			return p.castExpr()
		}
		if strings.EqualFold(t.text, "CONVERT") && p.peekN(1).kind == tLParen {
			return p.convertExpr()
		}
		if p.peekN(1).kind == tLParen {
			fn := strings.ToUpper(t.text)
			p.next()
			p.next()
			var args []*tds.ValueExpr
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
			return &tds.ValueExpr{Kind: tds.ValFunc, Func: fn, Args: args}, nil
		}
		name, _ := p.qualifiedName()
		return &tds.ValueExpr{Kind: tds.ValCol, Col: name}, nil
	}
	return nil, fmt.Errorf("tsql: unexpected %q in expression", t.text)
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
		if p.peek().kind == tIdent {
			a := p.peek().text
			p.next()
			return a
		}
	}
	return ""
}

func isAggName(s string) bool {
	switch strings.ToUpper(s) {
	case "COUNT", "SUM", "AVG", "MIN", "MAX":
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
	}
	return tds.AggNone
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
			return &tds.Expr{Pred: &tds.Predicate{Column: col, LeftExpr: leftExpr, Op: tds.OpIn, Sub: sub}}, nil
		}
		vals, err := p.literalList()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tRParen {
			return nil, fmt.Errorf("tsql: expected ')' after IN list, got %q", p.peek().text)
		}
		p.next()
		return mk(tds.OpIn, vals), nil

	case p.isKeyword("LIKE"):
		p.next()
		t := p.peek()
		if t.kind != tString {
			return nil, fmt.Errorf("tsql: expected string after LIKE, got %q", t.text)
		}
		p.next()
		return mk(tds.OpLike, t.text), nil

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
		return &tds.Expr{And: []*tds.Expr{mk(tds.OpGe, lo), mk(tds.OpLe, hi)}}, nil
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

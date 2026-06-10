// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/internal/exec"
	"github.com/RSKGroup/haystak-tds-spi/internal/infoschema"
	"github.com/RSKGroup/haystak-tds-spi/internal/sysviews"
	"github.com/RSKGroup/haystak-tds-spi/internal/tsql"
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

// Query runs a T-SQL batch (one or more ';'-separated statements) and returns the last
// result set. This is the entry point the wire calls.
func Query(ctx context.Context, b tds.Backend, sql string) (tds.Rows, error) {
	rows, _, err := Exec(ctx, b, sql)
	return rows, err
}

// Exec runs a batch and returns the last result set, or rows-affected (>=0) for a write/DDL.
func Exec(ctx context.Context, b tds.Backend, sql string) (tds.Rows, int64, error) {
	var lastRows tds.Rows
	lastAffected := int64(-1)
	for _, s := range splitBatch(sql) {
		if strings.TrimSpace(s) == "" {
			continue
		}
		rs, aff, err := queryOne(ctx, b, s)
		if err != nil {
			return nil, -1, err
		}
		if rs != nil {
			lastRows, lastAffected = rs, -1
		} else if aff >= 0 {
			lastRows, lastAffected = nil, aff
		}
	}
	return lastRows, lastAffected, nil
}

func execWrite(ctx context.Context, b tds.Backend, sql string) (int64, bool, error) {
	stmt, isWrite, err := tsql.ParseWrite(sql)
	if !isWrite {
		return 0, false, nil
	}
	if err != nil {
		return 0, true, err
	}
	switch {
	case stmt.Insert != nil:
		w, ok := b.(tds.Writer)
		if !ok {
			return 0, true, tds.ErrUnsupported
		}
		res, err := w.Insert(ctx, stmt.Insert)
		return res.RowsAffected, true, err
	case stmt.Update != nil:
		w, ok := b.(tds.Writer)
		if !ok {
			return 0, true, tds.ErrUnsupported
		}
		res, err := w.Update(ctx, stmt.Update)
		return res.RowsAffected, true, err
	case stmt.Delete != nil:
		w, ok := b.(tds.Writer)
		if !ok {
			return 0, true, tds.ErrUnsupported
		}
		res, err := w.Delete(ctx, stmt.Delete)
		return res.RowsAffected, true, err
	case stmt.CreateTable != nil:
		d, ok := b.(tds.DDL)
		if !ok {
			return 0, true, tds.ErrUnsupported
		}
		return 0, true, d.CreateTable(ctx, stmt.CreateTable)
	case stmt.Alter != nil:
		d, ok := b.(tds.DDL)
		if !ok {
			return 0, true, tds.ErrUnsupported
		}
		return 0, true, d.AlterTable(ctx, stmt.Alter)
	case stmt.DropTable != "":
		d, ok := b.(tds.DDL)
		if !ok {
			return 0, true, tds.ErrUnsupported
		}
		return 0, true, d.DropTable(ctx, stmt.DropTable)
	case stmt.CreateDB != "":
		d, ok := b.(tds.DatabaseDDL)
		if !ok {
			return 0, true, tds.ErrUnsupported
		}
		return 0, true, d.CreateDatabase(ctx, stmt.CreateDB)
	case stmt.DropDB != "":
		d, ok := b.(tds.DatabaseDDL)
		if !ok {
			return 0, true, tds.ErrUnsupported
		}
		return 0, true, d.DropDatabase(ctx, stmt.DropDB)
	}
	return 0, false, nil
}

func splitBatch(sql string) []string {
	var stmts []string
	var sb strings.Builder
	inStr := false
	for i := 0; i < len(sql); i++ {
		c := sql[i]
		switch {
		case c == '\'':
			inStr = !inStr
			sb.WriteByte(c)
		case c == ';' && !inStr:
			stmts = append(stmts, sb.String())
			sb.Reset()
		default:
			sb.WriteByte(c)
		}
	}
	if strings.TrimSpace(sb.String()) != "" {
		stmts = append(stmts, sb.String())
	}
	return stmts
}

func queryOne(ctx context.Context, b tds.Backend, sql string) (tds.Rows, int64, error) {
	if rs, handled, err := probe(sql); handled {
		return rs, -1, err
	}
	if affected, isWrite, err := execWrite(ctx, b, sql); isWrite {
		return nil, affected, err
	}
	q, err := tsql.Parse(sql)
	if err != nil {
		return nil, -1, err
	}
	if q.Union != nil {
		rs, err := unionRun(ctx, b, q)
		return rs, -1, err
	}
	rs, err := runParsed(ctx, b, q)
	return rs, -1, err
}

func unionRun(ctx context.Context, b tds.Backend, head *tds.Query) (tds.Rows, error) {
	var arms []*tds.Query
	var ops []tds.SetOp
	for a := head; a != nil; a = a.Union {
		arms = append(arms, a)
		if a.Union != nil {
			ops = append(ops, a.SetOp)
		}
	}
	last := arms[len(arms)-1]
	order, limit, offset := last.OrderBy, last.Limit, last.Offset

	var outCols []catalog.Column
	armRows := make([][][]any, len(arms))
	for i, a := range arms {
		arm := *a
		arm.Union = nil
		arm.OrderBy = nil
		arm.Limit = 0
		arm.Offset = 0
		rs, err := runParsed(ctx, b, &arm)
		if err != nil {
			return nil, err
		}
		cols, data, err := materialize(rs)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			outCols = cols
		}
		armRows[i] = data
	}
	result := armRows[0]
	for i := 1; i < len(arms); i++ {
		switch ops[i-1] {
		case tds.SetIntersect:
			result = intersectRows(result, armRows[i])
		case tds.SetExcept:
			result = exceptRows(result, armRows[i])
		default: // SetUnion, SetUnionAll
			result = append(result, armRows[i]...)
		}
	}
	dedup := false
	for _, op := range ops {
		if op != tds.SetUnionAll {
			dedup = true
		}
	}
	return exec.Apply(outCols, result, &tds.Query{Distinct: dedup, OrderBy: order, Limit: limit, Offset: offset})
}

func intersectRows(a, b [][]any) [][]any {
	in := map[string]bool{}
	for _, r := range b {
		in[rowKey(r)] = true
	}
	seen := map[string]bool{}
	var out [][]any
	for _, r := range a {
		k := rowKey(r)
		if in[k] && !seen[k] {
			seen[k] = true
			out = append(out, r)
		}
	}
	return out
}

func exceptRows(a, b [][]any) [][]any {
	in := map[string]bool{}
	for _, r := range b {
		in[rowKey(r)] = true
	}
	seen := map[string]bool{}
	var out [][]any
	for _, r := range a {
		k := rowKey(r)
		if !in[k] && !seen[k] {
			seen[k] = true
			out = append(out, r)
		}
	}
	return out
}

func rowKey(r []any) string { return fmt.Sprintf("%v", r) }

func runParsed(ctx context.Context, b tds.Backend, q *tds.Query) (tds.Rows, error) {
	if q.CTEs != nil && q.Table != "" {
		if cte, ok := q.CTEs[q.Table]; ok {
			if isRecursiveCTE(cte, q.Table) {
				cols, data, err := runRecursiveCTE(ctx, b, q.Table, cte)
				if err != nil {
					return nil, err
				}
				return exec.Apply(cols, data, q)
			}
			sub := *cte
			sub.CTEs = q.CTEs
			q2 := *q
			q2.FromSub = &sub
			q2.Table = ""
			return runParsed(ctx, b, &q2)
		}
	}
	if strings.EqualFold(q.Schema, "INFORMATION_SCHEMA") {
		schema, err := b.Describe(ctx)
		if err != nil {
			return nil, err
		}
		rows, handled, err := infoschema.Resolve(schema, q)
		if err != nil {
			return nil, err
		}
		if handled {
			return rows, nil
		}
	}

	if strings.EqualFold(q.Schema, "sys") {
		schema, err := b.Describe(ctx)
		if err != nil {
			return nil, err
		}
		rows, handled, err := sysviews.Resolve(schema, q)
		if err != nil {
			return nil, err
		}
		if handled {
			return rows, nil
		}
	}

	if err := resolveSubqueries(ctx, b, q.Where, q.FromAlias, q.Table); err != nil {
		return nil, err
	}
	for _, it := range q.Select {
		if it.Expr != nil {
			if err := resolveValueSubqueries(ctx, b, it.Expr); err != nil {
				return nil, err
			}
		}
	}

	if q.FromSub != nil {
		cols, data, err := runMaterialize(ctx, b, q.FromSub)
		if err != nil {
			return nil, err
		}
		return exec.Apply(cols, data, q)
	}

	if q.Table == "" && len(q.Joins) == 0 {
		return exec.Apply(nil, [][]any{{}}, q)
	}

	caps := b.Capabilities()
	if caps.FullQuery {
		if qe, ok := b.(tds.QueryExecutor); ok {
			return qe.ExecuteQuery(ctx, q)
		}
	}
	if caps.Pushdown {
		if sc, ok := b.(tds.Scanner); ok {
			if len(q.Joins) > 0 {
				return joinQuery(ctx, sc, q)
			}
			raw, err := sc.Scan(ctx, q)
			if err != nil {
				return nil, err
			}
			cols, data, err := materialize(raw)
			if err != nil {
				return nil, err
			}
			return exec.ApplyWith(cols, data, q, makeSubFn(ctx, b, q.FromAlias, q.Table))
		}
	}
	return nil, fmt.Errorf("engine: backend cannot answer query for %q", q.Table)
}

// resolveSubqueries evaluates NON-correlated subqueries once (IN→list, EXISTS→const, scalar→
// literal); correlated ones (referencing the outer alias/table) are left for exec's per-row SubFn.
func resolveSubqueries(ctx context.Context, b tds.Backend, e *tds.Expr, outerAlias, outerTable string) error {
	if e == nil {
		return nil
	}
	if p := e.Pred; p != nil {
		switch {
		case p.Op == tds.OpExists && p.Sub != nil:
			if isCorrelated(p.Sub, outerAlias, outerTable) {
				break
			}
			_, data, err := runMaterialize(ctx, b, p.Sub)
			if err != nil {
				return err
			}
			v := len(data) > 0
			e.Const = &v
			e.Pred = nil
		case p.Sub != nil: // IN (subquery)
			if isCorrelated(p.Sub, outerAlias, outerTable) {
				break
			}
			_, data, err := runMaterialize(ctx, b, p.Sub)
			if err != nil {
				return err
			}
			var vals []any
			for _, row := range data {
				if len(row) > 0 {
					vals = append(vals, row[0])
				}
			}
			p.Value = vals
			p.Sub = nil
		default:
			if err := resolveValueSubqueries(ctx, b, p.LeftExpr); err != nil {
				return err
			}
			if ve, ok := p.Value.(*tds.ValueExpr); ok {
				if err := resolveValueSubqueries(ctx, b, ve); err != nil {
					return err
				}
			}
		}
	}
	for _, c := range e.And {
		if err := resolveSubqueries(ctx, b, c, outerAlias, outerTable); err != nil {
			return err
		}
	}
	for _, c := range e.Or {
		if err := resolveSubqueries(ctx, b, c, outerAlias, outerTable); err != nil {
			return err
		}
	}
	return resolveSubqueries(ctx, b, e.Not, outerAlias, outerTable)
}

func isCorrelated(sub *tds.Query, alias, table string) bool {
	return exprRefsOuter(sub.Where, alias, table)
}

func exprRefsOuter(e *tds.Expr, alias, table string) bool {
	if e == nil {
		return false
	}
	if p := e.Pred; p != nil {
		if colRefsOuter(p.Column, alias, table) {
			return true
		}
		if cr, ok := p.Value.(tds.ColRef); ok && colRefsOuter(cr.Name, alias, table) {
			return true
		}
		if ve, ok := p.Value.(*tds.ValueExpr); ok && valRefsOuter(ve, alias, table) {
			return true
		}
		if valRefsOuter(p.LeftExpr, alias, table) {
			return true
		}
	}
	for _, c := range e.And {
		if exprRefsOuter(c, alias, table) {
			return true
		}
	}
	for _, c := range e.Or {
		if exprRefsOuter(c, alias, table) {
			return true
		}
	}
	return exprRefsOuter(e.Not, alias, table)
}

func valRefsOuter(ve *tds.ValueExpr, alias, table string) bool {
	if ve == nil {
		return false
	}
	if ve.Kind == tds.ValCol && colRefsOuter(ve.Col, alias, table) {
		return true
	}
	if valRefsOuter(ve.Left, alias, table) || valRefsOuter(ve.Right, alias, table) {
		return true
	}
	for _, a := range ve.Args {
		if valRefsOuter(a, alias, table) {
			return true
		}
	}
	return false
}

func colRefsOuter(col, alias, table string) bool {
	dot := strings.LastIndex(col, ".")
	if dot < 0 {
		return false
	}
	q := col[:dot]
	return strings.EqualFold(q, alias) || strings.EqualFold(q, table)
}

func makeSubFn(ctx context.Context, b tds.Backend, alias, table string) exec.SubFn {
	return func(outerRow []any, idx map[string]int, sub *tds.Query) ([][]any, error) {
		bound := bindOuter(sub, outerRow, idx, alias, table)
		_, data, err := runMaterialize(ctx, b, bound)
		return data, err
	}
}

func bindOuter(sub *tds.Query, row []any, idx map[string]int, alias, table string) *tds.Query {
	c := *sub
	c.Where = bindExpr(sub.Where, row, idx, alias, table)
	return &c
}

func bindExpr(e *tds.Expr, row []any, idx map[string]int, alias, table string) *tds.Expr {
	if e == nil {
		return nil
	}
	out := *e
	if e.Pred != nil {
		p := *e.Pred
		if v, ok := outerVal(p.Column, row, idx, alias, table); ok {
			p.LeftExpr = &tds.ValueExpr{Kind: tds.ValLit, Lit: v}
			p.Column = ""
		} else if p.LeftExpr != nil {
			p.LeftExpr = bindVal(p.LeftExpr, row, idx, alias, table)
		}
		if cr, ok := p.Value.(tds.ColRef); ok {
			if v, ok2 := outerVal(cr.Name, row, idx, alias, table); ok2 {
				p.Value = &tds.ValueExpr{Kind: tds.ValLit, Lit: v}
			}
		} else if ve, ok := p.Value.(*tds.ValueExpr); ok {
			p.Value = bindVal(ve, row, idx, alias, table)
		}
		out.Pred = &p
	}
	out.And = bindExprs(e.And, row, idx, alias, table)
	out.Or = bindExprs(e.Or, row, idx, alias, table)
	out.Not = bindExpr(e.Not, row, idx, alias, table)
	return &out
}

func bindExprs(es []*tds.Expr, row []any, idx map[string]int, alias, table string) []*tds.Expr {
	if es == nil {
		return nil
	}
	out := make([]*tds.Expr, len(es))
	for i, e := range es {
		out[i] = bindExpr(e, row, idx, alias, table)
	}
	return out
}

func bindVal(ve *tds.ValueExpr, row []any, idx map[string]int, alias, table string) *tds.ValueExpr {
	if ve == nil {
		return nil
	}
	if ve.Kind == tds.ValCol {
		if v, ok := outerVal(ve.Col, row, idx, alias, table); ok {
			return &tds.ValueExpr{Kind: tds.ValLit, Lit: v}
		}
		return ve
	}
	out := *ve
	out.Left = bindVal(ve.Left, row, idx, alias, table)
	out.Right = bindVal(ve.Right, row, idx, alias, table)
	if ve.Args != nil {
		out.Args = make([]*tds.ValueExpr, len(ve.Args))
		for i, a := range ve.Args {
			out.Args[i] = bindVal(a, row, idx, alias, table)
		}
	}
	return &out
}

func outerVal(col string, row []any, idx map[string]int, alias, table string) (any, bool) {
	if !colRefsOuter(col, alias, table) {
		return nil, false
	}
	short := col[strings.LastIndex(col, ".")+1:]
	if i, ok := idx[short]; ok {
		return row[i], true
	}
	return nil, false
}

func resolveValueSubqueries(ctx context.Context, b tds.Backend, ve *tds.ValueExpr) error {
	if ve == nil {
		return nil
	}
	if ve.Kind == tds.ValSubquery && ve.Sub != nil {
		_, data, err := runMaterialize(ctx, b, ve.Sub)
		if err != nil {
			return err
		}
		var val any
		if len(data) > 0 && len(data[0]) > 0 {
			val = data[0][0]
		}
		ve.Kind = tds.ValLit
		ve.Lit = val
		ve.Sub = nil
		return nil
	}
	for _, sub := range []*tds.ValueExpr{ve.Left, ve.Right, ve.Operand, ve.Else} {
		if err := resolveValueSubqueries(ctx, b, sub); err != nil {
			return err
		}
	}
	for _, a := range ve.Args {
		if err := resolveValueSubqueries(ctx, b, a); err != nil {
			return err
		}
	}
	for i := range ve.Whens {
		if err := resolveValueSubqueries(ctx, b, ve.Whens[i].Match); err != nil {
			return err
		}
		if err := resolveValueSubqueries(ctx, b, ve.Whens[i].Result); err != nil {
			return err
		}
		if err := resolveSubqueries(ctx, b, ve.Whens[i].Cond, "", ""); err != nil {
			return err
		}
	}
	return nil
}

const maxRecursionDepth = 100

func isRecursiveCTE(cte *tds.Query, name string) bool {
	for a := cte; a != nil; a = a.Union {
		if strings.EqualFold(a.Table, name) {
			return true
		}
	}
	return false
}

func runRecursiveCTE(ctx context.Context, b tds.Backend, name string, body *tds.Query) ([]catalog.Column, [][]any, error) {
	var anchors, recs []*tds.Query
	for a := body; a != nil; a = a.Union {
		arm := *a
		arm.Union = nil
		if strings.EqualFold(a.Table, name) {
			recs = append(recs, &arm)
		} else {
			anchors = append(anchors, &arm)
		}
	}
	var cols []catalog.Column
	var acc [][]any
	for i, a := range anchors {
		c, d, err := runMaterialize(ctx, b, a)
		if err != nil {
			return nil, nil, err
		}
		if i == 0 {
			cols = c
		}
		acc = append(acc, d...)
	}
	working := acc
	for depth := 0; depth < maxRecursionDepth && len(working) > 0; depth++ {
		var next [][]any
		for _, r := range recs {
			rs, err := exec.Apply(cols, working, r)
			if err != nil {
				return nil, nil, err
			}
			_, d, err := materialize(rs)
			if err != nil {
				return nil, nil, err
			}
			next = append(next, d...)
		}
		if len(next) == 0 {
			break
		}
		acc = append(acc, next...)
		working = next
	}
	return cols, acc, nil
}

func runMaterialize(ctx context.Context, b tds.Backend, q *tds.Query) ([]catalog.Column, [][]any, error) {
	rs, err := runParsed(ctx, b, q)
	if err != nil {
		return nil, nil, err
	}
	return materialize(rs)
}

func materialize(rows tds.Rows) ([]catalog.Column, [][]any, error) {
	defer rows.Close()
	cols := rows.Columns()
	var data [][]any
	for rows.Next() {
		v, err := rows.Values()
		if err != nil {
			return nil, nil, err
		}
		data = append(data, v)
	}
	return cols, data, rows.Err()
}

func joinQuery(ctx context.Context, sc tds.Scanner, q *tds.Query) (tds.Rows, error) {
	cols, rows, err := scanTable(ctx, sc, q.Schema, q.Table)
	if err != nil {
		return nil, err
	}
	cols = qualify(cols, effAlias(q.FromAlias, q.Table))
	for _, j := range q.Joins {
		rcols, rrows, err := scanTable(ctx, sc, j.Schema, j.Table)
		if err != nil {
			return nil, err
		}
		rcols = qualify(rcols, effAlias(j.Alias, j.Table))
		cols, rows, err = exec.Join(cols, rows, j.Type, rcols, rrows, j.On)
		if err != nil {
			return nil, err
		}
	}
	return exec.Apply(cols, rows, q)
}

func scanTable(ctx context.Context, sc tds.Scanner, schema, table string) ([]catalog.Column, [][]any, error) {
	raw, err := sc.Scan(ctx, &tds.Query{Schema: schema, Table: table})
	if err != nil {
		return nil, nil, err
	}
	return materialize(raw)
}

func qualify(cols []catalog.Column, alias string) []catalog.Column {
	out := make([]catalog.Column, len(cols))
	for i, c := range cols {
		c.Name = alias + "." + c.Name
		out[i] = c
	}
	return out
}

func effAlias(alias, table string) string {
	if alias != "" {
		return alias
	}
	return table
}

const serverVersion = "Microsoft SQL Server 2022 (haystak-tds-spi gateway) - TDS 7.4"

func probe(sql string) (tds.Rows, bool, error) {
	u := strings.TrimSuffix(strings.TrimSpace(sql), ";")
	u = strings.ToUpper(strings.TrimSpace(u))
	if u == "" || strings.HasPrefix(u, "SET ") || strings.HasPrefix(u, "USE ") {
		return nil, true, nil
	}
	if !strings.HasPrefix(u, "SELECT ") {
		return nil, false, nil
	}
	e := strings.TrimSpace(u[len("SELECT "):])
	if i := strings.Index(e, " AS "); i >= 0 {
		e = strings.TrimSpace(e[:i])
	}
	val, ok := probeValue(e)
	if !ok {
		return nil, false, nil
	}
	t := types.Type{Kind: types.String, MaxLen: 255}
	if _, isInt := val.(int64); isInt {
		t = types.Type{Kind: types.Int32}
	}
	rs, err := scalarRows("", t, val)
	return rs, true, err
}

func probeValue(e string) (any, bool) {
	switch e {
	case "@@VERSION":
		return serverVersion, true
	case "@@SPID":
		return int64(1), true
	case "@@SERVERNAME":
		return "haystak-tds-spi", true
	case "@@LANGUAGE":
		return "us_english", true
	case "@@ROWCOUNT", "@@ERROR", "@@TRANCOUNT", "@@FETCH_STATUS":
		return int64(0), true
	case "DB_NAME()", "ORIGINAL_DB_NAME()":
		return "master", true
	case "SCHEMA_NAME()":
		return "dbo", true
	case "SYSTEM_USER", "CURRENT_USER", "SESSION_USER", "SUSER_NAME()", "SUSER_SNAME()", "USER_NAME()", "USER":
		return "haystak", true
	case "HOST_NAME()", "APP_NAME()":
		return "haystak-tds-spi", true
	}
	switch {
	case strings.HasPrefix(e, "SERVERPROPERTY("):
		return serverProperty(e), true
	case strings.HasPrefix(e, "DATABASEPROPERTYEX("):
		return "ON", true
	}
	return nil, false
}

func serverProperty(e string) any {
	arg := ""
	if i := strings.Index(e, "'"); i >= 0 {
		if j := strings.Index(e[i+1:], "'"); j >= 0 {
			arg = e[i+1 : i+1+j]
		}
	}
	switch arg {
	case "PRODUCTVERSION":
		return "16.0.1000.6"
	case "PRODUCTLEVEL":
		return "RTM"
	case "EDITION":
		return "Developer Edition (64-bit)"
	case "ENGINEEDITION":
		return int64(3)
	case "COLLATION":
		return "SQL_Latin1_General_CP1_CI_AS"
	case "ISCLUSTERED", "ISINTEGRATEDSECURITYONLY":
		return int64(0)
	}
	return ""
}

func scalarRows(name string, t types.Type, val any) (tds.Rows, error) {
	return exec.Apply([]catalog.Column{{Name: name, Type: t}}, [][]any{{val}}, &tds.Query{})
}

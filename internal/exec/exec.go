// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package exec

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

// Apply runs a logical query against in-memory rows. It is the core thin-path evaluator,
// used for the catalog projection and any Scanner-style backend.
func Apply(cols []catalog.Column, data [][]any, q *tds.Query) (tds.Rows, error) {
	return ApplyWith(cols, data, q, nil)
}

func ApplyWith(cols []catalog.Column, data [][]any, q *tds.Query, sub SubFn) (tds.Rows, error) {
	idx := indexCols(cols)

	var filtered [][]any
	for _, row := range data {
		ok, err := evalExpr(idx, row, q.Where, sub)
		if err != nil {
			return nil, err
		}
		if ok {
			filtered = append(filtered, row)
		}
	}

	if isAggregate(q) {
		return aggregate(cols, idx, filtered, q)
	}

	mCols, mRows, mSel, err := materializeExprs(cols, idx, filtered, q.Select)
	if err != nil {
		return nil, err
	}
	cols, filtered, idx = mCols, mRows, indexCols(mCols)

	if len(q.OrderBy) > 0 {
		order := q.OrderBy
		if hasOrderExpr(order) {
			var err error
			cols, filtered, idx, order, err = materializeOrderExprs(cols, idx, filtered, order)
			if err != nil {
				return nil, err
			}
		}
		for _, o := range order {
			if _, ok := resolveCol(idx, o.Column); !ok {
				return nil, fmt.Errorf("exec: unknown column %q in ORDER BY", o.Column)
			}
		}
		sort.SliceStable(filtered, func(i, j int) bool {
			return less(idx, filtered[i], filtered[j], order)
		})
	}

	outCols, proj, err := projectItems(cols, idx, mSel)
	if err != nil {
		return nil, err
	}
	var out [][]any
	for _, row := range filtered {
		out = append(out, pick(row, proj))
	}
	if q.Distinct {
		out = dedupe(out)
	}
	out = paginate(out, q.Offset, effLimit(q, len(out)))
	return &memRows{cols: outCols, data: out}, nil
}

func isAggregate(q *tds.Query) bool {
	if len(q.GroupBy) > 0 {
		return true
	}
	for _, it := range q.Select {
		if it.Agg != tds.AggNone {
			return true
		}
	}
	return false
}

func aggregate(cols []catalog.Column, idx map[string]int, rows [][]any, q *tds.Query) (tds.Rows, error) {
	groupIdx := make([]int, 0, len(q.GroupBy))
	for _, g := range q.GroupBy {
		i, ok := idx[g]
		if !ok {
			return nil, fmt.Errorf("exec: unknown column %q in GROUP BY", g)
		}
		groupIdx = append(groupIdx, i)
	}

	outCols, err := aggOutCols(cols, idx, q.Select)
	if err != nil {
		return nil, err
	}

	var order []string
	groups := map[string][][]any{}
	for _, row := range rows {
		key := "__all__"
		if len(groupIdx) > 0 {
			parts := make([]any, len(groupIdx))
			for j, gi := range groupIdx {
				parts[j] = row[gi]
			}
			key = fmt.Sprintf("%v", parts)
		}
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], row)
	}
	if len(q.GroupBy) == 0 && len(order) == 0 {
		order = []string{"__all__"}
	}

	outIdx := indexCols(outCols)
	type aggregated struct {
		row   []any
		group [][]any
	}
	rowsOut := make([]aggregated, 0, len(order))
	for _, k := range order {
		row, err := aggRow(idx, q.Select, groups[k])
		if err != nil {
			return nil, err
		}
		rowsOut = append(rowsOut, aggregated{row: row, group: groups[k]})
	}

	// HAVING is evaluated in the group context: aggregate calls compute over the group, alias and
	// grouped-column references resolve against the aggregated output row.
	if q.Having != nil {
		var kept []aggregated
		for _, ar := range rowsOut {
			ok, err := evalAggExpr(idx, ar.group, outIdx, ar.row, q.Having)
			if err != nil {
				return nil, err
			}
			if ok {
				kept = append(kept, ar)
			}
		}
		rowsOut = kept
	}

	if len(q.OrderBy) > 0 {
		keys := make([][]any, len(rowsOut))
		for i, ar := range rowsOut {
			k := make([]any, len(q.OrderBy))
			for j, o := range q.OrderBy {
				kv, err := aggOrderKey(idx, outIdx, ar.group, ar.row, o)
				if err != nil {
					return nil, err
				}
				k[j] = kv
			}
			keys[i] = k
		}
		perm := make([]int, len(rowsOut))
		for i := range perm {
			perm[i] = i
		}
		sort.SliceStable(perm, func(a, b int) bool {
			for j, o := range q.OrderBy {
				c, ok := compare(keys[perm[a]][j], keys[perm[b]][j])
				if !ok || c == 0 {
					continue
				}
				if o.Desc {
					return c > 0
				}
				return c < 0
			}
			return false
		})
		sorted := make([]aggregated, len(rowsOut))
		for i, p := range perm {
			sorted[i] = rowsOut[p]
		}
		rowsOut = sorted
	}

	out := make([][]any, len(rowsOut))
	for i, ar := range rowsOut {
		out[i] = ar.row
	}
	out = paginate(out, q.Offset, effLimit(q, len(out)))
	return &memRows{cols: outCols, data: out}, nil
}

// aggOrderKey resolves one ORDER BY term to its sort value in the aggregate context.
func aggOrderKey(origIdx, outIdx map[string]int, group [][]any, outRow []any, o tds.OrderItem) (any, error) {
	switch {
	case o.Expr != nil:
		return evalAggValue(origIdx, group, outIdx, outRow, o.Expr)
	case o.Ordinal > 0:
		if o.Ordinal <= len(outRow) {
			return outRow[o.Ordinal-1], nil
		}
		return nil, nil
	default:
		if i, ok := resolveCol(outIdx, o.Column); ok {
			return outRow[i], nil
		}
		return nil, nil
	}
}

func aggFuncFromName(name string) tds.AggFunc {
	switch strings.ToUpper(name) {
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

func aggArg(args []*tds.ValueExpr) string {
	if len(args) == 0 {
		return "*"
	}
	if args[0].Kind == tds.ValCol {
		return args[0].Col
	}
	return ""
}

// evalAggValue evaluates a value expression in the GROUP context: aggregate calls (COUNT/SUM/…) compute
// over the group's rows via origIdx; everything else evaluates against the aggregated output row.
func evalAggValue(origIdx map[string]int, group [][]any, outIdx map[string]int, outRow []any, ve *tds.ValueExpr) (any, error) {
	switch ve.Kind {
	case tds.ValFunc:
		if fn := aggFuncFromName(ve.Func); fn != tds.AggNone {
			return computeAgg(fn, aggArg(ve.Args), origIdx, group)
		}
	case tds.ValBinary:
		l, err := evalAggValue(origIdx, group, outIdx, outRow, ve.Left)
		if err != nil {
			return nil, err
		}
		r, err := evalAggValue(origIdx, group, outIdx, outRow, ve.Right)
		if err != nil {
			return nil, err
		}
		return evalBinary(ve.Op, l, r), nil
	}
	return evalValue(outIdx, outRow, ve)
}

func evalAggExpr(origIdx map[string]int, group [][]any, outIdx map[string]int, outRow []any, e *tds.Expr) (bool, error) {
	switch {
	case e == nil:
		return true, nil
	case e.Const != nil:
		return *e.Const, nil
	case e.Pred != nil:
		return evalAggPred(origIdx, group, outIdx, outRow, e.Pred)
	case e.Not != nil:
		v, err := evalAggExpr(origIdx, group, outIdx, outRow, e.Not)
		return !v, err
	case len(e.And) > 0:
		for _, c := range e.And {
			v, err := evalAggExpr(origIdx, group, outIdx, outRow, c)
			if err != nil {
				return false, err
			}
			if !v {
				return false, nil
			}
		}
		return true, nil
	case len(e.Or) > 0:
		for _, c := range e.Or {
			v, err := evalAggExpr(origIdx, group, outIdx, outRow, c)
			if err != nil {
				return false, err
			}
			if v {
				return true, nil
			}
		}
		return false, nil
	}
	return true, nil
}

func evalAggPred(origIdx map[string]int, group [][]any, outIdx map[string]int, outRow []any, p *tds.Predicate) (bool, error) {
	var v any
	if p.LeftExpr != nil {
		lv, err := evalAggValue(origIdx, group, outIdx, outRow, p.LeftExpr)
		if err != nil {
			return false, err
		}
		v = lv
	} else {
		i, ok := resolveCol(outIdx, p.Column)
		if !ok {
			return false, fmt.Errorf("exec: unknown column %q in HAVING", p.Column)
		}
		v = outRow[i]
	}
	switch p.Op {
	case tds.OpIsNull:
		return v == nil, nil
	case tds.OpIsNotNull:
		return v != nil, nil
	case tds.OpIn:
		list, _ := p.Value.([]any)
		for _, item := range list {
			if c, ok := compare(v, item); ok && c == 0 {
				return true, nil
			}
		}
		return false, nil
	case tds.OpLike:
		pat, _ := p.Value.(string)
		return likeMatch(fmt.Sprintf("%v", v), pat), nil
	default:
		rhs := p.Value
		switch r := rhs.(type) {
		case *tds.ValueExpr:
			rv, err := evalAggValue(origIdx, group, outIdx, outRow, r)
			if err != nil {
				return false, err
			}
			rhs = rv
		case tds.ColRef:
			j, ok := resolveCol(outIdx, r.Name)
			if !ok {
				return false, fmt.Errorf("exec: unknown column %q in HAVING", r.Name)
			}
			rhs = outRow[j]
		}
		c, ok := compare(v, rhs)
		if !ok {
			return false, nil
		}
		return satisfies(p.Op, c), nil
	}
}

func aggOutCols(cols []catalog.Column, idx map[string]int, sel []tds.SelectItem) ([]catalog.Column, error) {
	var out []catalog.Column
	for _, it := range sel {
		name := it.Alias
		var typ types.Type
		switch it.Agg {
		case tds.AggNone:
			i, ok := idx[it.Column]
			if !ok {
				return nil, fmt.Errorf("exec: unknown column %q in SELECT", it.Column)
			}
			typ = cols[i].Type
			if name == "" {
				name = it.Column
			}
		case tds.AggCount:
			typ = types.Type{Kind: types.Int64}
			if name == "" {
				name = "count"
			}
		case tds.AggSum, tds.AggAvg:
			typ = types.Type{Kind: types.Float64}
			if name == "" {
				name = "agg"
			}
		case tds.AggMin, tds.AggMax:
			if i, ok := idx[it.Arg]; ok {
				typ = cols[i].Type
			} else {
				typ = types.Type{Kind: types.Float64}
			}
			if name == "" {
				name = it.Arg
			}
		}
		out = append(out, catalog.Column{Name: name, Type: typ})
	}
	return out, nil
}

func aggRow(idx map[string]int, sel []tds.SelectItem, rows [][]any) ([]any, error) {
	out := make([]any, len(sel))
	for j, it := range sel {
		if it.Agg == tds.AggNone {
			if i, ok := resolveCol(idx, it.Column); ok && len(rows) > 0 {
				out[j] = rows[0][i]
			}
			continue
		}
		v, err := computeAgg(it.Agg, it.Arg, idx, rows)
		if err != nil {
			return nil, err
		}
		out[j] = v
	}
	return out, nil
}

// computeAgg evaluates one aggregate function over a group's rows. arg is the column name ("*" or
// "" for COUNT-all); idx is the pre-aggregation column index.
func computeAgg(fn tds.AggFunc, arg string, idx map[string]int, rows [][]any) (any, error) {
	switch fn {
	case tds.AggCount:
		if arg == "*" || arg == "" {
			return int64(len(rows)), nil
		}
		i, ok := resolveCol(idx, arg)
		if !ok {
			return nil, fmt.Errorf("exec: unknown column %q in COUNT", arg)
		}
		var n int64
		for _, r := range rows {
			if r[i] != nil {
				n++
			}
		}
		return n, nil
	case tds.AggSum, tds.AggAvg:
		i, ok := resolveCol(idx, arg)
		if !ok {
			return nil, fmt.Errorf("exec: unknown column %q in aggregate", arg)
		}
		var sum float64
		var cnt int
		for _, r := range rows {
			if r[i] != nil {
				sum += toFloat(r[i])
				cnt++
			}
		}
		if fn == tds.AggSum {
			return sum, nil
		}
		if cnt > 0 {
			return sum / float64(cnt), nil
		}
		return nil, nil
	case tds.AggMin, tds.AggMax:
		i, ok := resolveCol(idx, arg)
		if !ok {
			return nil, fmt.Errorf("exec: unknown column %q in aggregate", arg)
		}
		var best any
		for _, r := range rows {
			v := r[i]
			if v == nil {
				continue
			}
			if best == nil {
				best = v
				continue
			}
			if c, ok := compare(v, best); ok {
				if (fn == tds.AggMin && c < 0) || (fn == tds.AggMax && c > 0) {
					best = v
				}
			}
		}
		return best, nil
	}
	return nil, nil
}

func toFloat(v any) float64 {
	switch x := v.(type) {
	case int64:
		return float64(x)
	case int:
		return float64(x)
	case float64:
		return x
	case float32:
		return float64(x)
	}
	return 0
}

func effLimit(q *tds.Query, total int) int {
	if q.LimitPercent && q.Limit > 0 {
		return (q.Limit*total + 99) / 100
	}
	return q.Limit
}

func paginate(out [][]any, offset, limit int) [][]any {
	if offset > 0 {
		if offset >= len(out) {
			return nil
		}
		out = out[offset:]
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func dedupe(rows [][]any) [][]any {
	seen := make(map[string]bool, len(rows))
	out := rows[:0]
	for _, row := range rows {
		k := fmt.Sprintf("%v", row)
		if !seen[k] {
			seen[k] = true
			out = append(out, row)
		}
	}
	return out
}

// Join nested-loops two materialized tables on the ON expr (INNER/LEFT/CROSS).
func Join(lcols []catalog.Column, lrows [][]any, jt tds.JoinType, rcols []catalog.Column, rrows [][]any, on *tds.Expr) ([]catalog.Column, [][]any, error) {
	cols := make([]catalog.Column, 0, len(lcols)+len(rcols))
	cols = append(cols, lcols...)
	cols = append(cols, rcols...)
	idx := indexCols(cols)
	rightMatched := make([]bool, len(rrows))
	var out [][]any
	for _, lr := range lrows {
		matched := false
		for ri, rr := range rrows {
			combined := make([]any, 0, len(lr)+len(rr))
			combined = append(combined, lr...)
			combined = append(combined, rr...)
			keep := true
			if on != nil {
				ok, err := evalExpr(idx, combined, on, nil)
				if err != nil {
					return nil, nil, err
				}
				keep = ok
			}
			if keep {
				out = append(out, combined)
				matched = true
				rightMatched[ri] = true
			}
		}
		if !matched && (jt == tds.JoinLeft || jt == tds.JoinFull) {
			combined := append(append([]any{}, lr...), make([]any, len(rcols))...)
			out = append(out, combined)
		}
	}
	if jt == tds.JoinRight || jt == tds.JoinFull {
		for ri, rr := range rrows {
			if !rightMatched[ri] {
				combined := append(make([]any, len(lcols)), rr...)
				out = append(out, combined)
			}
		}
	}
	return cols, out, nil
}

func resolveCol(idx map[string]int, name string) (int, bool) {
	if i, ok := idx[name]; ok {
		return i, true
	}
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		if i, ok := idx[name[dot+1:]]; ok {
			return i, true
		}
	}
	return -1, false
}

func indexCols(cols []catalog.Column) map[string]int {
	m := make(map[string]int, len(cols))
	for i, c := range cols {
		m[c.Name] = i
	}
	// Add unambiguous short names (alias.col → col) so bare refs resolve on joined rows.
	dup := map[string]bool{}
	for i, c := range cols {
		dot := strings.LastIndex(c.Name, ".")
		if dot < 0 {
			continue
		}
		short := c.Name[dot+1:]
		if _, taken := m[short]; taken {
			dup[short] = true
			continue
		}
		m[short] = i
	}
	for s := range dup {
		delete(m, s)
	}
	return m
}

func projectItems(cols []catalog.Column, idx map[string]int, sel []tds.SelectItem) ([]catalog.Column, []int, error) {
	if len(sel) == 0 {
		proj := make([]int, len(cols))
		for i := range cols {
			proj[i] = i
		}
		return cols, proj, nil
	}
	outCols := make([]catalog.Column, 0, len(sel))
	proj := make([]int, 0, len(sel))
	for _, it := range sel {
		i, ok := resolveCol(idx, it.Column)
		if !ok {
			return nil, nil, fmt.Errorf("exec: unknown column %q in SELECT", it.Column)
		}
		c := cols[i]
		if it.Alias != "" {
			c.Name = it.Alias
		} else if strings.HasPrefix(c.Name, "__expr") {
			c.Name = ""
		} else if dot := strings.LastIndex(c.Name, "."); dot >= 0 {
			c.Name = c.Name[dot+1:]
		}
		outCols = append(outCols, c)
		proj = append(proj, i)
	}
	return outCols, proj, nil
}

func pick(row []any, proj []int) []any {
	out := make([]any, len(proj))
	for j, i := range proj {
		out[j] = row[i]
	}
	return out
}

// SubFn evaluates a correlated subquery against the current outer row.
type SubFn func(outerRow []any, idx map[string]int, sub *tds.Query) ([][]any, error)

func evalExpr(idx map[string]int, row []any, e *tds.Expr, sub SubFn) (bool, error) {
	switch {
	case e == nil:
		return true, nil
	case e.Const != nil:
		return *e.Const, nil
	case e.Pred != nil:
		return evalPred(idx, row, e.Pred, sub)
	case e.Not != nil:
		v, err := evalExpr(idx, row, e.Not, sub)
		return !v, err
	case len(e.And) > 0:
		for _, c := range e.And {
			v, err := evalExpr(idx, row, c, sub)
			if err != nil {
				return false, err
			}
			if !v {
				return false, nil
			}
		}
		return true, nil
	case len(e.Or) > 0:
		for _, c := range e.Or {
			v, err := evalExpr(idx, row, c, sub)
			if err != nil {
				return false, err
			}
			if v {
				return true, nil
			}
		}
		return false, nil
	}
	return true, nil
}

func evalPred(idx map[string]int, row []any, p *tds.Predicate, sub SubFn) (bool, error) {
	if p.Op == tds.OpExists && p.Sub != nil {
		if sub == nil {
			return false, nil
		}
		rows, err := sub(row, idx, p.Sub)
		if err != nil {
			return false, err
		}
		return len(rows) > 0, nil
	}
	var v any
	if p.LeftExpr != nil {
		lv, err := evalValue(idx, row, p.LeftExpr)
		if err != nil {
			return false, err
		}
		v = lv
	} else {
		i, ok := resolveCol(idx, p.Column)
		if !ok {
			return false, fmt.Errorf("exec: unknown column %q in WHERE", p.Column)
		}
		v = row[i]
	}
	if p.Op == tds.OpIn && p.Sub != nil {
		if sub == nil {
			return false, nil
		}
		rows, err := sub(row, idx, p.Sub)
		if err != nil {
			return false, err
		}
		for _, r := range rows {
			if len(r) > 0 {
				if c, ok := compare(v, r[0]); ok && c == 0 {
					return true, nil
				}
			}
		}
		return false, nil
	}
	switch p.Op {
	case tds.OpIsNull:
		return v == nil, nil
	case tds.OpIsNotNull:
		return v != nil, nil
	case tds.OpIn:
		list, _ := p.Value.([]any)
		for _, item := range list {
			if c, ok := compare(v, item); ok && c == 0 {
				return true, nil
			}
		}
		return false, nil
	case tds.OpLike:
		pat, _ := p.Value.(string)
		return likeMatch(fmt.Sprintf("%v", v), pat), nil
	default:
		rhs := p.Value
		switch r := rhs.(type) {
		case *tds.ValueExpr:
			rv, err := evalValue(idx, row, r)
			if err != nil {
				return false, err
			}
			rhs = rv
		case tds.ColRef:
			j, ok := resolveCol(idx, r.Name)
			if !ok {
				return false, fmt.Errorf("exec: unknown column %q in WHERE", r.Name)
			}
			rhs = row[j]
		}
		c, ok := compare(v, rhs)
		if !ok {
			return false, nil
		}
		return satisfies(p.Op, c), nil
	}
}

func likeMatch(s, pattern string) bool {
	var b strings.Builder
	b.WriteString("(?is)^")
	for _, r := range pattern {
		switch r {
		case '%':
			b.WriteString(".*")
		case '_':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	if err != nil {
		return false
	}
	return re.MatchString(s)
}

func satisfies(op tds.Op, c int) bool {
	switch op {
	case tds.OpEq:
		return c == 0
	case tds.OpNe:
		return c != 0
	case tds.OpLt:
		return c < 0
	case tds.OpLe:
		return c <= 0
	case tds.OpGt:
		return c > 0
	case tds.OpGe:
		return c >= 0
	}
	return false
}

func less(idx map[string]int, a, b []any, order []tds.OrderItem) bool {
	for _, o := range order {
		i := idx[o.Column]
		c, ok := compare(a[i], b[i])
		if !ok || c == 0 {
			continue
		}
		if o.Desc {
			return c > 0
		}
		return c < 0
	}
	return false
}

func compare(a, b any) (int, bool) {
	if a == nil || b == nil {
		return 0, false
	}
	switch av := a.(type) {
	case int64:
		switch bv := b.(type) {
		case int64:
			return cmpInt(av, bv), true
		case float64:
			return cmpFloat(float64(av), bv), true
		}
	case float64:
		switch bv := b.(type) {
		case int64:
			return cmpFloat(av, float64(bv)), true
		case float64:
			return cmpFloat(av, bv), true
		}
	case string:
		if bv, ok := b.(string); ok {
			return strings.Compare(av, bv), true
		}
	case bool:
		if bv, ok := b.(bool); ok {
			switch {
			case av == bv:
				return 0, true
			case !av:
				return -1, true
			default:
				return 1, true
			}
		}
	case time.Time:
		if bv, ok := b.(time.Time); ok {
			switch {
			case av.Before(bv):
				return -1, true
			case av.After(bv):
				return 1, true
			default:
				return 0, true
			}
		}
	case []byte:
		if bv, ok := b.([]byte); ok {
			return bytes.Compare(av, bv), true
		}
	}
	return strings.Compare(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b)), true
}

func cmpInt(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func cmpFloat(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

type memRows struct {
	cols []catalog.Column
	data [][]any
	pos  int
}

func (r *memRows) Columns() []catalog.Column { return r.cols }

func (r *memRows) Next() bool {
	if r.pos >= len(r.data) {
		return false
	}
	r.pos++
	return true
}

func (r *memRows) Values() ([]any, error) { return r.data[r.pos-1], nil }
func (r *memRows) Err() error             { return nil }
func (r *memRows) Close() error           { return nil }

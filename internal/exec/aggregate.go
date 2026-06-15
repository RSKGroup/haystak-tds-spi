// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package exec

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

func IsAggregate(q *tds.Query) bool { return isAggregate(q) }

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
	return evalValue(outIdx, outRow, ve, nil)
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

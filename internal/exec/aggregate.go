// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package exec

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/internal/fold"
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

func IsAggregate(q *tds.Query) bool { return isAggregate(q) }

// HasDistinctAgg reports whether any select item is a DISTINCT aggregate (COUNT(DISTINCT col)).
func HasDistinctAgg(q *tds.Query) bool {
	for _, it := range q.Select {
		if it.AggDist {
			return true
		}
	}
	return false
}

func isAggregate(q *tds.Query) bool {
	if len(q.GroupBy) > 0 || len(q.GroupingSets) > 0 {
		return true
	}
	for _, it := range q.Select {
		if it.Agg != tds.AggNone || (it.Window == nil && hasAggCall(it.Expr)) {
			return true
		}
	}
	return false
}

// hasAggCall reports an aggregate call anywhere inside a scalar expression, such as UPPER(MIN(x)); a
// subquery's own aggregates belong to it.
func hasAggCall(ve *tds.ValueExpr) bool {
	if ve == nil || ve.Kind == tds.ValSubquery {
		return false
	}
	if ve.Kind == tds.ValFunc && aggFuncFromName(ve.Func) != tds.AggNone {
		return true
	}
	if hasAggCall(ve.Left) || hasAggCall(ve.Right) || hasAggCall(ve.Operand) || hasAggCall(ve.Else) {
		return true
	}
	for _, a := range ve.Args {
		if hasAggCall(a) {
			return true
		}
	}
	for _, w := range ve.Whens {
		if hasAggCall(w.Match) || hasAggCall(w.Result) || exprHasAggCall(w.Cond) {
			return true
		}
	}
	return false
}

func exprHasAggCall(e *tds.Expr) bool {
	if e == nil {
		return false
	}
	for _, c := range append(append([]*tds.Expr{}, e.And...), e.Or...) {
		if exprHasAggCall(c) {
			return true
		}
	}
	if exprHasAggCall(e.Not) {
		return true
	}
	if e.Pred != nil {
		v, _ := e.Pred.Value.(*tds.ValueExpr)
		return hasAggCall(e.Pred.LeftExpr) || hasAggCall(v)
	}
	return false
}

// bindAggregates copies ve with every aggregate call replaced by its value over group, leaving an
// ordinary expression the row evaluator can finish. An aggregate over an expression evaluates it per row.
func bindAggregates(ve *tds.ValueExpr, idx map[string]int, group [][]any, env *Env) (*tds.ValueExpr, error) {
	if ve == nil || ve.Kind == tds.ValSubquery {
		return ve, nil
	}
	if ve.Kind == tds.ValFunc {
		if fn := aggFuncFromName(ve.Func); fn != tds.AggNone {
			arg, argIdx, rows := aggArg(ve.Args), idx, group
			if arg == "" && len(ve.Args) > 0 {
				arg, argIdx = "__arg", map[string]int{"__arg": 0}
				rows = make([][]any, len(group))
				for i, r := range group {
					v, err := evalValue(idx, r, ve.Args[0], env)
					if err != nil {
						return nil, err
					}
					rows[i] = []any{v}
				}
			}
			v, err := computeAgg(fn, arg, aggSep(ve.Args), argIdx, rows, false)
			if err != nil {
				return nil, err
			}
			return &tds.ValueExpr{Kind: tds.ValLit, Lit: v}, nil
		}
	}
	out := *ve
	var err error
	if out.Left, err = bindAggregates(ve.Left, idx, group, env); err != nil {
		return nil, err
	}
	if out.Right, err = bindAggregates(ve.Right, idx, group, env); err != nil {
		return nil, err
	}
	if out.Operand, err = bindAggregates(ve.Operand, idx, group, env); err != nil {
		return nil, err
	}
	if out.Else, err = bindAggregates(ve.Else, idx, group, env); err != nil {
		return nil, err
	}
	if ve.Args != nil {
		out.Args = make([]*tds.ValueExpr, len(ve.Args))
		for i, a := range ve.Args {
			if out.Args[i], err = bindAggregates(a, idx, group, env); err != nil {
				return nil, err
			}
		}
	}
	if ve.Whens != nil {
		out.Whens = make([]tds.CaseWhen, len(ve.Whens))
		for i, w := range ve.Whens {
			nw := w
			if nw.Match, err = bindAggregates(w.Match, idx, group, env); err != nil {
				return nil, err
			}
			if nw.Result, err = bindAggregates(w.Result, idx, group, env); err != nil {
				return nil, err
			}
			if nw.Cond, err = bindAggExpr(w.Cond, idx, group, env); err != nil {
				return nil, err
			}
			out.Whens[i] = nw
		}
	}
	return &out, nil
}

func bindAggExpr(e *tds.Expr, idx map[string]int, group [][]any, env *Env) (*tds.Expr, error) {
	if e == nil || !exprHasAggCall(e) {
		return e, nil
	}
	out := *e
	var err error
	bindList := func(in []*tds.Expr) ([]*tds.Expr, error) {
		if in == nil {
			return nil, nil
		}
		list := make([]*tds.Expr, len(in))
		for i, c := range in {
			if list[i], err = bindAggExpr(c, idx, group, env); err != nil {
				return nil, err
			}
		}
		return list, nil
	}
	if out.And, err = bindList(e.And); err != nil {
		return nil, err
	}
	if out.Or, err = bindList(e.Or); err != nil {
		return nil, err
	}
	if out.Not, err = bindAggExpr(e.Not, idx, group, env); err != nil {
		return nil, err
	}
	if e.Pred != nil {
		pred := *e.Pred
		if pred.LeftExpr, err = bindAggregates(e.Pred.LeftExpr, idx, group, env); err != nil {
			return nil, err
		}
		if v, ok := e.Pred.Value.(*tds.ValueExpr); ok {
			if pred.Value, err = bindAggregates(v, idx, group, env); err != nil {
				return nil, err
			}
		}
		out.Pred = &pred
	}
	return &out, nil
}

// aggExprType types a select expression holding aggregates by standing each call in for its result.
func aggExprType(ve *tds.ValueExpr, cols []catalog.Column, idx map[string]int) types.Type {
	var stand func(*tds.ValueExpr) *tds.ValueExpr
	stand = func(v *tds.ValueExpr) *tds.ValueExpr {
		if v == nil || v.Kind != tds.ValFunc {
			if v != nil && (v.Left != nil || v.Right != nil || len(v.Args) > 0) {
				c := *v
				c.Left, c.Right = stand(v.Left), stand(v.Right)
				if v.Args != nil {
					c.Args = make([]*tds.ValueExpr, len(v.Args))
					for i, a := range v.Args {
						c.Args[i] = stand(a)
					}
				}
				return &c
			}
			return v
		}
		switch aggFuncFromName(v.Func) {
		case tds.AggNone:
			c := *v
			c.Args = make([]*tds.ValueExpr, len(v.Args))
			for i, a := range v.Args {
				c.Args[i] = stand(a)
			}
			return &c
		case tds.AggCount, tds.AggCountBig, tds.AggChecksumAgg, tds.AggApproxCountDistinct:
			return &tds.ValueExpr{Kind: tds.ValLit, Lit: int64(0)}
		case tds.AggMin, tds.AggMax:
			if len(v.Args) > 0 && v.Args[0].Kind != tds.ValCol || len(v.Args) > 0 && v.Args[0].Col != "*" {
				return stand(v.Args[0])
			}
			return &tds.ValueExpr{Kind: tds.ValLit, Lit: ""}
		case tds.AggStringAgg:
			return &tds.ValueExpr{Kind: tds.ValLit, Lit: ""}
		default:
			return &tds.ValueExpr{Kind: tds.ValLit, Lit: float64(0)}
		}
	}
	return exprType(stand(ve), cols, idx)
}

type aggregated struct {
	row   []any
	group [][]any
}

func aggregate(cols []catalog.Column, idx map[string]int, rows [][]any, q *tds.Query, env *Env) (tds.Rows, error) {
	outCols, err := aggOutCols(cols, idx, q.Select)
	if err != nil {
		return nil, err
	}

	universe := make(map[string]bool, len(q.GroupBy))
	for _, g := range q.GroupBy {
		universe[g] = true
	}
	sets := q.GroupingSets
	if len(sets) == 0 {
		sets = [][]string{q.GroupBy}
	}

	var rowsOut []aggregated
	for _, set := range sets {
		setRows, err := groupOneSet(idx, rows, q.Select, set, universe, q.GroupByCaseSensitive, env)
		if err != nil {
			return nil, err
		}
		rowsOut = append(rowsOut, setRows...)
	}

	outIdx := indexCols(outCols)

	// HAVING is evaluated in the group context: aggregate calls compute over the group, alias and
	// grouped-column references resolve against the aggregated output row.
	if q.Having != nil {
		var kept []aggregated
		for _, ar := range rowsOut {
			ok, err := evalAggExpr(idx, ar.group, outIdx, ar.row, q.Having, env)
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
				kv, err := aggOrderKey(idx, outIdx, ar.group, ar.row, o, env)
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
				c, ok := compareCS(keys[perm[a]][j], keys[perm[b]][j], o.CaseSensitive)
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
func aggOrderKey(origIdx, outIdx map[string]int, group [][]any, outRow []any, o tds.OrderItem, env *Env) (any, error) {
	switch {
	case o.Expr != nil:
		return evalAggValue(origIdx, group, outIdx, outRow, o.Expr, env)
	case o.Ordinal > 0:
		if o.Ordinal <= len(outRow) {
			return outRow[o.Ordinal-1], nil
		}
		return nil, nil
	default:
		if i, ok := resolveCol(outIdx, o.Column); ok {
			return outRow[i], nil
		}
		// A grouped column the select list leaves out still orders the groups; its value is shared by the group.
		if i, ok := resolveCol(origIdx, o.Column); ok && len(group) > 0 {
			return group[0][i], nil
		}
		return nil, nil
	}
}

// aggByName is the aggregate dispatch table; its keys are the wired set AggregateNames reports.
var aggByName = map[string]tds.AggFunc{
	"COUNT":                 tds.AggCount,
	"SUM":                   tds.AggSum,
	"AVG":                   tds.AggAvg,
	"MIN":                   tds.AggMin,
	"MAX":                   tds.AggMax,
	"COUNT_BIG":             tds.AggCountBig,
	"STDEV":                 tds.AggStdev,
	"STDEVP":                tds.AggStdevp,
	"VAR":                   tds.AggVar,
	"VARP":                  tds.AggVarp,
	"STRING_AGG":            tds.AggStringAgg,
	"CHECKSUM_AGG":          tds.AggChecksumAgg,
	"APPROX_COUNT_DISTINCT": tds.AggApproxCountDistinct,
}

func aggFuncFromName(name string) tds.AggFunc {
	if fn, ok := aggByName[strings.ToUpper(name)]; ok {
		return fn
	}
	return tds.AggNone
}

// AggregateNames returns every wired aggregate name, upper-cased and sorted.
func AggregateNames() []string {
	names := make([]string, 0, len(aggByName))
	for name := range aggByName {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
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

// aggSep is the STRING_AGG separator: the literal second argument, or "" if absent.
func aggSep(args []*tds.ValueExpr) string {
	if len(args) >= 2 && args[1].Kind == tds.ValLit {
		if s, ok := args[1].Lit.(string); ok {
			return s
		}
	}
	return ""
}

// evalAggValue evaluates a value expression in the GROUP context: aggregate calls (COUNT/SUM/…) compute
// over the group's rows via origIdx; everything else evaluates against the aggregated output row.
func evalAggValue(origIdx map[string]int, group [][]any, outIdx map[string]int, outRow []any, ve *tds.ValueExpr, env *Env) (any, error) {
	bound, err := bindAggregates(ve, origIdx, group, env)
	if err != nil {
		return nil, err
	}
	if bound.Kind == tds.ValLit {
		return bound.Lit, nil
	}
	return evalValue(outIdx, outRow, bound, env)
}

func evalAggExpr(origIdx map[string]int, group [][]any, outIdx map[string]int, outRow []any, e *tds.Expr, env *Env) (bool, error) {
	switch {
	case e == nil:
		return true, nil
	case e.Const != nil:
		return *e.Const, nil
	case e.Pred != nil:
		return evalAggPred(origIdx, group, outIdx, outRow, e.Pred, env)
	case e.Not != nil:
		v, err := evalAggExpr(origIdx, group, outIdx, outRow, e.Not, env)
		return !v, err
	case len(e.And) > 0:
		for _, c := range e.And {
			v, err := evalAggExpr(origIdx, group, outIdx, outRow, c, env)
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
			v, err := evalAggExpr(origIdx, group, outIdx, outRow, c, env)
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

func evalAggPred(origIdx map[string]int, group [][]any, outIdx map[string]int, outRow []any, p *tds.Predicate, env *Env) (bool, error) {
	var v any
	if p.LeftExpr != nil {
		lv, err := evalAggValue(origIdx, group, outIdx, outRow, p.LeftExpr, env)
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
			if c, ok := compareCS(v, item, p.CaseSensitive); ok && c == 0 {
				return true, nil
			}
		}
		return false, nil
	case tds.OpLike:
		pat, _ := p.Value.(string)
		return likeMatch(fmt.Sprintf("%v", v), pat, p.CaseSensitive), nil
	default:
		rhs := p.Value
		switch r := rhs.(type) {
		case *tds.ValueExpr:
			rv, err := evalAggValue(origIdx, group, outIdx, outRow, r, env)
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
		c, ok := compareCS(v, rhs, p.CaseSensitive)
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
		if _, _, ok := groupingFn(it); ok {
			if name == "" {
				name = "grouping"
			}
			out = append(out, catalog.Column{Name: name, Type: types.Type{Kind: types.Int64}})
			continue
		}
		var typ types.Type
		if it.Agg == tds.AggNone && it.Expr != nil {
			out = append(out, catalog.Column{Name: name, Type: aggExprType(it.Expr, cols, idx)})
			continue
		}
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
		case tds.AggCount, tds.AggCountBig:
			typ = types.Type{Kind: types.Int64}
			if name == "" {
				name = "count"
			}
		case tds.AggSum, tds.AggAvg, tds.AggStdev, tds.AggStdevp, tds.AggVar, tds.AggVarp:
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
		case tds.AggStringAgg:
			typ = types.Type{Kind: types.String, MaxLen: 4000}
			if name == "" {
				name = "agg"
			}
		case tds.AggChecksumAgg, tds.AggApproxCountDistinct:
			typ = types.Type{Kind: types.Int64}
			if name == "" {
				name = "agg"
			}
		}
		out = append(out, catalog.Column{Name: name, Type: typ})
	}
	return out, nil
}

// groupOneSet aggregates rows over one grouping set; universe columns not in the set roll up to NULL.
func groupOneSet(idx map[string]int, rows [][]any, sel []tds.SelectItem, set []string, universe map[string]bool, caseSensitive []string, env *Env) ([]aggregated, error) {
	setIdx := make([]int, 0, len(set))
	setCols := make(map[string]bool, len(set))
	exact := make([]bool, len(set))
	for j, g := range set {
		i, ok := idx[g]
		if !ok {
			return nil, fmt.Errorf("exec: unknown column %q in GROUP BY", g)
		}
		setIdx = append(setIdx, i)
		setCols[g] = true
		for _, cs := range caseSensitive {
			if cs == g {
				exact[j] = true
			}
		}
	}
	var order []string
	groups := map[string][][]any{}
	for _, row := range rows {
		key := "__all__"
		if len(setIdx) > 0 {
			parts := make([]any, len(setIdx))
			for j, gi := range setIdx {
				parts[j] = row[gi]
			}
			key = fold.RowKey(parts, exact)
		}
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], row)
	}
	if len(setIdx) == 0 && len(order) == 0 {
		order = []string{"__all__"}
	}
	out := make([]aggregated, 0, len(order))
	for _, k := range order {
		row, err := aggRowSet(idx, sel, groups[k], setCols, universe, env)
		if err != nil {
			return nil, err
		}
		out = append(out, aggregated{row: row, group: groups[k]})
	}
	return out, nil
}

func aggRowSet(idx map[string]int, sel []tds.SelectItem, rows [][]any, setCols, universe map[string]bool, env *Env) ([]any, error) {
	out := make([]any, len(sel))
	for j, it := range sel {
		if gcols, isID, ok := groupingFn(it); ok {
			out[j] = groupingValue(gcols, isID, setCols)
			continue
		}
		if it.Agg == tds.AggNone && it.Expr != nil {
			v, err := groupExprValue(idx, rows, it.Expr, setCols, universe, env)
			if err != nil {
				return nil, err
			}
			out[j] = v
			continue
		}
		if it.Agg == tds.AggNone {
			if i, ok := resolveCol(idx, it.Column); ok && len(rows) > 0 {
				if universe[it.Column] && !setCols[it.Column] {
					out[j] = nil // rolled up in this grouping set
				} else {
					out[j] = rows[0][i]
				}
			}
			continue
		}
		v, err := computeAgg(it.Agg, it.Arg, it.Sep, idx, rows, it.AggDist)
		if err != nil {
			return nil, err
		}
		out[j] = v
	}
	return out, nil
}

// groupingFn reports whether a select item is GROUPING(col) / GROUPING_ID(cols), with its column args.
func groupingFn(it tds.SelectItem) ([]string, bool, bool) {
	ve := it.Expr
	if it.Agg != tds.AggNone || ve == nil || ve.Kind != tds.ValFunc {
		return nil, false, false
	}
	name := strings.ToUpper(ve.Func)
	if name != "GROUPING" && name != "GROUPING_ID" {
		return nil, false, false
	}
	cols := make([]string, 0, len(ve.Args))
	for _, a := range ve.Args {
		if a.Kind == tds.ValCol {
			cols = append(cols, a.Col)
		}
	}
	return cols, name == "GROUPING_ID", true
}

// groupingValue is GROUPING(col) (1 when col is rolled up) or GROUPING_ID(c1,…) (a bitmask, c1 the MSB).
func groupingValue(gcols []string, isID bool, setCols map[string]bool) any {
	if !isID {
		if len(gcols) == 0 || setCols[gcols[0]] {
			return int64(0)
		}
		return int64(1)
	}
	var mask int64
	for _, c := range gcols {
		mask <<= 1
		if !setCols[c] {
			mask |= 1
		}
	}
	return mask
}

// computeAgg evaluates one aggregate function over a group's rows. arg is the column name ("*" or
// "" for COUNT-all); idx is the pre-aggregation column index.
func computeAgg(fn tds.AggFunc, arg, sep string, idx map[string]int, rows [][]any, distinct bool) (any, error) {
	if distinct && arg != "" && arg != "*" {
		if i, ok := resolveCol(idx, arg); ok {
			seen := make(map[string]bool, len(rows))
			deduped := make([][]any, 0, len(rows))
			for _, r := range rows {
				if r[i] == nil {
					continue
				}
				k := fmt.Sprintf("%v", fold.Value(r[i]))
				if seen[k] {
					continue
				}
				seen[k] = true
				deduped = append(deduped, r)
			}
			rows = deduped
		}
	}
	switch fn {
	case tds.AggChecksumAgg:
		i, ok := resolveCol(idx, arg)
		if !ok {
			return nil, fmt.Errorf("exec: unknown column %q in CHECKSUM_AGG", arg)
		}
		var acc int64 // XOR is order-independent, matching CHECKSUM_AGG semantics
		for _, r := range rows {
			if n, ok := toInt(r[i]); ok {
				acc ^= n
			}
		}
		return acc, nil
	case tds.AggApproxCountDistinct:
		i, ok := resolveCol(idx, arg)
		if !ok {
			return nil, fmt.Errorf("exec: unknown column %q in APPROX_COUNT_DISTINCT", arg)
		}
		seen := map[string]bool{}
		for _, r := range rows {
			if r[i] != nil {
				seen[fmt.Sprintf("%v", fold.Value(r[i]))] = true
			}
		}
		return int64(len(seen)), nil
	case tds.AggStringAgg:
		i, ok := resolveCol(idx, arg)
		if !ok {
			return nil, fmt.Errorf("exec: unknown column %q in STRING_AGG", arg)
		}
		var parts []string
		for _, r := range rows {
			if r[i] != nil {
				parts = append(parts, fmt.Sprintf("%v", r[i]))
			}
		}
		if len(parts) == 0 {
			return nil, nil
		}
		return strings.Join(parts, sep), nil
	case tds.AggCount, tds.AggCountBig:
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
	case tds.AggStdev, tds.AggStdevp, tds.AggVar, tds.AggVarp:
		i, ok := resolveCol(idx, arg)
		if !ok {
			return nil, fmt.Errorf("exec: unknown column %q in aggregate", arg)
		}
		var vals []float64
		for _, r := range rows {
			if r[i] != nil {
				vals = append(vals, toFloat(r[i]))
			}
		}
		sample := fn == tds.AggStdev || fn == tds.AggVar
		if len(vals) == 0 || (sample && len(vals) < 2) {
			return nil, nil
		}
		var mean float64
		for _, v := range vals {
			mean += v
		}
		mean /= float64(len(vals))
		var ss float64
		for _, v := range vals {
			d := v - mean
			ss += d * d
		}
		denom := float64(len(vals))
		if sample {
			denom = float64(len(vals) - 1)
		}
		variance := ss / denom
		if fn == tds.AggVar || fn == tds.AggVarp {
			return variance, nil
		}
		return math.Sqrt(variance), nil
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

// groupExprValue evaluates a select expression for one group: aggregates over the group, column
// references against its first row, with columns rolled up in this grouping set read as NULL.
func groupExprValue(idx map[string]int, group [][]any, ve *tds.ValueExpr, setCols, universe map[string]bool, env *Env) (any, error) {
	bound, err := bindAggregates(ve, idx, group, env)
	if err != nil {
		return nil, err
	}
	if bound.Kind == tds.ValLit {
		return bound.Lit, nil
	}
	width := 0
	for _, i := range idx {
		if i+1 > width {
			width = i + 1
		}
	}
	row := make([]any, width)
	if len(group) > 0 {
		copy(row, group[0])
	}
	for c := range universe {
		if !setCols[c] {
			if i, ok := idx[c]; ok {
				row[i] = nil
			}
		}
	}
	return evalValue(idx, row, bound, env)
}

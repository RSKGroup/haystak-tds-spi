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
)

// Apply runs a logical query against in-memory rows. It is the core thin-path evaluator,
// used for the catalog projection and any Scanner-style backend.
func Apply(cols []catalog.Column, data [][]any, q *tds.Query) (tds.Rows, error) {
	return ApplyWith(cols, data, q, nil)
}

func ApplyWith(cols []catalog.Column, data [][]any, q *tds.Query, env *Env) (tds.Rows, error) {
	idx := indexCols(cols)

	var filtered [][]any
	for _, row := range data {
		ok, err := evalExpr(idx, row, q.Where, env)
		if err != nil {
			return nil, err
		}
		if ok {
			filtered = append(filtered, row)
		}
	}

	if isAggregate(q) {
		mCols, mRows, mSel, err := materializeAggArgs(cols, idx, filtered, q.Select, env)
		if err != nil {
			return nil, err
		}
		q2 := *q
		q2.Select = mSel
		return aggregate(mCols, indexCols(mCols), mRows, &q2, env)
	}

	wCols, wRows, wSel, err := materializeWindows(cols, idx, filtered, q.Select, env)
	if err != nil {
		return nil, err
	}
	cols, filtered, idx = wCols, wRows, indexCols(wCols)

	mCols, mRows, mSel, err := materializeExprs(cols, idx, filtered, wSel, env)
	if err != nil {
		return nil, err
	}
	cols, filtered, idx = mCols, mRows, indexCols(mCols)

	if len(q.OrderBy) > 0 {
		order := q.OrderBy
		if hasOrderExpr(order) {
			var err error
			cols, filtered, idx, order, err = materializeOrderExprs(cols, idx, filtered, order, env)
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

// IsAggregate reports whether q is a GROUP BY / aggregate-function query.
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

// Env carries query-time context the pure evaluator can't infer: the correlated-subquery runner plus
// the catalog id→name resolvers behind OBJECT_NAME/DB_NAME. A nil *Env disables all three.
type Env struct {
	Sub        SubFn
	ObjectName func(int64) (string, bool)
	DBName     func(int64) (string, bool)
	Table      func(int64) (catalog.Table, bool) // table by object id (COL_*/OBJECTPROPERTY)
	ObjectKind func(int64) (string, bool)        // "U"/"V"/"P"/"FN"/"TR" by object id
	RoutineDef func(int64) (string, bool)        // CREATE text by object id (OBJECT_DEFINITION)
	CurrentDB  string
}

func (e *Env) subFn() SubFn {
	if e == nil {
		return nil
	}
	return e.Sub
}

func evalExpr(idx map[string]int, row []any, e *tds.Expr, env *Env) (bool, error) {
	switch {
	case e == nil:
		return true, nil
	case e.Const != nil:
		return *e.Const, nil
	case e.Pred != nil:
		return evalPred(idx, row, e.Pred, env)
	case e.Not != nil:
		v, err := evalExpr(idx, row, e.Not, env)
		return !v, err
	case len(e.And) > 0:
		for _, c := range e.And {
			v, err := evalExpr(idx, row, c, env)
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
			v, err := evalExpr(idx, row, c, env)
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

func evalPred(idx map[string]int, row []any, p *tds.Predicate, env *Env) (bool, error) {
	sub := env.subFn()
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
		lv, err := evalValue(idx, row, p.LeftExpr, env)
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
			rv, err := evalValue(idx, row, r, env)
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

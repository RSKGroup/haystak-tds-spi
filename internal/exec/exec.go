// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package exec

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/functions"
	"github.com/RSKGroup/haystak-tds-spi/internal/fold"
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

// Apply runs a logical query against in-memory rows. It is the core thin-path evaluator,
// used for the catalog projection and any Scanner-style backend.
func Apply(cols []catalog.Column, data [][]any, q *tds.Query) (tds.Rows, error) {
	return ApplyWith(cols, data, q, nil)
}

func ApplyWith(cols []catalog.Column, data [][]any, q *tds.Query, env *Env) (tds.Rows, error) {
	idx := indexCols(cols)
	if env != nil {
		e := *env
		e.colType = func(name string) (types.Type, bool) {
			if i, ok := resolveCol(idx, name); ok {
				return cols[i].Type, true
			}
			return types.Type{}, false
		}
		env = &e
	}

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
		order := resolveOrderAliases(q.OrderBy, mSel)
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
		k := fold.RowKey(row, nil)
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
		if strings.HasSuffix(it.Column, ".*") {
			prefix := strings.ToLower(it.Column[:len(it.Column)-1])
			// Rows from a single source carry bare names, so alias.* is each of its own columns.
			single := !anyQualified(cols)
			for i, c := range cols {
				if strings.HasPrefix(strings.ToLower(c.Name), prefix) || (single && !strings.HasPrefix(c.Name, "__")) {
					cc := c
					if dot := strings.LastIndex(cc.Name, "."); dot >= 0 {
						cc.Name = cc.Name[dot+1:]
					}
					outCols = append(outCols, cc)
					proj = append(proj, i)
				}
			}
			continue
		}
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

func anyQualified(cols []catalog.Column) bool {
	for _, c := range cols {
		if strings.Contains(c.Name, ".") {
			return true
		}
	}
	return false
}

func resolveOrderAliases(order []tds.OrderItem, sel []tds.SelectItem) []tds.OrderItem {
	alias := map[string]string{}
	for _, it := range sel {
		if it.Alias != "" && it.Column != "" {
			alias[strings.ToLower(it.Alias)] = it.Column
		}
	}
	out := make([]tds.OrderItem, len(order))
	for i, o := range order {
		out[i] = o
		if col, ok := alias[strings.ToLower(o.Column)]; ok {
			out[i].Column = col
		}
	}
	return out
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
	SPID       int            // @@SPID; 0 = no session
	Error      *tds.ErrorInfo // the caught error inside a CATCH block (nil otherwise), for ERROR_*

	colType func(string) (types.Type, bool) // the declared type of a source column, for CAST's default text
}

func (e *Env) subFn() SubFn {
	if e == nil {
		return nil
	}
	return e.Sub
}

// evalExpr reports whether e is TRUE for row; FALSE and UNKNOWN both filter it out.
func evalExpr(idx map[string]int, row []any, e *tds.Expr, env *Env) (bool, error) {
	t, err := evalTruth(e, func(p *tds.Predicate) (truth, error) { return evalPred(idx, row, p, env) })
	return t == truthTrue, err
}

func evalPred(idx map[string]int, row []any, p *tds.Predicate, env *Env) (truth, error) {
	sub := env.subFn()
	if p.Op == tds.OpExists && p.Sub != nil {
		if sub == nil {
			return truthFalse, nil
		}
		rows, err := sub(row, idx, p.Sub)
		if err != nil {
			return truthFalse, err
		}
		return truthOf(len(rows) > 0), nil
	}
	var v any
	if p.LeftExpr != nil {
		lv, err := evalValue(idx, row, p.LeftExpr, env)
		if err != nil {
			return truthFalse, err
		}
		v = lv
	} else {
		i, ok := resolveCol(idx, p.Column)
		if !ok {
			return truthFalse, fmt.Errorf("exec: unknown column %q in WHERE", p.Column)
		}
		v = row[i]
	}
	if p.Op == tds.OpIn && p.Sub != nil {
		if sub == nil {
			return truthFalse, nil
		}
		rows, err := sub(row, idx, p.Sub)
		if err != nil {
			return truthFalse, err
		}
		vals := make([]any, 0, len(rows))
		for _, r := range rows {
			if len(r) > 0 {
				vals = append(vals, r[0])
			}
		}
		return inTruth(v, vals, p.CaseSensitive), nil
	}
	switch p.Op {
	case tds.OpIsNull, tds.OpIsNotNull:
		return nullTruth(p.Op, v), nil
	case tds.OpIn:
		list, _ := p.Value.([]any)
		return inTruth(v, list, p.CaseSensitive), nil
	case tds.OpLike:
		pat := p.Value
		if ve, ok := pat.(*tds.ValueExpr); ok {
			pv, err := evalValue(idx, row, ve, env)
			if err != nil {
				return truthFalse, err
			}
			pat = pv
		}
		return likeTruth(v, pat, p.CaseSensitive, p.Escape), nil
	default:
		rhs := p.Value
		switch r := rhs.(type) {
		case *tds.ValueExpr:
			rv, err := evalValue(idx, row, r, env)
			if err != nil {
				return truthFalse, err
			}
			rhs = rv
		case tds.ColRef:
			j, ok := resolveCol(idx, r.Name)
			if !ok {
				return truthFalse, fmt.Errorf("exec: unknown column %q in WHERE", r.Name)
			}
			rhs = row[j]
		}
		return cmpTruth(p.Op, v, rhs, p.CaseSensitive), nil
	}
}

// likeMatch is T-SQL's LIKE (% _ [set] [a-f] [^set], esc before a character makes it literal; 0 for none).
func likeMatch(s, pattern string, exact bool, esc rune) bool {
	if !exact {
		s, pattern = fold.Key(s), fold.Key(pattern)
		if esc >= 'A' && esc <= 'Z' {
			esc += 'a' - 'A'
		}
	}
	str, pat := []rune(s), []rune(pattern)
	si, pi, starP, starS := 0, 0, -1, 0
	for si < len(str) {
		matched := false
		if pi < len(pat) {
			switch c := pat[pi]; {
			case esc != 0 && c == esc:
				if pi+1 >= len(pat) {
					return false // an escape with nothing after it makes the pattern invalid: FALSE
				}
				matched, pi = pat[pi+1] == str[si], pi+2
			case c == '%':
				starP, starS = pi, si
				pi++
				continue
			case c == '_':
				matched, pi = true, pi+1
			case c == '[':
				if in, n := likeClass(pat[pi:], str[si], esc); n > 0 {
					matched, pi = in, pi+n
				} else {
					matched, pi = str[si] == '[', pi+1
				}
			default:
				matched, pi = c == str[si], pi+1
			}
		}
		if matched {
			si++
			continue
		}
		if starP < 0 {
			return false
		}
		starS++
		si, pi = starS, starP+1
	}
	for pi < len(pat) && pat[pi] == '%' {
		pi++
	}
	return pi == len(pat)
}

// likeClass matches c against the [set] opening p (n its length, 0 if unclosed); esc makes ^ - ] plain members.
func likeClass(p []rune, c rune, esc rune) (in bool, n int) {
	i := 1
	neg := i < len(p) && p[i] == '^'
	if neg {
		i++
	}
	start := i
	var set []rune
	var plain []bool
	for i < len(p) && (p[i] != ']' || i == start) {
		if esc != 0 && p[i] == esc && i+1 < len(p) {
			set, plain = append(set, p[i+1]), append(plain, true)
			i += 2
			continue
		}
		set, plain = append(set, p[i]), append(plain, false)
		i++
	}
	if i >= len(p) {
		return false, 0
	}
	for k := 0; k < len(set); k++ {
		if k+2 < len(set) && set[k+1] == '-' && !plain[k+1] {
			in = in || (c >= set[k] && c <= set[k+2])
			k += 2
			continue
		}
		in = in || set[k] == c
	}
	return in != neg, i + 1
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
		c, ok := compareCS(a[i], b[i], o.CaseSensitive)
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

func compare(a, b any) (int, bool) { return compareCS(a, b, false) }

// compareCS orders two values; strings compare ASCII case-insensitively unless exact.
func compareCS(a, b any, exact bool) (int, bool) {
	if a == nil || b == nil {
		return 0, false
	}
	if _, ok := a.(bool); ok || isBoolType(b) { // bit compared to a number/string coerces both to bit
		ab, aok := boolish(a)
		bb, bok := boolish(b)
		if aok && bok {
			return cmpBool(ab, bb), true
		}
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
			if exact {
				return strings.Compare(fold.Pad(av), fold.Pad(bv)), true
			}
			return fold.Compare(av, bv), true
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
	// SQL Server converts the lower-precedence operand; it does not compare the two as text.
	// Formatting both with %v and comparing the strings returned a CONFIDENT wrong answer:
	// "99.50" sorts after "100", so `amount > 100` matched a row holding 99.50.
	if c, ok := compareCoerced(a, b, exact); ok {
		return c, true
	}
	// Unorderable. The WHERE path reads this as UNKNOWN and drops the row, and ORDER BY treats it
	// as a tie - the same handling NULL already gets, and far better than inventing an order.
	return 0, false
}

// compareCoerced applies the implicit conversions SQL Server would, and reports whether the pair
// could be ordered at all.
func compareCoerced(a, b any, exact bool) (int, bool) {
	// number vs text: the text converts to the number, and fails the comparison if it cannot.
	if af, ok := toFloatOk(a); ok {
		if s, ok := b.(string); ok {
			if bf, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
				return cmpFloat(af, bf), true
			}
			return 0, false
		}
	}
	if bf, ok := toFloatOk(b); ok {
		if s, ok := a.(string); ok {
			if af, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
				return cmpFloat(af, bf), true
			}
			return 0, false
		}
	}
	// datetime vs text: same direction, the text converts.
	if at, ok := a.(time.Time); ok {
		if s, ok := b.(string); ok {
			if bt, ok := parseCompareTime(s); ok {
				return cmpTime(at, bt), true
			}
			return 0, false
		}
	}
	if bt, ok := b.(time.Time); ok {
		if s, ok := a.(string); ok {
			if at, ok := parseCompareTime(s); ok {
				return cmpTime(at, bt), true
			}
			return 0, false
		}
	}
	// binary vs text compares the bytes, which is what the storage holds either way.
	if ab, ok := a.([]byte); ok {
		if s, ok := b.(string); ok {
			return bytes.Compare(ab, []byte(s)), true
		}
	}
	if bb, ok := b.([]byte); ok {
		if s, ok := a.(string); ok {
			return bytes.Compare([]byte(s), bb), true
		}
	}
	return 0, false
}

func cmpTime(a, b time.Time) int {
	switch {
	case a.Before(b):
		return -1
	case a.After(b):
		return 1
	default:
		return 0
	}
}

// parseCompareTime reads text as SQL Server's implicit conversion does (mdy), so '01/01/2025' compares as a date.
func parseCompareTime(s string) (time.Time, bool) { return functions.ParseDate(s, functions.NoStyle) }

func isBoolType(v any) bool { _, ok := v.(bool); return ok }

func boolish(v any) (bool, bool) {
	switch x := v.(type) {
	case bool:
		return x, true
	case int64:
		return x != 0, true
	case float64:
		return x != 0, true
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "1", "true":
			return true, true
		case "0", "false":
			return false, true
		}
	}
	return false, false
}

func cmpBool(a, b bool) int {
	switch {
	case a == b:
		return 0
	case !a:
		return -1
	default:
		return 1
	}
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

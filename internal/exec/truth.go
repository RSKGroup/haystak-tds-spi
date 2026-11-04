// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package exec

import (
	"github.com/RSKGroup/haystak-tds-spi/internal/fold"
	"github.com/RSKGroup/haystak-tds-spi/tds"
)

// truth is a SQL truth value. A comparison with NULL is unknown, NOT keeps it unknown, and only true keeps a row.
type truth int8

const (
	truthFalse truth = iota
	truthTrue
	truthUnknown
)

func truthOf(b bool) truth {
	if b {
		return truthTrue
	}
	return truthFalse
}

func (t truth) not() truth {
	switch t {
	case truthTrue:
		return truthFalse
	case truthFalse:
		return truthTrue
	}
	return truthUnknown
}

// evalTruth folds a predicate tree: AND is false on any false, OR true on any true, else an unknown operand wins.
func evalTruth(e *tds.Expr, leaf func(*tds.Predicate) (truth, error)) (truth, error) {
	switch {
	case e == nil:
		return truthTrue, nil
	case e.Const != nil:
		return truthOf(*e.Const), nil
	case e.Pred != nil:
		return leaf(e.Pred)
	case e.Not != nil:
		t, err := evalTruth(e.Not, leaf)
		return t.not(), err
	case len(e.And) > 0:
		out := truthTrue
		for _, c := range e.And {
			t, err := evalTruth(c, leaf)
			if err != nil || t == truthFalse {
				return truthFalse, err
			}
			if t == truthUnknown {
				out = truthUnknown
			}
		}
		return out, nil
	case len(e.Or) > 0:
		out := truthFalse
		for _, c := range e.Or {
			t, err := evalTruth(c, leaf)
			if err != nil {
				return truthFalse, err
			}
			if t == truthTrue {
				return truthTrue, nil
			}
			if t == truthUnknown {
				out = truthUnknown
			}
		}
		return out, nil
	}
	return truthTrue, nil
}

// cmpTruth compares two operands; NULL, or a pair SQL Server could not convert, is unknown.
func cmpTruth(op tds.Op, v, rhs any, exact bool) truth {
	c, ok := compareCS(v, rhs, exact)
	if !ok {
		return truthUnknown
	}
	return truthOf(satisfies(op, c))
}

// inTruth is x IN (list): true on a match, unknown when x or an unmatched member is NULL, false otherwise.
func inTruth(v any, list []any, exact bool) truth {
	if len(list) == 0 {
		return truthFalse
	}
	out := truthFalse
	for _, item := range list {
		switch t := cmpTruth(tds.OpEq, v, item, exact); t {
		case truthTrue:
			return truthTrue
		case truthUnknown:
			out = truthUnknown
		}
	}
	return out
}

// likeTruth matches v against pat; trailing blanks of v are ignored, those of the pattern are significant.
func likeTruth(v, pat any, exact bool, esc rune) truth {
	if v == nil || pat == nil {
		return truthUnknown
	}
	s, p := toStr(v), toStr(pat)
	return truthOf(likeMatch(s, p, exact, esc) || (fold.Pad(s) != s && likeMatch(fold.Pad(s), p, exact, esc)))
}

func nullTruth(op tds.Op, v any) truth {
	if op == tds.OpIsNull {
		return truthOf(v == nil)
	}
	return truthOf(v != nil)
}

// RejectsNulls reports whether e is never TRUE with all of cols NULL, so an outer join's NULL side may take it.
func RejectsNulls(e *tds.Expr, cols []string) bool {
	idx := make(map[string]int, len(cols))
	for _, c := range cols {
		idx[c] = 0
	}
	ok, err := evalExpr(idx, []any{nil}, e, nil)
	return err == nil && !ok
}

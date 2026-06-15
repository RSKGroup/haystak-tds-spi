// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package control

import (
	"context"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/routines"
)

// Deferred (documented): DECLARE @t TABLE needs a Scanner+Writer overlay backend that presents the table
// variable to the engine through the Runner seam -- its own architectural addition. GOTO needs a
// program-counter executor for non-structured jumps and is a discouraged pattern.

// trySelectAssign handles `SELECT @a = expr [, @b = expr2] [FROM …]`: it runs the right-hand expressions
// and binds the variables to the last row's values. Returns handled=false for an ordinary SELECT.
func trySelectAssign(ctx context.Context, sql string, sc *scope, run routines.Runner) (bool, error) {
	s := strings.TrimSpace(sql)
	if !hasPrefixFold(s, "SELECT ") {
		return false, nil
	}
	list, tail := splitSelectFrom(strings.TrimSpace(s[len("SELECT "):]))
	type asg struct{ name, expr string }
	var asgs []asg
	for _, it := range splitTopCommas(list) {
		it = strings.TrimSpace(it)
		eq := topEq(it)
		if !strings.HasPrefix(it, "@") || eq < 0 {
			return false, nil // not an assignment select
		}
		asgs = append(asgs, asg{
			name: strings.ToLower(strings.TrimPrefix(strings.TrimSpace(it[:eq]), "@")),
			expr: strings.TrimSpace(it[eq+1:]),
		})
	}
	if len(asgs) == 0 {
		return false, nil
	}
	exprs := make([]string, len(asgs))
	for i, a := range asgs {
		exprs[i] = a.expr
	}
	q := "SELECT " + strings.Join(exprs, ", ")
	if tail != "" {
		q += " " + tail
	}
	rows, err := run.Exec(ctx, subst(q, sc))
	if err != nil || rows == nil {
		return true, err
	}
	defer rows.Close()
	var last []any
	for rows.Next() {
		v, err := rows.Values()
		if err != nil {
			return true, err
		}
		last = v
	}
	for i, a := range asgs {
		if i < len(last) {
			sc.vars[a.name] = litOf(last[i])
		}
	}
	return true, nil
}

// splitSelectFrom splits a select body into its select list and the FROM-onward tail at paren depth 0.
func splitSelectFrom(s string) (list, tail string) {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'':
			i = skipStr(s, i)
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 && (s[i] == 'F' || s[i] == 'f') && i > 0 && isSpaceByte(s[i-1]) &&
				hasPrefixFold(s[i:], "FROM") && (i+4 >= len(s) || isSpaceByte(s[i+4])) {
				return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i:])
			}
		}
	}
	return strings.TrimSpace(s), ""
}

func hasPrefixFold(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
}

func isSpaceByte(c byte) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' }

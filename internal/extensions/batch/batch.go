// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package batch resolves T-SQL batch variables — DECLARE/SET @var — before the core engine parses a
// statement. It binds each initialized variable and substitutes @name with its literal in the remaining
// statements, then drops the DECLARE/SET lines. Substitution is string-literal aware (@x inside '…' or
// [bracketed] names is never touched), so the core lexer/parser never sees a `@` and stays frozen.
package batch

import (
	"fmt"
	"strings"
)

// Resolve returns the runnable SQL with DECLARE/SET variables bound, substituted, and removed. A batch
// with no `@` is returned unchanged.
func Resolve(sql string) (string, error) {
	if !strings.ContainsRune(sql, '@') {
		return sql, nil
	}
	vars := map[string]string{}
	var out []string
	for _, st := range splitStatements(sql) {
		t := strings.TrimSpace(st)
		if t == "" {
			continue
		}
		switch strings.ToUpper(firstWord(t)) {
		case "DECLARE":
			if err := bindDeclare(t, vars); err != nil {
				return "", err
			}
		case "SET":
			done, err := bindSet(t, vars)
			if err != nil {
				return "", err
			}
			if !done { // SET NOCOUNT ON etc. — not a variable assignment, keep it
				out = append(out, substitute(t, vars))
			}
		default:
			out = append(out, substitute(t, vars))
		}
	}
	return strings.Join(out, ";\n"), nil
}

// splitStatements splits sql at top-level ';' (ignoring ';' inside '…' strings or [bracketed] names).
func splitStatements(sql string) []string {
	var stmts []string
	start := 0
	for i := 0; i < len(sql); {
		switch sql[i] {
		case '\'':
			i = skipString(sql, i)
		case '[':
			i = skipBracket(sql, i)
		case ';':
			stmts = append(stmts, sql[start:i])
			i++
			start = i
		default:
			i++
		}
	}
	if start < len(sql) {
		stmts = append(stmts, sql[start:])
	}
	return stmts
}

// substitute replaces every @name with its bound value, skipping string and [bracket] literals.
func substitute(s string, vars map[string]string) string {
	if len(vars) == 0 {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		switch {
		case s[i] == '\'':
			j := skipString(s, i)
			b.WriteString(s[i:j])
			i = j
		case s[i] == '[':
			j := skipBracket(s, i)
			b.WriteString(s[i:j])
			i = j
		case s[i] == '@':
			j := i + 1
			for j < len(s) && isNamePart(s[j]) {
				j++
			}
			if v, ok := vars[strings.ToLower(s[i+1:j])]; ok {
				b.WriteString(v)
			} else {
				b.WriteString(s[i:j]) // unknown @var — leave untouched
			}
			i = j
		default:
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

// bindDeclare parses `DECLARE @a <type> [= <lit>] [, @b …]` and records each initialized variable.
func bindDeclare(stmt string, vars map[string]string) error {
	for _, part := range splitTopCommas(strings.TrimSpace(stmt[len("DECLARE"):])) {
		p := strings.TrimSpace(part)
		name, rest, ok := varName(p)
		if !ok {
			return fmt.Errorf("batch: DECLARE expects @name, got %q", p)
		}
		if eq := topIndex(rest, '='); eq >= 0 { // `<type> = <value>` — bind to the value; no '=' ⇒ NULL, unbound
			vars[name] = strings.TrimSpace(rest[eq+1:])
		}
	}
	return nil
}

// bindSet handles `SET @v = <value>`; ok is false for a non-variable SET (e.g. SET NOCOUNT ON).
func bindSet(stmt string, vars map[string]string) (bool, error) {
	name, rest, ok := varName(strings.TrimSpace(stmt[len("SET"):]))
	if !ok {
		return false, nil
	}
	eq := topIndex(rest, '=')
	if eq < 0 {
		return false, fmt.Errorf("batch: SET @%s expects '='", name)
	}
	vars[name] = strings.TrimSpace(rest[eq+1:])
	return true, nil
}

// varName reads a leading @name, returning the lower-cased name and the remainder after it.
func varName(s string) (name, rest string, ok bool) {
	if len(s) == 0 || s[0] != '@' {
		return "", s, false
	}
	k := 1
	for k < len(s) && isNamePart(s[k]) {
		k++
	}
	return strings.ToLower(s[1:k]), s[k:], true
}

// splitTopCommas splits on commas at paren-depth 0, outside strings/brackets (so decimal(10,2) is intact).
func splitTopCommas(s string) []string {
	var parts []string
	start, depth := 0, 0
	for i := 0; i < len(s); {
		switch s[i] {
		case '\'':
			i = skipString(s, i)
		case '[':
			i = skipBracket(s, i)
		case '(':
			depth++
			i++
		case ')':
			if depth > 0 {
				depth--
			}
			i++
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
			i++
		default:
			i++
		}
	}
	return append(parts, s[start:])
}

// topIndex finds the first ch at paren-depth 0, outside strings/brackets; -1 if none.
func topIndex(s string, ch byte) int {
	depth := 0
	for i := 0; i < len(s); {
		switch s[i] {
		case '\'':
			i = skipString(s, i)
		case '[':
			i = skipBracket(s, i)
		case '(':
			depth++
			i++
		case ')':
			if depth > 0 {
				depth--
			}
			i++
		case ch:
			if depth == 0 {
				return i
			}
			i++
		default:
			i++
		}
	}
	return -1
}

func skipString(s string, i int) int {
	for i++; i < len(s); i++ {
		if s[i] == '\'' {
			if i+1 < len(s) && s[i+1] == '\'' {
				i++
				continue
			}
			return i + 1
		}
	}
	return i
}

func skipBracket(s string, i int) int {
	for i++; i < len(s); i++ {
		if s[i] == ']' {
			return i + 1
		}
	}
	return i
}

func firstWord(s string) string {
	s = strings.TrimSpace(s)
	j := 0
	for j < len(s) && !isSpace(s[j]) {
		j++
	}
	return s[:j]
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }
func isNamePart(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

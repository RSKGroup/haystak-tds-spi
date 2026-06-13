// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package routines is the shared base for stored database objects — the views and procedures
// packages build on it. It holds the engine seam (Runner) and the DDL-text helpers those packages
// use to parse CREATE/ALTER/DROP. It imports neither the engine nor the feature packages, so there
// is no cycle.
package routines

import (
	"context"
	"strconv"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/tds"
)

// Runner is the engine seam: feature packages parse SQL themselves but execute it through these.
type Runner interface {
	Exec(ctx context.Context, sql string) (tds.Rows, error)       // run a SQL batch (procedure bodies)
	RunQuery(ctx context.Context, q *tds.Query) (tds.Rows, error) // run a parsed query (view expansion)
	CurrentDB(ctx context.Context) string
}

// Arg is one EXEC argument: a value, optionally named (@p = v).
type Arg struct {
	Name string
	Val  any
}

// HasHead reports whether the upper-cased SQL starts with `verb object` (whitespace-tolerant).
func HasHead(up, verb, object string) bool {
	rest := strings.TrimSpace(strings.TrimPrefix(up, verb))
	return rest != up && strings.HasPrefix(rest, object+" ")
}

// AfterDrop returns the object name in a DROP statement (past the object keyword + optional IF EXISTS).
func AfterDrop(s string, objects ...string) string {
	rest := s
	for _, o := range objects {
		if i := IndexFold(rest, o); i >= 0 {
			rest = rest[i+len(o):]
			break
		}
	}
	rest = strings.TrimSpace(rest)
	if IndexFold(rest, "IF EXISTS") == 0 {
		rest = strings.TrimSpace(rest[len("IF EXISTS"):])
	}
	return Unqualify(rest)
}

// Drop removes a stored routine by name (no-op for an empty name).
func Drop(ctx context.Context, store tds.RoutineStore, db, name string) error {
	if name == "" {
		return nil
	}
	return store.DropRoutine(ctx, db, name)
}

// QualifyDB qualifies every unqualified real-table reference in q — its joins, union arms, and nested
// derived tables — with db, so a routine body resolves in the routine's database, not the session's.
func QualifyDB(q *tds.Query, db string) {
	if q == nil || db == "" {
		return
	}
	for a := q; a != nil; a = a.Union {
		if a.Table != "" && a.Database == "" && !isSystemSchema(a.Schema) {
			a.Database = db
		}
		for i := range a.Joins {
			if a.Joins[i].Table != "" && a.Joins[i].Database == "" && !isSystemSchema(a.Joins[i].Schema) {
				a.Joins[i].Database = db
			}
		}
		QualifyDB(a.FromSub, db)
	}
}

func isSystemSchema(s string) bool {
	return strings.EqualFold(s, "INFORMATION_SCHEMA") || strings.EqualFold(s, "sys")
}

// SplitAS splits a definition at its top-level ` AS ` separator (the one after the header).
func SplitAS(s string) (head, body string, ok bool) {
	for i := 0; i+2 <= len(s); i++ {
		if (s[i] == 'a' || s[i] == 'A') && (s[i+1] == 's' || s[i+1] == 'S') {
			before := i == 0 || isWS(s[i-1])
			after := i+2 == len(s) || isWS(s[i+2])
			if before && after {
				return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+2:]), true
			}
		}
	}
	return "", "", false
}

func isWS(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

// Unqualify strips brackets/quotes and any db./schema. prefix, returning the bare first token.
func Unqualify(name string) string {
	name = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(name), ";"))
	if i := strings.IndexAny(name, " \t\r\n"); i >= 0 {
		name = name[:i]
	}
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	return strings.Trim(name, "[]\"`")
}

// IndexFold is strings.Index, case-insensitive.
func IndexFold(s, sub string) int { return strings.Index(strings.ToUpper(s), strings.ToUpper(sub)) }

// LitSQL renders a Go value as a T-SQL literal (for parameter substitution).
func LitSQL(v any) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case string:
		return "'" + strings.ReplaceAll(x, "'", "''") + "'"
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case bool:
		if x {
			return "1"
		}
		return "0"
	}
	return "NULL"
}

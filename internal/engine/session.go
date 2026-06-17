// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"strconv"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/batch"
	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/procedures/control"
	"github.com/RSKGroup/haystak-tds-spi/tds"
)

type ctxKey int

const dbKey ctxKey = 0

// WithDatabase attaches the session's current database to ctx for default-qualification.
func WithDatabase(ctx context.Context, db string) context.Context {
	return context.WithValue(ctx, dbKey, db)
}

func currentDB(ctx context.Context) string {
	db, _ := ctx.Value(dbKey).(string)
	return db
}

// principalOf is the authenticated identity in ctx (zero value when unauthenticated) for sys.*_principals.
func principalOf(ctx context.Context) tds.Principal {
	p, _ := tds.PrincipalFromContext(ctx)
	return p
}

// sessionOf returns the current session info from ctx, or nil for the runtime DMVs' empty shape.
func sessionOf(ctx context.Context) *tds.SessionInfo {
	if s, ok := tds.SessionInfoFromContext(ctx); ok {
		return &s
	}
	return nil
}

// Session carries per-connection state (current database, SET ROWCOUNT) across a batch sequence.
type Session struct {
	b        tds.Backend
	db       string
	rowCount int // SET ROWCOUNT n; 0 = unlimited
}

// NewSession makes a session whose current database defaults to db (or "master" if empty).
func NewSession(b tds.Backend, db string) *Session {
	if db == "" {
		db = "master"
	}
	return &Session{b: b, db: db}
}

// Database is the session's current database.
func (s *Session) Database() string { return s.db }

// Exec runs a batch under the session's current database; envDB is the new db when USE changed it.
func (s *Session) Exec(ctx context.Context, sql string) (tds.Rows, int64, string, error) {
	// Opt-in: a batch with control flow runs through the procedural interpreter; everything else keeps
	// the flat path untouched.
	if control.HasControlFlow(sql) {
		rows, err := control.Run(WithDatabase(ctx, s.db), sql, engineRunner{s.b})
		return rows, -1, "", err
	}
	sql, err := batch.Resolve(sql) // bind + substitute DECLARE/SET @var batch variables
	if err != nil {
		return nil, -1, "", err
	}
	var lastRows tds.Rows
	lastAffected := int64(-1)
	envDB := ""
	for _, stmt := range splitBatch(sql) {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if db, ok := parseUse(stmt); ok {
			s.db, envDB = db, db
			lastRows, lastAffected = nil, -1
			continue
		}
		if n, ok := parseRowcount(stmt); ok {
			s.rowCount = n
			lastRows, lastAffected = nil, -1
			continue
		}
		rs, aff, err := queryOne(WithDatabase(ctx, s.db), s.b, stmt)
		if err != nil {
			return nil, -1, envDB, err
		}
		if rs != nil {
			if s.rowCount > 0 {
				rs = &limitRows{Rows: rs, n: s.rowCount}
			}
			lastRows, lastAffected = rs, -1
		} else if aff >= 0 {
			lastRows, lastAffected = nil, aff
		}
	}
	return lastRows, lastAffected, envDB, nil
}

// parseRowcount returns n from a `SET ROWCOUNT n` statement.
func parseRowcount(sql string) (int, bool) {
	f := strings.Fields(strings.TrimSuffix(strings.TrimSpace(sql), ";"))
	if len(f) == 3 && strings.EqualFold(f[0], "SET") && strings.EqualFold(f[1], "ROWCOUNT") {
		if n, err := strconv.Atoi(f[2]); err == nil && n >= 0 {
			return n, true
		}
	}
	return 0, false
}

// limitRows caps a result at n rows for SET ROWCOUNT.
type limitRows struct {
	tds.Rows
	n, seen int
}

func (l *limitRows) Next() bool {
	if l.seen >= l.n {
		return false
	}
	if l.Rows.Next() {
		l.seen++
		return true
	}
	return false
}

// parseUse returns the target database of a `USE [db]` statement.
func parseUse(sql string) (string, bool) {
	s := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(sql), ";"))
	if !strings.HasPrefix(strings.ToUpper(s), "USE ") {
		return "", false
	}
	return strings.Trim(strings.TrimSpace(s[4:]), "[]\"`"), true
}

// applyDefaultDB qualifies unqualified real-table queries (each union arm and its joins) with the session db.
func applyDefaultDB(q *tds.Query, db string) {
	if db == "" {
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
	}
}

func isSystemSchema(s string) bool {
	return strings.EqualFold(s, "INFORMATION_SCHEMA") || strings.EqualFold(s, "sys")
}

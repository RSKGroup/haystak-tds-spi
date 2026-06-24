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

// sessionOf returns the live-session snapshot from ctx, or nil for the runtime DMVs' empty shape.
func sessionOf(ctx context.Context) []tds.SessionInfo {
	s, _ := tds.SessionsFromContext(ctx)
	return s
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

type Result struct {
	Rows     tds.Rows
	Affected int64
}

func (s *Session) Exec(ctx context.Context, sql string) (tds.Rows, int64, string, error) {
	res, envDB, err := s.ExecBatch(ctx, sql)
	if err != nil || len(res) == 0 {
		return nil, -1, envDB, err
	}
	last := res[len(res)-1]
	return last.Rows, last.Affected, envDB, nil
}

func (s *Session) ExecBatch(ctx context.Context, sql string) ([]Result, string, error) {
	ctx = withTempStore(ctx)
	if control.HasControlFlow(sql) {
		dbBefore := s.db
		rows, err := control.RunAll(WithDatabase(ctx, s.db), sql, engineRunner{b: s.b, sess: s})
		if err != nil {
			return nil, "", err
		}
		res := make([]Result, len(rows))
		for i, r := range rows {
			res[i] = Result{Rows: r, Affected: -1}
		}
		envDB := "" // a control-flow batch that ran USE [db] must still report the context change
		if s.db != dbBefore {
			envDB = s.db
		}
		return res, envDB, nil
	}
	sql, err := batch.Resolve(sql) // bind + substitute DECLARE/SET @var batch variables
	if err != nil {
		return nil, "", err
	}
	var res []Result
	envDB := ""
	for _, stmt := range splitBatch(sql) {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if db, ok := parseUse(stmt); ok {
			s.db, envDB = db, db
			continue
		}
		if n, ok := parseRowcount(stmt); ok {
			s.rowCount = n
			continue
		}
		rs, aff, err := queryOne(WithDatabase(ctx, s.db), s.b, stmt)
		if err != nil {
			return nil, envDB, err
		}
		if rs != nil {
			if s.rowCount > 0 {
				rs = &limitRows{Rows: rs, n: s.rowCount}
			}
			res = append(res, Result{Rows: rs, Affected: -1})
		} else if aff >= 0 {
			res = append(res, Result{Affected: aff})
		}
	}
	return res, envDB, nil
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

// applyDefaultDB qualifies every unqualified real-table reference (incl. subqueries) with the session db.
func applyDefaultDB(q *tds.Query, db string) {
	if db == "" || q == nil {
		return
	}
	for a := q; a != nil; a = a.Union {
		if a.Table != "" && a.Database == "" && !isSystemSchema(a.Schema) {
			a.Database = db
		}
		applyDefaultDB(a.FromSub, db)
		for i := range a.Joins {
			j := &a.Joins[i]
			if j.Table != "" && j.Database == "" && !isSystemSchema(j.Schema) {
				j.Database = db
			}
			applyDefaultDB(j.FromSub, db)
		}
		defaultDBExpr(a.Where, db)
		defaultDBExpr(a.Having, db)
		for k := range a.Select {
			defaultDBVal(a.Select[k].Expr, db)
			defaultDBVal(a.Select[k].ArgExpr, db)
		}
	}
}

func defaultDBExpr(e *tds.Expr, db string) {
	if e == nil {
		return
	}
	if e.Pred != nil {
		applyDefaultDB(e.Pred.Sub, db)
		defaultDBVal(e.Pred.LeftExpr, db)
		if ve, ok := e.Pred.Value.(*tds.ValueExpr); ok {
			defaultDBVal(ve, db)
		}
	}
	for _, c := range e.And {
		defaultDBExpr(c, db)
	}
	for _, c := range e.Or {
		defaultDBExpr(c, db)
	}
	defaultDBExpr(e.Not, db)
}

func defaultDBVal(ve *tds.ValueExpr, db string) {
	if ve == nil {
		return
	}
	applyDefaultDB(ve.Sub, db)
	defaultDBVal(ve.Left, db)
	defaultDBVal(ve.Right, db)
	for _, a := range ve.Args {
		defaultDBVal(a, db)
	}
	defaultDBVal(ve.Operand, db)
	defaultDBVal(ve.Else, db)
	for _, w := range ve.Whens {
		defaultDBExpr(w.Cond, db)
		defaultDBVal(w.Match, db)
		defaultDBVal(w.Result, db)
	}
}

func isSystemSchema(s string) bool {
	return strings.EqualFold(s, "INFORMATION_SCHEMA") || strings.EqualFold(s, "sys")
}

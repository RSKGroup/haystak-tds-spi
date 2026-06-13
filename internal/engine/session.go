// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/batch"
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

// Session carries per-connection state (the current database) across a batch sequence.
type Session struct {
	b  tds.Backend
	db string
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
		rs, aff, err := queryOne(WithDatabase(ctx, s.db), s.b, stmt)
		if err != nil {
			return nil, -1, envDB, err
		}
		if rs != nil {
			lastRows, lastAffected = rs, -1
		} else if aff >= 0 {
			lastRows, lastAffected = nil, aff
		}
	}
	return lastRows, lastAffected, envDB, nil
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

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tds

import (
	"context"
	"time"
)

// SessionInfo describes a live connection, set by the server at LOGIN7 and read by the runtime DMVs.
type SessionInfo struct {
	SessionID int
	LoginName string
	Host      string
	Program   string
	LoginTime time.Time
}

// SessionEvent is a login or logout, delivered to the server's audit hook.
type SessionEvent struct {
	Kind    string // "login" or "logout"
	Session SessionInfo
	At      time.Time
}

type sessionsKey struct{}

// WithSessions returns a context carrying a snapshot of the live sessions for the runtime DMVs.
func WithSessions(ctx context.Context, sessions []SessionInfo) context.Context {
	return context.WithValue(ctx, sessionsKey{}, sessions)
}

// SessionsFromContext returns the live-session snapshot carried in ctx, if any.
func SessionsFromContext(ctx context.Context) ([]SessionInfo, bool) {
	s, ok := ctx.Value(sessionsKey{}).([]SessionInfo)
	return s, ok
}

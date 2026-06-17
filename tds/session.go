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

type sessionInfoKey struct{}

// WithSessionInfo returns a context carrying the current session's info.
func WithSessionInfo(ctx context.Context, s SessionInfo) context.Context {
	return context.WithValue(ctx, sessionInfoKey{}, s)
}

// SessionInfoFromContext returns the current session info carried in ctx, if any.
func SessionInfoFromContext(ctx context.Context) (SessionInfo, bool) {
	s, ok := ctx.Value(sessionInfoKey{}).(SessionInfo)
	return s, ok
}

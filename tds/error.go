// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tds

import "context"

// ErrorInfo is the error a TRY block caught; it rides in the request context for the duration of the
// matching CATCH block so the ERROR_* functions can report it.
type ErrorInfo struct {
	Number    int64
	Message   string
	Severity  int64
	State     int64
	Line      int64
	Procedure string
}

type errorKey struct{}

// WithError returns a context carrying the caught error (for ERROR_* inside a CATCH block).
func WithError(ctx context.Context, e *ErrorInfo) context.Context {
	return context.WithValue(ctx, errorKey{}, e)
}

// ErrorFromContext returns the caught error carried in ctx, or nil outside a CATCH block.
func ErrorFromContext(ctx context.Context) *ErrorInfo {
	e, _ := ctx.Value(errorKey{}).(*ErrorInfo)
	return e
}

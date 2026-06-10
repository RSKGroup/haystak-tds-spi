// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tds

import "context"

// Login is the credential set a TDS client presents at LOGIN7.
type Login struct {
	Username string
	Password string
	Database string
	AppName  string
	Host     string
}

// Principal is the authenticated identity; it rides in the request context for the whole
// connection so backends can do per-user authorization and audit.
type Principal struct {
	Username   string
	Roles      []string
	Attributes map[string]string
}

// Authenticator validates a Login and returns the authenticated Principal, or an error to reject
// the connection. The Backend usually implements it: the gateway plumbs the LOGIN7 credentials to
// the backend and enforces its go/no-go. Operators may also set one directly on the server.
type Authenticator interface {
	Authenticate(ctx context.Context, login Login) (Principal, error)
}

// AuthFunc adapts a function to Authenticator.
type AuthFunc func(ctx context.Context, login Login) (Principal, error)

func (f AuthFunc) Authenticate(ctx context.Context, login Login) (Principal, error) {
	return f(ctx, login)
}

type principalKey struct{}

// WithPrincipal returns a context carrying the authenticated principal.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// PrincipalFromContext returns the authenticated principal carried in ctx, if any.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}

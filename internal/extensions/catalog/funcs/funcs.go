// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package funcs implements SQL Server scalar/system functions the expression evaluator can't infer
// generically — catalog and metadata functions like DB_ID, HAS_DBACCESS, SCHEMA_NAME, QUOTENAME.
//
// To add a function: register it from the relevant group file's init (catalog.go, and future
// string.go / datetime.go / security.go as groups grow). The evaluator calls Eval for any function
// it doesn't handle itself, so registration is the whole ritual — no engine changes.
package funcs

import "strings"

// registry maps an upper-cased function name to its implementation.
var registry = map[string]func([]any) any{}

func register(name string, fn func([]any) any) { registry[strings.ToUpper(name)] = fn }

// Eval evaluates a registered function; ok is false when the name isn't one of ours.
func Eval(name string, args []any) (any, bool) {
	if fn, ok := registry[strings.ToUpper(name)]; ok {
		return fn(args), true
	}
	return nil, false
}

// argStr returns the i-th argument as a string, or "" if absent/NULL.
func argStr(a []any, i int) string {
	if i < len(a) && a[i] != nil {
		return toStr(a[i])
	}
	return ""
}

func toStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

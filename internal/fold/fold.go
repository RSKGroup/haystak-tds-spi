// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package fold implements the default string collation: ASCII case-insensitive, everything else exact.
package fold

import (
	"fmt"
	"strings"
)

func lower(c byte) byte {
	if 'A' <= c && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}

func Key(s string) string {
	for i := 0; i < len(s); i++ {
		if 'A' <= s[i] && s[i] <= 'Z' {
			b := []byte(s)
			for j := i; j < len(b); j++ {
				b[j] = lower(b[j])
			}
			return string(b)
		}
	}
	return s
}

// Compare orders a and b as strings.Compare(Key(a), Key(b)) without allocating.
func Compare(a, b string) int {
	n := min(len(a), len(b))
	for i := 0; i < n; i++ {
		ca, cb := lower(a[i]), lower(b[i])
		if ca != cb {
			if ca < cb {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(a) < len(b):
		return -1
	case len(a) > len(b):
		return 1
	}
	return 0
}

func Equal(a, b string) bool { return len(a) == len(b) && Compare(a, b) == 0 }

func Value(v any) any {
	if s, ok := v.(string); ok {
		return Key(s)
	}
	return v
}

// RowKey is a grouping key for vals with string parts folded; exact[i] true keeps part i unfolded.
func RowKey(vals []any, exact []bool) string {
	parts := make([]any, len(vals))
	for i, v := range vals {
		if i < len(exact) && exact[i] {
			parts[i] = v
			continue
		}
		parts[i] = Value(v)
	}
	return fmt.Sprintf("%v", parts)
}

func CaseSensitive(collation string) bool {
	return strings.Contains(strings.ToUpper(collation), "_CS_")
}

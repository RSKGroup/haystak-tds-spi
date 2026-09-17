// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package fold

import (
	"strings"
	"testing"
)

func TestEqualASCIIOnly(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"Smith", "SMITH", true},
		{"smith", "Smith", true},
		{"", "", true},
		{"Émile", "émile", false},
		{"Émile", "ÉMILE", true},
		{"straße", "STRASSE", false},
		{"ǅ", "ǆ", false},
		{"K", "K", false},
		{"@", "`", false},
		{"[", "{", false},
		{"Smith", "Smit", false},
	}
	for _, c := range cases {
		if got := Equal(c.a, c.b); got != c.want {
			t.Errorf("Equal(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestCompareMatchesKey(t *testing.T) {
	vals := []string{"", "a", "A", "_", "Z", "z", "[", "`", "{", "ab", "AB", "aB", "Émile", "émile", "e", "abc", "ABD"}
	for _, a := range vals {
		for _, b := range vals {
			if got, want := Compare(a, b), strings.Compare(Key(a), Key(b)); got != want {
				t.Errorf("Compare(%q, %q) = %d, want %d", a, b, got, want)
			}
		}
	}
	if Compare("_", "a") >= 0 || Compare("_", "A") >= 0 {
		t.Errorf("underscore must sort before letters of either case")
	}
}

func TestKey(t *testing.T) {
	if got := Key("HeLLo Émile"); got != "hello Émile" {
		t.Errorf("Key = %q", got)
	}
	if got := Key("already"); got != "already" {
		t.Errorf("Key = %q", got)
	}
}

func TestRowKey(t *testing.T) {
	if RowKey([]any{"Boston", int64(1)}, nil) != RowKey([]any{"BOSTON", int64(1)}, nil) {
		t.Errorf("folded row keys differ")
	}
	if RowKey([]any{"Boston", int64(1)}, []bool{true}) == RowKey([]any{"BOSTON", int64(1)}, []bool{true}) {
		t.Errorf("exact part folded")
	}
	if RowKey([]any{"a", nil}, nil) == RowKey([]any{"A", int64(0)}, nil) {
		t.Errorf("non-string parts must stay distinct")
	}
}

func TestCaseSensitive(t *testing.T) {
	for name, want := range map[string]bool{
		"Latin1_General_CS_AS":         true,
		"SQL_Latin1_General_CP1_CS_AS": true,
		"latin1_general_100_cs_as_sc":  true,
		"SQL_Latin1_General_CP1_CI_AS": false,
		"Latin1_General_BIN2":          false,
		"":                             false,
	} {
		if got := CaseSensitive(name); got != want {
			t.Errorf("CaseSensitive(%q) = %v, want %v", name, got, want)
		}
	}
}

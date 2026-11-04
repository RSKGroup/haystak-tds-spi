// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package batch

import (
	"strings"
	"testing"
)

func TestResolve(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"simple int", `declare @loanId int = 1; SELECT id FROM t WHERE id = @loanId`, "SELECT id FROM t WHERE id = 1"},
		{"string literal untouched", `declare @x int = 5; SELECT '@x' AS a, id FROM t WHERE id = @x`, "SELECT '@x' AS a, id FROM t WHERE id = 5"},
		{"multi declare", `declare @a int = 1, @b int = 2; SELECT @a, @b`, "SELECT 1, 2"},
		{"string value", `declare @n varchar(50) = 'Acme'; SELECT name FROM t WHERE name = @n`, "SELECT name FROM t WHERE name = 'Acme'"},
		{"no vars passthrough", `SELECT 1`, "SELECT 1"},
	}
	for _, c := range cases {
		got, err := Resolve(c.in)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if strings.TrimSpace(got) != c.want {
			t.Errorf("%s:\n  got  %q\n  want %q", c.name, strings.TrimSpace(got), c.want)
		}
	}
}

func TestResolveComments(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"line comment before DECLARE", "DECLARE @x int = 1; -- a\nDECLARE @y int = 2;\nSELECT @x + @y AS z", "SELECT 1 + 2 AS z"},
		{"block comment before DECLARE", "DECLARE @x int = 1; /* a */ DECLARE @y int = 2; SELECT @y", "SELECT 2"},
		{"apostrophe in a comment", "DECLARE @n int = 7; /* the user's id; */ SELECT @n AS a -- it's\n", "SELECT 7 AS a -- it's"},
		{"variable in a comment", "DECLARE @n int = 7; SELECT @n /* @n */", "SELECT 7 /* @n */"},
		{"comment after the value", "DECLARE @n int = 7 /* seven */; SELECT @n", "SELECT 7"},
		{"nested block comment", "DECLARE @n int = 7; /* a /* b; */ c; */ SELECT @n", "SELECT 7"},
	}
	for _, c := range cases {
		got, err := Resolve(c.in)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if strings.TrimSpace(got) != c.want {
			t.Errorf("%s:\n  got  %q\n  want %q", c.name, strings.TrimSpace(got), c.want)
		}
	}
}

func TestSplitComments(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"SELECT 1 AS x /* a; b */", []string{"SELECT 1 AS x /* a; b */"}},
		{"SELECT 1 AS x -- a; b", []string{"SELECT 1 AS x -- a; b"}},
		{"SELECT 1 -- a\n;\nSELECT 2", []string{"SELECT 1 -- a", "SELECT 2"}},
		{"-- lead\nSELECT ';' AS s; /* x */ SELECT [a;b] FROM t", []string{"SELECT ';' AS s", "SELECT [a;b] FROM t"}},
		{"/* only */ ; -- only", nil},
	}
	for _, c := range cases {
		got := Split(c.in)
		if strings.Join(got, "|") != strings.Join(c.want, "|") || len(got) != len(c.want) {
			t.Errorf("Split(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

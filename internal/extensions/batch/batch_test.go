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

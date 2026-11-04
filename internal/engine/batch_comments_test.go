// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"fmt"
	"testing"
)

func TestBatchComments(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"DECLARE @x int = 1; -- a\nDECLARE @y int = 2;\nSELECT @x + @y AS z", "[[3]]"},
		{"SELECT 1 AS x /* a; b */", "[[1]]"},
		{"SELECT 1 AS x -- a; b", "[[1]]"},
		{"/* the user's list; */ SELECT name FROM users WHERE id = 2 -- don't; stop", "[[alan]]"},
		{"DECLARE @id int = 1; /* a /* nested; */ still a comment; */ SELECT name FROM users WHERE id = @id", "[[ada]]"},
		{"SELECT name FROM users /* a /* b */ c */ WHERE id = 1", "[[ada]]"},
		{"-- lead\nSET NOCOUNT ON; -- two\nSELECT COUNT(*) AS n FROM users", "[[2]]"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(qry(t, c.sql)); got != c.want {
			t.Errorf("%q = %s, want %s", c.sql, got, c.want)
		}
	}
}

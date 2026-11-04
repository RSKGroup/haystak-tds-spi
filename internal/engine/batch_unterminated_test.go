// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"fmt"
	"testing"
)

func TestBatchWithoutSemicolons(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"DECLARE @x int = 1 SELECT @x AS x", "[[1]]"},
		{"DECLARE @x int = 1\nDECLARE @y int = 2\nSELECT @x + @y AS z", "[[3]]"},
		{"DECLARE @n varchar(10) = 'ada' SELECT id FROM users WHERE name = @n", "[[1]]"},
		{"SET NOCOUNT ON\nSELECT name FROM users WHERE id = 1", "[[ada]]"},
		{"SELECT 1 AS a\nSELECT name FROM users WHERE id = 2", "[[alan]]"},
		{"SELECT id FROM users WHERE id = 1 UNION ALL SELECT id FROM users WHERE id = 2 ORDER BY id", "[[1] [2]]"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(qry(t, c.sql)); got != c.want {
			t.Errorf("%q = %s, want %s", c.sql, got, c.want)
		}
	}
}

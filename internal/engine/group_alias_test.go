// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
)

func TestGroupByAliasQualifiedColumn(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"SELECT u.name, COUNT(*) AS n FROM users u GROUP BY u.name ORDER BY u.name", "[[ada 1] [alan 1]]"},
		{"SELECT d.k, COUNT(*) AS n FROM (SELECT user_id AS k FROM orders) d GROUP BY d.k ORDER BY d.k", "[[1 1] [2 2]]"},
		{"SELECT o.user_id, MAX(o.amount) AS hi FROM orders o GROUP BY o.user_id ORDER BY o.user_id", "[[1 100] [2 200]]"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(qry(t, c.sql)); got != c.want {
			t.Errorf("%s\n  = %s, want %s", c.sql, got, c.want)
		}
	}
	rs, err := engine.Query(context.Background(), inmem.New(), "SELECT u.name, COUNT(*) FROM users u GROUP BY u.name")
	if err != nil {
		t.Fatal(err)
	}
	if name := rs.Columns()[0].Name; name != "name" {
		t.Errorf("grouped column is named %q, want name", name)
	}
	rs.Close()
}

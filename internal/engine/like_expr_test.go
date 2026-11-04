// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
	"github.com/RSKGroup/haystak-tds-spi/tds"
)

func TestLikeExpressionPattern(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"SELECT name FROM users WHERE name LIKE 'a' + '%' ORDER BY name", "[[ada] [alan]]"},
		{"DECLARE @p varchar(10) = 'al'; SELECT name FROM users WHERE name LIKE @p + '%'", "[[alan]]"},
		{"SELECT name FROM users WHERE name NOT LIKE 'a' + 'l%'", "[[ada]]"},
		{"SELECT name FROM users WHERE name LIKE name ORDER BY name", "[[ada] [alan]]"},
		{"SELECT name FROM users WHERE name LIKE UPPER(LEFT(name, 2)) + '%' ORDER BY name", "[[ada] [alan]]"},
		{"SELECT u.name FROM users u WHERE 'xadax' LIKE '%' + u.name + '%'", "[[ada]]"},
		{"SELECT e.name, d.name FROM emps e JOIN depts d ON e.name LIKE LEFT(d.name, 1) + '%'", "[[orphan ops]]"},
		{"SELECT c.name FROM categories c WHERE CAST(c.parent_id AS varchar(5)) LIKE '%' ORDER BY c.name", "[[Electronics] [Phones]]"},
		{"SELECT CASE WHEN 'Smith-Jones' LIKE '%[ -]Jones' THEN 1 ELSE 0 END", "[[1]]"},
		{"SELECT CASE WHEN 'ab' LIKE '[a-c]b' AND 'db' LIKE '[^a-c]b' AND '[x' LIKE '[[]x' THEN 1 ELSE 0 END", "[[1]]"},
		{"SELECT CASE WHEN 'a%' LIKE 'a[%]' AND 'ab' NOT LIKE 'a[%]' AND 'a-' LIKE 'a[-]' THEN 1 ELSE 0 END", "[[1]]"},
		{"SELECT id FROM orders WHERE amount BETWEEN 40 + 10 AND (SELECT MAX(amount) FROM orders) - 100 ORDER BY id", "[[10] [12]]"},
		{"SELECT o.id FROM orders o JOIN users u ON u.id = o.user_id WHERE o.amount BETWEEN u.id * 50 AND u.id * 100 ORDER BY o.id", "[[10] [11]]"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(qry(t, c.sql)); got != c.want {
			t.Errorf("%s\n  = %s, want %s", c.sql, got, c.want)
		}
	}
}

// recordingBackend keeps the WHERE each Scan receives, so a test can see what a backend is asked to push.
type recordingBackend struct {
	*inmem.Backend
	wheres []*tds.Expr
}

func (r *recordingBackend) Scan(ctx context.Context, q *tds.Query) (tds.Rows, error) {
	r.wheres = append(r.wheres, q.Where)
	return r.Backend.Scan(ctx, q)
}

func TestLikePatternReachesBackendFolded(t *testing.T) {
	for _, c := range []struct {
		sql      string
		wantPush bool
	}{
		{"SELECT name FROM users WHERE name LIKE 'a' + '%'", true},
		{"SELECT u.name FROM users u CROSS APPLY (SELECT x.id FROM users x WHERE x.name LIKE u.name + '%') m", true},
		{"SELECT name FROM users WHERE name LIKE name + '%'", false},
	} {
		b := &recordingBackend{Backend: inmem.New()}
		rs, err := engine.Query(context.Background(), b, c.sql)
		if err != nil {
			t.Fatalf("%s: %v", c.sql, err)
		}
		collect(t, rs)
		pushed := false
		for _, w := range b.wheres {
			if w != nil && w.Pred != nil && w.Pred.Op == tds.OpLike {
				_, pushed = w.Pred.Value.(string)
			}
		}
		if pushed != c.wantPush {
			t.Errorf("%s: LIKE reached the backend as a string = %v, want %v", c.sql, pushed, c.wantPush)
		}
	}
}

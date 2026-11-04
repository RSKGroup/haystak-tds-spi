// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
	"github.com/RSKGroup/haystak-tds-spi/internal/exec"
	"github.com/RSKGroup/haystak-tds-spi/tds"
)

// categories: (1 All NULL) (2 Electronics 1) (3 Phones 2); parent_id is NULL only for All.
func TestNotOverUnknownKeepsNoRow(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"SELECT name FROM categories WHERE NOT (parent_id = 1) ORDER BY name", "[[Phones]]"},
		{"SELECT name FROM categories WHERE parent_id <> 1 ORDER BY name", "[[Phones]]"},
		{"SELECT name FROM categories WHERE NOT (parent_id = 1 AND name = 'All')", "[[Electronics] [Phones]]"},
		{"SELECT name FROM categories WHERE NOT (parent_id = 1 OR name = 'Phones')", "[]"},
		{"SELECT name FROM categories WHERE NOT (parent_id = 9 OR name = 'All') ORDER BY name", "[[Electronics] [Phones]]"},
		{"SELECT name FROM categories WHERE NOT NOT (parent_id = 1)", "[[Electronics]]"},
		{"SELECT name FROM categories WHERE parent_id NOT IN (1, 5)", "[[Phones]]"},
		{"SELECT name FROM categories WHERE category_id NOT IN (1, NULL)", "[]"},
		{"SELECT name FROM categories WHERE category_id NOT IN (SELECT parent_id FROM categories)", "[]"},
		{"SELECT name FROM categories WHERE category_id NOT IN (SELECT parent_id FROM categories WHERE parent_id IS NOT NULL)", "[[Phones]]"},
		{"SELECT name FROM categories WHERE parent_id NOT BETWEEN 2 AND 5", "[[Electronics]]"},
		{"SELECT name FROM categories WHERE CAST(parent_id AS varchar(5)) NOT LIKE '1%'", "[[Phones]]"},
		{"SELECT name FROM categories WHERE parent_id IS NULL OR NOT (parent_id = 1) ORDER BY name", "[[All] [Phones]]"},
		{"SELECT name, CASE WHEN NOT (parent_id = 1) THEN 'y' ELSE 'n' END FROM categories ORDER BY category_id", "[[All n] [Electronics n] [Phones y]]"},
		{"SELECT c.name FROM categories c JOIN categories p ON NOT (c.parent_id = p.category_id) AND p.name = 'All' ORDER BY c.name", "[[Phones]]"},
		{"SELECT parent_id, COUNT(*) FROM categories GROUP BY parent_id HAVING NOT (parent_id = 1)", "[[2 1]]"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(qry(t, c.sql)); got != c.want {
			t.Errorf("%s\n  = %s, want %s", c.sql, got, c.want)
		}
	}
}

// filteringBackend applies every WHERE it is pushed, as a backend that honours pushdown does.
type filteringBackend struct{ *inmem.Backend }

func (f filteringBackend) Scan(ctx context.Context, q *tds.Query) (tds.Rows, error) {
	rs, err := f.Backend.Scan(ctx, q)
	if err != nil || q.Where == nil {
		return rs, err
	}
	cols := rs.Columns()
	var data [][]any
	for rs.Next() {
		v, err := rs.Values()
		if err != nil {
			return nil, err
		}
		data = append(data, v)
	}
	return exec.Apply(cols, data, &tds.Query{Where: q.Where})
}

func TestPushdownToOuterJoinKeepsMeaning(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"SELECT d.name FROM depts d LEFT JOIN emps e ON e.dept_id = d.id WHERE e.id IS NULL", "[[ops]]"},
		{"SELECT d.name FROM depts d LEFT JOIN emps e ON e.dept_id = d.id WHERE NOT (e.name = 'amy') ORDER BY d.name", "[[sales]]"},
		{"SELECT d.name FROM depts d LEFT JOIN emps e ON e.dept_id = d.id WHERE e.name = 'bob' OR e.name IS NULL ORDER BY d.name", "[[ops] [sales]]"},
		{"SELECT e.name FROM depts d RIGHT JOIN emps e ON e.dept_id = d.id WHERE d.id IS NULL", "[[orphan]]"},
		{"SELECT e.name FROM emps e JOIN depts d ON e.dept_id = d.id WHERE NOT (d.name = 'eng')", "[[bob]]"},
	}
	for _, c := range cases {
		rs, err := engine.Query(context.Background(), filteringBackend{inmem.New()}, c.sql)
		if err != nil {
			t.Fatalf("%s: %v", c.sql, err)
		}
		if got := fmt.Sprint(collect(t, rs)); got != c.want {
			t.Errorf("%s\n  = %s, want %s", c.sql, got, c.want)
		}
	}
}

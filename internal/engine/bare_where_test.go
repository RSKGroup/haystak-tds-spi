// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
	"github.com/RSKGroup/haystak-tds-spi/tds"
)

func rowsOf(t *testing.T, sql string) []string {
	t.Helper()
	rs, err := engine.Query(context.Background(), inmem.New(), sql)
	if err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	var out []string
	for _, r := range collect(t, rs) {
		out = append(out, fmt.Sprint(r))
	}
	return out
}

// A bare WHERE column is legal T-SQL and is what a hand-written correlated subquery usually holds.
// singleTableWhere kept a conjunct only when every column carried an alias., so a bare one was
// dropped from the pushdown and each side was scanned whole. Qualifying it must not change the
// ANSWER - only how much gets scanned - so the bare and qualified spellings are compared directly.
func TestBareWhereColumnMatchesItsQualifiedSpelling(t *testing.T) {
	bare := rowsOf(t, "SELECT u.name FROM users u JOIN orders o ON u.id = o.user_id WHERE amount > 50 ORDER BY u.name")
	qual := rowsOf(t, "SELECT u.name FROM users u JOIN orders o ON u.id = o.user_id WHERE o.amount > 50 ORDER BY u.name")
	if len(qual) == 0 {
		t.Fatal("the qualified control matched nothing; the fixture cannot prove anything")
	}
	if fmt.Sprint(bare) != fmt.Sprint(qual) {
		t.Errorf("bare %v != qualified %v", bare, qual)
	}
}

// "id" exists on BOTH users and orders, so it cannot be attributed to one side. It must stay bare
// and be REFUSED, the way SQL Server refuses an ambiguous column name. Silently picking a side
// would return a plausible subset of the answer, which is the failure mode this whole diagnosis is
// about. A returned row set here would be the bug, not a pass.
func TestAmbiguousBareColumnIsRefusedNotGuessed(t *testing.T) {
	_, err := engine.Query(context.Background(), inmem.New(),
		"SELECT u.name FROM users u JOIN orders o ON u.id = o.user_id WHERE id = 1")
	if err == nil {
		t.Fatal("an ambiguous bare column was attributed to a side and answered")
	}
}

// A column on no table in scope must also be left alone rather than invented.
func TestUnknownBareColumnIsLeftAlone(t *testing.T) {
	_, err := engine.Query(context.Background(), inmem.New(),
		"SELECT u.name FROM users u JOIN orders o ON u.id = o.user_id WHERE nosuchcol = 1")
	if err == nil {
		t.Error("an unknown bare column silently succeeded")
	}
}

// scanSpy records which predicate COLUMNS each side's scan receives. Asserting merely that a
// WHERE arrived proves nothing: the join key puts one there either way, which is why the first
// version of this test passed with the fix disabled.
type scanSpy struct {
	*inmem.Backend
	cols map[string][]string // table -> predicate columns that reached its scan
}

func (s *scanSpy) Scan(ctx context.Context, q *tds.Query) (tds.Rows, error) {
	if s.cols == nil {
		s.cols = map[string][]string{}
	}
	s.cols[q.Table] = append(s.cols[q.Table], predCols(q.Where)...)
	return s.Backend.Scan(ctx, q)
}

func predCols(e *tds.Expr) []string {
	if e == nil {
		return nil
	}
	var out []string
	if e.Pred != nil && e.Pred.Column != "" {
		out = append(out, e.Pred.Column)
	}
	for _, c := range e.And {
		out = append(out, predCols(c)...)
	}
	for _, c := range e.Or {
		out = append(out, predCols(c)...)
	}
	return out
}

// The point of the fix is not the answer - that was already right, exec filtered what the scan
// over-returned - it is that the filter REACHES the scan. Without it, orders was scanned whole.
func TestBareWhereReachesTheScan(t *testing.T) {
	spy := &scanSpy{Backend: inmem.New()}
	_, err := engine.Query(context.Background(), spy,
		"SELECT u.name FROM users u JOIN orders o ON u.id = o.user_id WHERE amount > 50")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	var got bool
	for _, c := range spy.cols["orders"] {
		if strings.EqualFold(c, "o.amount") {
			got = true
		}
	}
	if !got {
		t.Errorf("the bare filter never reached the orders scan (saw %v); that side is still read whole",
			spy.cols["orders"])
	}
}

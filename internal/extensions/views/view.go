// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package views implements stored views: CREATE/ALTER/DROP VIEW persist the raw SELECT, and a query
// whose FROM target is a view is rewritten into a derived-table query over that SELECT at read time.
package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/routines"
	"github.com/RSKGroup/haystak-tds-spi/internal/tsql"
	"github.com/RSKGroup/haystak-tds-spi/tds"
)

// HandleDDL persists CREATE/ALTER/DROP VIEW; handled is false for any other SQL.
func HandleDDL(ctx context.Context, store tds.RoutineStore, db, sql string) (bool, error) {
	s := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(sql), ";"))
	up := strings.ToUpper(s)
	switch {
	case routines.HasHead(up, "CREATE", "VIEW"), routines.HasHead(up, "CREATE OR ALTER", "VIEW"), routines.HasHead(up, "ALTER", "VIEW"):
		return true, createView(ctx, store, db, s)
	case routines.HasHead(up, "DROP", "VIEW"):
		return true, routines.Drop(ctx, store, db, routines.AfterDrop(s, "VIEW"))
	}
	return false, nil
}

func createView(ctx context.Context, store tds.RoutineStore, db, s string) error {
	head, body, ok := routines.SplitAS(s)
	if !ok {
		return fmt.Errorf("CREATE VIEW: missing AS")
	}
	name := nameAfter(head, "VIEW")
	if name == "" {
		return fmt.Errorf("CREATE VIEW: missing name")
	}
	return store.PutRoutine(ctx, &tds.Routine{Database: db, Schema: "dbo", Name: name, Kind: tds.RoutineView, Body: body})
}

// ExpandView rewrites a query whose FROM target is a stored view into a derived-table query over the
// view's body and runs it. handled is false when the target isn't a view.
func ExpandView(ctx context.Context, store tds.RoutineStore, run routines.Runner, q *tds.Query) (tds.Rows, bool, error) {
	db := q.Database
	if db == "" {
		db = run.CurrentDB(ctx)
	}
	r, found, err := store.GetRoutine(ctx, db, q.Table)
	if err != nil || !found || r.Kind != tds.RoutineView {
		return nil, false, nil
	}
	sub, perr := tsql.Parse(r.Body)
	if perr != nil {
		return nil, true, fmt.Errorf("view %q: %w", q.Table, perr)
	}
	routines.QualifyDB(sub, db) // body resolves in the view's database
	q2 := *q
	q2.FromSub = sub
	if q2.FromAlias == "" {
		q2.FromAlias = q.Table
	}
	q2.Table = ""
	q2.Database = ""
	rs, e := run.RunQuery(ctx, &q2)
	return rs, true, e
}

func nameAfter(s, keyword string) string {
	i := routines.IndexFold(s, keyword)
	if i < 0 {
		return ""
	}
	return routines.Unqualify(s[i+len(keyword):])
}

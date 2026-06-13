// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"

	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/procedures"
	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/routines"
	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/views"
	"github.com/RSKGroup/haystak-tds-spi/tds"
)

// engineRunner adapts the engine to routines.Runner so the views/procedures packages can execute SQL
// without importing the engine.
type engineRunner struct{ b tds.Backend }

func (r engineRunner) Exec(ctx context.Context, sql string) (tds.Rows, error) {
	rows, _, err := Exec(ctx, r.b, sql)
	return rows, err
}

func (r engineRunner) RunQuery(ctx context.Context, q *tds.Query) (tds.Rows, error) {
	return runParsed(ctx, r.b, q)
}

func (r engineRunner) CurrentDB(ctx context.Context) string { return currentDB(ctx) }

// handleRoutineDDL routes CREATE/ALTER/DROP VIEW|PROCEDURE to the feature packages.
func handleRoutineDDL(ctx context.Context, b tds.Backend, sql string) (bool, error) {
	store, ok := b.(tds.RoutineStore)
	if !ok {
		return false, nil
	}
	db := currentDB(ctx)
	if handled, err := views.HandleDDL(ctx, store, db, sql); handled {
		return true, err
	}
	return procedures.HandleDDL(ctx, store, db, sql)
}

// expandViewIfAny rewrites and runs q if its FROM target is a stored view.
func expandViewIfAny(ctx context.Context, b tds.Backend, q *tds.Query) (tds.Rows, bool, error) {
	store, ok := b.(tds.RoutineStore)
	if !ok {
		return nil, false, nil
	}
	return views.ExpandView(ctx, store, engineRunner{b}, q)
}

// execStoredProc runs a stored procedure by name; found is false when it isn't one.
func execStoredProc(ctx context.Context, b tds.Backend, name string, args []procArg) (tds.Rows, bool, error) {
	store, ok := b.(tds.RoutineStore)
	if !ok {
		return nil, false, nil
	}
	ra := make([]routines.Arg, len(args))
	for i, a := range args {
		ra[i] = routines.Arg{Name: a.name, Val: a.val}
	}
	return procedures.ExecProc(ctx, store, engineRunner{b}, name, ra)
}

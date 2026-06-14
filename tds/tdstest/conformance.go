// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package tdstest provides a conformance harness for tds.Backend implementations.
package tdstest

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
	"github.com/RSKGroup/haystak-tds-spi/tds"
)

// RunConformance checks that a backend's Caps match the interfaces it implements, then drives real
// SELECT and catalog queries through the gateway engine. Call it from a Test in your backend's package.
func RunConformance(t testing.TB, b tds.Backend) {
	t.Helper()
	ctx := context.Background()

	schema, err := b.Describe(ctx)
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if len(schema.Tables) == 0 {
		t.Fatalf("Describe returned an empty schema")
	}

	caps := b.Capabilities()
	if !caps.FullQuery && !caps.Pushdown {
		t.Fatalf("backend is not queryable: neither FullQuery nor Pushdown is set")
	}

	_, isQE := b.(tds.QueryExecutor)
	_, isSc := b.(tds.Scanner)
	_, isWr := b.(tds.Writer)
	_, isDDL := b.(tds.DDL)
	_, isTx := b.(tds.TxBeginner)
	_, isRS := b.(tds.RoutineStore)
	requires(t, caps.FullQuery, isQE, "FullQuery", "QueryExecutor")
	requires(t, caps.Pushdown, isSc, "Pushdown", "Scanner")
	requires(t, caps.Writable, isWr, "Writable", "Writer")
	requires(t, caps.DDL, isDDL, "DDL", "DDL")
	requires(t, caps.Tx != tds.TxNone, isTx, "Tx", "TxBeginner")
	requires(t, caps.Routines, isRS, "Routines", "RoutineStore")

	if caps.FullQuery {
		qe := b.(tds.QueryExecutor)
		rs, err := qe.ExecuteQuery(ctx, &tds.Query{Table: schema.Tables[0].Name})
		if err != nil {
			t.Fatalf("ExecuteQuery: %v", err)
		}
		defer rs.Close()
		if len(rs.Columns()) == 0 {
			t.Fatalf("ExecuteQuery returned no columns")
		}
		for rs.Next() {
			if _, err := rs.Values(); err != nil {
				t.Fatalf("Values: %v", err)
			}
		}
		if err := rs.Err(); err != nil {
			t.Fatalf("Rows.Err: %v", err)
		}
	}

	first := schema.Tables[0].Name
	mustQuery(t, b, "SELECT * FROM "+first)
	mustQuery(t, b, "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES")

	if caps.Routines {
		conformRoutines(t, b)
	}

	if d, ok := b.(tds.Databaser); ok {
		dbs, err := d.Databases(ctx)
		if err != nil {
			t.Fatalf("Databaser.Databases: %v", err)
		}
		if len(dbs) == 0 {
			t.Fatalf("Databaser.Databases returned no databases")
		}
	}
}

// conformRoutines verifies a RoutineStore-capable backend surfaces stored routines through the catalog
// the wire exposes — sys.objects/views/procedures/sql_modules, INFORMATION_SCHEMA.ROUTINES, and
// sp_helptext. It writes one routine of each kind (under a dedicated db), checks them, then drops them.
func conformRoutines(t testing.TB, b tds.Backend) {
	t.Helper()
	store := b.(tds.RoutineStore)
	ctx := context.Background()
	const db = "tdstest"
	qctx := engine.WithDatabase(ctx, db)
	rts := []*tds.Routine{
		{Database: db, Schema: "dbo", Name: "tdstest_view", Kind: tds.RoutineView, Body: "SELECT 1 AS one"},
		{Database: db, Schema: "dbo", Name: "tdstest_proc", Kind: tds.RoutineProc,
			Params: []tds.RoutineParam{{Name: "@id", Type: "int"}}, Body: "SELECT @id AS id"},
		{Database: db, Schema: "dbo", Name: "tdstest_func", Kind: tds.RoutineFunc, Body: "RETURNS int\nAS\nBEGIN\nRETURN 1\nEND"},
		{Database: db, Schema: "dbo", Name: "tdstest_trig", Kind: tds.RoutineTrigger, Body: "ON t AFTER INSERT\nAS\nBEGIN\nEND"},
	}
	for _, r := range rts {
		if err := store.PutRoutine(ctx, r); err != nil {
			t.Fatalf("PutRoutine(%s): %v", r.Name, err)
		}
	}
	defer func() {
		for _, r := range rts {
			_ = store.DropRoutine(ctx, db, r.Name)
		}
	}()

	objs := map[string]string{}
	for _, row := range rowsOf(t, qctx, b, "SELECT name, type_desc FROM sys.objects") {
		objs[str(row[0])] = strings.TrimSpace(str(row[1]))
	}
	for name, want := range map[string]string{
		"tdstest_view": "VIEW", "tdstest_proc": "SQL_STORED_PROCEDURE",
		"tdstest_func": "SQL_SCALAR_FUNCTION", "tdstest_trig": "SQL_TRIGGER",
	} {
		if objs[name] != want {
			t.Errorf("sys.objects[%s] = %q, want %q", name, objs[name], want)
		}
	}

	if views := set(rowsOf(t, qctx, b, "SELECT name FROM sys.views")); !views["tdstest_view"] || views["tdstest_proc"] {
		t.Errorf("sys.views = %v", views)
	}
	if procs := set(rowsOf(t, qctx, b, "SELECT name FROM sys.procedures")); !procs["tdstest_proc"] {
		t.Errorf("sys.procedures = %v", procs)
	}
	if rIS := set(rowsOf(t, qctx, b, "SELECT ROUTINE_NAME FROM INFORMATION_SCHEMA.ROUTINES")); !rIS["tdstest_proc"] || !rIS["tdstest_func"] {
		t.Errorf("INFORMATION_SCHEMA.ROUTINES = %v", rIS)
	}

	mods := rowsOf(t, qctx, b, "SELECT definition FROM sys.sql_modules WHERE object_id = OBJECT_ID('tdstest_view')")
	if len(mods) != 1 || !strings.Contains(str(mods[0][0]), "CREATE VIEW [tdstest_view]") {
		t.Errorf("sys.sql_modules(tdstest_view) = %v", mods)
	}

	var help []string
	for _, row := range rowsOf(t, qctx, b, "EXEC sp_helptext 'tdstest_proc'") {
		help = append(help, str(row[0]))
	}
	if !strings.Contains(strings.Join(help, "\n"), "CREATE PROCEDURE [tdstest_proc]") {
		t.Errorf("sp_helptext(tdstest_proc) = %v", help)
	}
}

func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func set(rows [][]any) map[string]bool {
	m := map[string]bool{}
	for _, r := range rows {
		if len(r) > 0 {
			m[str(r[0])] = true
		}
	}
	return m
}

// rowsOf drives a query through the engine and returns every row.
func rowsOf(t testing.TB, ctx context.Context, b tds.Backend, sql string) [][]any {
	t.Helper()
	rs, err := engine.Query(ctx, b, sql)
	if err != nil {
		t.Fatalf("engine.Query(%q): %v", sql, err)
	}
	if rs == nil {
		return nil
	}
	defer rs.Close()
	var out [][]any
	for rs.Next() {
		v, err := rs.Values()
		if err != nil {
			t.Fatalf("Values for %q: %v", sql, err)
		}
		out = append(out, v)
	}
	if err := rs.Err(); err != nil {
		t.Fatalf("Rows.Err for %q: %v", sql, err)
	}
	return out
}

// mustQuery drives the real gateway engine end-to-end and drains the result.
func mustQuery(t testing.TB, b tds.Backend, sql string) {
	t.Helper()
	rs, err := engine.Query(context.Background(), b, sql)
	if err != nil {
		t.Fatalf("engine.Query(%q): %v", sql, err)
	}
	if rs == nil {
		return
	}
	defer rs.Close()
	for rs.Next() {
		if _, err := rs.Values(); err != nil {
			t.Fatalf("Values for %q: %v", sql, err)
		}
	}
	if err := rs.Err(); err != nil {
		t.Fatalf("Rows.Err for %q: %v", sql, err)
	}
}

func requires(t testing.TB, declared, implemented bool, capName, ifaceName string) {
	t.Helper()
	if declared && !implemented {
		t.Fatalf("Caps.%s is set but backend does not implement %s", capName, ifaceName)
	}
}

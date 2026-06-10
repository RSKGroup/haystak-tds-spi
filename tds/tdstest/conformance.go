// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package tdstest provides a conformance harness for tds.Backend implementations.
package tdstest

import (
	"context"
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
	requires(t, caps.FullQuery, isQE, "FullQuery", "QueryExecutor")
	requires(t, caps.Pushdown, isSc, "Pushdown", "Scanner")
	requires(t, caps.Writable, isWr, "Writable", "Writer")
	requires(t, caps.DDL, isDDL, "DDL", "DDL")
	requires(t, caps.Tx != tds.TxNone, isTx, "Tx", "TxBeginner")

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

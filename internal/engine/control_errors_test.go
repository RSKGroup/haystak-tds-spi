// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
)

func TestTryCatchThrow(t *testing.T) {
	const q = "BEGIN TRY THROW 50001, 'boom', 1 END TRY BEGIN CATCH SELECT ERROR_NUMBER() AS n, ERROR_MESSAGE() AS m END CATCH"
	r := qry(t, q)[0]
	if cell(r[0]) != "50001" || cell(r[1]) != "boom" {
		t.Errorf("THROW caught = %v/%v, want 50001/boom", r[0], r[1])
	}
}

func TestTryCatchSqlError(t *testing.T) {
	// A SQL error inside TRY is caught; ERROR_MESSAGE reports it.
	const q = "BEGIN TRY SELECT 1 FROM nonexistent_table END TRY BEGIN CATCH SELECT ERROR_MESSAGE() AS m END CATCH"
	if got := cell(qry(t, q)[0][0]); got == "" || got == "<nil>" {
		t.Errorf("caught SQL error message = %q, want non-empty", got)
	}
}

func TestTryNoError(t *testing.T) {
	const q = "BEGIN TRY SELECT 'ok' AS r END TRY BEGIN CATCH SELECT 'caught' AS r END CATCH"
	if got := cell(qry(t, q)[0][0]); got != "ok" {
		t.Errorf("TRY no-error = %q, want ok (catch must not run)", got)
	}
}

func TestRaiserrorAndErrorScalars(t *testing.T) {
	const q = "BEGIN TRY RAISERROR('oops', 16, 5) END TRY BEGIN CATCH SELECT ERROR_MESSAGE() AS m, ERROR_SEVERITY() AS s, ERROR_STATE() AS st END CATCH"
	r := qry(t, q)[0]
	if cell(r[0]) != "oops" || cell(r[1]) != "16" || cell(r[2]) != "5" {
		t.Errorf("RAISERROR caught = %v, want oops/16/5", r)
	}
}

func TestErrorScalarsNullOutsideCatch(t *testing.T) {
	if got := cell(qry(t, "SELECT ERROR_MESSAGE() AS m")[0][0]); got != "<nil>" {
		t.Errorf("ERROR_MESSAGE outside CATCH = %q, want NULL", got)
	}
}

func TestUncaughtThrowPropagates(t *testing.T) {
	_, err := engine.Query(context.Background(), inmem.New(), "THROW 50005, 'unhandled', 1")
	if err == nil || err.Error() != "unhandled" {
		t.Errorf("uncaught THROW err = %v, want 'unhandled'", err)
	}
}

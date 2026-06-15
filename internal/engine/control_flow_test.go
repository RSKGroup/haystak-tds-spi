// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
)

func TestControlIfElse(t *testing.T) {
	if got := cell(qry(t, "IF 1 = 1 SELECT 'yes' AS r")[0][0]); got != "yes" {
		t.Errorf("IF true = %q, want yes", got)
	}
	if got := cell(qry(t, "IF 1 > 2 SELECT 'a' AS r ELSE SELECT 'b' AS r")[0][0]); got != "b" {
		t.Errorf("IF/ELSE = %q, want b", got)
	}
}

func TestControlWhileSum(t *testing.T) {
	const q = "DECLARE @i INT = 1; DECLARE @s INT = 0; WHILE @i <= 5 BEGIN SET @s = @s + @i; SET @i = @i + 1; END; SELECT @s AS total"
	if got := cell(qry(t, q)[0][0]); got != "15" {
		t.Errorf("WHILE sum 1..5 = %q, want 15", got)
	}
}

func TestControlBreakContinue(t *testing.T) {
	brk := "DECLARE @i INT = 0; WHILE 1 = 1 BEGIN SET @i = @i + 1; IF @i >= 3 BREAK; END; SELECT @i AS r"
	if got := cell(qry(t, brk)[0][0]); got != "3" {
		t.Errorf("BREAK = %q, want 3", got)
	}
	cont := "DECLARE @i INT = 0; DECLARE @c INT = 0; WHILE @i < 5 BEGIN SET @i = @i + 1; IF @i = 3 CONTINUE; SET @c = @c + 1; END; SELECT @c AS r"
	if got := cell(qry(t, cont)[0][0]); got != "4" {
		t.Errorf("CONTINUE = %q, want 4 (skip i=3)", got)
	}
}

func TestControlIfExistsAndReturn(t *testing.T) {
	exists := "IF EXISTS (SELECT 1 FROM users WHERE id = 1) SELECT 'found' AS r ELSE SELECT 'no' AS r"
	if got := cell(qry(t, exists)[0][0]); got != "found" {
		t.Errorf("IF EXISTS = %q, want found", got)
	}
	// RETURN halts before the SELECT, so the batch yields no result set.
	rs, err := engine.Query(context.Background(), inmem.New(), "DECLARE @i INT = 1; IF @i = 1 RETURN; SELECT 'after' AS r")
	if err != nil {
		t.Fatalf("RETURN batch: %v", err)
	}
	if rs != nil {
		t.Errorf("RETURN produced a result set, want none")
	}
}

func TestControlNestedAndStringVar(t *testing.T) {
	nested := "DECLARE @x INT = 10; IF @x > 5 BEGIN IF @x > 8 SELECT 'big' AS r ELSE SELECT 'mid' AS r END ELSE SELECT 'small' AS r"
	if got := cell(qry(t, nested)[0][0]); got != "big" {
		t.Errorf("nested IF = %q, want big", got)
	}
	if got := cell(qry(t, "DECLARE @n VARCHAR(20) = 'world'; SELECT 'hi ' + @n AS r")[0][0]); got != "hi world" {
		t.Errorf("string var = %q, want 'hi world'", got)
	}
}

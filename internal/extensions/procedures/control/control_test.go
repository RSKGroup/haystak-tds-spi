// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package control

import "testing"

// The interpreter is opt-in: plain batches (no control statement) must not route to it, so the flat
// query path stays untouched. A column or alias merely named like a keyword must not trigger it either.
func TestHasControlFlowIsolation(t *testing.T) {
	plain := []string{
		"SELECT name FROM users WHERE id = 2",
		"SELECT 1 + 2",
		"INSERT INTO t (a) VALUES (1)",
		"UPDATE t SET a = 1 WHERE id = 2",
		"SELECT iff AS while_col FROM t",                     // names that look like keywords, not statements
		"EXEC sp_help",                                       // bare proc
		"SELECT * FROM (SELECT 1 WHERE EXISTS (SELECT 1)) x", // EXISTS deep in a subquery is not a statement
		"BEGIN TRANSACTION",                                  // transaction, not a control block
	}
	for _, sql := range plain {
		if HasControlFlow(sql) {
			t.Errorf("HasControlFlow(%q) = true, want false (must keep the flat path)", sql)
		}
	}
	control := []string{
		"IF 1 = 1 SELECT 1",
		"WHILE @i < 3 SET @i = @i + 1",
		"BEGIN SELECT 1 END",
		"PRINT 'hi'",
		"SELECT 1; RETURN",
		"BEGIN TRY SELECT 1 END TRY BEGIN CATCH SELECT 2 END CATCH",
	}
	for _, sql := range control {
		if !HasControlFlow(sql) {
			t.Errorf("HasControlFlow(%q) = false, want true", sql)
		}
	}
}

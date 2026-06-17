// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

func TestSpExecuteSQL(t *testing.T) {
	if got := qry(t, "EXEC sp_executesql N'SELECT name FROM users WHERE id = 2'"); len(got) != 1 || cell(got[0][0]) != "alan" {
		t.Errorf("sp_executesql plain = %v, want [[alan]]", got)
	}
	if got := qry(t, "EXEC sp_executesql N'SELECT name FROM users WHERE id = @id', N'@id int', @id = 1"); len(got) != 1 || cell(got[0][0]) != "ada" {
		t.Errorf("sp_executesql @id=1 = %v, want [[ada]]", got)
	}
	if got := qry(t, "EXEC sp_executesql N'SELECT name FROM users WHERE name = @n', N'@n nvarchar(10)', @n = N'alan'"); len(got) != 1 || cell(got[0][0]) != "alan" {
		t.Errorf("sp_executesql @n=alan = %v, want [[alan]]", got)
	}
}

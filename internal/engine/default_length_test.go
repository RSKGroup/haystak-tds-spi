// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"fmt"
	"testing"
)

// CAST and CONVERT to a text type with no length use 30; a variable declared with no length holds 1.
func TestTextTypeDefaultLengths(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"SELECT LEN(CAST(REPLICATE('x', 40) AS varchar))", "30"},
		{"SELECT LEN(CONVERT(nvarchar, REPLICATE('y', 35)))", "30"},
		{"SELECT LEN(TRY_CONVERT(varchar, REPLICATE('z', 31)))", "30"},
		{"SELECT LEN(CAST('ab' AS char) + '|')", "31"},
		{"SELECT LEN(CAST(REPLICATE('x', 40) AS varchar(max)))", "40"},
		{"SELECT CAST(12 AS varchar)", "12"},
		{"DECLARE @x varchar = 'abc'; SELECT @x", "a"},
		{"DECLARE @x nvarchar = N'xyz'; SELECT @x", "x"},
		{"DECLARE @x varchar(2) = 'abc'; SELECT @x", "ab"},
		{"DECLARE @x char(4) = 'ab'; SELECT @x + '|'", "ab  |"},
		{"DECLARE @x varchar(max) = 'abcdef'; SELECT @x", "abcdef"},
		{"DECLARE @x varchar(3); SET @x = 'abcdef'; SELECT @x", "abc"},
		{"DECLARE @x varchar(2) = 123; SELECT @x", "*"},
		{"DECLARE @x varchar(3) = UPPER('abcdef'); SELECT @x", "ABC"},
		{"DECLARE @n int = 7; SELECT @n + 1", "8"},
		{"DECLARE @x varchar = 'abc'; IF 1 = 1 SELECT @x AS x", "a"},
		{"DECLARE @x varchar(2); IF 1 = 1 BEGIN SET @x = 'hello' SELECT @x AS x END", "he"},
		{"DECLARE @x varchar(2); SELECT @x = 'hello'; SELECT @x AS x", "he"},
	}
	for _, c := range cases {
		rows := qry(t, c.sql)
		if len(rows) != 1 || fmt.Sprint(rows[0][0]) != c.want {
			t.Errorf("%s = %v, want %q", c.sql, rows, c.want)
		}
	}
}

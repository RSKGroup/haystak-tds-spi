// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
)

// 373: string functions counted and cut bytes, so accented or non-Latin text read long and could be split
// mid-character. Positions and lengths are characters, as in SQL Server's nvarchar.
func TestStringFunctionsCountCharacters(t *testing.T) {
	b := collationBackend(t)
	cases := []struct{ sql, want string }{
		{"SELECT LEN(N'Émile')", "5"},
		{"SELECT LEN(N'Émile  ')", "5"},
		{"SELECT LEN('ab  ')", "2"},
		{"SELECT LEN(N'  ab')", "4"},
		{"SELECT LEN(N'日本語')", "3"},
		{"SELECT LEN(name) FROM people WHERE id = 5", "5"},
		{"SELECT LEFT(N'Émile', 2)", "Ém"},
		{"SELECT RIGHT(N'日本語', 2)", "本語"},
		{"SELECT SUBSTRING(N'Émile', 2, 2)", "mi"},
		{"SELECT SUBSTRING(N'日本語テキスト', 2, 3)", "本語テ"},
		{"SELECT SUBSTRING(N'Émile', 0, 2)", "É"},
		{"SELECT CHARINDEX('l', N'Émile')", "4"},
		{"SELECT CHARINDEX(N'語', N'日本語')", "3"},
		{"SELECT CHARINDEX('E', N'ÉmileÉmile', 3)", "5"},
		{"SELECT CHARINDEX('x', N'Émile')", "0"},
		{"SELECT PATINDEX('%l%', N'Émile')", "4"},
		{"SELECT PATINDEX('%テ%', N'日本語テキスト')", "4"},
		{"SELECT STUFF(N'Émile', 2, 1, 'X')", "ÉXile"},
		{"SELECT STUFF(N'日本語', 3, 1, N'人')", "日本人"},
		{"SELECT LEFT('abcdef', 3), RIGHT('abcdef', 2), SUBSTRING('abcdef', 2, 3)", "abc|ef|bcd"},
	}
	for _, c := range cases {
		rs, err := engine.Query(context.Background(), b, c.sql)
		if err != nil {
			t.Errorf("%s: %v", c.sql, err)
			continue
		}
		if got := render(collect(t, rs)); got != c.want {
			t.Errorf("%s\n got  %s\n want %s", c.sql, got, c.want)
		}
	}
}

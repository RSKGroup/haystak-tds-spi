// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package batch

import (
	"strings"
	"testing"
)

func TestResolve(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"simple int", `declare @loanId int = 1; SELECT id FROM t WHERE id = @loanId`, "SELECT id FROM t WHERE id = 1"},
		{"string literal untouched", `declare @x int = 5; SELECT '@x' AS a, id FROM t WHERE id = @x`, "SELECT '@x' AS a, id FROM t WHERE id = 5"},
		{"multi declare", `declare @a int = 1, @b int = 2; SELECT @a, @b`, "SELECT 1, 2"},
		{"string value", `declare @n varchar(50) = 'Acme'; SELECT name FROM t WHERE name = @n`, "SELECT name FROM t WHERE name = 'Acme'"},
		{"no vars passthrough", `SELECT 1`, "SELECT 1"},
	}
	for _, c := range cases {
		got, err := Resolve(c.in)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if strings.TrimSpace(got) != c.want {
			t.Errorf("%s:\n  got  %q\n  want %q", c.name, strings.TrimSpace(got), c.want)
		}
	}
}

func TestResolveComments(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"line comment before DECLARE", "DECLARE @x int = 1; -- a\nDECLARE @y int = 2;\nSELECT @x + @y AS z", "SELECT 1 + 2 AS z"},
		{"block comment before DECLARE", "DECLARE @x int = 1; /* a */ DECLARE @y int = 2; SELECT @y", "SELECT 2"},
		{"apostrophe in a comment", "DECLARE @n int = 7; /* the user's id; */ SELECT @n AS a -- it's\n", "SELECT 7 AS a -- it's"},
		{"variable in a comment", "DECLARE @n int = 7; SELECT @n /* @n */", "SELECT 7 /* @n */"},
		{"comment after the value", "DECLARE @n int = 7 /* seven */; SELECT @n", "SELECT 7"},
		{"nested block comment", "DECLARE @n int = 7; /* a /* b; */ c; */ SELECT @n", "SELECT 7"},
	}
	for _, c := range cases {
		got, err := Resolve(c.in)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if strings.TrimSpace(got) != c.want {
			t.Errorf("%s:\n  got  %q\n  want %q", c.name, strings.TrimSpace(got), c.want)
		}
	}
}

func TestSplitComments(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"SELECT 1 AS x /* a; b */", []string{"SELECT 1 AS x /* a; b */"}},
		{"SELECT 1 AS x -- a; b", []string{"SELECT 1 AS x -- a; b"}},
		{"SELECT 1 -- a\n;\nSELECT 2", []string{"SELECT 1 -- a", "SELECT 2"}},
		{"-- lead\nSELECT ';' AS s; /* x */ SELECT [a;b] FROM t", []string{"SELECT ';' AS s", "SELECT [a;b] FROM t"}},
		{"/* only */ ; -- only", nil},
	}
	for _, c := range cases {
		got := Split(c.in)
		if strings.Join(got, "|") != strings.Join(c.want, "|") || len(got) != len(c.want) {
			t.Errorf("Split(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSplitWithoutSemicolons(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"DECLARE @x int = 1 SELECT @x", []string{"DECLARE @x int = 1", "SELECT @x"}},
		{"DECLARE @x int = 1\nDECLARE @y int = 2\nSELECT @x + @y", []string{"DECLARE @x int = 1", "DECLARE @y int = 2", "SELECT @x + @y"}},
		{"DECLARE @x int = CASE WHEN 1 = 1 THEN (SELECT 1) ELSE 2 END SELECT @x", []string{"DECLARE @x int = CASE WHEN 1 = 1 THEN (SELECT 1) ELSE 2 END", "SELECT @x"}},
		{"DECLARE @s varchar(20) = 'SELECT x' -- SET y\nSELECT @s", []string{"DECLARE @s varchar(20) = 'SELECT x' -- SET y", "SELECT @s"}},
		{"SET NOCOUNT ON\nSELECT 1", []string{"SET NOCOUNT ON", "SELECT 1"}},
		{"USE [db one] SELECT 1", []string{"USE [db one]", "SELECT 1"}},
		{"SELECT 1 SELECT 2", []string{"SELECT 1", "SELECT 2"}},
		{"SELECT 1 UNION ALL SELECT 2 UNION /* x */ SELECT 3", []string{"SELECT 1 UNION ALL SELECT 2 UNION /* x */ SELECT 3"}},
		{"WITH c AS (SELECT 1 AS a) SELECT a FROM c", []string{"WITH c AS (SELECT 1 AS a) SELECT a FROM c"}},
		{"WITH c AS (SELECT 1 AS a) SELECT a FROM c SELECT 2", []string{"WITH c AS (SELECT 1 AS a) SELECT a FROM c", "SELECT 2"}},
		{"SELECT TOP 1 WITH TIES a FROM t WITH (NOLOCK) ORDER BY a", []string{"SELECT TOP 1 WITH TIES a FROM t WITH (NOLOCK) ORDER BY a"}},
		{"UPDATE t SET a = 1 WHERE b = 2 DECLARE @x int = 1", []string{"UPDATE t SET a = 1 WHERE b = 2", "DECLARE @x int = 1"}},
		{"INSERT INTO t SELECT a FROM u WHERE a IN (SELECT b FROM v)", []string{"INSERT INTO t SELECT a FROM u WHERE a IN (SELECT b FROM v)"}},
		{"CREATE PROCEDURE p AS DECLARE @x int = 1 SELECT @x", []string{"CREATE PROCEDURE p AS DECLARE @x int = 1 SELECT @x"}},
		{"EXEC p @a = 1 SELECT 2", []string{"EXEC p @a = 1", "SELECT 2"}},
		{"INSERT INTO t VALUES (1), (2) SELECT a FROM t", []string{"INSERT INTO t VALUES (1), (2)", "SELECT a FROM t"}},
		{"INSERT INTO t (a) SELECT b FROM u", []string{"INSERT INTO t (a) SELECT b FROM u"}},
	}
	for _, c := range cases {
		got := Split(c.in)
		if strings.Join(got, "|") != strings.Join(c.want, "|") || len(got) != len(c.want) {
			t.Errorf("Split(%q)\n  = %q\n  want %q", c.in, got, c.want)
		}
	}
	got, err := Resolve("DECLARE @x int = 1\nDECLARE @y int = 2\nSELECT @x + @y AS z")
	if err != nil || strings.TrimSpace(got) != "SELECT 1 + 2 AS z" {
		t.Errorf("Resolve without semicolons = %q, %v", got, err)
	}
}

func TestResolveFitsDeclaredTextTypes(t *testing.T) {
	cases := []struct{ in, want string }{
		{"DECLARE @x varchar = 'abc'; SELECT @x", "SELECT 'a'"},
		{"DECLARE @x nvarchar(2) = N'it''s'; SELECT @x", "SELECT N'it'"},
		{"DECLARE @x char(3) = 'a'; SELECT @x", "SELECT 'a  '"},
		{"DECLARE @x varchar(2) = 12345; SELECT @x", "SELECT '*'"},
		{"DECLARE @x varchar(3) = UPPER(@y); SELECT @x", "SELECT CAST((UPPER(@y)) AS varchar(3))"},
		{"DECLARE @x varchar(1); SET @x = 'zz'; SELECT @x", "SELECT 'z'"},
		{"DECLARE @x varchar(max) = 'abc'; SELECT @x", "SELECT 'abc'"},
		{"DECLARE @n int = 12345; SELECT @n", "SELECT 12345"},
		{"DECLARE @x varchar(3) = NULL; SELECT @x", "SELECT NULL"},
	}
	for _, c := range cases {
		got, err := Resolve(c.in)
		if err != nil || strings.TrimSpace(got) != c.want {
			t.Errorf("Resolve(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
}

func TestSplitAfterUpdateAndMerge(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"UPDATE t SET a = 1 SET NOCOUNT ON", []string{"UPDATE t SET a = 1", "SET NOCOUNT ON"}},
		{"UPDATE t SET a = 1, b = 2 WHERE c IN (SELECT d FROM u) SET XACT_ABORT ON", []string{"UPDATE t SET a = 1, b = 2 WHERE c IN (SELECT d FROM u)", "SET XACT_ABORT ON"}},
		{"UPDATE t SET a = 1 SELECT @@ROWCOUNT", []string{"UPDATE t SET a = 1", "SELECT @@ROWCOUNT"}},
		{"UPDATE t WITH (ROWLOCK) SET a = 1 WHERE b = 2", []string{"UPDATE t WITH (ROWLOCK) SET a = 1 WHERE b = 2"}},
		{"UPDATE t SET nocount = 1", []string{"UPDATE t SET nocount = 1"}},
		{"WITH c AS (SELECT 1 AS a) UPDATE c SET a = 2 SET NOCOUNT OFF", []string{"WITH c AS (SELECT 1 AS a) UPDATE c SET a = 2", "SET NOCOUNT OFF"}},
		{"DELETE FROM t WHERE a = 1 SET ROWCOUNT 0", []string{"DELETE FROM t WHERE a = 1", "SET ROWCOUNT 0"}},
		{"SET DATEFORMAT dmy UPDATE t SET a = 1", []string{"SET DATEFORMAT dmy", "UPDATE t SET a = 1"}},
		{"MERGE t USING u ON t.a = u.a WHEN MATCHED THEN UPDATE SET b = u.b WHEN NOT MATCHED THEN INSERT (a) VALUES (u.a) SET NOCOUNT ON",
			[]string{"MERGE t USING u ON t.a = u.a WHEN MATCHED THEN UPDATE SET b = u.b WHEN NOT MATCHED THEN INSERT (a) VALUES (u.a)", "SET NOCOUNT ON"}},
		{"MERGE t USING (SELECT a FROM u) s ON t.a = s.a WHEN MATCHED THEN DELETE SET TRANSACTION ISOLATION LEVEL READ COMMITTED",
			[]string{"MERGE t USING (SELECT a FROM u) s ON t.a = s.a WHEN MATCHED THEN DELETE", "SET TRANSACTION ISOLATION LEVEL READ COMMITTED"}},
		{"INSERT INTO t (a) EXEC p", []string{"INSERT INTO t (a) EXEC p"}},
	}
	for _, c := range cases {
		got := Split(c.in)
		if strings.Join(got, "|") != strings.Join(c.want, "|") || len(got) != len(c.want) {
			t.Errorf("Split(%q)\n  = %q\n  want %q", c.in, got, c.want)
		}
	}
}

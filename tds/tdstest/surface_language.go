// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tdstest

// Expression-language features that are not named wired elements (so Element is empty and they are
// exempt from the completeness gate); they still run on every backend as extra coverage.
var languageCases = []Case{
	{Name: "lang/arith-precedence", SQL: "SELECT 2 + 3 * 4", Want: []any{14}},
	{Name: "lang/paren-grouping", SQL: "SELECT (10 - 2) * 2", Want: []any{16}},
	{Name: "lang/string-concat", SQL: "SELECT 'a' + 'b' + 'c'", Want: []any{"abc"}},
	{Name: "lang/case-searched", SQL: "SELECT CASE WHEN 1 = 1 THEN 'eq' ELSE 'ne' END", Want: []any{"eq"}},
	{Name: "lang/case-simple", SQL: "SELECT CASE 2 WHEN 1 THEN 'one' WHEN 2 THEN 'two' ELSE '?' END", Want: []any{"two"}},
	{Name: "lang/nested-call", SQL: "SELECT UPPER(LEFT('abcdef',3))", Want: []any{"ABC"}},
	{Name: "lang/column-alias", SQL: "SELECT 7 AS n", Want: []any{7}},
	{Name: "lang/ci-equal", SQL: "SELECT CASE WHEN 'Smith' = 'SMITH' THEN 1 ELSE 0 END", Want: []any{1}},
	{Name: "lang/ci-ascii-only", SQL: "SELECT CASE WHEN N'Émile' = N'émile' THEN 1 ELSE 0 END", Want: []any{0}},
	{Name: "lang/ci-in", SQL: "SELECT CASE WHEN 'x' IN ('X', 'Y') THEN 1 ELSE 0 END", Want: []any{1}},
	{Name: "lang/ci-between", SQL: "SELECT CASE WHEN 'M' BETWEEN 'a' AND 'z' THEN 1 ELSE 0 END", Want: []any{1}},
	{Name: "lang/ci-like", SQL: "SELECT CASE WHEN 'ABC' LIKE 'a_c' THEN 1 ELSE 0 END", Want: []any{1}},
	{Name: "lang/like-expression", SQL: "SELECT CASE WHEN 'Smith' LIKE 'Smi' + '%' THEN 1 ELSE 0 END", Want: []any{1}},
	{Name: "lang/like-set", SQL: "SELECT CASE WHEN 'Smith-Jones' LIKE '%[ -]Jones' AND 'db' LIKE '[^a-c]b' THEN 1 ELSE 0 END", Want: []any{1}},
	{Name: "lang/like-escape", SQL: "SELECT CASE WHEN '50%' LIKE '50!%' ESCAPE '!' AND '500' NOT LIKE '50!%' ESCAPE '!' AND '^' LIKE '[!^a]' ESCAPE '!' THEN 1 ELSE 0 END", Want: []any{1}},
	{Name: "lang/like-null", SQL: "SELECT CASE WHEN NULL LIKE '%' THEN 1 ELSE 0 END", Want: []any{0}},
	{Name: "lang/not-unknown", SQL: "SELECT CASE WHEN NOT (NULL = 1) THEN 1 ELSE 0 END", Want: []any{0}},
	{Name: "lang/not-in-with-null", SQL: "SELECT CASE WHEN 1 NOT IN (2, NULL) THEN 1 ELSE 0 END", Want: []any{0}},
	{Name: "lang/not-unknown-and-false", SQL: "SELECT CASE WHEN NOT (NULL = 1 AND 1 = 0) THEN 1 ELSE 0 END", Want: []any{1}},
	{Name: "lang/not-filters-null-rows", SQL: "SELECT COUNT(*) FROM (VALUES (1), (NULL)) v(x) WHERE NOT (v.x = 2)", Want: []any{1}},
	{Name: "lang/between-expression", SQL: "SELECT CASE WHEN 5 BETWEEN 2 + 2 AND 3 * 2 THEN 1 ELSE 0 END", Want: []any{1}},
	{Name: "lang/apply-outer-row", SQL: "SELECT c.x FROM (SELECT 41 AS n) t CROSS APPLY (SELECT t.n + 1 AS x) c", Want: []any{42}},
	{Name: "lang/derived-union", SQL: "SELECT COUNT(*) FROM (SELECT 1 AS a UNION ALL SELECT 2) u", Want: []any{2}},
	{Name: "lang/values-derived", SQL: "SELECT MIN(v.x) FROM (VALUES ('b'), ('A')) v(x)", Want: []any{"A"}},
	{Name: "lang/aggregate-operand", SQL: "SELECT COUNT(*) + 1 FROM (VALUES (1), (2)) v(x)", Want: []any{3}},
	{Name: "lang/batch-without-semicolons", SQL: "DECLARE @x int = 40\nDECLARE @y int = 2\nSELECT @x + @y", Want: []any{42}},
	{Name: "lang/trailing-spaces-equal", SQL: "SELECT CASE WHEN 'ab' = 'ab   ' AND 'ab  ' LIKE 'ab' AND 'ab' NOT LIKE 'ab ' THEN 1 ELSE 0 END", Want: []any{1}},
	{Name: "lang/group-by-alias-column", SQL: "SELECT COUNT(*) FROM (SELECT t.schema_id FROM sys.types t GROUP BY t.schema_id) g", Want: []any{1}},
	{Name: "lang/comment-in-batch", SQL: "SELECT 1 AS first /* a; b */ -- c; d", Want: []any{1}},
	{Name: "lang/ci-case-simple", SQL: "SELECT CASE 'yes' WHEN 'YES' THEN 1 ELSE 0 END", Want: []any{1}},
	{Name: "lang/ci-nullif", SQL: "SELECT NULLIF('abc', 'ABC')", Want: []any{P(isNull)}},
	{Name: "lang/ci-union", SQL: "SELECT 'a' AS v UNION SELECT 'A'", Want: []any{"a"}},
	{Name: "lang/ci-intersect", SQL: "SELECT 'Smith' AS v INTERSECT SELECT 'SMITH'", Want: []any{"Smith"}},
	{Name: "lang/ci-order-by", SQL: "SELECT 'b' AS v UNION ALL SELECT 'A' UNION ALL SELECT 'a' UNION ALL SELECT 'B' ORDER BY v", Check: firstCol("A", "a", "b", "B")},
	{Name: "lang/collate-cs-equal", SQL: "SELECT CASE WHEN 'Smith' COLLATE Latin1_General_CS_AS = 'SMITH' THEN 1 ELSE 0 END", Want: []any{0}},
	{Name: "lang/collate-cs-like", SQL: "SELECT CASE WHEN 'ABC' LIKE 'a%' COLLATE Latin1_General_CS_AS THEN 1 ELSE 0 END", Want: []any{0}},
	{Name: "lang/collate-ci-name", SQL: "SELECT CASE WHEN 'Smith' = 'SMITH' COLLATE SQL_Latin1_General_CP1_CI_AS THEN 1 ELSE 0 END", Want: []any{1}},
	{Name: "lang/collate-cs-order-by", SQL: "SELECT 'b' AS v UNION ALL SELECT 'A' UNION ALL SELECT 'a' UNION ALL SELECT 'B' ORDER BY v COLLATE Latin1_General_CS_AS", Check: firstCol("A", "B", "a", "b")},
}

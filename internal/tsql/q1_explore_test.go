// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tsql

import (
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/batch"
)

// TestQ1Bits drives each construct of the loan-rollup target query (Q1) through the real pipeline
// (batch.Resolve → Parse), so we can see the bits land. Exploratory — fold into real tests once Q1
// runs end-to-end against a backend.
func TestQ1Bits(t *testing.T) {
	const q1 = `declare @loanId int = 1;
	WITH LoanScope AS (
		SELECT StartDate, EndDate FROM Lenders.Loan WHERE LoanId = @loanId
	),
	CandidateVisits AS (
		SELECT v.VisitId, v.DateStarted,
			WorkFlow = CASE WHEN wf.Name = '_Template' THEN wf1.Name ELSE wf.Name END
		FROM LoanScope l
		JOIN Visit v ON v.DateStarted >= l.StartDate AND v.DateStarted <= l.EndDate AND v.Voided = 0
		JOIN TemplateWorkFlow wf ON wf.WorkFlowId = v.WorkFlowId
		LEFT JOIN TemplateWorkFlow wf1 ON wf1.WorkFlowId = wf.ParentId
	),
	TaskRollup AS (
		SELECT t.VisitId,
			HasOpenRequired = MAX(CASE WHEN tt.IsRequired = 1 AND t.Status NOT IN (3,4,5) THEN 1 ELSE 0 END),
			HasStarted = MAX(CASE WHEN t.Status <> 0 THEN 1 ELSE 0 END)
		FROM Task t
		JOIN CandidateVisits cv ON cv.VisitId = t.VisitId
		LEFT JOIN TemplateTask tt ON tt.TemplateTaskId = t.TemplateTaskId
		GROUP BY t.VisitId
	)
	SELECT cv.DateStarted, cv.WorkFlow, MIN(tr.HasStarted) AS S, SUM(tr.HasOpenRequired) AS O
	FROM CandidateVisits cv
	LEFT JOIN TaskRollup tr ON tr.VisitId = cv.VisitId
	GROUP BY cv.DateStarted, cv.WorkFlow
	ORDER BY cv.DateStarted`

	cases := []struct{ name, sql string }{
		{"col = expr alias", `SELECT WorkFlow = CASE WHEN n = '_T' THEN p ELSE n END FROM t`},
		{"multi-predicate ON (range+eq)", `SELECT x FROM a JOIN b ON b.d >= a.s AND b.d <= a.e AND b.v = 0`},
		{"self-join (same table, 2 aliases)", `SELECT x FROM t wf JOIN t wf1 ON wf1.id = wf.pid`},
		{"schema-qualified table (non-dbo)", `SELECT StartDate FROM Lenders.Loan WHERE LoanId = 1`},
		{"NOT IN list", `SELECT x FROM t WHERE s NOT IN (3,4,5)`},
		{"<> operator", `SELECT x FROM t WHERE s <> 0`},
		{"aggregates MIN/MAX/SUM", `SELECT MIN(a) AS mn, MAX(b) AS mx, SUM(c) AS sm FROM t GROUP BY g`},
		{"ISNULL fn", `SELECT ISNULL(x, 0) AS v FROM t`},
		{"agg over CASE", `SELECT MAX(CASE WHEN a = 1 THEN 1 ELSE 0 END) AS m FROM t GROUP BY g`},
		{"chained CTE (b refs a)", `WITH a AS (SELECT 1 AS x), b AS (SELECT x FROM a) SELECT x FROM b`},
		{"DECLARE @var (Resolve→Parse)", `declare @id int = 7; SELECT x FROM t WHERE id = @id`},
		{"full Q1 (DECLARE + 3 chained CTEs)", q1},
	}
	for _, c := range cases {
		sql, rerr := batch.Resolve(c.sql)
		if rerr != nil {
			t.Logf("  MISSING  %-36s  resolve: %v", c.name, rerr)
			continue
		}
		if _, err := Parse(sql); err != nil {
			t.Logf("  MISSING  %-36s  %v", c.name, err)
		} else {
			t.Logf("  ok       %-36s", c.name)
		}
	}
}

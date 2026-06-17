// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

// sales amounts by region: E=[10,20], W=[5,30]. Median (0.5) per region.
func TestPercentileCont(t *testing.T) {
	got := perRegion(t, "SELECT region, PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY amount) OVER (PARTITION BY region) AS m FROM sales")
	if got["E"] != 15 { // interpolated 10..20
		t.Errorf("PERCENTILE_CONT E = %v, want 15", got["E"])
	}
	if got["W"] != 17.5 { // interpolated 5..30
		t.Errorf("PERCENTILE_CONT W = %v, want 17.5", got["W"])
	}
}

func TestPercentileDisc(t *testing.T) {
	got := perRegion(t, "SELECT region, PERCENTILE_DISC(0.5) WITHIN GROUP (ORDER BY amount) OVER (PARTITION BY region) AS m FROM sales")
	if got["E"] != 10 { // an actual value, not interpolated
		t.Errorf("PERCENTILE_DISC E = %v, want 10", got["E"])
	}
	if got["W"] != 5 {
		t.Errorf("PERCENTILE_DISC W = %v, want 5", got["W"])
	}
}

func perRegion(t *testing.T, sql string) map[string]float64 {
	t.Helper()
	m := map[string]float64{}
	for _, r := range qry(t, sql) {
		m[cell(r[0])] = num(r[1])
	}
	return m
}

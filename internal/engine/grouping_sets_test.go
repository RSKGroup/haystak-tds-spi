// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

// sales: E/A=10, E/B=20, W/A=30, W/A=5. region totals E=30 W=35; product totals A=45 B=20; grand=65.

func num(v any) float64 {
	switch n := v.(type) {
	case int64:
		return float64(n)
	case float64:
		return n
	}
	return -1
}

// salesAgg keys each result row by "region|product" ("_" for a rolled-up NULL) -> summed amount.
func salesAgg(t *testing.T, sql string) map[string]float64 {
	t.Helper()
	m := map[string]float64{}
	for _, r := range qry(t, sql) {
		key := nullKey(r[0]) + "|" + nullKey(r[1])
		m[key] = num(r[2])
	}
	return m
}

func nullKey(v any) string {
	if v == nil {
		return "_"
	}
	return cell(v)
}

func TestRollupSubtotals(t *testing.T) {
	got := salesAgg(t, "SELECT region, product, SUM(amount) FROM sales GROUP BY ROLLUP(region, product)")
	want := map[string]float64{"E|A": 10, "E|B": 20, "W|A": 35, "E|_": 30, "W|_": 35, "_|_": 65}
	if len(got) != len(want) {
		t.Fatalf("ROLLUP rows = %d (%v), want %d", len(got), got, len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("ROLLUP[%s] = %v, want %v", k, got[k], v)
		}
	}
}

func TestCubeSubtotals(t *testing.T) {
	got := salesAgg(t, "SELECT region, product, SUM(amount) FROM sales GROUP BY CUBE(region, product)")
	want := map[string]float64{"E|A": 10, "E|B": 20, "W|A": 35, "E|_": 30, "W|_": 35, "_|A": 45, "_|B": 20, "_|_": 65}
	if len(got) != len(want) {
		t.Fatalf("CUBE rows = %d (%v), want %d", len(got), got, len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("CUBE[%s] = %v, want %v", k, got[k], v)
		}
	}
}

func TestGroupingSetsExplicit(t *testing.T) {
	got := salesAgg(t, "SELECT region, product, SUM(amount) FROM sales GROUP BY GROUPING SETS((region),(product),())")
	want := map[string]float64{"E|_": 30, "W|_": 35, "_|A": 45, "_|B": 20, "_|_": 65}
	if len(got) != len(want) {
		t.Fatalf("GROUPING SETS rows = %d (%v), want %d", len(got), got, len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("GROUPING SETS[%s] = %v, want %v", k, got[k], v)
		}
	}
}

// GROUPING(col) is 1 on the row where col is rolled up to its subtotal/grand total, else 0.
func TestGroupingFunction(t *testing.T) {
	rows := qry(t, "SELECT region, GROUPING(region) AS g, SUM(amount) AS t FROM sales GROUP BY ROLLUP(region)")
	var grand, detail int
	for _, r := range rows {
		switch r[1].(int64) {
		case 1:
			grand++
			if r[0] != nil {
				t.Errorf("grand-total row region = %v, want NULL", r[0])
			}
			if num(r[2]) != 65 {
				t.Errorf("grand-total = %v, want 65", r[2])
			}
		case 0:
			detail++
		}
	}
	if grand != 1 || detail != 2 {
		t.Errorf("got %d grand-total + %d detail rows, want 1 + 2", grand, detail)
	}
}

// GROUPING_ID(a,b) is a bitmask: bit set when that column is rolled up (a is the high bit).
func TestGroupingIDBitmask(t *testing.T) {
	for _, r := range qry(t, "SELECT region, product, GROUPING_ID(region, product) AS gid, SUM(amount) FROM sales GROUP BY CUBE(region, product)") {
		gid := r[2].(int64)
		regionRolled := r[0] == nil
		productRolled := r[1] == nil
		var want int64
		if regionRolled {
			want |= 2
		}
		if productRolled {
			want |= 1
		}
		if gid != want {
			t.Errorf("GROUPING_ID for (region=%v, product=%v) = %d, want %d", r[0], r[1], gid, want)
		}
	}
}

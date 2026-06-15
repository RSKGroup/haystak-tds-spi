// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

// emps: (10,1,amy) (11,2,bob) (12,99,orphan). orders: amounts 100/200/50 for user_ids 1/2/2.

func TestWindowRowNumber(t *testing.T) {
	rows := qry(t, "SELECT id, ROW_NUMBER() OVER (ORDER BY id) AS rn FROM emps")
	want := []string{"1", "2", "3"}
	for i, r := range rows {
		if cell(r[1]) != want[i] {
			t.Errorf("row %d rn = %v, want %s", i, r[1], want[i])
		}
	}
}

func TestWindowPartition(t *testing.T) {
	// user 2 has amounts {200,50}; ordered ascending -> 50 is rn 1, 200 is rn 2.
	rows := qry(t, "SELECT user_id, amount, ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY amount) AS rn FROM orders")
	got := map[string]string{}
	for _, r := range rows {
		got[cell(r[0])+"/"+cell(r[1])] = cell(r[2])
	}
	if got["1/100"] != "1" || got["2/50"] != "1" || got["2/200"] != "2" {
		t.Errorf("partition row numbers = %v", got)
	}
}

func TestWindowRankDenseRank(t *testing.T) {
	// products price: Widget 9.99, Gadget 19.99 -> two product_tags rows share product 1 (sale,new) for ties.
	// Use orders amounts with a duplicate to exercise ties: 100,200,50 are distinct, so RANK==DENSE_RANK.
	rows := qry(t, "SELECT amount, RANK() OVER (ORDER BY amount) AS rk, DENSE_RANK() OVER (ORDER BY amount) AS dr FROM orders")
	for _, r := range rows {
		switch cell(r[0]) {
		case "50":
			if cell(r[1]) != "1" || cell(r[2]) != "1" {
				t.Errorf("amount 50 rank/dense = %v/%v, want 1/1", r[1], r[2])
			}
		case "100":
			if cell(r[1]) != "2" || cell(r[2]) != "2" {
				t.Errorf("amount 100 rank/dense = %v/%v, want 2/2", r[1], r[2])
			}
		case "200":
			if cell(r[1]) != "3" || cell(r[2]) != "3" {
				t.Errorf("amount 200 rank/dense = %v/%v, want 3/3", r[1], r[2])
			}
		}
	}
}

func TestWindowNtile(t *testing.T) {
	// 3 emps into 2 buckets: bucket 1 gets 2 rows (10,11), bucket 2 gets 1 (12).
	got := map[string]string{}
	for _, r := range qry(t, "SELECT id, NTILE(2) OVER (ORDER BY id) AS b FROM emps") {
		got[cell(r[0])] = cell(r[1])
	}
	if got["10"] != "1" || got["11"] != "1" || got["12"] != "2" {
		t.Errorf("NTILE(2) = %v, want 10->1 11->1 12->2", got)
	}
}

func TestWindowFirstLastValue(t *testing.T) {
	for _, r := range qry(t, "SELECT id, FIRST_VALUE(id) OVER (ORDER BY id) AS fv, LAST_VALUE(id) OVER (ORDER BY id) AS lv FROM emps") {
		if cell(r[1]) != "10" || cell(r[2]) != "12" {
			t.Errorf("id %v first/last = %v/%v, want 10/12", r[0], r[1], r[2])
		}
	}
}

func TestWindowLagLead(t *testing.T) {
	rows := qry(t, "SELECT id, LAG(id) OVER (ORDER BY id) AS lg, LEAD(id) OVER (ORDER BY id) AS ld FROM emps")
	// id 10: lag NULL, lead 11; id 11: lag 10, lead 12; id 12: lag 11, lead NULL.
	exp := map[string][2]string{"10": {"<nil>", "11"}, "11": {"10", "12"}, "12": {"11", "<nil>"}}
	for _, r := range rows {
		e := exp[cell(r[0])]
		if cell(r[1]) != e[0] || cell(r[2]) != e[1] {
			t.Errorf("id %v lag/lead = %v/%v, want %v/%v", r[0], r[1], r[2], e[0], e[1])
		}
	}
}

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

// vActiveProducts = "SELECT product_id, name, price FROM products WHERE in_stock = 1".
func TestViewColumnUsage(t *testing.T) {
	rows := qry(t, "SELECT TABLE_NAME, COLUMN_NAME FROM INFORMATION_SCHEMA.VIEW_COLUMN_USAGE WHERE VIEW_NAME = 'vActiveProducts' ORDER BY COLUMN_NAME")
	got := map[string]bool{}
	for _, r := range rows {
		if cell(r[0]) != "products" {
			t.Errorf("TABLE_NAME = %v, want products", r[0])
		}
		got[cell(r[1])] = true
	}
	for _, c := range []string{"product_id", "name", "price", "in_stock"} {
		if !got[c] {
			t.Errorf("VIEW_COLUMN_USAGE missing column %q (got %v)", c, got)
		}
	}
	if got["sku"] || got["weight"] {
		t.Errorf("VIEW_COLUMN_USAGE reported unreferenced columns: %v", got)
	}
}

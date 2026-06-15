// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

// A3: catalog/metadata scalars (OBJECT_NAME, COL_*, *PROPERTY) resolve in every predicate clause now that
// the evaluators thread *Env, not just in SELECT/ORDER BY. products has 2 rows; product_tags has 3.

func TestCatalogScalarInWhere(t *testing.T) {
	if n := len(qry(t, "SELECT name FROM products WHERE OBJECT_NAME(OBJECT_ID('products')) = 'products'")); n != 2 {
		t.Errorf("WHERE OBJECT_NAME = %d rows, want 2 (scalar should resolve, not NULL)", n)
	}
	if n := len(qry(t, "SELECT name FROM products WHERE OBJECT_NAME(OBJECT_ID('products')) = 'nope'")); n != 0 {
		t.Errorf("WHERE OBJECT_NAME mismatch = %d rows, want 0 (proves real comparison)", n)
	}
}

func TestCatalogScalarInCaseWhenCond(t *testing.T) {
	if n := len(qry(t, "SELECT name FROM products WHERE CASE WHEN COL_LENGTH('products','sku') > 0 THEN 1 ELSE 0 END = 1")); n != 2 {
		t.Errorf("CASE WHEN COL_LENGTH = %d rows, want 2", n)
	}
}

func TestCatalogScalarInJoinOn(t *testing.T) {
	const q = "SELECT p.name FROM products p JOIN product_tags t ON t.product_id = p.product_id AND OBJECT_NAME(OBJECT_ID('products')) = 'products'"
	if n := len(qry(t, q)); n != 3 {
		t.Errorf("JOIN ON OBJECT_NAME = %d rows, want 3", n)
	}
}

func TestCatalogScalarInHaving(t *testing.T) {
	const q = "SELECT name, COUNT(*) AS c FROM products GROUP BY name HAVING OBJECT_NAME(OBJECT_ID('products')) = 'products'"
	if n := len(qry(t, q)); n != 2 {
		t.Errorf("HAVING OBJECT_NAME = %d rows, want 2", n)
	}
}

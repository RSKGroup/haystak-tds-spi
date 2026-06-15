// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

// products has PK_products (clustered, unique) and UX_products_sku (unique); product_id is column 1.
func TestIndexPropertyResolvesRealIndex(t *testing.T) {
	r := qry(t, "SELECT INDEXPROPERTY(OBJECT_ID('products'),'PK_products','IsClustered'), "+
		"INDEXPROPERTY(OBJECT_ID('products'),'UX_products_sku','IsUnique'), "+
		"INDEXKEY_PROPERTY(OBJECT_ID('products'),1,1,'ColumnId')")[0]
	if cell(r[0]) != "1" || cell(r[1]) != "1" || cell(r[2]) != "1" {
		t.Errorf("INDEXPROPERTY/INDEXKEY_PROPERTY = %v, want 1/1/1", r)
	}
}

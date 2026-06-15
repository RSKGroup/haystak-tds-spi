// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

func TestSpPkeys(t *testing.T) {
	rows := qry(t, "EXEC sp_pkeys 'product_tags'")
	if len(rows) != 2 {
		t.Fatalf("sp_pkeys product_tags = %v, want 2 PK columns", rows)
	}
	if cell(rows[0][3]) != "product_id" || cell(rows[0][4]) != "1" || cell(rows[1][3]) != "tag" || cell(rows[1][4]) != "2" {
		t.Errorf("sp_pkeys = %v, want product_id(1), tag(2)", rows)
	}
}

func TestSpFkeys(t *testing.T) {
	rows := qry(t, "EXEC sp_fkeys @fktable_name = 'product_tags'")
	if len(rows) != 1 {
		t.Fatalf("sp_fkeys product_tags = %v, want 1 FK column pair", rows)
	}
	// PKTABLE_NAME(2), PKCOLUMN_NAME(3), FKTABLE_NAME(6), FKCOLUMN_NAME(7)
	if cell(rows[0][2]) != "products" || cell(rows[0][3]) != "product_id" ||
		cell(rows[0][6]) != "product_tags" || cell(rows[0][7]) != "product_id" {
		t.Errorf("sp_fkeys = %v, want products.product_id <- product_tags.product_id", rows[0])
	}
}

func TestSpStatistics(t *testing.T) {
	got := map[string]bool{}
	for _, r := range qry(t, "EXEC sp_statistics 'products'") {
		got[cell(r[5])] = true // INDEX_NAME
	}
	if !got["PK_products"] || !got["UX_products_sku"] {
		t.Errorf("sp_statistics = %v, want PK_products + UX_products_sku", got)
	}
}

func TestSpSpecialColumns(t *testing.T) {
	rows := qry(t, "EXEC sp_special_columns 'products'")
	if len(rows) != 1 || cell(rows[0][1]) != "product_id" { // COLUMN_NAME
		t.Fatalf("sp_special_columns = %v, want product_id", rows)
	}
}

func TestSpStoredProcedures(t *testing.T) {
	got := map[string]bool{}
	for _, r := range qry(t, "EXEC sp_stored_procedures") {
		got[cell(r[2])] = true // PROCEDURE_NAME
	}
	if !got["uspGetUser;1"] || !got["uspAddOrder;1"] || !got["ufnPriceWithTax;1"] {
		t.Errorf("sp_stored_procedures = %v", got)
	}
}

func TestSpSprocColumns(t *testing.T) {
	var got []string
	for _, r := range qry(t, "EXEC sp_sproc_columns 'uspAddOrder'") {
		got = append(got, cell(r[3])) // COLUMN_NAME
	}
	if len(got) != 2 || got[0] != "@user_id" || got[1] != "@amount" {
		t.Errorf("sp_sproc_columns = %v, want @user_id + @amount", got)
	}
}

func TestSpHelpindex(t *testing.T) {
	got := map[string]string{}
	for _, r := range qry(t, "EXEC sp_helpindex 'products'") {
		got[cell(r[0])] = cell(r[2]) // index_name -> index_keys
	}
	if got["PK_products"] != "product_id" || got["UX_products_sku"] != "sku" {
		t.Errorf("sp_helpindex = %v", got)
	}
}

func TestSpHelpconstraint(t *testing.T) {
	kinds := map[string]bool{}
	for _, r := range qry(t, "EXEC sp_helpconstraint 'product_tags'") {
		kinds[cell(r[0])] = true // constraint_type
	}
	if !kinds["PRIMARY KEY (clustered)"] || !kinds["FOREIGN KEY"] {
		t.Errorf("sp_helpconstraint = %v, want PK + FK", kinds)
	}
}

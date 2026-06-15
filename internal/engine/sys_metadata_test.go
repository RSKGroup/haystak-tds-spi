// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
)

func qry(t *testing.T, sql string) [][]any {
	t.Helper()
	rs, err := engine.Query(context.Background(), inmem.New(), sql)
	if err != nil {
		t.Fatalf("query %q: %v", sql, err)
	}
	return collect(t, rs)
}

func cell(v any) string { return fmt.Sprintf("%v", v) }

func TestSysIndexes(t *testing.T) {
	rows := qry(t, "SELECT name, type_desc, is_unique, is_primary_key FROM sys.indexes WHERE object_id = OBJECT_ID('products') ORDER BY index_id")
	if len(rows) != 2 {
		t.Fatalf("products indexes = %v, want 2 (PK + unique sku)", rows)
	}
	if cell(rows[0][0]) != "PK_products" || strings.TrimSpace(cell(rows[0][1])) != "CLUSTERED" ||
		cell(rows[0][2]) != "1" || cell(rows[0][3]) != "1" {
		t.Errorf("index[0] = %v, want PK_products CLUSTERED unique pk", rows[0])
	}
	if cell(rows[1][0]) != "UX_products_sku" || strings.TrimSpace(cell(rows[1][1])) != "NONCLUSTERED" ||
		cell(rows[1][2]) != "1" || cell(rows[1][3]) != "0" {
		t.Errorf("index[1] = %v, want UX_products_sku NONCLUSTERED unique non-pk", rows[1])
	}
}

func TestSysIndexesSynthesizesPK(t *testing.T) {
	// product_tags declares a composite PrimaryKey but no explicit Index — the PK index is synthesized.
	rows := qry(t, "SELECT name FROM sys.indexes WHERE object_id = OBJECT_ID('product_tags')")
	if len(rows) != 1 || cell(rows[0][0]) != "PK_product_tags" {
		t.Fatalf("product_tags indexes = %v, want synthesized PK_product_tags", rows)
	}
	cols := qry(t, "SELECT column_id, key_ordinal FROM sys.index_columns WHERE object_id = OBJECT_ID('product_tags') ORDER BY key_ordinal")
	if len(cols) != 2 || cell(cols[0][0]) != "1" || cell(cols[1][0]) != "2" {
		t.Fatalf("product_tags PK columns = %v, want product_id(1) + tag(2)", cols)
	}
}

func TestSysKeyConstraints(t *testing.T) {
	got := map[string]string{}
	for _, r := range qry(t, "SELECT name, type FROM sys.key_constraints WHERE parent_object_id = OBJECT_ID('products')") {
		got[cell(r[0])] = strings.TrimSpace(cell(r[1]))
	}
	if got["PK_products"] != "PK" || got["UX_products_sku"] != "UQ" {
		t.Errorf("key_constraints = %v, want PK_products=PK, UX_products_sku=UQ", got)
	}
}

func TestSysForeignKeyColumns(t *testing.T) {
	rows := qry(t, "SELECT parent_column_id, referenced_object_id, referenced_column_id FROM sys.foreign_key_columns WHERE parent_object_id = OBJECT_ID('product_tags')")
	if len(rows) != 1 {
		t.Fatalf("product_tags FK columns = %v, want 1", rows)
	}
	if cell(rows[0][1]) != cell(objectID(t, "products")) {
		t.Errorf("referenced_object_id = %v, want OBJECT_ID(products)=%v", rows[0][1], objectID(t, "products"))
	}
	if cell(rows[0][0]) != "1" || cell(rows[0][2]) != "1" {
		t.Errorf("FK col pair = %v, want parent_column_id 1 -> referenced 1", rows[0])
	}
}

func TestSysCheckConstraints(t *testing.T) {
	rows := qry(t, "SELECT name, definition FROM sys.check_constraints WHERE parent_object_id = OBJECT_ID('products')")
	if len(rows) != 1 || cell(rows[0][0]) != "CK_products_price" || !strings.Contains(cell(rows[0][1]), ">=") {
		t.Fatalf("check_constraints = %v, want CK_products_price with >=", rows)
	}
}

func TestSysDefaultConstraints(t *testing.T) {
	rows := qry(t, "SELECT name, definition FROM sys.default_constraints WHERE parent_object_id = OBJECT_ID('products')")
	if len(rows) != 1 || cell(rows[0][0]) != "DF_products_created" || !strings.Contains(cell(rows[0][1]), "GETDATE") {
		t.Fatalf("default_constraints = %v, want DF_products_created GETDATE()", rows)
	}
}

func TestSysIdentityColumns(t *testing.T) {
	names := map[string]bool{}
	for _, r := range qry(t, "SELECT OBJECT_NAME(object_id) AS t, name FROM sys.identity_columns") {
		names[cell(r[0])+"."+cell(r[1])] = true
	}
	if !names["products.product_id"] || !names["categories.category_id"] {
		t.Errorf("identity_columns = %v, want products.product_id + categories.category_id", names)
	}
}

func TestSysComputedColumns(t *testing.T) {
	rows := qry(t, "SELECT name, definition FROM sys.computed_columns WHERE object_id = OBJECT_ID('products')")
	if len(rows) != 1 || cell(rows[0][0]) != "margin" || !strings.Contains(cell(rows[0][1]), "price") {
		t.Fatalf("computed_columns = %v, want margin computed from price", rows)
	}
}

func TestSysSqlExpressionDependencies(t *testing.T) {
	deps := func(view string) [][]any {
		return qry(t, "SELECT referenced_entity_name, referenced_id FROM sys.sql_expression_dependencies WHERE referencing_id = OBJECT_ID('"+view+"')")
	}
	// view-on-table: resolves to the table's object_id
	if d := deps("vActiveProducts"); len(d) != 1 || cell(d[0][0]) != "products" || cell(d[0][1]) != cell(objectID(t, "products")) {
		t.Fatalf("vActiveProducts deps = %v, want products with resolved id", d)
	}
	// view-on-view chain
	if d := deps("vPremiumProducts"); len(d) != 1 || cell(d[0][0]) != "vActiveProducts" {
		t.Fatalf("vPremiumProducts deps = %v, want vActiveProducts", d)
	}
	if d := deps("vOrderTotals"); len(d) != 1 || cell(d[0][0]) != "orders" {
		t.Fatalf("vOrderTotals deps = %v, want orders", d)
	}
}

// objectID resolves OBJECT_ID('name') through the engine so tests compare against the live id scheme.
func objectID(t *testing.T, name string) any {
	t.Helper()
	rows := qry(t, "SELECT OBJECT_ID('"+name+"')")
	if len(rows) != 1 {
		t.Fatalf("OBJECT_ID(%q) returned %v", name, rows)
	}
	return rows[0][0]
}

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

func TestSysTriggers(t *testing.T) {
	rows := qry(t, "SELECT name, type_desc, parent_id FROM sys.triggers")
	if len(rows) != 1 || cell(rows[0][0]) != "trgOrdersAudit" || strings.TrimSpace(cell(rows[0][1])) != "SQL_TRIGGER" {
		t.Fatalf("sys.triggers = %v, want trgOrdersAudit SQL_TRIGGER", rows)
	}
	if cell(rows[0][2]) != cell(objectID(t, "orders")) {
		t.Errorf("trgOrdersAudit parent_id = %v, want OBJECT_ID(orders)", cell(rows[0][2]))
	}
}

func TestSysExtendedProperties(t *testing.T) {
	if rows := qry(t, "SELECT class, name, value FROM sys.extended_properties"); len(rows) != 0 {
		t.Errorf("sys.extended_properties = %v, want empty set", rows)
	}
}

func TestSysAllObjectsAndColumns(t *testing.T) {
	objs := nameSet(t, "SELECT name FROM sys.all_objects")
	if !objs["products"] || !objs["vActiveProducts"] {
		t.Errorf("sys.all_objects = %v, want products + vActiveProducts", objs)
	}
	cols := qry(t, "SELECT name FROM sys.all_columns WHERE object_id = OBJECT_ID('products')")
	if len(cols) == 0 {
		t.Error("sys.all_columns returned no products columns")
	}
}

func TestSysSequencesSynonymsEmpty(t *testing.T) {
	if rows := qry(t, "SELECT name FROM sys.sequences"); len(rows) != 0 {
		t.Errorf("sys.sequences = %v, want empty", rows)
	}
	if rows := qry(t, "SELECT name FROM sys.synonyms"); len(rows) != 0 {
		t.Errorf("sys.synonyms = %v, want empty", rows)
	}
}

func TestSysLegacyCompatViews(t *testing.T) {
	got := map[string]string{}
	for _, r := range qry(t, "SELECT name, xtype FROM sys.sysobjects") {
		got[cell(r[0])] = cell(r[1])
	}
	if got["products"] != "U" || got["vActiveProducts"] != "V" {
		t.Errorf("sys.sysobjects = %v, want products=U, vActiveProducts=V", got)
	}
	if cols := qry(t, "SELECT name FROM sys.syscolumns WHERE id = OBJECT_ID('products')"); len(cols) == 0 {
		t.Error("sys.syscolumns returned no products columns")
	}
	if types := nameSet(t, "SELECT name FROM sys.systypes"); !types["int"] || !types["nvarchar"] {
		t.Errorf("sys.systypes = %v, want int + nvarchar", types)
	}
}

func TestInfoSchemaConstraintTableUsageAndDomains(t *testing.T) {
	got := map[string]bool{}
	for _, r := range qry(t, "SELECT TABLE_NAME, CONSTRAINT_NAME FROM INFORMATION_SCHEMA.CONSTRAINT_TABLE_USAGE") {
		got[cell(r[1])+":"+cell(r[0])] = true
	}
	if !got["PK_products:products"] || !got["FK_product_tags_products:products"] {
		t.Errorf("CONSTRAINT_TABLE_USAGE = %v", got)
	}
	if rows := qry(t, "SELECT DOMAIN_NAME FROM INFORMATION_SCHEMA.DOMAINS"); len(rows) != 0 {
		t.Errorf("DOMAINS = %v, want empty", rows)
	}
}

func TestSysTableTypes(t *testing.T) {
	// projection over the backend's declared table types (inmem seeds OrderLineType)
	rows := qry(t, "SELECT name, is_table_type, type_table_object_id FROM sys.table_types WHERE name = 'OrderLineType'")
	if len(rows) != 1 || cell(rows[0][1]) != "1" {
		t.Fatalf("sys.table_types = %v, want OrderLineType is_table_type=1", rows)
	}
	if cell(rows[0][2]) != cell(objectID(t, "OrderLineType")) {
		t.Errorf("type_table_object_id = %v, want OBJECT_ID(OrderLineType)", rows[0][2])
	}
}

func TestSysIndexesCompatAndPermissionsEmpty(t *testing.T) {
	got := map[string]bool{}
	for _, r := range qry(t, "SELECT name FROM sys.sysindexes WHERE id = OBJECT_ID('products')") {
		got[cell(r[0])] = true
	}
	if !got["PK_products"] || !got["UX_products_sku"] {
		t.Errorf("sys.sysindexes = %v, want PK_products + UX_products_sku", got)
	}
	if rows := qry(t, "SELECT permission_name FROM sys.database_permissions"); len(rows) != 0 {
		t.Errorf("sys.database_permissions = %v, want empty", rows)
	}
	if rows := qry(t, "SELECT permission_name FROM sys.server_permissions"); len(rows) != 0 {
		t.Errorf("sys.server_permissions = %v, want empty", rows)
	}
}

func TestSpConfigureAndLock(t *testing.T) {
	cfg := map[string]bool{}
	for _, r := range qry(t, "EXEC sp_configure") {
		cfg[cell(r[0])] = true
	}
	if !cfg["max degree of parallelism"] || !cfg["max server memory (MB)"] {
		t.Errorf("sp_configure = %v, want MAXDOP + max memory options", cfg)
	}
	if rows := qry(t, "EXEC sp_lock"); len(rows) != 0 {
		t.Errorf("sp_lock = %v, want empty", rows)
	}
}

func TestSpHelpdb(t *testing.T) {
	names := map[string]bool{}
	for _, r := range qry(t, "EXEC sp_helpdb") {
		names[cell(r[0])] = true
	}
	if !names["master"] || !names["model"] {
		t.Errorf("sp_helpdb = %v, want master + model", names)
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

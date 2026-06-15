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

func nameSet(t *testing.T, sql string) map[string]bool {
	t.Helper()
	rs, err := engine.Query(context.Background(), inmem.New(), sql)
	if err != nil {
		t.Fatalf("query %q: %v", sql, err)
	}
	m := map[string]bool{}
	for _, r := range collect(t, rs) {
		m[fmt.Sprintf("%v", r[0])] = true
	}
	return m
}

func TestInmemViewsAndProcedures(t *testing.T) {
	v := nameSet(t, "SELECT name FROM sys.views")
	if !v["vActiveProducts"] || !v["vOrderTotals"] || v["uspGetUser"] || v["ufnPriceWithTax"] {
		t.Errorf("sys.views = %v", v)
	}
	p := nameSet(t, "SELECT name FROM sys.procedures")
	if !p["uspGetUser"] || !p["uspAddOrder"] || p["vActiveProducts"] {
		t.Errorf("sys.procedures = %v", p)
	}
}

func TestInmemObjectsSpanAllKinds(t *testing.T) {
	rs, err := engine.Query(context.Background(), inmem.New(), "SELECT name, type_desc FROM sys.objects")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, r := range collect(t, rs) {
		got[fmt.Sprintf("%v", r[0])] = strings.TrimSpace(fmt.Sprintf("%v", r[1]))
	}
	want := map[string]string{
		"users":           "USER_TABLE",
		"products":        "USER_TABLE",
		"vActiveProducts": "VIEW",
		"uspGetUser":      "SQL_STORED_PROCEDURE",
		"ufnPriceWithTax": "SQL_SCALAR_FUNCTION",
		"trgOrdersAudit":  "SQL_TRIGGER",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("sys.objects[%s] = %q, want %q", k, got[k], v)
		}
	}
}

func TestInmemSqlModulesScriptOut(t *testing.T) {
	rs, err := engine.Query(context.Background(), inmem.New(),
		"SELECT definition FROM sys.sql_modules WHERE object_id = OBJECT_ID('vOrderTotals')")
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, rs)
	if len(got) != 1 {
		t.Fatalf("want 1 module row, got %d", len(got))
	}
	def := fmt.Sprintf("%v", got[0][0])
	if !strings.Contains(def, "CREATE VIEW [vOrderTotals]") || !strings.Contains(def, "SUM(amount)") {
		t.Errorf("definition not scriptable: %q", def)
	}
}

func TestInmemParameters(t *testing.T) {
	rs, err := engine.Query(context.Background(), inmem.New(),
		"SELECT name FROM sys.parameters WHERE object_id = OBJECT_ID('uspAddOrder')")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, r := range collect(t, rs) {
		got[fmt.Sprintf("%v", r[0])] = true
	}
	if !got["@user_id"] || !got["@amount"] {
		t.Errorf("sys.parameters(uspAddOrder) = %v, want @user_id + @amount", got)
	}
}

func TestInmemInformationSchemaViews(t *testing.T) {
	rs, err := engine.Query(context.Background(), inmem.New(),
		"SELECT VIEW_DEFINITION FROM INFORMATION_SCHEMA.VIEWS WHERE TABLE_NAME = 'vOrderTotals'")
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, rs)
	if len(got) != 1 || !strings.Contains(fmt.Sprintf("%v", got[0][0]), "SUM(amount)") {
		t.Fatalf("INFORMATION_SCHEMA.VIEWS = %v, want vOrderTotals definition", got)
	}
}

func TestInmemInformationSchemaRoutines(t *testing.T) {
	rs, err := engine.Query(context.Background(), inmem.New(),
		"SELECT ROUTINE_NAME, ROUTINE_TYPE FROM INFORMATION_SCHEMA.ROUTINES")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, r := range collect(t, rs) {
		got[fmt.Sprintf("%v", r[0])] = fmt.Sprintf("%v", r[1])
	}
	if got["uspGetUser"] != "PROCEDURE" || got["ufnPriceWithTax"] != "FUNCTION" {
		t.Errorf("ROUTINES = %v", got)
	}
	if _, ok := got["vActiveProducts"]; ok {
		t.Errorf("ROUTINES should exclude views, got %v", got)
	}
}

func TestInmemInformationSchemaParameters(t *testing.T) {
	rs, err := engine.Query(context.Background(), inmem.New(),
		"SELECT PARAMETER_NAME, DATA_TYPE FROM INFORMATION_SCHEMA.PARAMETERS WHERE SPECIFIC_NAME = 'ufnPriceWithTax'")
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, rs)
	if len(got) != 1 || fmt.Sprintf("%v", got[0][0]) != "@price" || fmt.Sprintf("%v", got[0][1]) != "decimal" {
		t.Fatalf("PARAMETERS = %v, want [@price decimal]", got)
	}
}

func TestInmemSpHelptext(t *testing.T) {
	rs, err := engine.Query(context.Background(), inmem.New(), "EXEC sp_helptext 'vOrderTotals'")
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, r := range collect(t, rs) {
		lines = append(lines, fmt.Sprintf("%v", r[0]))
	}
	full := strings.Join(lines, "\n")
	if len(lines) < 2 || !strings.Contains(full, "CREATE VIEW [vOrderTotals]") || !strings.Contains(full, "SUM(amount)") {
		t.Fatalf("sp_helptext = %q", full)
	}
}

func TestInmemSpHelpNamedAndList(t *testing.T) {
	rs, err := engine.Query(context.Background(), inmem.New(), "EXEC sp_help 'uspGetUser'")
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, rs)
	if len(got) != 1 || fmt.Sprintf("%v", got[0][0]) != "uspGetUser" || fmt.Sprintf("%v", got[0][2]) != "stored procedure" {
		t.Fatalf("sp_help 'uspGetUser' = %v", got)
	}

	rs, err = engine.Query(context.Background(), inmem.New(), "sp_help")
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]string{}
	for _, r := range collect(t, rs) {
		kinds[fmt.Sprintf("%v", r[0])] = fmt.Sprintf("%v", r[2])
	}
	if kinds["users"] != "user table" || kinds["uspGetUser"] != "stored procedure" || kinds["ufnPriceWithTax"] != "scalar function" {
		t.Errorf("sp_help list = %v", kinds)
	}
}

func TestInmemCreateViewSurfacesAndExpands(t *testing.T) {
	ctx := context.Background()
	b := inmem.New()
	if _, _, err := engine.Exec(ctx, b, "CREATE VIEW vTwoUsers AS SELECT id FROM users"); err != nil {
		t.Fatalf("CREATE VIEW on inmem (now a RoutineStore): %v", err)
	}
	rs, err := engine.Query(ctx, b, "SELECT name FROM sys.views WHERE name = 'vTwoUsers'")
	if err != nil {
		t.Fatal(err)
	}
	if got := collect(t, rs); len(got) != 1 {
		t.Fatalf("sys.views after CREATE = %v, want vTwoUsers visible", got)
	}
	rs, err = engine.Query(ctx, b, "SELECT id FROM vTwoUsers ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, rs)
	if len(got) != 2 || fmt.Sprintf("%v", got[0][0]) != "1" || fmt.Sprintf("%v", got[1][0]) != "2" {
		t.Fatalf("view expansion = %v, want ids 1,2", got)
	}
}

func TestInfoSchemaSchemata(t *testing.T) {
	got := nameSet(t, "SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA")
	if !got["dbo"] || !got["sys"] || !got["INFORMATION_SCHEMA"] {
		t.Errorf("SCHEMATA = %v, want dbo + sys + INFORMATION_SCHEMA", got)
	}
}

func TestInfoSchemaCheckConstraints(t *testing.T) {
	rows := qry(t, "SELECT CONSTRAINT_NAME, CHECK_CLAUSE FROM INFORMATION_SCHEMA.CHECK_CONSTRAINTS WHERE CONSTRAINT_NAME = 'CK_products_price'")
	if len(rows) != 1 || !strings.Contains(cell(rows[0][1]), "price") {
		t.Fatalf("CHECK_CONSTRAINTS = %v, want CK_products_price referencing price", rows)
	}
}

func TestInfoSchemaConstraintColumnUsage(t *testing.T) {
	got := map[string]bool{}
	for _, r := range qry(t, "SELECT TABLE_NAME, COLUMN_NAME, CONSTRAINT_NAME FROM INFORMATION_SCHEMA.CONSTRAINT_COLUMN_USAGE") {
		got[cell(r[2])+":"+cell(r[0])+"."+cell(r[1])] = true
	}
	if !got["PK_products:products.product_id"] {
		t.Errorf("missing PK column usage in %v", got)
	}
	if !got["FK_product_tags_products:products.product_id"] { // FK reports the referenced table's column
		t.Errorf("missing FK column usage in %v", got)
	}
	if !got["CK_products_price:products.price"] {
		t.Errorf("missing CHECK column usage in %v", got)
	}
}

func TestInfoSchemaViewTableUsage(t *testing.T) {
	got := map[string]string{}
	for _, r := range qry(t, "SELECT VIEW_NAME, TABLE_NAME FROM INFORMATION_SCHEMA.VIEW_TABLE_USAGE") {
		got[cell(r[0])] = cell(r[1])
	}
	if got["vActiveProducts"] != "products" || got["vOrderTotals"] != "orders" {
		t.Errorf("VIEW_TABLE_USAGE = %v, want vActiveProducts->products, vOrderTotals->orders", got)
	}
	if _, ok := got["vPremiumProducts"]; ok { // references a view, not a base table
		t.Errorf("vPremiumProducts should surface no base table, got %v", got["vPremiumProducts"])
	}
}

func TestInfoSchemaRoutineColumns(t *testing.T) {
	if rows := qry(t, "SELECT TABLE_NAME, COLUMN_NAME FROM INFORMATION_SCHEMA.ROUTINE_COLUMNS"); len(rows) != 0 {
		t.Errorf("ROUTINE_COLUMNS = %v, want empty set (no table-valued functions)", rows)
	}
}

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package sysviews_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/functions"
	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/sysviews"
	"github.com/RSKGroup/haystak-tds-spi/internal/tsql"
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

func fixture() (catalog.Schema, []*tds.Routine, []string) {
	schema := catalog.Schema{Tables: []catalog.Table{{
		Name: "Customer",
		Columns: []catalog.Column{
			{Name: "id", Type: types.Type{Kind: types.Int64}},
			{Name: "active", Type: types.Type{Kind: types.Int32}},
		},
	}}}
	rts := []*tds.Routine{
		{Name: "ActiveCustomers", Kind: tds.RoutineView, Body: "SELECT * FROM Customer WHERE active = 1"},
		{Name: "GetCustomer", Kind: tds.RoutineProc, Params: []tds.RoutineParam{{Name: "@id", Type: "int"}},
			Body: "SELECT * FROM Customer WHERE id = @id"},
	}
	return schema, rts, []string{"shop"}
}

func run(t *testing.T, sql string) [][]any { return runAs(t, tds.Principal{}, sql) }

func runAs(t *testing.T, p tds.Principal, sql string) [][]any {
	t.Helper()
	schema, rts, dbs := fixture()
	q, err := tsql.Parse(sql)
	if err != nil {
		t.Fatalf("parse %q: %v", sql, err)
	}
	rows, handled, err := sysviews.Resolve(schema, rts, dbs, p, nil, 0, q)
	if err != nil {
		t.Fatalf("resolve %q: %v", sql, err)
	}
	if !handled {
		t.Fatalf("not handled: %q", sql)
	}
	var out [][]any
	for rows.Next() {
		v, err := rows.Values()
		if err != nil {
			t.Fatalf("values: %v", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err: %v", err)
	}
	return out
}

func str(v any) string { return fmt.Sprintf("%v", v) }

// set collects column c of every row into a string set.
func set(rows [][]any, c int) map[string]bool {
	m := map[string]bool{}
	for _, r := range rows {
		m[str(r[c])] = true
	}
	return m
}

func TestObjectNameReversesSqlModules(t *testing.T) {
	got := set(run(t, "SELECT OBJECT_NAME(object_id) AS n FROM sys.sql_modules"), 0)
	for _, want := range []string{"ActiveCustomers", "GetCustomer"} {
		if !got[want] {
			t.Errorf("OBJECT_NAME missing %q in %v", want, got)
		}
	}
}

func TestSqlModulesScriptsOutDefinition(t *testing.T) {
	rows := run(t, "SELECT definition FROM sys.sql_modules WHERE object_id = OBJECT_ID('ActiveCustomers')")
	if len(rows) != 1 {
		t.Fatalf("want 1 row joined by OBJECT_ID, got %d", len(rows))
	}
	def := str(rows[0][0])
	if !strings.Contains(def, "CREATE VIEW [ActiveCustomers]") || !strings.Contains(def, "active = 1") {
		t.Errorf("definition not scriptable: %q", def)
	}
}

func TestObjectIDMatchesSqlModulesColumn(t *testing.T) {
	rows := run(t, "SELECT object_id FROM sys.sql_modules WHERE object_id = OBJECT_ID('GetCustomer')")
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if got, want := rows[0][0].(int64), functions.ObjectID("GetCustomer"); got != want {
		t.Errorf("object_id = %d, want %d", got, want)
	}
}

func TestViewsAndProcedures(t *testing.T) {
	if v := set(run(t, "SELECT name FROM sys.views"), 0); !v["ActiveCustomers"] || v["GetCustomer"] {
		t.Errorf("sys.views = %v", v)
	}
	if p := set(run(t, "SELECT name FROM sys.procedures"), 0); !p["GetCustomer"] || p["ActiveCustomers"] {
		t.Errorf("sys.procedures = %v", p)
	}
}

func TestObjectsSpansTablesViewsProcs(t *testing.T) {
	rows := run(t, "SELECT name, type_desc FROM sys.objects")
	got := map[string]string{}
	for _, r := range rows {
		got[str(r[0])] = strings.TrimSpace(str(r[1]))
	}
	want := map[string]string{"Customer": "USER_TABLE", "ActiveCustomers": "VIEW", "GetCustomer": "SQL_STORED_PROCEDURE"}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("sys.objects[%s] = %q, want %q", k, got[k], v)
		}
	}
}

func TestParametersWithTypeNameAndSchema(t *testing.T) {
	rows := run(t, "SELECT name, TYPE_NAME(system_type_id) AS t, OBJECT_SCHEMA_NAME(object_id) AS s FROM sys.parameters")
	if len(rows) != 1 {
		t.Fatalf("want 1 parameter, got %d", len(rows))
	}
	if n, ty, sc := str(rows[0][0]), str(rows[0][1]), str(rows[0][2]); n != "@id" || ty != "int" || sc != "dbo" {
		t.Errorf("parameter = (%q,%q,%q), want (@id,int,dbo)", n, ty, sc)
	}
}

func TestDBNameReversesDatabaseID(t *testing.T) {
	rows := run(t, "SELECT DB_NAME(database_id) AS dn FROM sys.databases WHERE name = 'shop'")
	if len(rows) != 1 || str(rows[0][0]) != "shop" {
		t.Fatalf("DB_NAME(database_id) = %v, want [shop]", rows)
	}
}

func TestDatabasePrincipalsReflectLiveIdentity(t *testing.T) {
	p := tds.Principal{Username: "alice", Roles: []string{"db_datareader"}}
	names := map[string]string{}
	for _, r := range runAs(t, p, "SELECT name, type_desc FROM sys.database_principals") {
		names[fmt.Sprintf("%v", r[0])] = fmt.Sprintf("%v", r[1])
	}
	if names["dbo"] != "SQL_USER" || names["alice"] != "SQL_USER" || names["db_datareader"] != "DATABASE_ROLE" {
		t.Errorf("database_principals = %v, want dbo + alice (user) + db_datareader (role)", names)
	}
	// role membership links the role to the live user
	mem := runAs(t, p, "SELECT role_principal_id, member_principal_id FROM sys.database_role_members")
	if len(mem) != 1 || fmt.Sprintf("%v", mem[0][1]) != "5" {
		t.Errorf("database_role_members = %v, want one row for member id 5 (alice)", mem)
	}
}

func TestPrincipalsDefaultWhenUnauthenticated(t *testing.T) {
	// empty Principal -> well-knowns + the default "haystak" user, no roles
	names := map[string]bool{}
	for _, r := range run(t, "SELECT name FROM sys.database_principals") {
		names[fmt.Sprintf("%v", r[0])] = true
	}
	if !names["dbo"] || !names["public"] || !names["haystak"] {
		t.Errorf("database_principals = %v, want dbo + public + haystak", names)
	}
	if rows := run(t, "SELECT role_principal_id FROM sys.database_role_members"); len(rows) != 0 {
		t.Errorf("database_role_members = %v, want empty (no roles)", rows)
	}
}

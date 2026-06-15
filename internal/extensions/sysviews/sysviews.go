// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package sysviews

import (
	"fmt"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/internal/exec"
	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/functions"
	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/routines"
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

const dbName = "haystak"

// Resolve answers a query against sys.* catalog views from a backend's declared schema and stored
// routines. Returns handled=false when the query does not target the sys schema.
func Resolve(schema catalog.Schema, rts []*tds.Routine, dbs []string, q *tds.Query) (tds.Rows, bool, error) {
	if !strings.EqualFold(q.Schema, "sys") {
		return nil, false, nil
	}
	var cols []catalog.Column
	var data [][]any
	switch strings.ToLower(q.Table) {
	case "databases":
		cols, data = databasesRows(dbs)
	case "schemas":
		cols, data = schemasRows()
	case "tables":
		cols, data = tablesRows(schema)
	case "objects", "all_objects":
		cols, data = objectsRows(schema, rts)
	case "views":
		cols, data = viewsRows(rts)
	case "procedures":
		cols, data = proceduresRows(rts)
	case "sql_modules":
		cols, data = sqlModulesRows(rts)
	case "parameters":
		cols, data = parametersRows(rts)
	case "columns", "all_columns":
		cols, data = columnsRows(schema)
	case "types":
		cols, data = typesRows()
	case "foreign_keys":
		cols, data = foreignKeysRows(schema)
	case "indexes":
		cols, data = indexesRows(schema)
	case "index_columns":
		cols, data = indexColumnsRows(schema)
	case "key_constraints":
		cols, data = keyConstraintsRows(schema)
	case "foreign_key_columns":
		cols, data = foreignKeyColumnsRows(schema)
	case "check_constraints":
		cols, data = checkConstraintsRows(schema)
	case "default_constraints":
		cols, data = defaultConstraintsRows(schema)
	case "identity_columns":
		cols, data = identityColumnsRows(schema)
	case "computed_columns":
		cols, data = computedColumnsRows(schema)
	case "sql_expression_dependencies":
		cols, data = sqlExpressionDependenciesRows(schema, rts)
	case "triggers":
		cols, data = triggersRows(rts)
	case "extended_properties":
		cols, data = extendedPropertiesRows()
	case "sequences":
		cols, data = sequencesRows()
	case "synonyms":
		cols, data = synonymsRows()
	case "sysobjects":
		cols, data = sysobjectsRows(schema, rts)
	case "syscolumns":
		cols, data = syscolumnsRows(schema)
	case "systypes":
		cols, data = systypesRows()
	default:
		return nil, true, fmt.Errorf("sysviews: sys.%s not supported", q.Table)
	}
	obj, db := exec.CatalogResolvers(schema, rts, dbs)
	tbl, kind := exec.CatalogObjects(schema, rts)
	env := &exec.Env{ObjectName: obj, DBName: db, Table: tbl, ObjectKind: kind, RoutineDef: exec.RoutineDefs(rts), CurrentDB: q.Database}
	r, err := exec.ApplyWith(cols, data, q, env)
	return r, true, err
}

func databasesRows(dbs []string) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		sname("name"), intc("database_id"), intc("state"), sname("state_desc"),
		intc("is_read_only"), sname("collation_name"), intc("compatibility_level"),
	}
	mk := func(name string, id int64) []any {
		return []any{name, id, int64(0), "ONLINE", int64(0), "SQL_Latin1_General_CP1_CI_AS", int64(160)}
	}
	if len(dbs) == 0 { // single-database backend: keep reporting the default catalog
		dbs = []string{dbName}
	}
	rows := [][]any{mk("master", 1), mk("tempdb", 2), mk("model", 3), mk("msdb", 4)}
	for _, db := range dbs {
		rows = append(rows, mk(db, functions.DBID(db)))
	}
	return cols, rows
}

func schemasRows() ([]catalog.Column, [][]any) {
	cols := []catalog.Column{sname("name"), intc("schema_id"), intc("principal_id")}
	return cols, [][]any{
		{"dbo", int64(1), int64(1)},
		{"sys", int64(4), int64(4)},
		{"INFORMATION_SCHEMA", int64(3), int64(4)},
	}
}

func objectCols() []catalog.Column {
	return []catalog.Column{
		sname("name"), intc("object_id"), intc("schema_id"), sname("type"),
		sname("type_desc"), intc("is_ms_shipped"),
	}
}

func tablesRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	var rows [][]any
	for _, t := range schema.Tables {
		rows = append(rows, []any{t.Name, oid(t.Name), int64(1), "U ", "USER_TABLE", int64(0)})
	}
	return objectCols(), rows
}

// objectsRows is sys.objects: tables plus the stored views and procedures (sys.objects spans all object kinds).
func objectsRows(schema catalog.Schema, rts []*tds.Routine) ([]catalog.Column, [][]any) {
	var rows [][]any
	for _, t := range schema.Tables {
		rows = append(rows, []any{t.Name, oid(t.Name), int64(1), "U ", "USER_TABLE", int64(0)})
	}
	for _, r := range rts {
		typ, desc := routineTypeCodes(r.Kind)
		rows = append(rows, []any{r.Name, oid(r.Name), int64(1), typ, desc, int64(0)})
	}
	return objectCols(), rows
}

func viewsRows(rts []*tds.Routine) ([]catalog.Column, [][]any) {
	var rows [][]any
	for _, r := range rts {
		if r.Kind == tds.RoutineView {
			rows = append(rows, []any{r.Name, oid(r.Name), int64(1), "V ", "VIEW", int64(0)})
		}
	}
	return objectCols(), rows
}

func proceduresRows(rts []*tds.Routine) ([]catalog.Column, [][]any) {
	var rows [][]any
	for _, r := range rts {
		if r.Kind == tds.RoutineProc {
			rows = append(rows, []any{r.Name, oid(r.Name), int64(1), "P ", "SQL_STORED_PROCEDURE", int64(0)})
		}
	}
	return objectCols(), rows
}

// sqlModulesRows is sys.sql_modules: one row per routine carrying the reconstructed CREATE text in
// definition, so a client can script the object back out.
func sqlModulesRows(rts []*tds.Routine) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		intc("object_id"), sname("definition"),
		intc("uses_ansi_nulls"), intc("uses_quoted_identifier"), intc("is_schema_bound"),
		intc("uses_database_collation"), intc("is_recompiled"), intc("null_on_null_input"),
	}
	var rows [][]any
	for _, r := range rts {
		rows = append(rows, []any{
			oid(r.Name), routines.ScriptDefinition(r),
			int64(1), int64(1), int64(0), int64(1), int64(0), int64(0),
		})
	}
	return cols, rows
}

// parametersRows is sys.parameters: one row per procedure parameter, in declared order.
func parametersRows(rts []*tds.Routine) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		intc("object_id"), sname("name"), intc("parameter_id"),
		intc("system_type_id"), intc("user_type_id"), intc("max_length"),
		intc("precision"), intc("scale"), intc("is_output"),
	}
	var rows [][]any
	for _, r := range rts {
		for i, p := range r.Params {
			st := sysTypeIDByName(p.Type)
			rows = append(rows, []any{
				oid(r.Name), p.Name, int64(i + 1),
				st, st, int64(-1), int64(0), int64(0), int64(0),
			})
		}
	}
	return cols, rows
}

func columnsRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		intc("object_id"), sname("name"), intc("column_id"),
		intc("system_type_id"), intc("user_type_id"), intc("max_length"),
		intc("precision"), intc("scale"), intc("is_nullable"),
	}
	var rows [][]any
	for _, t := range schema.Tables {
		for j, c := range t.Columns {
			st := sysTypeID(c.Type)
			rows = append(rows, []any{
				oid(t.Name), c.Name, int64(j + 1),
				st, st, sysTypeLen(c.Type),
				int64(c.Type.Precision), int64(c.Type.Scale), boolInt(c.Type.Nullable),
			})
		}
	}
	return cols, rows
}

func typesRows() ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		sname("name"), intc("system_type_id"), intc("user_type_id"), intc("schema_id"),
		intc("max_length"), intc("precision"), intc("scale"), intc("is_nullable"), intc("is_user_defined"),
	}
	builtins := []struct {
		name string
		id   int64
		ml   int64
	}{
		{"bit", 104, 1}, {"tinyint", 48, 1}, {"smallint", 52, 2}, {"int", 56, 4}, {"bigint", 127, 8},
		{"decimal", 106, 17}, {"numeric", 108, 17}, {"float", 62, 8}, {"real", 59, 4},
		{"date", 40, 3}, {"time", 41, 5}, {"datetime", 61, 8}, {"datetime2", 42, 8},
		{"char", 175, -1}, {"varchar", 167, -1}, {"nchar", 239, -1}, {"nvarchar", 231, -1},
		{"binary", 173, -1}, {"varbinary", 165, -1}, {"uniqueidentifier", 36, 16},
	}
	var rows [][]any
	for _, b := range builtins {
		rows = append(rows, []any{b.name, b.id, b.id, int64(4), b.ml, int64(0), int64(0), int64(1), int64(0)})
	}
	return cols, rows
}

func foreignKeysRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		sname("name"), intc("object_id"), intc("parent_object_id"), intc("referenced_object_id"),
		intc("schema_id"), sname("type"), sname("type_desc"), intc("is_disabled"),
	}
	idxOf := map[string]int{}
	for i, t := range schema.Tables {
		idxOf[strings.ToLower(t.Name)] = i
	}
	var rows [][]any
	fkID := int64(200)
	for _, t := range schema.Tables {
		for _, fk := range t.ForeignKeys {
			refOID := int64(0)
			if ri, ok := idxOf[strings.ToLower(fk.RefTable)]; ok {
				refOID = oid(schema.Tables[ri].Name)
			}
			rows = append(rows, []any{
				"FK_" + t.Name + "_" + fk.RefTable, fkID, oid(t.Name), refOID,
				int64(1), "F ", "FOREIGN_KEY_CONSTRAINT", int64(0),
			})
			fkID++
		}
	}
	return cols, rows
}

// TableIndexes returns a table's declared indexes, synthesizing the clustered PK index from PrimaryKey
// when no explicit primary index was declared (SQL Server always backs a PK with an index).
func TableIndexes(t catalog.Table) []catalog.Index {
	idxs := append([]catalog.Index{}, t.Indexes...)
	for _, ix := range idxs {
		if ix.Primary {
			return idxs
		}
	}
	if len(t.PrimaryKey) > 0 {
		pk := catalog.Index{Name: "PK_" + t.Name, Columns: t.PrimaryKey, Unique: true, Primary: true, Clustered: true}
		idxs = append([]catalog.Index{pk}, idxs...)
	}
	return idxs
}

func indexesRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		intc("object_id"), sname("name"), intc("index_id"), intc("type"), sname("type_desc"),
		intc("is_unique"), intc("is_primary_key"), intc("is_unique_constraint"), intc("is_disabled"),
	}
	var rows [][]any
	for _, t := range schema.Tables {
		for i, ix := range TableIndexes(t) {
			typ, desc := int64(2), "NONCLUSTERED"
			if ix.Clustered {
				typ, desc = 1, "CLUSTERED"
			}
			rows = append(rows, []any{
				oid(t.Name), ix.Name, int64(i + 1), typ, desc,
				boolInt(ix.Unique), boolInt(ix.Primary), boolInt(ix.Unique && !ix.Primary), int64(0),
			})
		}
	}
	return cols, rows
}

func indexColumnsRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		intc("object_id"), intc("index_id"), intc("index_column_id"), intc("column_id"),
		intc("key_ordinal"), intc("is_descending_key"), intc("is_included_column"),
	}
	var rows [][]any
	for _, t := range schema.Tables {
		for i, ix := range TableIndexes(t) {
			for j, c := range ix.Columns {
				rows = append(rows, []any{
					oid(t.Name), int64(i + 1), int64(j + 1), colID(t, c), int64(j + 1), int64(0), int64(0),
				})
			}
		}
	}
	return cols, rows
}

func keyConstraintsRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		sname("name"), intc("object_id"), intc("parent_object_id"), intc("schema_id"),
		sname("type"), sname("type_desc"), intc("unique_index_id"),
	}
	var rows [][]any
	for _, t := range schema.Tables {
		for i, ix := range TableIndexes(t) {
			switch {
			case ix.Primary:
				rows = append(rows, []any{ix.Name, oid(ix.Name), oid(t.Name), int64(1), "PK", "PRIMARY_KEY_CONSTRAINT", int64(i + 1)})
			case ix.Unique:
				rows = append(rows, []any{ix.Name, oid(ix.Name), oid(t.Name), int64(1), "UQ", "UNIQUE_CONSTRAINT", int64(i + 1)})
			}
		}
	}
	return cols, rows
}

func foreignKeyColumnsRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		intc("constraint_object_id"), intc("constraint_column_id"), intc("parent_object_id"),
		intc("parent_column_id"), intc("referenced_object_id"), intc("referenced_column_id"),
	}
	byName := tableByName(schema)
	var rows [][]any
	for _, t := range schema.Tables {
		for _, fk := range t.ForeignKeys {
			cname := "FK_" + t.Name + "_" + fk.RefTable
			ref := byName[strings.ToLower(fk.RefTable)]
			for k, c := range fk.Columns {
				var refOID, refColID int64
				if ref != nil {
					refOID = oid(ref.Name)
					if k < len(fk.RefColumns) {
						refColID = colID(*ref, fk.RefColumns[k])
					}
				}
				rows = append(rows, []any{oid(cname), int64(k + 1), oid(t.Name), colID(t, c), refOID, refColID})
			}
		}
	}
	return cols, rows
}

func checkConstraintsRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		sname("name"), intc("object_id"), intc("parent_object_id"), intc("schema_id"),
		sname("type"), sname("type_desc"), sname("definition"), intc("is_disabled"),
	}
	var rows [][]any
	for _, t := range schema.Tables {
		for _, ck := range t.Checks {
			rows = append(rows, []any{ck.Name, oid(ck.Name), oid(t.Name), int64(1), "C", "CHECK_CONSTRAINT", ck.Expression, int64(0)})
		}
	}
	return cols, rows
}

func defaultConstraintsRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		sname("name"), intc("object_id"), intc("parent_object_id"), intc("parent_column_id"),
		intc("schema_id"), sname("type"), sname("type_desc"), sname("definition"),
	}
	var rows [][]any
	for _, t := range schema.Tables {
		for j, c := range t.Columns {
			if c.Default == "" {
				continue
			}
			name := "DF_" + t.Name + "_" + c.Name
			rows = append(rows, []any{name, oid(name), oid(t.Name), int64(j + 1), int64(1), "D", "DEFAULT_CONSTRAINT", c.Default})
		}
	}
	return cols, rows
}

func identityColumnsRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		intc("object_id"), sname("name"), intc("column_id"), intc("system_type_id"),
		intc("is_identity"), intc("seed_value"), intc("increment_value"), nintc("last_value"),
	}
	var rows [][]any
	for _, t := range schema.Tables {
		for j, c := range t.Columns {
			if !c.Identity {
				continue
			}
			rows = append(rows, []any{oid(t.Name), c.Name, int64(j + 1), sysTypeID(c.Type), int64(1), int64(1), int64(1), nil})
		}
	}
	return cols, rows
}

func computedColumnsRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		intc("object_id"), sname("name"), intc("column_id"), sname("definition"),
		intc("is_persisted"), intc("is_computed"),
	}
	var rows [][]any
	for _, t := range schema.Tables {
		for j, c := range t.Columns {
			if c.Computed == "" {
				continue
			}
			rows = append(rows, []any{oid(t.Name), c.Name, int64(j + 1), c.Computed, int64(0), int64(1)})
		}
	}
	return cols, rows
}

// sqlExpressionDependenciesRows is sys.sql_expression_dependencies: view -> table/view references
// parsed from each view body's FROM/JOIN clauses; referenced_id resolves when the target is known.
func sqlExpressionDependenciesRows(schema catalog.Schema, rts []*tds.Routine) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		intc("referencing_id"), intc("referencing_minor_id"), intc("referencing_class"), sname("referencing_class_desc"),
		intc("referenced_class"), sname("referenced_class_desc"), nsname("referenced_server_name"), nsname("referenced_database_name"),
		nsname("referenced_schema_name"), sname("referenced_entity_name"), nintc("referenced_id"), intc("referenced_minor_id"),
		intc("is_caller_dependent"), intc("is_ambiguous"),
	}
	known := objectNameSet(schema, rts)
	var rows [][]any
	for _, r := range rts {
		if r.Kind != tds.RoutineView {
			continue
		}
		for _, name := range routines.ReferencedNames(r.Body) {
			var refID, refSchema any
			if known[strings.ToLower(name)] {
				refID, refSchema = oid(name), "dbo"
			}
			rows = append(rows, []any{
				oid(r.Name), int64(0), int64(1), "OBJECT_OR_COLUMN",
				int64(1), "OBJECT_OR_COLUMN", nil, nil,
				refSchema, name, refID, int64(0),
				int64(0), int64(0),
			})
		}
	}
	return cols, rows
}

// triggersRows is sys.triggers: one row per trigger, parent_id resolved from the body's ON <table>.
func triggersRows(rts []*tds.Routine) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		sname("name"), intc("object_id"), intc("parent_class"), sname("parent_class_desc"),
		intc("parent_id"), sname("type"), sname("type_desc"), intc("is_ms_shipped"),
		intc("is_disabled"), intc("is_instead_of_trigger"),
	}
	var rows [][]any
	for _, r := range rts {
		if r.Kind != tds.RoutineTrigger {
			continue
		}
		rows = append(rows, []any{
			r.Name, oid(r.Name), int64(1), "OBJECT_OR_COLUMN",
			oid(triggerParent(r.Body)), "TR", "SQL_TRIGGER", int64(0),
			int64(0), int64(0),
		})
	}
	return cols, rows
}

// triggerParent extracts the table a trigger fires on from its body's leading ON <table>.
func triggerParent(body string) string {
	tokens := strings.Fields(body)
	for i := 0; i+1 < len(tokens); i++ {
		if strings.EqualFold(tokens[i], "ON") {
			return routines.Unqualify(tokens[i+1])
		}
	}
	return ""
}

// extendedPropertiesRows is sys.extended_properties: empty, no extended properties are stored.
func extendedPropertiesRows() ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		intc("class"), sname("class_desc"), intc("major_id"), intc("minor_id"), sname("name"), nsname("value"),
	}
	return cols, nil
}

func objectNameSet(schema catalog.Schema, rts []*tds.Routine) map[string]bool {
	m := map[string]bool{}
	for _, t := range schema.Tables {
		m[strings.ToLower(t.Name)] = true
	}
	for _, r := range rts {
		m[strings.ToLower(r.Name)] = true
	}
	return m
}

// sequencesRows is sys.sequences: empty, no sequences are modeled.
func sequencesRows() ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		sname("name"), intc("object_id"), intc("schema_id"),
		intc("start_value"), intc("increment"), intc("current_value"), intc("is_cycling"),
	}
	return cols, nil
}

// synonymsRows is sys.synonyms: empty, no synonyms are modeled.
func synonymsRows() ([]catalog.Column, [][]any) {
	cols := []catalog.Column{sname("name"), intc("object_id"), intc("schema_id"), sname("base_object_name")}
	return cols, nil
}

// sysobjectsRows is the legacy sys.sysobjects compatibility view (id/xtype/uid naming).
func sysobjectsRows(schema catalog.Schema, rts []*tds.Routine) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{sname("name"), intc("id"), sname("xtype"), intc("uid")}
	var rows [][]any
	for _, t := range schema.Tables {
		rows = append(rows, []any{t.Name, oid(t.Name), "U", int64(1)})
	}
	for _, r := range rts {
		x, _ := routineTypeCodes(r.Kind)
		rows = append(rows, []any{r.Name, oid(r.Name), strings.TrimSpace(x), int64(1)})
	}
	return cols, rows
}

// syscolumnsRows is the legacy sys.syscolumns compatibility view (id/colid/xtype naming).
func syscolumnsRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{sname("name"), intc("id"), intc("colid"), intc("xtype"), intc("length"), intc("isnullable")}
	var rows [][]any
	for _, t := range schema.Tables {
		for j, c := range t.Columns {
			rows = append(rows, []any{c.Name, oid(t.Name), int64(j + 1), sysTypeID(c.Type), sysTypeLen(c.Type), boolInt(c.Type.Nullable)})
		}
	}
	return cols, rows
}

// systypesRows is the legacy sys.systypes compatibility view, reusing the modern sys.types data.
func systypesRows() ([]catalog.Column, [][]any) {
	cols := []catalog.Column{sname("name"), intc("xtype"), intc("xusertype"), intc("length")}
	_, src := typesRows()
	var rows [][]any
	for _, r := range src {
		rows = append(rows, []any{r[0], r[1], r[2], r[4]}) // name, system_type_id, user_type_id, max_length
	}
	return cols, rows
}

// colID is a column's 1-based ordinal within its table (the column_id the catalog reports).
func colID(t catalog.Table, name string) int64 {
	for i, c := range t.Columns {
		if strings.EqualFold(c.Name, name) {
			return int64(i + 1)
		}
	}
	return 0
}

func tableByName(schema catalog.Schema) map[string]*catalog.Table {
	m := map[string]*catalog.Table{}
	for i := range schema.Tables {
		m[strings.ToLower(schema.Tables[i].Name)] = &schema.Tables[i]
	}
	return m
}

// oid is the one object_id scheme, shared with OBJECT_ID/OBJECT_NAME so the catalog views join.
func oid(name string) int64 { return functions.ObjectID(name) }

func routineTypeCodes(k tds.RoutineKind) (string, string) {
	switch k {
	case tds.RoutineProc:
		return "P ", "SQL_STORED_PROCEDURE"
	case tds.RoutineFunc:
		return "FN", "SQL_SCALAR_FUNCTION"
	case tds.RoutineTrigger:
		return "TR", "SQL_TRIGGER"
	}
	return "V ", "VIEW"
}

func sname(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.String}}
}
func intc(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.Int64}}
}
func nintc(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.Int64, Nullable: true}}
}
func nsname(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.String, Nullable: true}}
}

func sysTypeID(t types.Type) int64 {
	switch t.Kind {
	case types.Bool:
		return 104
	case types.Int32:
		return 56
	case types.Int64:
		return 127
	case types.Float64:
		return 62
	case types.Decimal:
		return 106
	case types.String:
		return 231
	case types.Bytes:
		return 165
	case types.Time:
		return 42
	case types.UUID:
		return 36
	}
	return 231
}

// sysTypeIDByName maps a declared parameter type ("int", "nvarchar(50)") to its system_type_id.
func sysTypeIDByName(decl string) int64 {
	name := strings.ToLower(strings.TrimSpace(decl))
	if i := strings.IndexByte(name, '('); i >= 0 {
		name = strings.TrimSpace(name[:i])
	}
	switch name {
	case "bit":
		return 104
	case "tinyint":
		return 48
	case "smallint":
		return 52
	case "int", "integer":
		return 56
	case "bigint":
		return 127
	case "decimal", "dec":
		return 106
	case "numeric":
		return 108
	case "float":
		return 62
	case "real":
		return 59
	case "money":
		return 60
	case "smallmoney":
		return 122
	case "date":
		return 40
	case "time":
		return 41
	case "datetime":
		return 61
	case "datetime2":
		return 42
	case "char":
		return 175
	case "varchar":
		return 167
	case "nchar":
		return 239
	case "nvarchar":
		return 231
	case "text":
		return 35
	case "ntext":
		return 99
	case "binary":
		return 173
	case "varbinary":
		return 165
	case "uniqueidentifier":
		return 36
	case "xml":
		return 241
	}
	return 231
}

// sysTypeLen is sys.columns.max_length: the storage width in bytes, -1 for max/unbounded.
func sysTypeLen(t types.Type) int64 {
	switch t.Kind {
	case types.Bool:
		return 1
	case types.Int32:
		return 4
	case types.Int64, types.Float64, types.Time:
		return 8
	case types.UUID:
		return 16
	case types.String:
		if t.MaxLen > 0 {
			return int64(t.MaxLen * 2)
		}
		return -1
	case types.Bytes:
		if t.MaxLen > 0 {
			return int64(t.MaxLen)
		}
		return -1
	case types.Decimal:
		return 17
	}
	return -1
}

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

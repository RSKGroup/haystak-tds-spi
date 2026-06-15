// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package infoschema

import (
	"fmt"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/internal/exec"
	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/routines"
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

const (
	catalogName = "haystak"
	schemaName  = "dbo"
)

// Resolve answers a query against INFORMATION_SCHEMA.* from a backend's declared schema and routines.
// Returns handled=false when the query does not target INFORMATION_SCHEMA.
func Resolve(schema catalog.Schema, rts []*tds.Routine, q *tds.Query) (rows tds.Rows, handled bool, err error) {
	if !strings.EqualFold(q.Schema, "INFORMATION_SCHEMA") {
		return nil, false, nil
	}
	var cols []catalog.Column
	var data [][]any
	switch strings.ToUpper(q.Table) {
	case "TABLES":
		cols, data = tablesRows(schema)
	case "COLUMNS":
		cols, data = columnsRows(schema)
	case "VIEWS":
		cols, data = viewsRows(rts)
	case "ROUTINES":
		cols, data = routinesRows(rts)
	case "PARAMETERS":
		cols, data = parametersRows(rts)
	case "TABLE_CONSTRAINTS":
		cols, data = tableConstraintsRows(schema)
	case "KEY_COLUMN_USAGE":
		cols, data = keyColumnUsageRows(schema)
	case "REFERENTIAL_CONSTRAINTS":
		cols, data = referentialConstraintsRows(schema)
	default:
		return nil, true, fmt.Errorf("infoschema: INFORMATION_SCHEMA.%s not supported", q.Table)
	}
	r, err := exec.Apply(cols, data, q)
	return r, true, err
}

// TypeName maps the canonical type model to the T-SQL type name reported by the catalog.
func TypeName(t types.Type) string {
	switch t.Kind {
	case types.Bool:
		return "bit"
	case types.Int32:
		return "int"
	case types.Int64:
		return "bigint"
	case types.Float64:
		return "float"
	case types.Decimal:
		return "decimal"
	case types.String:
		return "nvarchar"
	case types.Bytes:
		return "varbinary"
	case types.Time:
		return "datetime2"
	case types.UUID:
		return "uniqueidentifier"
	}
	return "sql_variant"
}

func tablesRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		strCol("TABLE_CATALOG"), strCol("TABLE_SCHEMA"), strCol("TABLE_NAME"), strCol("TABLE_TYPE"),
	}
	var rows [][]any
	for _, t := range schema.Tables {
		rows = append(rows, []any{catalogOf(t), schemaName, t.Name, "BASE TABLE"})
	}
	return cols, rows
}

func columnsRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		strCol("TABLE_CATALOG"), strCol("TABLE_SCHEMA"), strCol("TABLE_NAME"),
		strCol("COLUMN_NAME"), intCol("ORDINAL_POSITION"), strCol("IS_NULLABLE"),
		strCol("DATA_TYPE"), nintCol("CHARACTER_MAXIMUM_LENGTH"),
		nintCol("NUMERIC_PRECISION"), nintCol("NUMERIC_SCALE"),
	}
	var rows [][]any
	for _, t := range schema.Tables {
		for i, c := range t.Columns {
			rows = append(rows, []any{
				catalogOf(t), schemaName, t.Name, c.Name, int64(i + 1),
				yesNo(c.Type.Nullable), TypeName(c.Type),
				charLen(c.Type), numPrec(c.Type), numScale(c.Type),
			})
		}
	}
	return cols, rows
}

// viewsRows is INFORMATION_SCHEMA.VIEWS: the portable view-definition read.
func viewsRows(rts []*tds.Routine) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		strCol("TABLE_CATALOG"), strCol("TABLE_SCHEMA"), strCol("TABLE_NAME"),
		strCol("VIEW_DEFINITION"), strCol("CHECK_OPTION"), strCol("IS_UPDATABLE"),
	}
	var rows [][]any
	for _, r := range rts {
		if r.Kind == tds.RoutineView {
			rows = append(rows, []any{catalogName, schemaName, r.Name, routines.ScriptDefinition(r), "NONE", "NO"})
		}
	}
	return cols, rows
}

// routinesRows is INFORMATION_SCHEMA.ROUTINES: procedures and functions (views live in VIEWS).
func routinesRows(rts []*tds.Routine) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		strCol("SPECIFIC_CATALOG"), strCol("SPECIFIC_SCHEMA"), strCol("SPECIFIC_NAME"),
		strCol("ROUTINE_CATALOG"), strCol("ROUTINE_SCHEMA"), strCol("ROUTINE_NAME"),
		strCol("ROUTINE_TYPE"), nstrCol("DATA_TYPE"), strCol("ROUTINE_DEFINITION"),
	}
	var rows [][]any
	for _, r := range rts {
		var rtype string
		switch r.Kind {
		case tds.RoutineProc:
			rtype = "PROCEDURE"
		case tds.RoutineFunc:
			rtype = "FUNCTION"
		default:
			continue
		}
		rows = append(rows, []any{
			catalogName, schemaName, r.Name, catalogName, schemaName, r.Name,
			rtype, nil, routines.ScriptDefinition(r),
		})
	}
	return cols, rows
}

// parametersRows is INFORMATION_SCHEMA.PARAMETERS: procedure/function parameters in declared order.
func parametersRows(rts []*tds.Routine) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		strCol("SPECIFIC_CATALOG"), strCol("SPECIFIC_SCHEMA"), strCol("SPECIFIC_NAME"),
		intCol("ORDINAL_POSITION"), strCol("PARAMETER_MODE"), strCol("PARAMETER_NAME"), strCol("DATA_TYPE"),
	}
	var rows [][]any
	for _, r := range rts {
		if r.Kind != tds.RoutineProc && r.Kind != tds.RoutineFunc {
			continue
		}
		for i, p := range r.Params {
			rows = append(rows, []any{
				catalogName, schemaName, r.Name, int64(i + 1), "IN", p.Name, baseTypeName(p.Type),
			})
		}
	}
	return cols, rows
}

// baseTypeName strips a declared parameter type ("decimal(10,2)") down to its base name ("decimal").
func baseTypeName(decl string) string {
	s := strings.ToLower(strings.TrimSpace(decl))
	if i := strings.IndexByte(s, '('); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}

func tableConstraintsRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		strCol("CONSTRAINT_CATALOG"), strCol("CONSTRAINT_SCHEMA"), strCol("CONSTRAINT_NAME"),
		strCol("TABLE_CATALOG"), strCol("TABLE_SCHEMA"), strCol("TABLE_NAME"), strCol("CONSTRAINT_TYPE"),
	}
	var rows [][]any
	for _, t := range schema.Tables {
		if len(t.PrimaryKey) > 0 {
			rows = append(rows, []any{catalogOf(t), schemaName, "PK_" + t.Name, catalogOf(t), schemaName, t.Name, "PRIMARY KEY"})
		}
		for _, fk := range t.ForeignKeys {
			rows = append(rows, []any{catalogOf(t), schemaName, "FK_" + t.Name + "_" + fk.RefTable, catalogOf(t), schemaName, t.Name, "FOREIGN KEY"})
		}
	}
	return cols, rows
}

func keyColumnUsageRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		strCol("CONSTRAINT_CATALOG"), strCol("CONSTRAINT_SCHEMA"), strCol("CONSTRAINT_NAME"),
		strCol("TABLE_CATALOG"), strCol("TABLE_SCHEMA"), strCol("TABLE_NAME"),
		strCol("COLUMN_NAME"), intCol("ORDINAL_POSITION"),
	}
	var rows [][]any
	for _, t := range schema.Tables {
		for i, c := range t.PrimaryKey {
			rows = append(rows, []any{catalogOf(t), schemaName, "PK_" + t.Name, catalogOf(t), schemaName, t.Name, c, int64(i + 1)})
		}
		for _, fk := range t.ForeignKeys {
			for i, c := range fk.Columns {
				rows = append(rows, []any{catalogOf(t), schemaName, "FK_" + t.Name + "_" + fk.RefTable, catalogOf(t), schemaName, t.Name, c, int64(i + 1)})
			}
		}
	}
	return cols, rows
}

func referentialConstraintsRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		strCol("CONSTRAINT_CATALOG"), strCol("CONSTRAINT_SCHEMA"), strCol("CONSTRAINT_NAME"),
		strCol("UNIQUE_CONSTRAINT_CATALOG"), strCol("UNIQUE_CONSTRAINT_SCHEMA"), strCol("UNIQUE_CONSTRAINT_NAME"),
		strCol("MATCH_OPTION"), strCol("UPDATE_RULE"), strCol("DELETE_RULE"),
	}
	var rows [][]any
	for _, t := range schema.Tables {
		for _, fk := range t.ForeignKeys {
			rows = append(rows, []any{
				catalogOf(t), schemaName, "FK_" + t.Name + "_" + fk.RefTable,
				catalogOf(t), schemaName, "PK_" + fk.RefTable,
				"NONE", "NO ACTION", "NO ACTION",
			})
		}
	}
	return cols, rows
}

// catalogOf is the table's database (TABLE_CATALOG), or the default catalog if untagged.
func catalogOf(t catalog.Table) string {
	if t.Catalog != "" {
		return t.Catalog
	}
	return catalogName
}

func strCol(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.String}}
}
func intCol(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.Int64}}
}
func nintCol(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.Int64, Nullable: true}}
}
func nstrCol(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.String, Nullable: true}}
}

func yesNo(b bool) string {
	if b {
		return "YES"
	}
	return "NO"
}

func charLen(t types.Type) any {
	if (t.Kind == types.String || t.Kind == types.Bytes) && t.MaxLen > 0 {
		return int64(t.MaxLen)
	}
	return nil
}

func numPrec(t types.Type) any {
	if t.Kind == types.Decimal {
		return int64(t.Precision)
	}
	return nil
}

func numScale(t types.Type) any {
	if t.Kind == types.Decimal {
		return int64(t.Scale)
	}
	return nil
}

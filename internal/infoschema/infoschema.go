// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package infoschema

import (
	"fmt"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/internal/exec"
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

const (
	catalogName = "haystak"
	schemaName  = "dbo"
)

// Resolve answers a query against INFORMATION_SCHEMA.* from a backend's declared schema.
// Returns handled=false when the query does not target INFORMATION_SCHEMA.
func Resolve(schema catalog.Schema, q *tds.Query) (rows tds.Rows, handled bool, err error) {
	if !strings.EqualFold(q.Schema, "INFORMATION_SCHEMA") {
		return nil, false, nil
	}
	switch strings.ToUpper(q.Table) {
	case "TABLES":
		cols, data := tablesRows(schema)
		r, err := exec.Apply(cols, data, q)
		return r, true, err
	case "COLUMNS":
		cols, data := columnsRows(schema)
		r, err := exec.Apply(cols, data, q)
		return r, true, err
	case "TABLE_CONSTRAINTS":
		cols, data := tableConstraintsRows(schema)
		r, err := exec.Apply(cols, data, q)
		return r, true, err
	case "KEY_COLUMN_USAGE":
		cols, data := keyColumnUsageRows(schema)
		r, err := exec.Apply(cols, data, q)
		return r, true, err
	case "REFERENTIAL_CONSTRAINTS":
		cols, data := referentialConstraintsRows(schema)
		r, err := exec.Apply(cols, data, q)
		return r, true, err
	}
	return nil, true, fmt.Errorf("infoschema: INFORMATION_SCHEMA.%s not supported", q.Table)
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
		rows = append(rows, []any{catalogName, schemaName, t.Name, "BASE TABLE"})
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
				catalogName, schemaName, t.Name, c.Name, int64(i + 1),
				yesNo(c.Type.Nullable), TypeName(c.Type),
				charLen(c.Type), numPrec(c.Type), numScale(c.Type),
			})
		}
	}
	return cols, rows
}

func tableConstraintsRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		strCol("CONSTRAINT_CATALOG"), strCol("CONSTRAINT_SCHEMA"), strCol("CONSTRAINT_NAME"),
		strCol("TABLE_CATALOG"), strCol("TABLE_SCHEMA"), strCol("TABLE_NAME"), strCol("CONSTRAINT_TYPE"),
	}
	var rows [][]any
	for _, t := range schema.Tables {
		if len(t.PrimaryKey) > 0 {
			rows = append(rows, []any{catalogName, schemaName, "PK_" + t.Name, catalogName, schemaName, t.Name, "PRIMARY KEY"})
		}
		for _, fk := range t.ForeignKeys {
			rows = append(rows, []any{catalogName, schemaName, "FK_" + t.Name + "_" + fk.RefTable, catalogName, schemaName, t.Name, "FOREIGN KEY"})
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
			rows = append(rows, []any{catalogName, schemaName, "PK_" + t.Name, catalogName, schemaName, t.Name, c, int64(i + 1)})
		}
		for _, fk := range t.ForeignKeys {
			for i, c := range fk.Columns {
				rows = append(rows, []any{catalogName, schemaName, "FK_" + t.Name + "_" + fk.RefTable, catalogName, schemaName, t.Name, c, int64(i + 1)})
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
				catalogName, schemaName, "FK_" + t.Name + "_" + fk.RefTable,
				catalogName, schemaName, "PK_" + fk.RefTable,
				"NONE", "NO ACTION", "NO ACTION",
			})
		}
	}
	return cols, rows
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

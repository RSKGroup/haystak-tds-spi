// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package sysviews

import (
	"fmt"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/internal/exec"
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

const dbName = "haystak"

// Resolve answers a query against sys.* catalog views from a backend's declared schema.
// Returns handled=false when the query does not target the sys schema.
func Resolve(schema catalog.Schema, dbs []string, q *tds.Query) (tds.Rows, bool, error) {
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
	case "tables", "objects":
		cols, data = tablesRows(schema)
	case "columns":
		cols, data = columnsRows(schema)
	case "types":
		cols, data = typesRows()
	case "foreign_keys":
		cols, data = foreignKeysRows(schema)
	default:
		return nil, true, fmt.Errorf("sysviews: sys.%s not supported", q.Table)
	}
	r, err := exec.Apply(cols, data, q)
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
	for i, db := range dbs {
		rows = append(rows, mk(db, int64(5+i)))
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

func tablesRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		sname("name"), intc("object_id"), intc("schema_id"), sname("type"),
		sname("type_desc"), intc("is_ms_shipped"),
	}
	var rows [][]any
	for i, t := range schema.Tables {
		rows = append(rows, []any{t.Name, objectID(i), int64(1), "U ", "USER_TABLE", int64(0)})
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
	for i, t := range schema.Tables {
		for j, c := range t.Columns {
			st := sysTypeID(c.Type)
			rows = append(rows, []any{
				objectID(i), c.Name, int64(j + 1),
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
	for i, t := range schema.Tables {
		for _, fk := range t.ForeignKeys {
			refOID := int64(0)
			if ri, ok := idxOf[strings.ToLower(fk.RefTable)]; ok {
				refOID = objectID(ri)
			}
			rows = append(rows, []any{
				"FK_" + t.Name + "_" + fk.RefTable, fkID, objectID(i), refOID,
				int64(1), "F ", "FOREIGN_KEY_CONSTRAINT", int64(0),
			})
			fkID++
		}
	}
	return cols, rows
}

func objectID(i int) int64 { return int64(100 + i) }

func sname(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.String}}
}
func intc(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.Int64}}
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

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package sysviews

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/RSKGroup/haystak-tds-spi/internal/exec"
	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/functions"
	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/routines"
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

const dbName = "haystak"

var dbCreateDate = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

type viewBuilder func() ([]catalog.Column, [][]any)

// viewBuilders is the sys.* dispatch table; its keys are the wired set SupportedViews reports.
func viewBuilders(schema catalog.Schema, rts []*tds.Routine, dbs []string, p tds.Principal, sess []tds.SessionInfo) map[string]viewBuilder {
	return map[string]viewBuilder{
		"databases":                   func() ([]catalog.Column, [][]any) { return databasesRows(dbs) },
		"configurations":              configurationsRows,
		"schemas":                     schemasRows,
		"tables":                      func() ([]catalog.Column, [][]any) { return tablesRows(schema) },
		"objects":                     func() ([]catalog.Column, [][]any) { return objectsRows(schema, rts) },
		"all_objects":                 func() ([]catalog.Column, [][]any) { return objectsRows(schema, rts) },
		"views":                       func() ([]catalog.Column, [][]any) { return viewsRows(rts) },
		"procedures":                  func() ([]catalog.Column, [][]any) { return proceduresRows(rts) },
		"sql_modules":                 func() ([]catalog.Column, [][]any) { return sqlModulesRows(rts) },
		"parameters":                  func() ([]catalog.Column, [][]any) { return parametersRows(rts) },
		"columns":                     func() ([]catalog.Column, [][]any) { return columnsRows(schema) },
		"all_columns":                 func() ([]catalog.Column, [][]any) { return columnsRows(schema) },
		"types":                       typesRows,
		"foreign_keys":                func() ([]catalog.Column, [][]any) { return foreignKeysRows(schema) },
		"indexes":                     func() ([]catalog.Column, [][]any) { return indexesRows(schema) },
		"index_columns":               func() ([]catalog.Column, [][]any) { return indexColumnsRows(schema) },
		"key_constraints":             func() ([]catalog.Column, [][]any) { return keyConstraintsRows(schema) },
		"foreign_key_columns":         func() ([]catalog.Column, [][]any) { return foreignKeyColumnsRows(schema) },
		"check_constraints":           func() ([]catalog.Column, [][]any) { return checkConstraintsRows(schema) },
		"default_constraints":         func() ([]catalog.Column, [][]any) { return defaultConstraintsRows(schema) },
		"identity_columns":            func() ([]catalog.Column, [][]any) { return identityColumnsRows(schema) },
		"computed_columns":            func() ([]catalog.Column, [][]any) { return computedColumnsRows(schema) },
		"sql_expression_dependencies": func() ([]catalog.Column, [][]any) { return sqlExpressionDependenciesRows(schema, rts) },
		"triggers":                    func() ([]catalog.Column, [][]any) { return triggersRows(rts) },
		"extended_properties":         extendedPropertiesRows,
		"sequences":                   sequencesRows,
		"synonyms":                    synonymsRows,
		"sysobjects":                  func() ([]catalog.Column, [][]any) { return sysobjectsRows(schema, rts) },
		"syscolumns":                  func() ([]catalog.Column, [][]any) { return syscolumnsRows(schema) },
		"systypes":                    systypesRows,
		"table_types":                 func() ([]catalog.Column, [][]any) { return tableTypesRows(schema) },
		"database_principals":         func() ([]catalog.Column, [][]any) { return databasePrincipalsRows(p) },
		"server_principals":           func() ([]catalog.Column, [][]any) { return serverPrincipalsRows(p) },
		"database_role_members":       func() ([]catalog.Column, [][]any) { return databaseRoleMembersRows(p) },
		"sysusers":                    func() ([]catalog.Column, [][]any) { return sysusersRows(p) },
		"sysindexes":                  func() ([]catalog.Column, [][]any) { return sysindexesRows(schema) },
		"database_permissions":        databasePermissionsRows,
		"server_permissions":          serverPermissionsRows,
		"system_objects":              func() ([]catalog.Column, [][]any) { return objectCols(), nil },
		"stats":                       statsRows,
		"stats_columns":               statsColumnsRows,
		"partitions":                  func() ([]catalog.Column, [][]any) { return partitionsRows(schema) },
		"database_files":              databaseFilesRows,
		"filegroups":                  filegroupsRows,
		"dm_exec_sessions":            func() ([]catalog.Column, [][]any) { return dmExecSessionsRows(sess) },
		"dm_exec_connections":         func() ([]catalog.Column, [][]any) { return dmExecConnectionsRows(sess) },
		"dm_exec_requests":            func() ([]catalog.Column, [][]any) { return dmExecRequestsRows(sess) },
		"dm_exec_query_stats":         func() ([]catalog.Column, [][]any) { return dmExecQueryStatsRows() },
		"dm_os_waiting_tasks":         func() ([]catalog.Column, [][]any) { return dmOsWaitingTasksRows() },
		"dm_os_host_info":             func() ([]catalog.Column, [][]any) { return dmOsHostInfoRows() },
		"internal_tables":             internalTablesRows,
		"data_spaces":                 dataSpacesRows,
		"partition_schemes":           partitionSchemesRows,
		"xml_schema_collections":      xmlSchemaCollectionsRows,
		"column_encryption_keys":      columnEncryptionKeysRows,
		"masked_columns":              maskedColumnsRows,
		"xml_indexes":                 xmlIndexesRows,
		"spatial_indexes":             spatialIndexesRows,
		"assembly_modules":            assemblyModulesRows,
		"trigger_events":              triggerEventsRows,
		"system_sql_modules":          systemSQLModulesRows,
		"master_files":                masterFilesRows,
		"partition_functions":         partitionFunctionsRows,
		"partition_range_values":      partitionRangeValuesRows,
		"partition_parameters":        partitionParametersRows,
		"destination_data_spaces":     destinationDataSpacesRows,
		"allocation_units":            allocationUnitsRows,
		"dm_db_partition_stats":       dmDbPartitionStatsRows,
		"availability_replicas":       availabilityReplicasRows,
		"availability_groups":         availabilityGroupsRows,
		"dm_hadr_database_replica_states": dmHadrDatabaseReplicaStatesRows,
		"database_mirroring":          databaseMirroringRows,
		"dm_hadr_cluster":             dmHadrClusterRows,
		"dm_hadr_availability_replica_states": dmHadrAvailabilityReplicaStatesRows,
		"dm_hadr_availability_group_states":   dmHadrAvailabilityGroupStatesRows,
	}
}

// SupportedViews returns every wired sys.* view name, lower-cased and sorted.
func SupportedViews() []string {
	builders := viewBuilders(catalog.Schema{}, nil, nil, tds.Principal{}, nil)
	names := make([]string, 0, len(builders))
	for name := range builders {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Resolve answers a query against sys.* catalog views from a backend's declared schema and stored
// routines. Returns handled=false when the query does not target the sys schema.
func Resolve(schema catalog.Schema, rts []*tds.Routine, dbs []string, p tds.Principal, sess []tds.SessionInfo, spid int, q *tds.Query) (tds.Rows, bool, error) {
	if !strings.EqualFold(q.Schema, "sys") {
		return nil, false, nil
	}
	build, ok := viewBuilders(schema, rts, dbs, p, sess)[strings.ToLower(q.Table)]
	if !ok {
		return nil, true, fmt.Errorf("sysviews: sys.%s not supported", q.Table)
	}
	cols, data := build()
	obj, db := exec.CatalogResolvers(schema, rts, dbs)
	tbl, kind := exec.CatalogObjects(schema, rts)
	env := &exec.Env{ObjectName: obj, DBName: db, Table: tbl, ObjectKind: kind, RoutineDef: exec.RoutineDefs(rts), CurrentDB: q.Database, SPID: spid}
	r, err := exec.ApplyWith(cols, data, q, env)
	return r, true, err
}

func databasesRows(dbs []string) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		sname("name"), int32c("database_id"), nint32c("source_database_id"), tinyc("state"), sname("state_desc"),
		bitc("is_read_only"), sname("collation_name"), tinyc("compatibility_level"),
		tinyc("containment"), tinyc("recovery_model"), tinyc("user_access"),
		bitc("is_in_standby"), bitc("is_fulltext_enabled"),
		bitc("is_distributor"), bitc("is_published"), bitc("is_subscribed"), bitc("is_merge_published"),
		tinyc("catalog_collation_type"), bitc("is_ledger_on"), dtc("create_date"),
		{Name: "owner_sid", Type: types.Type{Kind: types.Bytes, Nullable: true}},
	}
	mk := func(name string, id int64) []any {
		return []any{name, id, nil, int64(0), "ONLINE",
			false, "SQL_Latin1_General_CP1_CI_AS", int64(160),
			int64(0), int64(1), int64(0),
			false, false,
			false, false, false, false,
			int64(0), false, dbCreateDate,
			nil}
	}
	if len(dbs) == 0 { // single-database backend: keep reporting the default catalog
		dbs = []string{dbName}
	}
	rows := [][]any{mk("master", 1), mk("tempdb", 2), mk("model", 3), mk("msdb", 4)}
	for _, db := range dbs {
		rows = append(rows, mk(db, functions.DBIDForList(db, dbs)))
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

// tablesRows is the full sys.tables surface SMO/Power BI read for the Tables node (name+object_id per table, the rest static).
func tablesRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	defs := []struct {
		col catalog.Column
		val any
	}{
		{sname("name"), nil}, {int32c("object_id"), nil}, {nint32c("principal_id"), nil},
		{int32c("schema_id"), int64(1)}, {int32c("parent_object_id"), int64(0)},
		{sname("type"), "U "}, {sname("type_desc"), "USER_TABLE"},
		{dtc("create_date"), dbCreateDate}, {dtc("modify_date"), dbCreateDate},
		{bitc("is_ms_shipped"), false}, {bitc("is_published"), false}, {bitc("is_schema_published"), false},
		{nint32c("lob_data_space_id"), int64(0)}, {nint32c("filestream_data_space_id"), nil},
		{int32c("max_column_id_used"), int64(0)}, {bitc("lock_on_bulk_load"), false}, {bitc("uses_ansi_nulls"), true},
		{bitc("is_replicated"), false}, {bitc("has_replication_filter"), false}, {bitc("is_merge_published"), false},
		{bitc("is_sync_tran_subscribed"), false}, {bitc("has_unchecked_assembly_data"), false},
		{int32c("text_in_row_limit"), int64(0)}, {bitc("large_value_types_out_of_row"), false}, {bitc("is_tracked_by_cdc"), false},
		{tinyc("lock_escalation"), int64(0)}, {sname("lock_escalation_desc"), "TABLE"},
		{bitc("is_filetable"), false}, {bitc("is_memory_optimized"), false},
		{tinyc("durability"), int64(0)}, {sname("durability_desc"), "SCHEMA_AND_DATA"},
		{tinyc("temporal_type"), int64(0)}, {sname("temporal_type_desc"), "NON_TEMPORAL_TABLE"}, {nint32c("history_table_id"), nil},
		{bitc("is_remote_data_archive_enabled"), false}, {bitc("is_external"), false},
		{nint32c("history_retention_period"), nil}, {nint32c("history_retention_period_unit"), nil}, {nsname("history_retention_period_unit_desc"), nil},
		{bitc("is_node"), false}, {bitc("is_edge"), false},
		{nint32c("data_retention_period"), int64(-1)}, {nint32c("data_retention_period_unit"), int64(0)}, {nsname("data_retention_period_unit_desc"), "INFINITE"},
		{tinyc("ledger_type"), int64(0)}, {sname("ledger_type_desc"), "NONE"}, {nint32c("ledger_view_id"), nil}, {bitc("is_dropped_ledger_table"), false},
	}
	cols := make([]catalog.Column, len(defs))
	base := make([]any, len(defs))
	for i, d := range defs {
		cols[i], base[i] = d.col, d.val
	}
	var rows [][]any
	for _, t := range schema.Tables {
		row := make([]any, len(defs))
		copy(row, base)
		row[0], row[1] = t.Name, oid(t.Name)
		rows = append(rows, row)
	}
	return cols, rows
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
		int32c("object_id"), sname("name"), int32c("column_id"),
		tinyc("system_type_id"), int32c("user_type_id"), smallc("max_length"),
		tinyc("precision"), tinyc("scale"), bitc("is_nullable"), bitc("is_sparse"),
		nsname("collation_name"), bitc("is_ansi_padded"), bitc("is_rowguidcol"), bitc("is_identity"),
		bitc("is_computed"), bitc("is_filestream"), bitc("is_replicated"), bitc("is_non_sql_subscribed"),
		bitc("is_merge_published"), bitc("is_dts_replicated"), bitc("is_xml_document"),
		int32c("xml_collection_id"), int32c("default_object_id"), int32c("rule_object_id"), bitc("is_column_set"),
		tinyc("generated_always_type"), sname("generated_always_type_desc"),
		nint32c("encryption_type"), nsname("encryption_type_desc"), nsname("encryption_algorithm_name"),
		nint32c("column_encryption_key_id"), nsname("column_encryption_key_database_name"),
		bitc("is_hidden"), bitc("is_masked"), nint32c("graph_type"), nsname("graph_type_desc"),
		tinyc("ledger_view_column_type"), nsname("ledger_view_column_type_desc"), bitc("is_dropped_ledger_column"),
	}
	var rows [][]any
	for _, t := range schema.Tables {
		for j, c := range t.Columns {
			st := sysTypeID(c.Type)
			rows = append(rows, []any{
				oid(t.Name), c.Name, int64(j + 1),
				st, st, sysTypeLen(c.Type),
				int64(c.Type.Precision), int64(c.Type.Scale), c.Type.Nullable, false,
				nil, true, false, false,
				false, false, false, false,
				false, false, false,
				int64(0), int64(0), int64(0), false,
				int64(0), "NOT_APPLICABLE",
				nil, nil, nil,
				nil, nil,
				false, false, nil, nil,
				int64(0), nil, false,
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
		intc("object_id"), nsname("name"), intc("index_id"), intc("type"), sname("type_desc"),
		intc("is_unique"), intc("is_primary_key"), intc("is_unique_constraint"), intc("is_disabled"),
		int32c("data_space_id"), bitc("is_hypothetical"),
	}
	var rows [][]any
	for _, t := range schema.Tables {
		ixs := TableIndexes(t)
		clustered := false
		for _, ix := range ixs {
			clustered = clustered || ix.Clustered
		}
		if !clustered { // a table with no clustered index is a heap: one row at index_id 0 (SMO joins this)
			rows = append(rows, []any{oid(t.Name), nil, int64(0), int64(0), "HEAP", int64(0), int64(0), int64(0), int64(0), int64(1), false})
		}
		next := int64(2)
		for _, ix := range ixs {
			id, typ, desc := next, int64(2), "NONCLUSTERED"
			if ix.Clustered {
				id, typ, desc = int64(1), int64(1), "CLUSTERED"
			} else {
				next++
			}
			rows = append(rows, []any{
				oid(t.Name), ix.Name, id, typ, desc,
				boolInt(ix.Unique), boolInt(ix.Primary), boolInt(ix.Unique && !ix.Primary), int64(0), int64(1), false,
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

// wellKnownPrincipals are the fixed database principals SQL Server always exposes.
var wellKnownPrincipals = []struct {
	id   int64
	name string
	typ  string // S = user, R = role
}{
	{0, "public", "R"}, {1, "dbo", "S"}, {2, "guest", "S"}, {3, "INFORMATION_SCHEMA", "S"}, {4, "sys", "S"},
}

func principalName(p tds.Principal) string {
	if p.Username != "" {
		return p.Username
	}
	return "haystak"
}

func wellKnownID(name string) (int64, bool) {
	for _, w := range wellKnownPrincipals {
		if strings.EqualFold(w.name, name) {
			return w.id, true
		}
	}
	return 0, false
}

func userPrincipalID(p tds.Principal) int64 {
	if id, ok := wellKnownID(principalName(p)); ok {
		return id
	}
	return 5
}

func defaultSchema(name string) any {
	switch strings.ToLower(name) {
	case "dbo":
		return "dbo"
	case "guest":
		return "guest"
	}
	return nil
}

// databasePrincipalsRows is sys.database_principals: the fixed well-knowns plus the live authenticated
// user and its roles (the Principal seam).
func databasePrincipalsRows(p tds.Principal) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		sname("name"), intc("principal_id"), sname("type"), sname("type_desc"),
		nsname("default_schema_name"), intc("is_fixed_role"),
	}
	desc := func(t string) string {
		if t == "R" {
			return "DATABASE_ROLE"
		}
		return "SQL_USER"
	}
	var rows [][]any
	for _, w := range wellKnownPrincipals {
		rows = append(rows, []any{w.name, w.id, w.typ, desc(w.typ), defaultSchema(w.name), int64(0)})
	}
	if u := principalName(p); !isWellKnownName(u) {
		rows = append(rows, []any{u, int64(5), "S", "SQL_USER", "dbo", int64(0)})
	}
	for i, r := range p.Roles {
		rows = append(rows, []any{r, int64(6 + i), "R", "DATABASE_ROLE", nil, int64(0)})
	}
	return cols, rows
}

// serverPrincipalsRows is sys.server_principals: sa, public, and the live login.
func serverPrincipalsRows(p tds.Principal) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{sname("name"), intc("principal_id"), sname("type"), sname("type_desc"), intc("is_disabled")}
	rows := [][]any{
		{"sa", int64(1), "S", "SQL_LOGIN", int64(0)},
		{"public", int64(2), "R", "SERVER_ROLE", int64(0)},
	}
	if u := principalName(p); !strings.EqualFold(u, "sa") {
		rows = append(rows, []any{u, int64(3), "S", "SQL_LOGIN", int64(0)})
	}
	return cols, rows
}

// databaseRoleMembersRows is sys.database_role_members: the live user's role memberships.
func databaseRoleMembersRows(p tds.Principal) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{intc("role_principal_id"), intc("member_principal_id")}
	var rows [][]any
	for i := range p.Roles {
		rows = append(rows, []any{int64(6 + i), userPrincipalID(p)})
	}
	return cols, rows
}

// sysusersRows is the legacy sys.sysusers compatibility view over the same principal set.
func sysusersRows(p tds.Principal) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{intc("uid"), sname("name")}
	var rows [][]any
	for _, w := range wellKnownPrincipals {
		rows = append(rows, []any{w.id, w.name})
	}
	if u := principalName(p); !isWellKnownName(u) {
		rows = append(rows, []any{int64(5), u})
	}
	return cols, rows
}

func isWellKnownName(name string) bool { _, ok := wellKnownID(name); return ok }

// sysindexesRows is the legacy sys.sysindexes compatibility view over the declared indexes.
func sysindexesRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{intc("id"), sname("name"), intc("indid")}
	var rows [][]any
	for _, t := range schema.Tables {
		for i, ix := range TableIndexes(t) {
			rows = append(rows, []any{oid(t.Name), ix.Name, int64(i + 1)})
		}
	}
	return cols, rows
}

// databasePermissionsRows is sys.database_permissions: empty, no grants are tracked.
func databasePermissionsRows() ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		intc("class"), sname("class_desc"), intc("major_id"), intc("minor_id"),
		intc("grantee_principal_id"), sname("permission_name"), sname("state"), sname("state_desc"),
	}
	return cols, nil
}

// serverPermissionsRows is sys.server_permissions: empty, no grants are tracked.
func serverPermissionsRows() ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		intc("class"), sname("class_desc"), intc("major_id"),
		intc("grantee_principal_id"), sname("permission_name"), sname("state_desc"),
	}
	return cols, nil
}

// tableTypesRows is sys.table_types: a projection over the backend's declared table types (empty if none).
func tableTypesRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		sname("name"), intc("system_type_id"), intc("user_type_id"), intc("schema_id"),
		intc("is_user_defined"), intc("is_table_type"), intc("type_table_object_id"),
	}
	var rows [][]any
	for _, tt := range schema.TableTypes {
		rows = append(rows, []any{tt.Name, int64(243), oid(tt.Name), int64(1), int64(1), int64(1), oid(tt.Name)})
	}
	return cols, rows
}

// sequencesRows is sys.sequences: empty, no sequences are modeled.
func statsRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{
		intc("object_id"), sname("name"), intc("stats_id"),
		intc("auto_created"), intc("user_created"), intc("no_recompute"), intc("has_filter"),
	}, nil
}

func statsColumnsRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{intc("object_id"), intc("stats_id"), intc("stats_column_id"), intc("column_id")}, nil
}

func partitionsRows(schema catalog.Schema) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{intc("partition_id"), intc("object_id"), intc("index_id"), intc("partition_number"), intc("rows")}
	var rows [][]any
	for _, t := range schema.Tables {
		rows = append(rows, []any{oid(t.Name), oid(t.Name), int64(1), int64(1), int64(0)})
	}
	return cols, rows
}

func databaseFilesRows() ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		intc("file_id"), sname("type_desc"), intc("data_space_id"),
		sname("name"), sname("physical_name"), sname("state_desc"), intc("size"),
	}
	return cols, [][]any{{int64(1), "ROWS", int64(1), dbName, dbName + ".mdf", "ONLINE", int64(0)}}
}

func filegroupsRows() ([]catalog.Column, [][]any) {
	cols := []catalog.Column{sname("name"), intc("data_space_id"), sname("type"), sname("type_desc"), intc("is_default"), intc("is_read_only")}
	return cols, [][]any{{"PRIMARY", int64(1), "FG", "ROWS_FILEGROUP", int64(1), int64(0)}}
}

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
func int32c(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.Int32}}
}
func nint32c(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.Int32, Nullable: true}}
}
func tinyc(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.Int8}}
}
func smallc(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.Int16}}
}
func nintc(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.Int64, Nullable: true}}
}
func nsname(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.String, Nullable: true}}
}

func sysTypeID(t types.Type) int64 {
	if t.Name != "" {
		return sysTypeIDByName(t.Name)
	}
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
	if t.Name != "" {
		name := strings.ToLower(strings.TrimSpace(t.Name))
		if i := strings.IndexByte(name, '('); i >= 0 {
			name = strings.TrimSpace(name[:i])
		}
		switch name {
		case "bit", "tinyint":
			return 1
		case "smallint":
			return 2
		case "int":
			return 4
		case "bigint", "float", "money", "datetime", "datetime2":
			return 8
		case "real", "smallmoney", "smalldatetime":
			return 4
		case "decimal", "numeric":
			return 17
		case "date":
			return 3
		case "time":
			return 5
		case "datetimeoffset":
			return 10
		case "char", "varchar", "binary", "varbinary":
			if t.MaxLen > 0 {
				return int64(t.MaxLen)
			}
			return -1
		case "nchar", "nvarchar":
			if t.MaxLen > 0 {
				return int64(t.MaxLen * 2)
			}
			return -1
		case "text", "ntext", "image":
			return 16
		case "uniqueidentifier":
			return 16
		case "xml":
			return -1
		}
	}
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

func tcol(n string) catalog.Column { return catalog.Column{Name: n, Type: types.Type{Kind: types.Time}} }
func bitc(n string) catalog.Column { return catalog.Column{Name: n, Type: types.Type{Kind: types.Bool}} }
func dtc(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.Time, Nullable: true}}
}
func uidc(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.UUID, Nullable: true}}
}

// AlwaysOn/mirroring catalog stubs the SSMS database-list query reads; a backend without HA reports none, so empty.
func availabilityReplicasRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{uidc("replica_id"), uidc("group_id"), sname("replica_server_name")}, nil
}

func availabilityGroupsRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{uidc("group_id"), sname("name")}, nil
}

func dmHadrDatabaseReplicaStatesRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{uidc("group_database_id"), tinyc("synchronization_state"), bitc("is_local"), uidc("group_id"), int32c("database_id")}, nil
}

func databaseMirroringRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{int32c("database_id"), int32c("mirroring_role"), int32c("mirroring_state"), uidc("mirroring_guid")}, nil
}

func dmHadrClusterRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{sname("cluster_name"), tinyc("quorum_type"), nsname("quorum_type_desc"), tinyc("quorum_state"), nsname("quorum_state_desc")}, nil
}

func dmHadrAvailabilityReplicaStatesRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{uidc("replica_id"), uidc("group_id"), bitc("is_local"), tinyc("role"), nsname("role_desc"), tinyc("operational_state"), nsname("operational_state_desc"), tinyc("connected_state"), nsname("connected_state_desc"), tinyc("recovery_health"), tinyc("synchronization_health"), nsname("synchronization_health_desc")}, nil
}

func dmHadrAvailabilityGroupStatesRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{uidc("group_id"), sname("primary_replica"), tinyc("primary_recovery_health"), tinyc("secondary_recovery_health"), tinyc("synchronization_health"), nsname("synchronization_health_desc")}, nil
}

// Storage/partitioning/feature-metadata catalog stubs SMO and Power BI LEFT JOIN; a storage-less backend reports them empty.
func internalTablesRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{int32c("object_id"), sname("name"), int32c("schema_id"), int32c("parent_object_id"), int32c("parent_id"), int32c("parent_minor_id"), int32c("internal_type"), sname("internal_type_desc"), int32c("lob_data_space_id")}, nil
}

func dataSpacesRows() ([]catalog.Column, [][]any) {
	cols := []catalog.Column{sname("name"), int32c("data_space_id"), sname("type"), sname("type_desc"), bitc("is_default"), bitc("is_system")}
	return cols, [][]any{{"PRIMARY", int64(1), "FG", "ROWS_FILEGROUP", true, false}}
}

func partitionSchemesRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{sname("name"), int32c("data_space_id"), sname("type"), sname("type_desc"), bitc("is_default"), bitc("is_system"), int32c("function_id")}, nil
}

func xmlSchemaCollectionsRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{int32c("xml_collection_id"), int32c("schema_id"), sname("name"), nint32c("principal_id"), dtc("create_date"), dtc("modify_date")}, nil
}

func columnEncryptionKeysRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{int32c("column_encryption_key_id"), sname("name"), dtc("create_date"), dtc("modify_date")}, nil
}

func maskedColumnsRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{int32c("object_id"), int32c("column_id"), sname("name"), bitc("is_masked"), nsname("masking_function")}, nil
}

func xmlIndexesRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{int32c("object_id"), int32c("index_id"), sname("name"), nint32c("using_xml_index_id"), nsname("secondary_type"), nsname("secondary_type_desc"), tinyc("xml_index_type"), nsname("xml_index_type_description")}, nil
}

func spatialIndexesRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{int32c("object_id"), int32c("index_id"), sname("name"), tinyc("spatial_index_type"), nsname("spatial_index_type_desc"), nsname("tessellation_scheme")}, nil
}

func assemblyModulesRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{int32c("object_id"), int32c("assembly_id"), nsname("assembly_class"), nsname("assembly_method"), nint32c("execute_as_principal_id")}, nil
}

func triggerEventsRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{int32c("object_id"), int32c("type"), sname("type_desc"), bitc("is_first"), bitc("is_last")}, nil
}

func systemSQLModulesRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{int32c("object_id"), nsname("definition"), bitc("uses_ansi_nulls"), bitc("uses_quoted_identifier")}, nil
}

func masterFilesRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{int32c("database_id"), int32c("file_id"), tinyc("type"), sname("type_desc"), int32c("data_space_id"), sname("name"), sname("physical_name"), tinyc("state"), sname("state_desc"), int32c("size"), int32c("max_size"), int32c("growth"), bitc("is_percent_growth"), bitc("is_read_only")}, nil
}

func partitionFunctionsRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{sname("name"), int32c("function_id"), sname("type"), sname("type_desc"), int32c("fanout"), bitc("boundary_value_on_right"), dtc("create_date"), dtc("modify_date")}, nil
}

func partitionRangeValuesRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{int32c("function_id"), int32c("boundary_id"), int32c("parameter_id"), nsname("value")}, nil
}

func partitionParametersRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{int32c("function_id"), int32c("parameter_id"), tinyc("system_type_id"), int32c("user_type_id"), int32c("max_length"), tinyc("precision"), tinyc("scale")}, nil
}

func destinationDataSpacesRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{int32c("partition_scheme_id"), int32c("destination_id"), int32c("data_space_id")}, nil
}

func allocationUnitsRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{intc("allocation_unit_id"), tinyc("type"), sname("type_desc"), intc("container_id"), int32c("data_space_id"), intc("total_pages"), intc("used_pages"), intc("data_pages")}, nil
}

func dmDbPartitionStatsRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{intc("partition_id"), int32c("object_id"), int32c("index_id"), int32c("partition_number"), intc("in_row_data_page_count"), intc("in_row_used_page_count"), intc("in_row_reserved_page_count"), intc("lob_used_page_count"), intc("lob_reserved_page_count"), intc("row_overflow_used_page_count"), intc("row_overflow_reserved_page_count"), intc("used_page_count"), intc("reserved_page_count"), intc("row_count")}, nil
}

func configurationsRows() ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		intc("configuration_id"), sname("name"), intc("value"), intc("minimum"),
		intc("maximum"), intc("value_in_use"), nsname("description"),
		bitc("is_dynamic"), bitc("is_advanced"),
	}
	rows := [][]any{
		{int64(16384), "user options", int64(0), int64(0), int64(32767), int64(0), "user options", false, false},
		{int64(1543), "default trace enabled", int64(1), int64(0), int64(1), int64(1), "default trace enabled", true, true},
	}
	return cols, rows
}

// Runtime DMVs enumerate every live session from the server's registry snapshot in ctx.
func dmExecSessionsRows(sessions []tds.SessionInfo) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		intc("session_id"), sname("login_name"), sname("host_name"), sname("program_name"),
		tcol("login_time"), sname("status"), intc("cpu_time"), intc("memory_usage"),
		intc("total_elapsed_time"), intc("reads"), intc("writes"), intc("logical_reads"),
		intc("database_id"), intc("is_user_process"), intc("authenticating_database_id"),
	}
	var rows [][]any
	for _, s := range sessions {
		rows = append(rows, []any{
			int64(s.SessionID), s.LoginName, s.Host, s.Program,
			s.LoginTime, "running", int64(0), int64(0),
			int64(0), int64(0), int64(0), int64(0), int64(0), int64(1), int64(1),
		})
	}
	return cols, rows
}

func dmExecConnectionsRows(sessions []tds.SessionInfo) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		intc("session_id"), tcol("connect_time"), sname("net_transport"), sname("protocol_type"),
		sname("auth_scheme"), sname("client_net_address"), sname("local_net_address"),
		intc("local_tcp_port"), intc("most_recent_session_id"),
	}
	var rows [][]any
	for _, s := range sessions {
		rows = append(rows, []any{
			int64(s.SessionID), s.LoginTime, "TCP", "TSQL",
			"SQL", "127.0.0.1", "127.0.0.1", int64(1433), int64(s.SessionID),
		})
	}
	return cols, rows
}

func dmExecRequestsRows(sessions []tds.SessionInfo) ([]catalog.Column, [][]any) {
	cols := []catalog.Column{
		intc("session_id"), sname("status"), sname("command"), intc("database_id"),
		tcol("start_time"), intc("cpu_time"), intc("total_elapsed_time"), intc("reads"),
		intc("writes"), intc("logical_reads"), nsname("wait_type"), intc("blocking_session_id"),
	}
	var rows [][]any
	for _, s := range sessions {
		rows = append(rows, []any{
			int64(s.SessionID), "running", "SELECT", int64(0),
			s.LoginTime, int64(0), int64(0), int64(0),
			int64(0), int64(0), nil, int64(0),
		})
	}
	return cols, rows
}

func dmExecQueryStatsRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{
		nsname("sql_handle"), nsname("plan_handle"), intc("execution_count"), intc("total_worker_time"),
		intc("total_elapsed_time"), intc("total_logical_reads"), intc("total_physical_reads"),
		tcol("creation_time"), tcol("last_execution_time"),
	}, nil
}

func dmOsWaitingTasksRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{
		nsname("waiting_task_address"), intc("session_id"), sname("wait_type"),
		intc("wait_duration_ms"), intc("blocking_session_id"), nsname("resource_description"),
	}, nil
}

func dmOsHostInfoRows() ([]catalog.Column, [][]any) {
	return []catalog.Column{
			sname("host_platform"), sname("host_distribution"), sname("host_release"),
			sname("host_service_pack_level"), intc("host_sku"), intc("os_language_version"),
		}, [][]any{
			{"Windows", "Windows", "10.0", "", int64(8), int64(1033)},
		}
}

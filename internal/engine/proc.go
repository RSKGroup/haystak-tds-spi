// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/internal/exec"
	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/infoschema"
	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/routines"
	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/sysviews"
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

type procFunc func(ctx context.Context, b tds.Backend, args []procArg) (tds.Rows, bool, error)

// procDispatch is the built-in sp_* dispatch table; its keys are the wired set SupportedProcs reports.
var procDispatch = map[string]procFunc{
	"sp_databases": func(ctx context.Context, b tds.Backend, _ []procArg) (tds.Rows, bool, error) {
		return spDatabases(ctx, b)
	},
	"sp_tables":            spTables,
	"sp_columns":           spColumns,
	"sp_columns_90":        spColumns,
	"sp_helptext":          spHelptext,
	"sp_help":              spHelp,
	"sp_pkeys":             spPkeys,
	"sp_fkeys":             spFkeys,
	"sp_statistics":        spStatistics,
	"sp_special_columns":   spSpecialColumns,
	"sp_stored_procedures": spStoredProcedures,
	"sp_sproc_columns":     spSprocColumns,
	"sp_helpindex":         spHelpindex,
	"sp_helpconstraint":    spHelpconstraint,
	"sp_helpdb":            spHelpdb,
	"sp_configure":         spConfigure,
	"sp_lock":              spLock,
	"sp_server_info":       spServerInfo,
	"sp_datatype_info":     spDatatypeInfo,
	"sp_datatype_info_100": spDatatypeInfo,
	"sp_table_privileges":  spTablePrivileges,
	"sp_column_privileges": spColumnPrivileges,
	"sp_helptrigger":       spHelptrigger,
	"sp_depends":           spDepends,
	"sp_who":               spWho,
}

// spWho lists every live session from the registry snapshot in ctx.
func spWho(ctx context.Context, _ tds.Backend, _ []procArg) (tds.Rows, bool, error) {
	cols := []catalog.Column{
		in16("spid"), in16("ecid"), sn("status"), sn("loginame"),
		sn("hostname"), sn("blk"), sn("dbname"), sn("cmd"),
	}
	var data [][]any
	for _, s := range sessionOf(ctx) {
		data = append(data, []any{
			int64(s.SessionID), int64(0), "running", s.LoginName,
			s.Host, "0", currentDB(ctx), "SELECT",
		})
	}
	rs, err := exec.Apply(cols, data, &tds.Query{})
	return rs, true, err
}

// Registered in init, not the literal, to break the cycle spExecuteSQL -> queryOne -> execProc -> procDispatch.
func init() { procDispatch["sp_executesql"] = spExecuteSQL }

// spExecuteSQL substitutes named params (@p = val) as literals into the statement and runs it.
func spExecuteSQL(ctx context.Context, b tds.Backend, args []procArg) (tds.Rows, bool, error) {
	if len(args) == 0 {
		return nil, true, nil
	}
	stmt, ok := args[0].val.(string)
	if !ok {
		return nil, true, fmt.Errorf("sp_executesql: @stmt must be a string")
	}
	type binding struct{ name, lit string }
	var subs []binding
	for _, a := range args {
		if a.name != "" {
			subs = append(subs, binding{a.name, sqlLiteral(a.val)})
		}
	}
	sort.Slice(subs, func(i, j int) bool { return len(subs[i].name) > len(subs[j].name) })
	for _, s := range subs {
		stmt = strings.ReplaceAll(stmt, s.name, s.lit)
	}
	rs, _, err := queryOne(ctx, b, stmt)
	return rs, true, err
}

func sqlLiteral(v any) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case string:
		return "'" + strings.ReplaceAll(x, "'", "''") + "'"
	case int64:
		return strconv.FormatInt(x, 10)
	default:
		return fmt.Sprintf("%v", x)
	}
}

// SupportedProcs returns every wired built-in sp_* proc name, lower-cased and sorted (aliases included).
func SupportedProcs() []string {
	names := make([]string, 0, len(procDispatch))
	for name := range procDispatch {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// execProc answers the ODBC catalog procs (sp_databases/sp_tables/sp_columns); other sp_* no-op.
func execProc(ctx context.Context, b tds.Backend, sql string) (tds.Rows, bool, error) {
	if inner, ok := parseExecDynamic(sql); ok {
		rows, _, err := queryOne(ctx, b, inner)
		return rows, true, err
	}
	name, args, ok := parseProcCall(sql)
	if !ok {
		return nil, false, nil
	}
	if fn, ok := procDispatch[strings.ToLower(name)]; ok {
		return fn(ctx, b, args)
	}
	if rs, found, err := execStoredProc(ctx, b, name, args); found || err != nil {
		return rs, true, err
	}
	if strings.HasPrefix(strings.ToLower(name), "sp_") {
		return nil, true, nil
	}
	return nil, true, fmt.Errorf("could not find stored procedure %q", name)
}

type procArg struct {
	name string
	val  any
}

// parseExecDynamic recognizes EXEC[UTE] ('sql' [+ 'sql']) and returns the literal SQL to run.
func parseExecDynamic(sql string) (string, bool) {
	s := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(sql), ";"))
	up := strings.ToUpper(s)
	switch {
	case strings.HasPrefix(up, "EXECUTE"):
		s = strings.TrimSpace(s[len("EXECUTE"):])
	case strings.HasPrefix(up, "EXEC"):
		s = strings.TrimSpace(s[len("EXEC"):])
	default:
		return "", false
	}
	if !strings.HasPrefix(s, "(") || !strings.HasSuffix(s, ")") {
		return "", false
	}
	inner := strings.TrimSpace(s[1 : len(s)-1])
	var b strings.Builder
	for i := 0; i < len(inner); {
		switch inner[i] {
		case '\'':
			j := i + 1
			for j < len(inner) {
				if inner[j] == '\'' {
					if j+1 < len(inner) && inner[j+1] == '\'' {
						b.WriteByte('\'')
						j += 2
						continue
					}
					break
				}
				b.WriteByte(inner[j])
				j++
			}
			i = j + 1
		case '+', ' ', '\t', '\r', '\n':
			i++
		default:
			return "", false
		}
	}
	if b.Len() == 0 {
		return "", false
	}
	return b.String(), true
}

// parseProcCall recognizes `EXEC[UTE] proc args` and bare `sp_proc args`; ok is false otherwise.
func parseProcCall(sql string) (string, []procArg, bool) {
	s := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(sql), ";"))
	up := strings.ToUpper(s)
	switch {
	case strings.HasPrefix(up, "EXECUTE "):
		s = strings.TrimSpace(s[len("EXECUTE "):])
	case strings.HasPrefix(up, "EXEC "):
		s = strings.TrimSpace(s[len("EXEC "):])
	case strings.HasPrefix(up, "SP_"):
	default:
		return "", nil, false
	}
	name, rest := splitName(s)
	name = unqualifyProc(name)
	if name == "" {
		return "", nil, false
	}
	return name, parseArgs(rest), true
}

// splitName takes the proc name (first whitespace- or comma-delimited token) and the argument tail.
func splitName(s string) (string, string) {
	i := strings.IndexAny(s, " \t\r\n")
	if i < 0 {
		return s, ""
	}
	return s[:i], strings.TrimSpace(s[i+1:])
}

func unqualifyProc(name string) string {
	name = strings.Trim(name, "[]\"")
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = strings.Trim(name[i+1:], "[]\"")
	}
	return name
}

func parseArgs(s string) []procArg {
	var args []procArg
	for _, part := range splitArgs(s) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name := ""
		if strings.HasPrefix(part, "@") {
			if eq := strings.Index(part, "="); eq >= 0 {
				name = strings.TrimSpace(part[:eq])
				part = strings.TrimSpace(part[eq+1:])
			}
		}
		args = append(args, procArg{name: name, val: parseLit(part)})
	}
	return args
}

// splitArgs splits on commas outside single-quoted strings.
func splitArgs(s string) []string {
	var out []string
	var sb strings.Builder
	inStr := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\'':
			inStr = !inStr
			sb.WriteByte(c)
		case c == ',' && !inStr:
			out = append(out, sb.String())
			sb.Reset()
		default:
			sb.WriteByte(c)
		}
	}
	if strings.TrimSpace(sb.String()) != "" {
		out = append(out, sb.String())
	}
	return out
}

func parseLit(s string) any {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "NULL") {
		return nil
	}
	if (strings.HasPrefix(s, "N'") || strings.HasPrefix(s, "n'")) && strings.HasSuffix(s, "'") {
		s = s[1:]
	}
	if len(s) >= 2 && strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		return strings.ReplaceAll(s[1:len(s)-1], "''", "'")
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	return s
}

// arg returns the proc argument by name (case-insensitive) or, failing that, by 0-based position.
func arg(args []procArg, name string, pos int) string {
	for _, a := range args {
		if a.name != "" && strings.EqualFold(a.name, name) {
			return str(a.val)
		}
	}
	named := 0
	for i, a := range args {
		if a.name != "" {
			named++
			continue
		}
		if i-named == pos {
			return str(a.val)
		}
	}
	return ""
}

func str(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if n, ok := v.(int64); ok {
		return strconv.FormatInt(n, 10)
	}
	return ""
}

func spDatabases(ctx context.Context, b tds.Backend) (tds.Rows, bool, error) {
	_, dbs, err := introspectSchema(ctx, b, &tds.Query{})
	if err != nil {
		return nil, true, err
	}
	cols := []catalog.Column{sn("DATABASE_NAME"), in32("DATABASE_SIZE"), nstr("REMARKS")}
	var data [][]any
	for _, db := range dbs {
		data = append(data, []any{db, nil, nil})
	}
	rs, err := exec.Apply(cols, data, &tds.Query{})
	return rs, true, err
}

func spTables(ctx context.Context, b tds.Backend, args []procArg) (tds.Rows, bool, error) {
	namePat := arg(args, "@table_name", 0)
	qualifier := defaultQualifier(ctx, arg(args, "@table_qualifier", 2))
	cols := []catalog.Column{sn("TABLE_QUALIFIER"), sn("TABLE_OWNER"), sn("TABLE_NAME"), str32("TABLE_TYPE"), nstr("REMARKS")}
	var data [][]any
	if !isSystemDB(qualifier) {
		schema, _, err := introspectSchema(ctx, b, &tds.Query{Database: qualifier})
		if err != nil {
			return nil, true, err
		}
		for _, t := range schema.Tables {
			if !matchLike(t.Name, namePat) {
				continue
			}
			data = append(data, []any{catOf(t, qualifier), "dbo", t.Name, "TABLE", nil})
		}
	}
	rs, err := exec.Apply(cols, data, &tds.Query{OrderBy: []tds.OrderItem{{Column: "TABLE_QUALIFIER"}, {Column: "TABLE_NAME"}}})
	return rs, true, err
}

func spColumns(ctx context.Context, b tds.Backend, args []procArg) (tds.Rows, bool, error) {
	tableName := arg(args, "@table_name", 0)
	qualifier := defaultQualifier(ctx, arg(args, "@table_qualifier", 2))
	colPat := arg(args, "@column_name", 3)
	cols := []catalog.Column{
		sn("TABLE_QUALIFIER"), sn("TABLE_OWNER"), sn("TABLE_NAME"), sn("COLUMN_NAME"),
		in16("DATA_TYPE"), sn("TYPE_NAME"), in32("PRECISION"), in32("LENGTH"),
		in16("SCALE"), in16("RADIX"), in16("NULLABLE"), nstr("REMARKS"),
		nstr("COLUMN_DEF"), in16("SQL_DATA_TYPE"), in16("SQL_DATETIME_SUB"), in32("CHAR_OCTET_LENGTH"),
		in32("ORDINAL_POSITION"), str32("IS_NULLABLE"),
	}
	var data [][]any
	if !isSystemDB(qualifier) {
		schema, _, err := introspectSchema(ctx, b, &tds.Query{Database: qualifier})
		if err != nil {
			return nil, true, err
		}
		for _, t := range schema.Tables {
			if !strings.EqualFold(t.Name, tableName) {
				continue
			}
			for i, c := range t.Columns {
				if colPat != "" && !matchLike(c.Name, colPat) {
					continue
				}
				odbc := odbcType(c.Type)
				data = append(data, []any{
					catOf(t, qualifier), "dbo", t.Name, c.Name,
					odbc, infoschema.TypeName(c.Type), typePrecision(c.Type), typeLength(c.Type),
					typeScale(c.Type), typeRadix(c.Type), nullableInt(c.Type), nil,
					nil, odbc, nil, charOctetLen(c.Type),
					int64(i + 1), yesNo(c.Type.Nullable),
				})
			}
		}
	}
	rs, err := exec.Apply(cols, data, &tds.Query{})
	return rs, true, err
}

// spHelptext is the classic definition dump: one row per line of the object's reconstructed CREATE text.
func spHelptext(ctx context.Context, b tds.Backend, args []procArg) (tds.Rows, bool, error) {
	name := unqualifyProc(arg(args, "@objname", 0))
	cols := []catalog.Column{sn("Text")}
	var data [][]any
	if store, ok := b.(tds.RoutineStore); ok && name != "" {
		if r, found, err := store.GetRoutine(ctx, currentDB(ctx), name); err == nil && found {
			for _, line := range strings.Split(routines.ScriptDefinition(r), "\n") {
				data = append(data, []any{line})
			}
		}
	}
	rs, err := exec.Apply(cols, data, &tds.Query{})
	return rs, true, err
}

// spHelp summarizes one object (Name/Owner/Type), or with no argument lists every object in the database.
func spHelp(ctx context.Context, b tds.Backend, args []procArg) (tds.Rows, bool, error) {
	name := unqualifyProc(arg(args, "@objname", 0))
	if name == "" {
		return spHelpList(ctx, b)
	}
	cols := []catalog.Column{sn("Name"), sn("Owner"), sn("Type"), nstr("Created_datetime")}
	var data [][]any
	if store, ok := b.(tds.RoutineStore); ok {
		if r, found, _ := store.GetRoutine(ctx, currentDB(ctx), name); found {
			data = append(data, []any{r.Name, "dbo", routineTypeLabel(r.Kind), nil})
		}
	}
	if len(data) == 0 {
		if schema, _, err := introspectSchema(ctx, b, &tds.Query{Database: currentDB(ctx)}); err == nil {
			for _, t := range schema.Tables {
				if strings.EqualFold(t.Name, name) {
					data = append(data, []any{t.Name, "dbo", "user table", nil})
					break
				}
			}
		}
	}
	rs, err := exec.Apply(cols, data, &tds.Query{})
	return rs, true, err
}

func spHelpList(ctx context.Context, b tds.Backend) (tds.Rows, bool, error) {
	cols := []catalog.Column{sn("Name"), sn("Owner"), sn("Object_type")}
	var data [][]any
	if schema, _, err := introspectSchema(ctx, b, &tds.Query{Database: currentDB(ctx)}); err == nil {
		for _, t := range schema.Tables {
			data = append(data, []any{t.Name, "dbo", "user table"})
		}
	}
	for _, r := range listRoutines(ctx, b, currentDB(ctx)) {
		data = append(data, []any{r.Name, "dbo", routineTypeLabel(r.Kind)})
	}
	rs, err := exec.Apply(cols, data, &tds.Query{OrderBy: []tds.OrderItem{{Column: "Name"}}})
	return rs, true, err
}

func routineTypeLabel(k tds.RoutineKind) string {
	switch k {
	case tds.RoutineProc:
		return "stored procedure"
	case tds.RoutineFunc:
		return "scalar function"
	case tds.RoutineTrigger:
		return "trigger"
	default:
		return "view"
	}
}

// spPkeys is the ODBC/JDBC getPrimaryKeys backing proc: the PK columns of a table, in key order.
func spPkeys(ctx context.Context, b tds.Backend, args []procArg) (tds.Rows, bool, error) {
	table := unqualifyProc(arg(args, "@table_name", 0))
	qualifier := defaultQualifier(ctx, arg(args, "@table_qualifier", 2))
	cols := []catalog.Column{sn("TABLE_QUALIFIER"), sn("TABLE_OWNER"), sn("TABLE_NAME"), sn("COLUMN_NAME"), in16("KEY_SEQ"), sn("PK_NAME")}
	var data [][]any
	if t, ok, err := findTable(ctx, b, qualifier, table); err != nil {
		return nil, true, err
	} else if ok {
		for i, c := range t.PrimaryKey {
			data = append(data, []any{catOf(t, qualifier), "dbo", t.Name, c, int64(i + 1), "PK_" + t.Name})
		}
	}
	rs, err := exec.Apply(cols, data, &tds.Query{})
	return rs, true, err
}

// spFkeys is the getImportedKeys/getExportedKeys backing proc: FK column pairs, filtered by the PK
// and/or FK table name.
func spFkeys(ctx context.Context, b tds.Backend, args []procArg) (tds.Rows, bool, error) {
	pktable := unqualifyProc(arg(args, "@pktable_name", 0))
	fktable := unqualifyProc(arg(args, "@fktable_name", 3))
	qualifier := defaultQualifier(ctx, arg(args, "@fktable_qualifier", 5))
	cols := []catalog.Column{
		sn("PKTABLE_QUALIFIER"), sn("PKTABLE_OWNER"), sn("PKTABLE_NAME"), sn("PKCOLUMN_NAME"),
		sn("FKTABLE_QUALIFIER"), sn("FKTABLE_OWNER"), sn("FKTABLE_NAME"), sn("FKCOLUMN_NAME"),
		in16("KEY_SEQ"), in16("UPDATE_RULE"), in16("DELETE_RULE"), sn("FK_NAME"), sn("PK_NAME"), in16("DEFERRABILITY"),
	}
	var data [][]any
	schema, _, err := introspectSchema(ctx, b, &tds.Query{Database: qualifier})
	if err != nil {
		return nil, true, err
	}
	for _, t := range schema.Tables {
		if fktable != "" && !strings.EqualFold(t.Name, fktable) {
			continue
		}
		for _, fk := range t.ForeignKeys {
			if pktable != "" && !strings.EqualFold(fk.RefTable, pktable) {
				continue
			}
			for k, c := range fk.Columns {
				pkcol := ""
				if k < len(fk.RefColumns) {
					pkcol = fk.RefColumns[k]
				}
				data = append(data, []any{
					qualifier, "dbo", fk.RefTable, pkcol, qualifier, "dbo", t.Name, c,
					int64(k + 1), int64(1), int64(1), "FK_" + t.Name + "_" + fk.RefTable, "PK_" + fk.RefTable, int64(7),
				})
			}
		}
	}
	rs, err := exec.Apply(cols, data, &tds.Query{})
	return rs, true, err
}

// spStatistics is the getIndexInfo backing proc: one row per index column.
func spStatistics(ctx context.Context, b tds.Backend, args []procArg) (tds.Rows, bool, error) {
	table := unqualifyProc(arg(args, "@table_name", 0))
	qualifier := defaultQualifier(ctx, arg(args, "@table_qualifier", 2))
	cols := []catalog.Column{
		sn("TABLE_QUALIFIER"), sn("TABLE_OWNER"), sn("TABLE_NAME"), in16("NON_UNIQUE"), sn("INDEX_QUALIFIER"),
		sn("INDEX_NAME"), in16("TYPE"), in16("SEQ_IN_INDEX"), sn("COLUMN_NAME"), str32("COLLATION"),
		in32("CARDINALITY"), in32("PAGES"), nstr("FILTER_CONDITION"),
	}
	var data [][]any
	if t, ok, err := findTable(ctx, b, qualifier, table); err != nil {
		return nil, true, err
	} else if ok {
		for _, ix := range sysviews.TableIndexes(t) {
			nonUnique, typ := int64(1), int64(3)
			if ix.Unique {
				nonUnique = 0
			}
			if ix.Clustered {
				typ = 1
			}
			for k, c := range ix.Columns {
				data = append(data, []any{
					catOf(t, qualifier), "dbo", t.Name, nonUnique, qualifier, ix.Name, typ, int64(k + 1), c, "A", nil, nil, nil,
				})
			}
		}
	}
	rs, err := exec.Apply(cols, data, &tds.Query{})
	return rs, true, err
}

// spSpecialColumns is the getBestRowIdentifier backing proc: the PK columns as the best row id.
func spSpecialColumns(ctx context.Context, b tds.Backend, args []procArg) (tds.Rows, bool, error) {
	table := unqualifyProc(arg(args, "@table_name", 0))
	qualifier := defaultQualifier(ctx, arg(args, "@table_qualifier", 2))
	cols := []catalog.Column{
		in16("SCOPE"), sn("COLUMN_NAME"), in16("DATA_TYPE"), sn("TYPE_NAME"),
		in32("PRECISION"), in32("LENGTH"), in16("SCALE"), in16("PSEUDO_COLUMN"),
	}
	var data [][]any
	if t, ok, err := findTable(ctx, b, qualifier, table); err != nil {
		return nil, true, err
	} else if ok {
		for _, c := range t.PrimaryKey {
			cd, found := colDef(t, c)
			if !found {
				continue
			}
			data = append(data, []any{
				int64(0), c, odbcType(cd.Type), infoschema.TypeName(cd.Type),
				typePrecision(cd.Type), typeLength(cd.Type), typeScale(cd.Type), int64(1),
			})
		}
	}
	rs, err := exec.Apply(cols, data, &tds.Query{})
	return rs, true, err
}

// spStoredProcedures is the getProcedures backing proc: stored procedures and functions.
func spStoredProcedures(ctx context.Context, b tds.Backend, args []procArg) (tds.Rows, bool, error) {
	qualifier := defaultQualifier(ctx, arg(args, "@procedure_qualifier", 2))
	namePat := arg(args, "@procedure_name", 0)
	cols := []catalog.Column{
		sn("PROCEDURE_QUALIFIER"), sn("PROCEDURE_OWNER"), sn("PROCEDURE_NAME"),
		in32("NUM_INPUT_PARAMS"), in32("NUM_OUTPUT_PARAMS"), in32("NUM_RESULT_SETS"), nstr("REMARKS"), in16("PROCEDURE_TYPE"),
	}
	var data [][]any
	for _, r := range listRoutines(ctx, b, qualifier) {
		if r.Kind != tds.RoutineProc && r.Kind != tds.RoutineFunc {
			continue
		}
		if !matchLike(r.Name, namePat) {
			continue
		}
		ptype := int64(1)
		if r.Kind == tds.RoutineFunc {
			ptype = 2
		}
		data = append(data, []any{qualifier, "dbo", r.Name + ";1", int64(len(r.Params)), int64(0), int64(-1), nil, ptype})
	}
	rs, err := exec.Apply(cols, data, &tds.Query{})
	return rs, true, err
}

// spSprocColumns is the getProcedureColumns backing proc: a procedure's/function's parameters.
func spSprocColumns(ctx context.Context, b tds.Backend, args []procArg) (tds.Rows, bool, error) {
	proc := unqualifyProc(arg(args, "@procedure_name", 0))
	qualifier := defaultQualifier(ctx, arg(args, "@procedure_qualifier", 2))
	cols := []catalog.Column{
		sn("PROCEDURE_QUALIFIER"), sn("PROCEDURE_OWNER"), sn("PROCEDURE_NAME"), sn("COLUMN_NAME"),
		in16("COLUMN_TYPE"), in16("DATA_TYPE"), sn("TYPE_NAME"), in32("PRECISION"), in32("LENGTH"),
		in16("SCALE"), in16("RADIX"), in16("NULLABLE"), nstr("REMARKS"),
	}
	var data [][]any
	for _, r := range listRoutines(ctx, b, qualifier) {
		if !strings.EqualFold(r.Name, proc) {
			continue
		}
		for _, p := range r.Params {
			ty := declType(p.Type)
			data = append(data, []any{
				qualifier, "dbo", r.Name + ";1", p.Name, int64(1), odbcType(ty), infoschema.TypeName(ty),
				typePrecision(ty), typeLength(ty), typeScale(ty), typeRadix(ty), int64(1), nil,
			})
		}
	}
	rs, err := exec.Apply(cols, data, &tds.Query{})
	return rs, true, err
}

// spHelpindex is the SSMS index summary: name, description, and key columns per index.
func spHelpindex(ctx context.Context, b tds.Backend, args []procArg) (tds.Rows, bool, error) {
	table := unqualifyProc(arg(args, "@objname", 0))
	qualifier := currentDB(ctx)
	cols := []catalog.Column{sn("index_name"), nstr("index_description"), nstr("index_keys")}
	var data [][]any
	if t, ok, err := findTable(ctx, b, qualifier, table); err != nil {
		return nil, true, err
	} else if ok {
		for _, ix := range sysviews.TableIndexes(t) {
			data = append(data, []any{ix.Name, indexDesc(ix), strings.Join(ix.Columns, ", ")})
		}
	}
	rs, err := exec.Apply(cols, data, &tds.Query{})
	return rs, true, err
}

// spHelpconstraint is the SSMS constraint summary: PK / FK / CHECK / DEFAULT on a table.
func spHelpconstraint(ctx context.Context, b tds.Backend, args []procArg) (tds.Rows, bool, error) {
	table := unqualifyProc(arg(args, "@objname", 0))
	qualifier := currentDB(ctx)
	cols := []catalog.Column{sn("constraint_type"), sn("constraint_name"), nstr("constraint_keys")}
	var data [][]any
	if t, ok, err := findTable(ctx, b, qualifier, table); err != nil {
		return nil, true, err
	} else if ok {
		if len(t.PrimaryKey) > 0 {
			data = append(data, []any{"PRIMARY KEY (clustered)", "PK_" + t.Name, strings.Join(t.PrimaryKey, ", ")})
		}
		for _, fk := range t.ForeignKeys {
			keys := strings.Join(fk.Columns, ", ") + " REFERENCES " + fk.RefTable + " (" + strings.Join(fk.RefColumns, ", ") + ")"
			data = append(data, []any{"FOREIGN KEY", "FK_" + t.Name + "_" + fk.RefTable, keys})
		}
		for _, ck := range t.Checks {
			data = append(data, []any{"CHECK", ck.Name, ck.Expression})
		}
		for _, c := range t.Columns {
			if c.Default != "" {
				data = append(data, []any{"DEFAULT on column " + c.Name, "DF_" + t.Name + "_" + c.Name, c.Default})
			}
		}
	}
	rs, err := exec.Apply(cols, data, &tds.Query{})
	return rs, true, err
}

// spServerInfo is the SQLGetInfo backing proc; @attribute_id narrows to one attribute.
func spServerInfo(_ context.Context, _ tds.Backend, args []procArg) (tds.Rows, bool, error) {
	want := arg(args, "@attribute_id", 0)
	cols := []catalog.Column{in32("attribute_id"), sn("attribute_name"), nstr("attribute_value")}
	var data [][]any
	for _, r := range serverInfoRows {
		if want != "" && strconv.FormatInt(r.id, 10) != want {
			continue
		}
		data = append(data, []any{r.id, r.name, r.value})
	}
	rs, err := exec.Apply(cols, data, &tds.Query{})
	return rs, true, err
}

type serverInfoRow struct {
	id    int64
	name  string
	value string
}

var serverInfoRows = []serverInfoRow{
	{1, "DBMS_NAME", "Microsoft SQL Server"},
	{2, "DBMS_VER", "Microsoft SQL Server 2022 - 16.0.1000.6"},
	{10, "OWNER_TERM", "owner"},
	{11, "TABLE_TERM", "table"},
	{12, "MAX_OWNER_NAME_LENGTH", "128"},
	{13, "TABLE_LENGTH", "128"},
	{14, "MAX_QUAL_LENGTH", "128"},
	{15, "COLUMN_LENGTH", "128"},
	{16, "IDENTIFIER_CASE", "MIXED"},
	{17, "TX_ISOLATION", "2"},
	{18, "COLLATION_SEQ", "charset=iso_1 sort_order=nocase_iso"},
	{19, "SAVEPOINT_SUPPORT", "Y"},
	{20, "MULTI_RESULT_SETS", "Y"},
	{22, "ACCESSIBLE_TABLES", "Y"},
	{100, "USERID_LENGTH", "128"},
	{101, "QUALIFIER_TERM", "database"},
	{102, "NAMED_TRANSACTIONS", "Y"},
	{103, "SPROC_AS_LANGUAGE", "Y"},
	{104, "ACCESSIBLE_PROC", "Y"},
	{105, "MAX_INDEX_COLS", "16"},
	{106, "RENAME_TABLE", "Y"},
	{107, "RENAME_COLUMN", "Y"},
	{108, "DROP_COLUMN", "Y"},
	{111, "DATA_SOURCE_NAME", "haystak-tds-spi"},
}

// spDatatypeInfo is the SQLGetTypeInfo backing proc; @data_type narrows to one ODBC type code.
func spDatatypeInfo(_ context.Context, _ tds.Backend, args []procArg) (tds.Rows, bool, error) {
	want := arg(args, "@data_type", 0)
	cols := []catalog.Column{
		sn("TYPE_NAME"), in16("DATA_TYPE"), in32("PRECISION"), str32("LITERAL_PREFIX"), str32("LITERAL_SUFFIX"),
		str32("CREATE_PARAMS"), in16("NULLABLE"), in16("CASE_SENSITIVE"), in16("SEARCHABLE"), in16("UNSIGNED_ATTRIBUTE"),
		in16("MONEY"), in16("AUTO_INCREMENT"), sn("LOCAL_TYPE_NAME"), in16("MINIMUM_SCALE"), in16("MAXIMUM_SCALE"),
		in16("SQL_DATA_TYPE"), in16("SQL_DATETIME_SUB"), in32("NUM_PREC_RADIX"), in16("INTERVAL_PRECISION"), in16("USERTYPE"),
	}
	var data [][]any
	for _, r := range datatypeInfoRows {
		if want != "" && strconv.FormatInt(r.dataType, 10) != want {
			continue
		}
		data = append(data, r.row())
	}
	rs, err := exec.Apply(cols, data, &tds.Query{})
	return rs, true, err
}

type datatypeRow struct {
	name     string
	dataType int64 // ODBC type code
	prec     int64
	prefix   string
	suffix   string
	params   string // CREATE_PARAMS
	money    int64
	numeric  bool // radix 10 + signed + identity-capable
	minScale any
	maxScale any
	userType int64 // sys.types xtype
}

func (d datatypeRow) row() []any {
	var radix, unsigned, autoInc any
	if d.numeric {
		radix, unsigned, autoInc = int64(10), int64(0), int64(0)
	}
	return []any{
		d.name, d.dataType, d.prec, nullStr(d.prefix), nullStr(d.suffix),
		nullStr(d.params), int64(1), int64(0), int64(3), unsigned,
		d.money, autoInc, d.name, d.minScale, d.maxScale,
		d.dataType, nil, radix, nil, d.userType,
	}
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

var datatypeInfoRows = []datatypeRow{
	{name: "bit", dataType: -7, prec: 1, money: 0, userType: 104},
	{name: "tinyint", dataType: -6, prec: 3, money: 0, numeric: true, minScale: int64(0), maxScale: int64(0), userType: 48},
	{name: "smallint", dataType: 5, prec: 5, money: 0, numeric: true, minScale: int64(0), maxScale: int64(0), userType: 52},
	{name: "int", dataType: 4, prec: 10, money: 0, numeric: true, minScale: int64(0), maxScale: int64(0), userType: 56},
	{name: "bigint", dataType: -5, prec: 19, money: 0, numeric: true, minScale: int64(0), maxScale: int64(0), userType: 127},
	{name: "real", dataType: 7, prec: 24, money: 0, numeric: true, userType: 59},
	{name: "float", dataType: 8, prec: 53, params: "length", money: 0, numeric: true, userType: 62},
	{name: "decimal", dataType: 3, prec: 38, params: "precision,scale", money: 0, numeric: true, minScale: int64(0), maxScale: int64(38), userType: 106},
	{name: "numeric", dataType: 2, prec: 38, params: "precision,scale", money: 0, numeric: true, minScale: int64(0), maxScale: int64(38), userType: 108},
	{name: "money", dataType: 3, prec: 19, prefix: "$", money: 1, numeric: true, minScale: int64(4), maxScale: int64(4), userType: 60},
	{name: "char", dataType: 1, prec: 8000, prefix: "'", suffix: "'", params: "length", money: 0, userType: 175},
	{name: "varchar", dataType: 12, prec: 8000, prefix: "'", suffix: "'", params: "max length", money: 0, userType: 167},
	{name: "nchar", dataType: -8, prec: 4000, prefix: "N'", suffix: "'", params: "length", money: 0, userType: 239},
	{name: "nvarchar", dataType: -9, prec: 4000, prefix: "N'", suffix: "'", params: "max length", money: 0, userType: 231},
	{name: "binary", dataType: -2, prec: 8000, prefix: "0x", params: "length", money: 0, userType: 173},
	{name: "varbinary", dataType: -3, prec: 8000, prefix: "0x", params: "max length", money: 0, userType: 165},
	{name: "uniqueidentifier", dataType: -11, prec: 36, prefix: "'", suffix: "'", money: 0, userType: 36},
	{name: "date", dataType: 91, prec: 10, prefix: "'", suffix: "'", money: 0, userType: 40},
	{name: "time", dataType: 92, prec: 16, prefix: "'", suffix: "'", params: "scale", money: 0, minScale: int64(0), maxScale: int64(7), userType: 41},
	{name: "datetime", dataType: 93, prec: 23, prefix: "'", suffix: "'", money: 0, minScale: int64(3), maxScale: int64(3), userType: 61},
	{name: "datetime2", dataType: 93, prec: 27, prefix: "'", suffix: "'", params: "scale", money: 0, minScale: int64(0), maxScale: int64(7), userType: 42},
}

// spConfigure lists server configuration options (no-argument form): name + min/max/config/run values.
func spConfigure(_ context.Context, _ tds.Backend, _ []procArg) (tds.Rows, bool, error) {
	cols := []catalog.Column{sn("name"), in32("minimum"), in32("maximum"), in32("config_value"), in32("run_value")}
	opts := []struct {
		name          string
		min, max, val int64
	}{
		{"cost threshold for parallelism", 0, 32767, 5},
		{"fill factor (%)", 0, 100, 0},
		{"max degree of parallelism", 0, 32767, 0},
		{"max server memory (MB)", 16, 2147483647, 2147483647},
		{"min server memory (MB)", 0, 2147483647, 0},
		{"optimize for ad hoc workloads", 0, 1, 0},
	}
	var data [][]any
	for _, o := range opts {
		data = append(data, []any{o.name, o.min, o.max, o.val, o.val})
	}
	rs, err := exec.Apply(cols, data, &tds.Query{})
	return rs, true, err
}

// spLock is the legacy lock report: empty, no lock manager exists.
func spLock(_ context.Context, _ tds.Backend, _ []procArg) (tds.Rows, bool, error) {
	cols := []catalog.Column{
		in16("spid"), in16("dbid"), in32("ObjId"), in16("IndId"), str32("Type"),
		nstr("Resource"), str32("Mode"), str32("Status"),
	}
	rs, err := exec.Apply(cols, nil, &tds.Query{})
	return rs, true, err
}

// spHelpdb lists the databases (no-argument form): name, size, owner, dbid, status, compat level.
func spHelpdb(ctx context.Context, b tds.Backend, args []procArg) (tds.Rows, bool, error) {
	_, dbs, err := introspectSchema(ctx, b, &tds.Query{})
	if err != nil {
		return nil, true, err
	}
	cols := []catalog.Column{
		sn("name"), str32("db_size"), sn("owner"), in16("dbid"), nstr("created"), nstr("status"), in16("compatibility_level"),
	}
	id := int64(0)
	row := func(name string) []any {
		id++
		return []any{name, "       16.00 MB", "sa", id, nil, "Status=ONLINE", int64(160)}
	}
	data := [][]any{row("master"), row("tempdb"), row("model"), row("msdb")}
	for _, db := range dbs {
		data = append(data, row(db))
	}
	rs, err := exec.Apply(cols, data, &tds.Query{OrderBy: []tds.OrderItem{{Column: "name"}}})
	return rs, true, err
}

// findTable looks up one table by name in a database. introspectSchema returns an empty schema for a
// multi-db backend's non-existent database (e.g. master), and all tables for a single-db backend.
func findTable(ctx context.Context, b tds.Backend, qualifier, name string) (catalog.Table, bool, error) {
	if name == "" {
		return catalog.Table{}, false, nil
	}
	schema, _, err := introspectSchema(ctx, b, &tds.Query{Database: qualifier})
	if err != nil {
		return catalog.Table{}, false, err
	}
	for _, t := range schema.Tables {
		if strings.EqualFold(t.Name, name) {
			return t, true, nil
		}
	}
	return catalog.Table{}, false, nil
}

func colDef(t catalog.Table, name string) (catalog.Column, bool) {
	for _, c := range t.Columns {
		if strings.EqualFold(c.Name, name) {
			return c, true
		}
	}
	return catalog.Column{}, false
}

func indexDesc(ix catalog.Index) string {
	parts := []string{"nonclustered"}
	if ix.Clustered {
		parts[0] = "clustered"
	}
	if ix.Unique {
		parts = append(parts, "unique")
	}
	if ix.Primary {
		parts = append(parts, "primary key")
	}
	return strings.Join(parts, ", ") + " located on PRIMARY"
}

// declType maps a declared T-SQL type ("int", "nvarchar(50)", "decimal(10,2)") to the value model.
func declType(decl string) types.Type {
	s := strings.ToLower(strings.TrimSpace(decl))
	inner := ""
	if i := strings.IndexByte(s, '('); i >= 0 {
		if j := strings.IndexByte(s, ')'); j > i {
			inner = s[i+1 : j]
		}
		s = strings.TrimSpace(s[:i])
	}
	t := types.Type{}
	switch s {
	case "bit":
		t.Kind = types.Bool
	case "tinyint", "smallint", "int", "integer":
		t.Kind = types.Int32
	case "bigint":
		t.Kind = types.Int64
	case "real", "float":
		t.Kind = types.Float64
	case "decimal", "dec", "numeric", "money", "smallmoney":
		t.Kind = types.Decimal
	case "date", "time", "datetime", "datetime2", "smalldatetime":
		t.Kind = types.Time
	case "uniqueidentifier":
		t.Kind = types.UUID
	case "binary", "varbinary", "image":
		t.Kind = types.Bytes
	default:
		t.Kind = types.String
	}
	if inner != "" {
		if t.Kind == types.Decimal {
			parts := strings.SplitN(inner, ",", 2)
			t.Precision = atoiOr(parts[0], 18)
			if len(parts) == 2 {
				t.Scale = atoiOr(parts[1], 0)
			}
		} else if n, err := strconv.Atoi(strings.TrimSpace(inner)); err == nil {
			t.MaxLen = n
		}
	}
	return t
}

func atoiOr(s string, d int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return n
	}
	return d
}

// defaultQualifier uses the explicit @table_qualifier if given, else the session's current database.
func defaultQualifier(ctx context.Context, explicit string) string {
	if explicit != "" {
		return explicit
	}
	return currentDB(ctx)
}

func isSystemDB(db string) bool {
	switch strings.ToLower(db) {
	case "master", "tempdb", "model", "msdb":
		return true
	}
	return false
}

func catOf(t catalog.Table, fallback string) any {
	if t.Catalog != "" {
		return t.Catalog
	}
	if fallback != "" {
		return fallback
	}
	return nil
}

// matchLike does a case-insensitive SQL LIKE with %/_ wildcards; empty/"%" matches everything.
func matchLike(s, pat string) bool {
	if pat == "" || pat == "%" {
		return true
	}
	s, pat = strings.ToLower(s), strings.ToLower(pat)
	return likeMatch(s, pat)
}

func likeMatch(s, pat string) bool {
	if pat == "" {
		return s == ""
	}
	if pat[0] == '%' {
		for i := 0; i <= len(s); i++ {
			if likeMatch(s[i:], pat[1:]) {
				return true
			}
		}
		return false
	}
	if s == "" {
		return false
	}
	if pat[0] == '_' || pat[0] == s[0] {
		return likeMatch(s[1:], pat[1:])
	}
	return false
}

func declBase(decl string) string {
	name := strings.ToLower(strings.TrimSpace(decl))
	if i := strings.IndexByte(name, '('); i >= 0 {
		name = strings.TrimSpace(name[:i])
	}
	return name
}

func odbcType(t types.Type) int64 {
	switch declBase(t.Name) {
	case "bit":
		return -7
	case "tinyint":
		return -6
	case "smallint":
		return 5
	case "int":
		return 4
	case "bigint":
		return -5
	case "real":
		return 7
	case "float":
		return 8
	case "decimal", "money", "smallmoney":
		return 3
	case "numeric":
		return 2
	case "char":
		return 1
	case "varchar":
		return 12
	case "nchar":
		return -8
	case "nvarchar":
		return -9
	case "binary":
		return -2
	case "varbinary":
		return -3
	case "uniqueidentifier":
		return -11
	case "date":
		return 91
	case "time":
		return 92
	case "datetime", "smalldatetime", "datetime2":
		return 93
	case "datetimeoffset":
		return -155
	case "text":
		return -1
	case "ntext":
		return -10
	case "image":
		return -4
	case "xml":
		return -152
	}
	switch t.Kind {
	case types.Bool:
		return -7
	case types.Int32:
		return 4
	case types.Int64:
		return -5
	case types.Float64:
		return 8
	case types.Decimal:
		return 3
	case types.Bytes:
		return -3
	case types.Time:
		return 93
	case types.UUID:
		return -11
	default:
		return -9
	}
}

func typePrecision(t types.Type) int64 {
	switch t.Kind {
	case types.Int32:
		return 10
	case types.Int64:
		return 19
	case types.Float64:
		return 53
	case types.Decimal:
		if t.Precision > 0 {
			return int64(t.Precision)
		}
		return 18
	case types.String:
		if t.MaxLen > 0 {
			return int64(t.MaxLen)
		}
		return 0
	}
	return 0
}

func typeLength(t types.Type) int64 {
	switch declBase(t.Name) {
	case "tinyint":
		return 1
	case "smallint":
		return 2
	case "bigint":
		return 8
	case "real":
		return 4
	case "char", "varchar", "binary", "varbinary":
		if t.MaxLen > 0 {
			return int64(t.MaxLen)
		}
		return 0
	case "nchar", "nvarchar":
		if t.MaxLen > 0 {
			return int64(t.MaxLen * 2)
		}
		return 0
	}
	switch t.Kind {
	case types.Int32:
		return 4
	case types.Int64:
		return 8
	case types.Float64:
		return 8
	case types.Bool:
		return 1
	case types.String:
		if t.MaxLen > 0 {
			return int64(t.MaxLen * 2)
		}
		return 0
	}
	return typePrecision(t)
}

func typeScale(t types.Type) any {
	if t.Kind == types.Decimal {
		return int64(t.Scale)
	}
	return nil
}

func typeRadix(t types.Type) any {
	switch t.Kind {
	case types.Int32, types.Int64, types.Float64, types.Decimal:
		return int64(10)
	}
	return nil
}

func charOctetLen(t types.Type) any {
	switch declBase(t.Name) {
	case "char", "varchar":
		if t.MaxLen > 0 {
			return int64(t.MaxLen)
		}
		return nil
	case "nchar", "nvarchar":
		if t.MaxLen > 0 {
			return int64(t.MaxLen * 2)
		}
		return nil
	}
	if t.Kind == types.String && t.MaxLen > 0 {
		return int64(t.MaxLen * 2)
	}
	return nil
}

func nullableInt(t types.Type) int64 {
	if t.Nullable {
		return 1
	}
	return 0
}

func yesNo(nullable bool) string {
	if nullable {
		return "YES"
	}
	return "NO"
}

func spTablePrivileges(_ context.Context, _ tds.Backend, _ []procArg) (tds.Rows, bool, error) {
	cols := []catalog.Column{
		sn("TABLE_QUALIFIER"), sn("TABLE_OWNER"), sn("TABLE_NAME"),
		nstr("GRANTOR"), nstr("GRANTEE"), sn("PRIVILEGE"), nstr("IS_GRANTABLE"),
	}
	rs, err := exec.Apply(cols, nil, &tds.Query{}) // no permission model: correct columns, zero rows
	return rs, true, err
}

func spColumnPrivileges(_ context.Context, _ tds.Backend, _ []procArg) (tds.Rows, bool, error) {
	cols := []catalog.Column{
		sn("TABLE_QUALIFIER"), sn("TABLE_OWNER"), sn("TABLE_NAME"), sn("COLUMN_NAME"),
		nstr("GRANTOR"), nstr("GRANTEE"), sn("PRIVILEGE"), nstr("IS_GRANTABLE"),
	}
	rs, err := exec.Apply(cols, nil, &tds.Query{})
	return rs, true, err
}

// spHelptrigger projects the trigger routines whose ON-table matches @tabname.
func spHelptrigger(ctx context.Context, b tds.Backend, args []procArg) (tds.Rows, bool, error) {
	table := unqualifyProc(arg(args, "@tabname", 0))
	cols := []catalog.Column{
		sn("trigger_name"), sn("trigger_owner"), in16("isupdate"), in16("isdelete"),
		in16("isinsert"), in16("isafter"), in16("isinsteadof"), nstr("trigger_schema"),
	}
	var data [][]any
	for _, r := range listRoutines(ctx, b, defaultQualifier(ctx, "")) {
		if r.Kind != tds.RoutineTrigger {
			continue
		}
		if table != "" && !strings.EqualFold(triggerOnTable(r.Body), table) {
			continue
		}
		data = append(data, []any{r.Name, "dbo", int64(1), int64(1), int64(1), int64(1), int64(0), "dbo"})
	}
	rs, err := exec.Apply(cols, data, &tds.Query{})
	return rs, true, err
}

// spDepends projects the objects the named routine references (its FROM/JOIN tables).
func spDepends(ctx context.Context, b tds.Backend, args []procArg) (tds.Rows, bool, error) {
	name := unqualifyProc(arg(args, "@objname", 0))
	cols := []catalog.Column{sn("name"), sn("type"), sn("updated"), sn("selected"), nstr("column")}
	var data [][]any
	for _, r := range listRoutines(ctx, b, defaultQualifier(ctx, "")) {
		if !strings.EqualFold(routines.Unqualify(r.Name), name) {
			continue
		}
		for _, ref := range routines.ReferencedNames(r.Body) {
			data = append(data, []any{ref, "user table", "no", "yes", nil})
		}
	}
	rs, err := exec.Apply(cols, data, &tds.Query{})
	return rs, true, err
}

func triggerOnTable(body string) string {
	tokens := strings.Fields(body)
	for i := 0; i+1 < len(tokens); i++ {
		if strings.EqualFold(tokens[i], "ON") {
			return routines.Unqualify(tokens[i+1])
		}
	}
	return ""
}

func sn(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.String, MaxLen: 128}}
}
func nstr(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.String, Nullable: true}}
}
func str32(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.String, MaxLen: 32}}
}
func in16(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.Int32, Nullable: true}}
}
func in32(n string) catalog.Column {
	return catalog.Column{Name: n, Type: types.Type{Kind: types.Int32, Nullable: true}}
}

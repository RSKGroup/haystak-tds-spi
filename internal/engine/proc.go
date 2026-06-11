// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"strconv"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/internal/exec"
	"github.com/RSKGroup/haystak-tds-spi/internal/infoschema"
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

// execProc answers the ODBC catalog procs (sp_databases/sp_tables/sp_columns); other sp_* no-op.
func execProc(ctx context.Context, b tds.Backend, sql string) (tds.Rows, bool, error) {
	name, args, ok := parseProcCall(sql)
	if !ok {
		return nil, false, nil
	}
	switch strings.ToLower(name) {
	case "sp_databases":
		return spDatabases(ctx, b)
	case "sp_tables":
		return spTables(ctx, b, args)
	case "sp_columns", "sp_columns_90":
		return spColumns(ctx, b, args)
	default:
		return nil, true, nil
	}
}

type procArg struct {
	name string
	val  any
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

func odbcType(t types.Type) int64 {
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

func sn(n string) catalog.Column   { return catalog.Column{Name: n, Type: types.Type{Kind: types.String, MaxLen: 128}} }
func nstr(n string) catalog.Column { return catalog.Column{Name: n, Type: types.Type{Kind: types.String, Nullable: true}} }
func str32(n string) catalog.Column { return catalog.Column{Name: n, Type: types.Type{Kind: types.String, MaxLen: 32}} }
func in16(n string) catalog.Column { return catalog.Column{Name: n, Type: types.Type{Kind: types.Int32, Nullable: true}} }
func in32(n string) catalog.Column { return catalog.Column{Name: n, Type: types.Type{Kind: types.Int32, Nullable: true}} }

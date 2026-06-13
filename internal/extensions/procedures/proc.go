// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package procedures implements stored procedures: CREATE/DROP PROCEDURE persist the parameter list
// plus the raw body, and EXEC substitutes arguments for parameters and runs it. Procedural control
// flow (DECLARE/IF/WHILE/…) lives one file per construct under control/.
package procedures

import (
	"context"
	"fmt"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/routines"
	"github.com/RSKGroup/haystak-tds-spi/tds"
)

// HandleDDL persists CREATE/ALTER/DROP PROCEDURE; handled is false for any other SQL.
func HandleDDL(ctx context.Context, store tds.RoutineStore, db, sql string) (bool, error) {
	s := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(sql), ";"))
	up := strings.ToUpper(s)
	switch {
	case routines.HasHead(up, "CREATE", "PROCEDURE"), routines.HasHead(up, "CREATE", "PROC"),
		routines.HasHead(up, "CREATE OR ALTER", "PROCEDURE"), routines.HasHead(up, "CREATE OR ALTER", "PROC"),
		routines.HasHead(up, "ALTER", "PROCEDURE"), routines.HasHead(up, "ALTER", "PROC"):
		return true, createProc(ctx, store, db, s)
	case routines.HasHead(up, "DROP", "PROCEDURE"), routines.HasHead(up, "DROP", "PROC"):
		return true, routines.Drop(ctx, store, db, routines.AfterDrop(s, "PROCEDURE", "PROC"))
	}
	return false, nil
}

func createProc(ctx context.Context, store tds.RoutineStore, db, s string) error {
	head, body, ok := routines.SplitAS(s)
	if !ok {
		return fmt.Errorf("CREATE PROCEDURE: missing AS")
	}
	name, params := parseProcHead(head)
	if name == "" {
		return fmt.Errorf("CREATE PROCEDURE: missing name")
	}
	return store.PutRoutine(ctx, &tds.Routine{Database: db, Schema: "dbo", Name: name, Kind: tds.RoutineProc, Body: body, Params: params})
}

// ExecProc runs a stored procedure: it substitutes the arguments for the parameters in the body and
// executes it. found is false when name isn't a stored procedure.
func ExecProc(ctx context.Context, store tds.RoutineStore, run routines.Runner, name string, args []routines.Arg) (tds.Rows, bool, error) {
	r, found, err := store.GetRoutine(ctx, run.CurrentDB(ctx), name)
	if err != nil || !found || r.Kind != tds.RoutineProc {
		return nil, false, err
	}
	rows, e := run.Exec(ctx, substituteParams(r, args))
	return rows, true, e
}

// parseProcHead splits a proc header ("CREATE [OR ALTER] PROC[EDURE] name @p type, …") into the name
// and parameters; the parameter list may optionally be wrapped in parentheses.
func parseProcHead(head string) (string, []tds.RoutineParam) {
	rest := head
	if i := routines.IndexFold(rest, "PROCEDURE"); i >= 0 {
		rest = rest[i+len("PROCEDURE"):]
	} else if i := routines.IndexFold(rest, "PROC"); i >= 0 {
		rest = rest[i+len("PROC"):]
	} else {
		return "", nil
	}
	rest = strings.TrimSpace(rest)
	name, after := splitName(rest)
	after = strings.TrimSpace(after)
	after = strings.TrimPrefix(after, "(")
	after = strings.TrimSuffix(strings.TrimSpace(after), ")")
	var params []tds.RoutineParam
	for _, part := range splitTopComma(after) {
		toks := strings.Fields(strings.TrimSpace(part))
		if len(toks) == 0 || !strings.HasPrefix(toks[0], "@") {
			continue
		}
		typ := ""
		if len(toks) > 1 {
			typ = toks[1]
		}
		params = append(params, tds.RoutineParam{Name: toks[0], Type: typ})
	}
	return routines.Unqualify(name), params
}

// substituteParams replaces each @parameter in the body with its supplied value as a literal
// (unsupplied parameters become NULL).
func substituteParams(r *tds.Routine, args []routines.Arg) string {
	vals := map[string]string{}
	pos := 0
	for _, a := range args {
		if a.Name != "" {
			vals[strings.ToLower(a.Name)] = routines.LitSQL(a.Val)
			continue
		}
		if pos < len(r.Params) {
			vals[strings.ToLower(r.Params[pos].Name)] = routines.LitSQL(a.Val)
		}
		pos++
	}
	body := r.Body
	for _, p := range r.Params {
		v, ok := vals[strings.ToLower(p.Name)]
		if !ok {
			v = "NULL"
		}
		body = replaceParam(body, p.Name, v)
	}
	return body
}

// replaceParam substitutes every case-insensitive, word-bounded occurrence of @name in body with val.
func replaceParam(body, name, val string) string {
	var sb strings.Builder
	n := len(name)
	for i := 0; i < len(body); {
		if i+n <= len(body) && strings.EqualFold(body[i:i+n], name) && (i+n == len(body) || !isIdentChar(body[i+n])) {
			sb.WriteString(val)
			i += n
			continue
		}
		sb.WriteByte(body[i])
		i++
	}
	return sb.String()
}

func isIdentChar(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// splitName returns the first whitespace/paren/@-delimited token and the remainder.
func splitName(s string) (string, string) {
	for i := 0; i < len(s); i++ {
		if c := s[i]; c == ' ' || c == '\t' || c == '(' || c == '@' {
			return s[:i], s[i:]
		}
	}
	return s, ""
}

// splitTopComma splits on commas at parenthesis-depth 0 and outside single-quoted strings.
func splitTopComma(s string) []string {
	var out []string
	var sb strings.Builder
	depth, inStr := 0, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\'':
			inStr = !inStr
			sb.WriteByte(c)
		case inStr:
			sb.WriteByte(c)
		case c == '(':
			depth++
			sb.WriteByte(c)
		case c == ')':
			depth--
			sb.WriteByte(c)
		case c == ',' && depth == 0:
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

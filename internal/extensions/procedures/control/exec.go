// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package control is the T-SQL procedural interpreter: it runs a batch that uses control flow
// (IF/WHILE/BEGIN-END/TRY-CATCH/PRINT/RETURN). The engine routes here only when HasControlFlow detects a
// control statement; plain batches keep the flat path. All SQL and expression evaluation is delegated
// back through the routines.Runner seam, so the interpreter adds a layer without touching the query path.
package control

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/routines"
	"github.com/RSKGroup/haystak-tds-spi/tds"
)

const maxWhileIterations = 1_000_000 // runaway guard for WHILE

type flowKind int

const (
	flowNone flowKind = iota
	flowReturn
	flowBreak
	flowContinue
)

type state struct {
	flow flowKind
	last tds.Rows
}

type scope struct{ vars map[string]string } // @name -> SQL-literal value

var controlLead = map[string]bool{
	"IF": true, "WHILE": true, "BEGIN": true, "RETURN": true, "PRINT": true,
	"THROW": true, "RAISERROR": true, "BREAK": true, "CONTINUE": true,
	"GOTO": true, "WAITFOR": true,
}

// HasControlFlow reports whether sql leads any statement with a control-flow keyword. Conservative:
// BEGIN TRAN(SACTION) is not control flow, and only a keyword at a statement boundary (depth 0) counts.
func HasControlFlow(sql string) bool {
	toks := lex(sql)
	depth, atStart := 0, true
	for i, t := range toks {
		switch t.kind {
		case tParenL:
			depth++
			atStart = false
		case tParenR:
			if depth > 0 {
				depth--
			}
			atStart = false
		case tSemi:
			if depth == 0 {
				atStart = true
			}
		case tWord:
			if atStart && depth == 0 {
				kw := strings.ToUpper(t.text)
				beginTxn := kw == "BEGIN" && i+1 < len(toks) && (eqKw(toks[i+1], "TRAN") || eqKw(toks[i+1], "TRANSACTION"))
				if !beginTxn && controlLead[kw] {
					return true
				}
			}
			atStart = false
		default:
			atStart = false
		}
	}
	return false
}

// Run parses and executes a control-flow batch, returning its last result set.
func Run(ctx context.Context, sql string, run routines.Runner) (tds.Rows, error) {
	blk, err := parse(sql)
	if err != nil {
		return nil, err
	}
	sc := &scope{vars: map[string]string{}}
	st := &state{}
	if err := execStmt(ctx, blk, sc, run, st); err != nil {
		return nil, err
	}
	return st.last, nil
}

func execStmt(ctx context.Context, s Stmt, sc *scope, run routines.Runner, st *state) error {
	switch n := s.(type) {
	case *Block:
		for _, inner := range n.Stmts {
			if err := execStmt(ctx, inner, sc, run, st); err != nil {
				return err
			}
			if st.flow != flowNone {
				return nil
			}
		}
	case *Raw:
		rows, err := run.Exec(ctx, subst(n.SQL, sc))
		if err != nil {
			return err
		}
		if rows != nil {
			st.last = rows
		}
	case *Declare:
		for _, v := range n.Vars {
			lit := "NULL"
			if v.init != "" {
				l, err := evalLit(ctx, run, v.init, sc)
				if err != nil {
					return err
				}
				lit = l
			}
			sc.vars[v.name] = lit
		}
	case *Set:
		lit, err := evalLit(ctx, run, n.Expr, sc)
		if err != nil {
			return err
		}
		sc.vars[n.Name] = lit
	case *Print:
		if _, err := evalScalar(ctx, run, n.Expr, sc); err != nil { // evaluated; the message is not surfaced
			return err
		}
	case *Return:
		st.flow = flowReturn
	case *Break:
		st.flow = flowBreak
	case *Continue:
		st.flow = flowContinue
	case *If:
		b, err := evalBool(ctx, run, n.Cond, sc)
		if err != nil {
			return err
		}
		if b {
			return execStmt(ctx, n.Then, sc, run, st)
		}
		if n.Else != nil {
			return execStmt(ctx, n.Else, sc, run, st)
		}
	case *While:
		for iter := 0; ; iter++ {
			if iter >= maxWhileIterations {
				return fmt.Errorf("control: WHILE exceeded %d iterations", maxWhileIterations)
			}
			b, err := evalBool(ctx, run, n.Cond, sc)
			if err != nil {
				return err
			}
			if !b {
				break
			}
			if err := execStmt(ctx, n.Body, sc, run, st); err != nil {
				return err
			}
			switch st.flow {
			case flowBreak:
				st.flow = flowNone
				return nil
			case flowContinue:
				st.flow = flowNone
			case flowReturn:
				return nil
			}
		}
	case *Try:
		if err := execStmt(ctx, n.Body, sc, run, st); err != nil {
			st.flow = flowNone // the error aborts the TRY body; the CATCH handles it
			st.last = nil
			catchCtx := tds.WithError(ctx, errorInfoOf(err))
			return execStmt(catchCtx, n.Catch, sc, run, st)
		}
	case *Throw:
		args := strings.TrimSpace(n.Args)
		if args == "" { // bare THROW: re-raise the current error (valid only inside a CATCH)
			if cur := tds.ErrorFromContext(ctx); cur != nil {
				return &ctrlError{cur}
			}
			return fmt.Errorf("control: THROW with no arguments is only valid in a CATCH block")
		}
		info := &tds.ErrorInfo{Number: 50000, Severity: 16, State: 1}
		parts := splitTopCommas(args)
		if len(parts) >= 1 {
			n, err := evalIntArg(ctx, run, parts[0], sc)
			if err != nil {
				return err
			}
			info.Number = n
		}
		if len(parts) >= 2 {
			m, err := evalStrArg(ctx, run, parts[1], sc)
			if err != nil {
				return err
			}
			info.Message = m
		}
		if len(parts) >= 3 {
			if v, err := evalIntArg(ctx, run, parts[2], sc); err == nil {
				info.State = v
			}
		}
		return &ctrlError{info}
	case *Raiserror:
		inner := strings.TrimSpace(n.Args)
		inner = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(inner, "("), ")"))
		info := &tds.ErrorInfo{Number: 50000, Severity: 16, State: 1}
		parts := splitTopCommas(inner)
		if len(parts) >= 1 {
			m, err := evalStrArg(ctx, run, parts[0], sc)
			if err != nil {
				return err
			}
			info.Message = m
		}
		if len(parts) >= 2 {
			if v, err := evalIntArg(ctx, run, parts[1], sc); err == nil {
				info.Severity = v
			}
		}
		if len(parts) >= 3 {
			if v, err := evalIntArg(ctx, run, parts[2], sc); err == nil {
				info.State = v
			}
		}
		if info.Severity < 11 { // severities below 11 are informational, like PRINT
			return nil
		}
		return &ctrlError{info}
	}
	return nil
}

type ctrlError struct{ info *tds.ErrorInfo }

func (e *ctrlError) Error() string { return e.info.Message }

func errorInfoOf(err error) *tds.ErrorInfo {
	if ce, ok := err.(*ctrlError); ok {
		return ce.info
	}
	return &tds.ErrorInfo{Number: 50000, Message: err.Error(), Severity: 16, State: 1}
}

func evalIntArg(ctx context.Context, run routines.Runner, expr string, sc *scope) (int64, error) {
	v, err := evalScalar(ctx, run, expr, sc)
	if err != nil {
		return 0, err
	}
	return toInt(v), nil
}

func evalStrArg(ctx context.Context, run routines.Runner, expr string, sc *scope) (string, error) {
	v, err := evalScalar(ctx, run, expr, sc)
	if err != nil || v == nil {
		return "", err
	}
	return fmt.Sprintf("%v", v), nil
}

func evalScalar(ctx context.Context, run routines.Runner, expr string, sc *scope) (any, error) {
	rows, err := run.Exec(ctx, "SELECT ("+subst(expr, sc)+")")
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()
	if rows.Next() {
		v, err := rows.Values()
		if err != nil {
			return nil, err
		}
		if len(v) > 0 {
			return v[0], nil
		}
	}
	return nil, nil
}

func evalBool(ctx context.Context, run routines.Runner, cond string, sc *scope) (bool, error) {
	rows, err := run.Exec(ctx, "SELECT CASE WHEN ("+subst(cond, sc)+") THEN 1 ELSE 0 END")
	if err != nil {
		return false, err
	}
	if rows == nil {
		return false, nil
	}
	defer rows.Close()
	if rows.Next() {
		v, err := rows.Values()
		if err != nil {
			return false, err
		}
		if len(v) > 0 {
			return toInt(v[0]) == 1, nil
		}
	}
	return false, nil
}

func evalLit(ctx context.Context, run routines.Runner, expr string, sc *scope) (string, error) {
	v, err := evalScalar(ctx, run, expr, sc)
	if err != nil {
		return "", err
	}
	return litOf(v), nil
}

// litOf renders a runtime value as the SQL literal that subst splices back into statement text.
func litOf(v any) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case string:
		return "'" + strings.ReplaceAll(x, "'", "''") + "'"
	case int64:
		return strconv.FormatInt(x, 10)
	case int:
		return strconv.Itoa(x)
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case bool:
		if x {
			return "1"
		}
		return "0"
	case time.Time:
		return "'" + x.Format("2006-01-02 15:04:05.000") + "'"
	case []byte:
		return "0x" + fmt.Sprintf("%x", x)
	}
	return "'" + fmt.Sprintf("%v", v) + "'"
}

func toInt(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	case bool:
		if n {
			return 1
		}
	case string:
		i, _ := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		return i
	}
	return 0
}

// subst replaces @vars with their bound literal, skipping '…' strings and [bracketed] names.
func subst(s string, sc *scope) string {
	if len(sc.vars) == 0 {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		switch s[i] {
		case '\'':
			j := i + 1
			for j < len(s) {
				if s[j] == '\'' {
					if j+1 < len(s) && s[j+1] == '\'' {
						j += 2
						continue
					}
					j++
					break
				}
				j++
			}
			b.WriteString(s[i:j])
			i = j
		case '[':
			j := i + 1
			for j < len(s) && s[j] != ']' {
				j++
			}
			if j < len(s) {
				j++
			}
			b.WriteString(s[i:j])
			i = j
		case '@':
			j := i + 1
			for j < len(s) && isNamePart(s[j]) {
				j++
			}
			if v, ok := sc.vars[strings.ToLower(s[i+1:j])]; ok {
				b.WriteString(v)
			} else {
				b.WriteString(s[i:j])
			}
			i = j
		default:
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

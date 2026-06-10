// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package exec

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

// materializeExprs computes each ValueExpr select item into a synthetic appended column.
func materializeExprs(cols []catalog.Column, idx map[string]int, rows [][]any, sel []tds.SelectItem) ([]catalog.Column, [][]any, []tds.SelectItem, error) {
	has := false
	for _, it := range sel {
		if it.Expr != nil {
			has = true
			break
		}
	}
	if !has {
		return cols, rows, sel, nil
	}
	newCols := append([]catalog.Column{}, cols...)
	newSel := make([]tds.SelectItem, len(sel))
	type comp struct {
		at int
		ve *tds.ValueExpr
	}
	var comps []comp
	for k, it := range sel {
		if it.Expr == nil {
			newSel[k] = it
			continue
		}
		name := fmt.Sprintf("__expr%d", k)
		newCols = append(newCols, catalog.Column{Name: name, Type: exprType(it.Expr, cols, idx)})
		comps = append(comps, comp{len(newCols) - 1, it.Expr})
		newSel[k] = tds.SelectItem{Column: name, Alias: it.Alias}
	}
	out := make([][]any, len(rows))
	for r, row := range rows {
		nr := make([]any, len(newCols))
		copy(nr, row)
		for _, c := range comps {
			v, err := evalValue(idx, row, c.ve)
			if err != nil {
				return nil, nil, nil, err
			}
			nr[c.at] = v
		}
		out[r] = nr
	}
	return newCols, out, newSel, nil
}

func evalValue(idx map[string]int, row []any, ve *tds.ValueExpr) (any, error) {
	switch ve.Kind {
	case tds.ValLit:
		return ve.Lit, nil
	case tds.ValCol:
		i, ok := resolveCol(idx, ve.Col)
		if !ok {
			return nil, fmt.Errorf("exec: unknown column %q in expression", ve.Col)
		}
		return row[i], nil
	case tds.ValBinary:
		l, err := evalValue(idx, row, ve.Left)
		if err != nil {
			return nil, err
		}
		r, err := evalValue(idx, row, ve.Right)
		if err != nil {
			return nil, err
		}
		return evalBinary(ve.Op, l, r), nil
	case tds.ValFunc:
		args := make([]any, len(ve.Args))
		for i, a := range ve.Args {
			v, err := evalValue(idx, row, a)
			if err != nil {
				return nil, err
			}
			args[i] = v
		}
		return evalFunc(ve.Func, args), nil
	case tds.ValCase:
		for _, w := range ve.Whens {
			matched := false
			if w.Cond != nil {
				ok, err := evalExpr(idx, row, w.Cond, nil)
				if err != nil {
					return nil, err
				}
				matched = ok
			} else {
				ov, err := evalValue(idx, row, ve.Operand)
				if err != nil {
					return nil, err
				}
				mv, err := evalValue(idx, row, w.Match)
				if err != nil {
					return nil, err
				}
				if c, ok := compare(ov, mv); ok && c == 0 {
					matched = true
				}
			}
			if matched {
				return evalValue(idx, row, w.Result)
			}
		}
		if ve.Else != nil {
			return evalValue(idx, row, ve.Else)
		}
		return nil, nil
	case tds.ValCast:
		v, err := evalValue(idx, row, ve.Left)
		if err != nil {
			return nil, err
		}
		return castValue(v, ve.Cast), nil
	}
	return nil, nil
}

func evalBinary(op string, l, r any) any {
	if op == "+" {
		if ls, ok := l.(string); ok {
			if rs, ok := r.(string); ok {
				return ls + rs
			}
		}
	}
	if li, ok := l.(int64); ok {
		if ri, ok := r.(int64); ok {
			switch op {
			case "+":
				return li + ri
			case "-":
				return li - ri
			case "*":
				return li * ri
			case "/":
				if ri == 0 {
					return nil
				}
				return li / ri
			case "%":
				if ri == 0 {
					return nil
				}
				return li % ri
			}
		}
	}
	lf, lok := toFloatOk(l)
	rf, rok := toFloatOk(r)
	if lok && rok {
		switch op {
		case "+":
			return lf + rf
		case "-":
			return lf - rf
		case "*":
			return lf * rf
		case "/":
			if rf == 0 {
				return nil
			}
			return lf / rf
		}
	}
	return nil
}

func evalFunc(name string, a []any) any {
	switch name {
	case "LEN", "DATALEN":
		if len(a) == 1 {
			return int64(len(toStr(a[0])))
		}
	case "UPPER":
		if len(a) == 1 {
			return strings.ToUpper(toStr(a[0]))
		}
	case "LOWER":
		if len(a) == 1 {
			return strings.ToLower(toStr(a[0]))
		}
	case "LTRIM":
		if len(a) == 1 {
			return strings.TrimLeft(toStr(a[0]), " ")
		}
	case "RTRIM":
		if len(a) == 1 {
			return strings.TrimRight(toStr(a[0]), " ")
		}
	case "TRIM":
		if len(a) == 1 {
			return strings.TrimSpace(toStr(a[0]))
		}
	case "ISNULL", "COALESCE":
		for _, v := range a {
			if v != nil {
				return v
			}
		}
		return nil
	case "NULLIF":
		if len(a) == 2 {
			if c, ok := compare(a[0], a[1]); ok && c == 0 {
				return nil
			}
			return a[0]
		}
	case "ABS":
		if len(a) == 1 {
			if i, ok := a[0].(int64); ok {
				if i < 0 {
					return -i
				}
				return i
			}
			if f, ok := toFloatOk(a[0]); ok {
				if f < 0 {
					return -f
				}
				return f
			}
		}
	case "CONCAT":
		var sb strings.Builder
		for _, v := range a {
			if v != nil {
				sb.WriteString(toStr(v))
			}
		}
		return sb.String()
	case "REPLACE":
		if len(a) == 3 {
			return strings.ReplaceAll(toStr(a[0]), toStr(a[1]), toStr(a[2]))
		}
	case "SUBSTRING":
		if len(a) == 3 {
			start, _ := toInt(a[1])
			length, _ := toInt(a[2])
			return substr(toStr(a[0]), int(start), int(length))
		}
	case "YEAR":
		if len(a) == 1 {
			if t, ok := a[0].(time.Time); ok {
				return int64(t.Year())
			}
		}
	case "MONTH":
		if len(a) == 1 {
			if t, ok := a[0].(time.Time); ok {
				return int64(t.Month())
			}
		}
	case "DAY":
		if len(a) == 1 {
			if t, ok := a[0].(time.Time); ok {
				return int64(t.Day())
			}
		}
	case "GETDATE", "GETUTCDATE", "SYSDATETIME", "SYSUTCDATETIME":
		return time.Now().UTC()
	}
	return nil
}

func castValue(v any, typ string) any {
	if v == nil {
		return nil
	}
	switch typ {
	case "INT", "BIGINT", "SMALLINT", "TINYINT":
		if i, ok := toInt(v); ok {
			return i
		}
		if s, ok := v.(string); ok {
			if n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil {
				return n
			}
		}
		return nil
	case "FLOAT", "REAL", "DECIMAL", "NUMERIC", "MONEY", "SMALLMONEY":
		if f, ok := toFloatOk(v); ok {
			return f
		}
		if s, ok := v.(string); ok {
			if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
				return f
			}
		}
		return nil
	case "BIT":
		if i, ok := toInt(v); ok {
			if i != 0 {
				return int64(1)
			}
			return int64(0)
		}
		return nil
	case "VARCHAR", "NVARCHAR", "CHAR", "NCHAR", "TEXT", "NTEXT":
		return toStr(v)
	}
	return v
}

func exprType(ve *tds.ValueExpr, cols []catalog.Column, idx map[string]int) types.Type {
	switch ve.Kind {
	case tds.ValLit:
		switch ve.Lit.(type) {
		case int64:
			return types.Type{Kind: types.Int64}
		case float64:
			return types.Type{Kind: types.Float64}
		}
		return types.Type{Kind: types.String, MaxLen: 255}
	case tds.ValCol:
		if i, ok := resolveCol(idx, ve.Col); ok {
			return cols[i].Type
		}
		return types.Type{Kind: types.String, MaxLen: 255}
	case tds.ValBinary:
		lt := exprType(ve.Left, cols, idx)
		rt := exprType(ve.Right, cols, idx)
		if ve.Op == "+" && lt.Kind == types.String {
			return types.Type{Kind: types.String, MaxLen: 255}
		}
		if lt.Kind == types.Float64 || rt.Kind == types.Float64 {
			return types.Type{Kind: types.Float64}
		}
		return types.Type{Kind: types.Int64}
	case tds.ValFunc:
		switch ve.Func {
		case "LEN", "DATALEN", "YEAR", "MONTH", "DAY":
			return types.Type{Kind: types.Int64}
		case "GETDATE", "GETUTCDATE", "SYSDATETIME", "SYSUTCDATETIME":
			return types.Type{Kind: types.Time}
		case "ABS":
			if len(ve.Args) == 1 {
				return exprType(ve.Args[0], cols, idx)
			}
		case "ISNULL", "COALESCE", "NULLIF":
			if len(ve.Args) > 0 {
				return exprType(ve.Args[0], cols, idx)
			}
		}
		return types.Type{Kind: types.String, MaxLen: 255}
	case tds.ValCase:
		if len(ve.Whens) > 0 {
			return exprType(ve.Whens[0].Result, cols, idx)
		}
		if ve.Else != nil {
			return exprType(ve.Else, cols, idx)
		}
		return types.Type{Kind: types.String, MaxLen: 255}
	case tds.ValCast:
		switch ve.Cast {
		case "INT", "SMALLINT", "TINYINT", "BIT":
			return types.Type{Kind: types.Int32}
		case "BIGINT":
			return types.Type{Kind: types.Int64}
		case "FLOAT", "REAL", "DECIMAL", "NUMERIC", "MONEY", "SMALLMONEY":
			return types.Type{Kind: types.Float64}
		}
		return types.Type{Kind: types.String, MaxLen: 255}
	}
	return types.Type{Kind: types.String, MaxLen: 255}
}

func toStr(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func toInt(v any) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case int:
		return int64(x), true
	case float64:
		return int64(x), true
	}
	return 0, false
}

func toFloatOk(v any) (float64, bool) {
	switch x := v.(type) {
	case int64:
		return float64(x), true
	case int:
		return float64(x), true
	case float64:
		return x, true
	case float32:
		return float64(x), true
	}
	return 0, false
}

func substr(s string, start, length int) string {
	if start < 1 {
		length += start - 1
		start = 1
	}
	if start > len(s) || length <= 0 {
		return ""
	}
	end := start - 1 + length
	if end > len(s) {
		end = len(s)
	}
	return s[start-1 : end]
}

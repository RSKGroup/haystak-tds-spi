// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package exec

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/functions"
	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/routines"
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

// materializeExprs computes each ValueExpr select item into a synthetic appended column.
func materializeExprs(cols []catalog.Column, idx map[string]int, rows [][]any, sel []tds.SelectItem, env *Env) ([]catalog.Column, [][]any, []tds.SelectItem, error) {
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
			v, err := evalValue(idx, row, c.ve, env)
			if err != nil {
				return nil, nil, nil, err
			}
			nr[c.at] = v
		}
		out[r] = nr
	}
	return newCols, out, newSel, nil
}

// materializeAggArgs evaluates each aggregate's expression argument (e.g. MAX(CASE …)) into a synthetic
// appended column and rewrites the item to aggregate that column, so the column-based fold can run.
func materializeAggArgs(cols []catalog.Column, idx map[string]int, rows [][]any, sel []tds.SelectItem, env *Env) ([]catalog.Column, [][]any, []tds.SelectItem, error) {
	has := false
	for _, it := range sel {
		if it.Agg != tds.AggNone && it.ArgExpr != nil {
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
		newSel[k] = it
		if it.Agg == tds.AggNone || it.ArgExpr == nil {
			continue
		}
		name := fmt.Sprintf("__agg%d", k)
		newCols = append(newCols, catalog.Column{Name: name, Type: exprType(it.ArgExpr, cols, idx)})
		comps = append(comps, comp{len(newCols) - 1, it.ArgExpr})
		newSel[k].Arg = name
		newSel[k].ArgExpr = nil
	}
	out := make([][]any, len(rows))
	for r, row := range rows {
		nr := make([]any, len(newCols))
		copy(nr, row)
		for _, c := range comps {
			v, err := evalValue(idx, row, c.ve, env)
			if err != nil {
				return nil, nil, nil, err
			}
			nr[c.at] = v
		}
		out[r] = nr
	}
	return newCols, out, newSel, nil
}

func hasOrderExpr(order []tds.OrderItem) bool {
	for _, o := range order {
		if o.Expr != nil {
			return true
		}
	}
	return false
}

// materializeOrderExprs computes each expression ORDER BY term into a synthetic appended column and
// rewrites the term to reference it, so the column-indexed sorter can order by it.
func materializeOrderExprs(cols []catalog.Column, idx map[string]int, rows [][]any, order []tds.OrderItem, env *Env) ([]catalog.Column, [][]any, map[string]int, []tds.OrderItem, error) {
	newCols := append([]catalog.Column{}, cols...)
	newOrder := append([]tds.OrderItem{}, order...)
	type comp struct {
		at int
		ve *tds.ValueExpr
	}
	var comps []comp
	for k := range newOrder {
		if newOrder[k].Expr == nil {
			continue
		}
		name := fmt.Sprintf("__ord%d", k)
		newCols = append(newCols, catalog.Column{Name: name, Type: exprType(newOrder[k].Expr, cols, idx)})
		comps = append(comps, comp{len(newCols) - 1, newOrder[k].Expr})
		newOrder[k] = tds.OrderItem{Column: name, Desc: newOrder[k].Desc}
	}
	out := make([][]any, len(rows))
	for r, row := range rows {
		nr := make([]any, len(newCols))
		copy(nr, row)
		for _, c := range comps {
			v, err := evalValue(idx, row, c.ve, env)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			nr[c.at] = v
		}
		out[r] = nr
	}
	return newCols, out, indexCols(newCols), newOrder, nil
}

func evalValue(idx map[string]int, row []any, ve *tds.ValueExpr, env *Env) (any, error) {
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
		l, err := evalValue(idx, row, ve.Left, env)
		if err != nil {
			return nil, err
		}
		r, err := evalValue(idx, row, ve.Right, env)
		if err != nil {
			return nil, err
		}
		return evalBinary(ve.Op, l, r), nil
	case tds.ValFunc:
		args := make([]any, len(ve.Args))
		for i, a := range ve.Args {
			v, err := evalValue(idx, row, a, env)
			if err != nil {
				return nil, err
			}
			args[i] = v
		}
		return evalFunc(ve.Func, args, env), nil
	case tds.ValCase:
		for _, w := range ve.Whens {
			matched := false
			if w.Cond != nil {
				ok, err := evalExpr(idx, row, w.Cond, env)
				if err != nil {
					return nil, err
				}
				matched = ok
			} else {
				ov, err := evalValue(idx, row, ve.Operand, env)
				if err != nil {
					return nil, err
				}
				mv, err := evalValue(idx, row, w.Match, env)
				if err != nil {
					return nil, err
				}
				if c, ok := compare(ov, mv); ok && c == 0 {
					matched = true
				}
			}
			if matched {
				return evalValue(idx, row, w.Result, env)
			}
		}
		if ve.Else != nil {
			return evalValue(idx, row, ve.Else, env)
		}
		return nil, nil
	case tds.ValCast:
		v, err := evalValue(idx, row, ve.Left, env)
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

// envScalars are the catalog scalars resolved against *Env below, not via the registry; keep in lockstep with evalFunc.
var envScalars = []string{
	"OBJECT_NAME", "DB_NAME", "COL_NAME", "COL_LENGTH",
	"COLUMNPROPERTY", "OBJECTPROPERTY", "OBJECTPROPERTYEX", "OBJECT_DEFINITION",
	"INDEXPROPERTY", "INDEXKEY_PROPERTY",
	"ERROR_MESSAGE", "ERROR_NUMBER", "ERROR_SEVERITY", "ERROR_STATE", "ERROR_LINE", "ERROR_PROCEDURE",
}

// EnvScalarNames returns the env-resolved scalar functions, sorted.
func EnvScalarNames() []string {
	out := append([]string(nil), envScalars...)
	sort.Strings(out)
	return out
}

func evalFunc(name string, a []any, env *Env) any {
	switch name {
	case "OBJECT_NAME":
		if env != nil && env.ObjectName != nil && len(a) >= 1 {
			if id, ok := toInt(a[0]); ok {
				if n, ok := env.ObjectName(id); ok {
					return n
				}
			}
		}
		return nil
	case "DB_NAME":
		if len(a) == 0 || a[0] == nil {
			if env != nil {
				return env.CurrentDB
			}
			return nil
		}
		if env != nil && env.DBName != nil {
			if id, ok := toInt(a[0]); ok {
				if n, ok := env.DBName(id); ok {
					return n
				}
			}
		}
		return nil
	case "COL_NAME":
		if env != nil && env.Table != nil && len(a) >= 2 {
			oid, ok1 := toInt(a[0])
			cid, ok2 := toInt(a[1])
			if ok1 && ok2 {
				if t, ok := env.Table(oid); ok && cid >= 1 && int(cid) <= len(t.Columns) {
					return t.Columns[cid-1].Name
				}
			}
		}
		return nil
	case "COL_LENGTH":
		if env != nil && env.Table != nil && len(a) >= 2 {
			tbl, _ := a[0].(string)
			col, _ := a[1].(string)
			if t, ok := env.Table(functions.ObjectID(tbl)); ok {
				for _, c := range t.Columns {
					if strings.EqualFold(c.Name, col) {
						return colByteLen(c.Type)
					}
				}
			}
		}
		return nil
	case "COLUMNPROPERTY":
		if env != nil && env.Table != nil && len(a) >= 3 {
			if oid, ok := toInt(a[0]); ok {
				col, _ := a[1].(string)
				prop, _ := a[2].(string)
				if t, ok := env.Table(oid); ok {
					for i, c := range t.Columns {
						if strings.EqualFold(c.Name, col) {
							return columnProperty(c, i+1, prop)
						}
					}
				}
			}
		}
		return nil
	case "OBJECTPROPERTY", "OBJECTPROPERTYEX":
		if env != nil && len(a) >= 2 {
			if oid, ok := toInt(a[0]); ok {
				prop, _ := a[1].(string)
				return objectProperty(env, oid, prop)
			}
		}
		return nil
	case "OBJECT_DEFINITION":
		if env != nil && env.RoutineDef != nil && len(a) >= 1 {
			if oid, ok := toInt(a[0]); ok {
				if def, ok := env.RoutineDef(oid); ok {
					return def
				}
			}
		}
		return nil
	case "INDEXPROPERTY":
		if env != nil && env.Table != nil && len(a) >= 3 {
			if oid, ok := toInt(a[0]); ok {
				name, _ := a[1].(string)
				prop, _ := a[2].(string)
				if t, ok := env.Table(oid); ok {
					for i, ix := range t.Indexes {
						if strings.EqualFold(ix.Name, name) {
							return indexProperty(ix, int64(i+1), prop)
						}
					}
				}
			}
		}
		return nil
	case "ERROR_MESSAGE":
		if env != nil && env.Error != nil {
			return env.Error.Message
		}
		return nil
	case "ERROR_NUMBER":
		if env != nil && env.Error != nil {
			return env.Error.Number
		}
		return nil
	case "ERROR_SEVERITY":
		if env != nil && env.Error != nil {
			return env.Error.Severity
		}
		return nil
	case "ERROR_STATE":
		if env != nil && env.Error != nil {
			return env.Error.State
		}
		return nil
	case "ERROR_LINE":
		if env != nil && env.Error != nil {
			return env.Error.Line
		}
		return nil
	case "ERROR_PROCEDURE":
		if env != nil && env.Error != nil && env.Error.Procedure != "" {
			return env.Error.Procedure
		}
		return nil
	case "INDEXKEY_PROPERTY":
		if env != nil && env.Table != nil && len(a) >= 4 {
			oid, ok1 := toInt(a[0])
			iid, ok2 := toInt(a[1])
			ord, ok3 := toInt(a[2])
			prop, _ := a[3].(string)
			if ok1 && ok2 && ok3 {
				if t, ok := env.Table(oid); ok && iid >= 1 && int(iid) <= len(t.Indexes) {
					return indexKeyProperty(t.Indexes[iid-1], t, int(ord), prop)
				}
			}
		}
		return nil
	}
	// Everything else (string / numeric / date / logical / catalog scalars) lives in its family file
	// under catalog/funcs and resolves through the registry — see that package's *.go.
	if v, ok := functions.Eval(name, a); ok {
		return v
	}
	return nil
}

// CatalogResolvers builds the OBJECT_NAME/DB_NAME reverse maps from a backend's tables, routines, and
// database list, so id→name lookups match the ids OBJECT_ID/DB_ID emit.
func CatalogResolvers(schema catalog.Schema, routines []*tds.Routine, dbs []string) (object, db func(int64) (string, bool)) {
	objs := map[int64]string{}
	for _, t := range schema.Tables {
		objs[functions.ObjectID(t.Name)] = t.Name
	}
	for _, r := range routines {
		objs[functions.ObjectID(r.Name)] = r.Name
	}
	dbm := map[int64]string{1: "master", 2: "tempdb", 3: "model", 4: "msdb"}
	for _, d := range dbs {
		dbm[functions.DBID(d)] = d
	}
	object = func(id int64) (string, bool) { n, ok := objs[id]; return n, ok }
	db = func(id int64) (string, bool) { n, ok := dbm[id]; return n, ok }
	return object, db
}

// CatalogObjects builds the COL_*/OBJECTPROPERTY resolvers: a table-by-object-id lookup plus an
// object-kind ("U"/"V"/"P"/"FN"/"TR") lookup, keyed by the ids OBJECT_ID emits.
func CatalogObjects(schema catalog.Schema, routines []*tds.Routine) (table func(int64) (catalog.Table, bool), kind func(int64) (string, bool)) {
	tbls := map[int64]catalog.Table{}
	kinds := map[int64]string{}
	for _, t := range schema.Tables {
		id := functions.ObjectID(t.Name)
		tbls[id] = t
		kinds[id] = "U"
	}
	for _, r := range routines {
		kinds[functions.ObjectID(r.Name)] = routineKindCode(r.Kind)
	}
	table = func(id int64) (catalog.Table, bool) { t, ok := tbls[id]; return t, ok }
	kind = func(id int64) (string, bool) { k, ok := kinds[id]; return k, ok }
	return table, kind
}

// RoutineDefs builds the OBJECT_DEFINITION resolver: object_id -> reconstructed CREATE text.
func RoutineDefs(rts []*tds.Routine) func(int64) (string, bool) {
	defs := map[int64]string{}
	for _, r := range rts {
		defs[functions.ObjectID(r.Name)] = routines.ScriptDefinition(r)
	}
	return func(id int64) (string, bool) { d, ok := defs[id]; return d, ok }
}

func routineKindCode(k tds.RoutineKind) string {
	switch k {
	case tds.RoutineView:
		return "V"
	case tds.RoutineProc:
		return "P"
	case tds.RoutineFunc:
		return "FN"
	case tds.RoutineTrigger:
		return "TR"
	}
	return ""
}

// colByteLen is COL_LENGTH's contract: the column's defined storage width in bytes, -1 for max.
func colByteLen(t types.Type) int64 {
	switch t.Kind {
	case types.Bool:
		return 1
	case types.Int32:
		return 4
	case types.Int64, types.Float64, types.Time:
		return 8
	case types.UUID:
		return 16
	case types.Decimal:
		return 17
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
	}
	return -1
}

func indexProperty(ix catalog.Index, id int64, prop string) any {
	switch strings.ToLower(prop) {
	case "indexid":
		return id
	case "isclustered":
		return boolToInt(ix.Clustered)
	case "isunique":
		return boolToInt(ix.Unique)
	case "ispadindex", "isdisabled", "isautostatistics", "isfulltextkey":
		return int64(0)
	}
	return nil
}

func indexKeyProperty(ix catalog.Index, t catalog.Table, ordinal int, prop string) any {
	if ordinal < 1 || ordinal > len(ix.Columns) {
		return nil
	}
	switch strings.ToLower(prop) {
	case "columnid":
		for i, c := range t.Columns {
			if strings.EqualFold(c.Name, ix.Columns[ordinal-1]) {
				return int64(i + 1)
			}
		}
		return nil
	case "isdescending":
		return int64(0)
	}
	return nil
}

func columnProperty(c catalog.Column, ordinal int, prop string) any {
	switch strings.ToLower(prop) {
	case "allowsnull":
		return boolToInt(c.Type.Nullable)
	case "columnid":
		return int64(ordinal)
	case "precision":
		return int64(c.Type.Precision)
	case "scale":
		return int64(c.Type.Scale)
	case "charmaxlen":
		if (c.Type.Kind == types.String || c.Type.Kind == types.Bytes) && c.Type.MaxLen > 0 {
			return int64(c.Type.MaxLen)
		}
		return int64(-1)
	case "isidentity":
		return boolToInt(c.Identity)
	case "iscomputed":
		return boolToInt(c.Computed != "")
	}
	return nil
}

func objectProperty(env *Env, oid int64, prop string) any {
	kind := ""
	if env.ObjectKind != nil {
		if k, ok := env.ObjectKind(oid); ok {
			kind = k
		}
	}
	t, haveTable := catalog.Table{}, false
	if env.Table != nil {
		t, haveTable = env.Table(oid)
	}
	switch strings.ToLower(prop) {
	case "istable", "isusertable":
		return boolToInt(kind == "U")
	case "isview":
		return boolToInt(kind == "V")
	case "isprocedure":
		return boolToInt(kind == "P")
	case "isscalarfunction":
		return boolToInt(kind == "FN")
	case "istrigger":
		return boolToInt(kind == "TR")
	case "tablehasprimarykey":
		return boolToInt(haveTable && len(t.PrimaryKey) > 0)
	case "tablehasidentity":
		if haveTable {
			for _, c := range t.Columns {
				if c.Identity {
					return int64(1)
				}
			}
		}
		return int64(0)
	case "ismsshipped":
		return int64(0)
	}
	return nil
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
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

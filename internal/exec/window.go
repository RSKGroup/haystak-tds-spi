// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package exec

import (
	"fmt"
	"sort"

	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

// windowFuncs are the wired window functions; keep in lockstep with the applyWindow switch.
var windowFuncs = []string{"ROW_NUMBER", "RANK", "DENSE_RANK", "LAG", "LEAD"}

// WindowFuncNames returns the wired window-function names, sorted.
func WindowFuncNames() []string {
	out := append([]string(nil), windowFuncs...)
	sort.Strings(out)
	return out
}

// materializeWindows computes each window function (ROW_NUMBER/RANK/DENSE_RANK/LAG/LEAD) into a synthetic
// appended column and rewrites the item to reference it, so projection picks it up like any other column.
func materializeWindows(cols []catalog.Column, idx map[string]int, rows [][]any, sel []tds.SelectItem, env *Env) ([]catalog.Column, [][]any, []tds.SelectItem, error) {
	has := false
	for _, it := range sel {
		if it.Window != nil {
			has = true
			break
		}
	}
	if !has {
		return cols, rows, sel, nil
	}
	newCols := append([]catalog.Column{}, cols...)
	newSel := make([]tds.SelectItem, len(sel))
	out := make([][]any, len(rows))
	for r := range rows {
		out[r] = append([]any{}, rows[r]...)
	}
	for k, it := range sel {
		if it.Window == nil {
			newSel[k] = it
			continue
		}
		vals, err := computeWindow(it.Window, idx, rows, env)
		if err != nil {
			return nil, nil, nil, err
		}
		name := fmt.Sprintf("__win%d", k)
		newCols = append(newCols, catalog.Column{Name: name, Type: windowColType(it.Window, cols, idx)})
		for r := range out {
			out[r] = append(out[r], vals[r])
		}
		newSel[k] = tds.SelectItem{Column: name, Alias: it.Alias}
	}
	return newCols, out, newSel, nil
}

func windowColType(w *tds.WindowSpec, cols []catalog.Column, idx map[string]int) types.Type {
	switch w.Func {
	case "ROW_NUMBER", "RANK", "DENSE_RANK":
		return types.Type{Kind: types.Int64}
	case "LAG", "LEAD":
		if len(w.Args) > 0 {
			return exprType(w.Args[0], cols, idx)
		}
	}
	return types.Type{Kind: types.String, MaxLen: 4000}
}

// computeWindow returns one value per input row (indexed by original position): partition the rows,
// order each partition, then apply the function over that ordered sequence.
func computeWindow(w *tds.WindowSpec, idx map[string]int, rows [][]any, env *Env) ([]any, error) {
	out := make([]any, len(rows))
	pIdx := make([]int, 0, len(w.PartitionBy))
	for _, pc := range w.PartitionBy {
		i, ok := resolveCol(idx, pc)
		if !ok {
			return nil, fmt.Errorf("exec: unknown PARTITION BY column %q", pc)
		}
		pIdx = append(pIdx, i)
	}
	partitions := map[string][]int{}
	var order []string
	for r := range rows {
		key := partKey(rows[r], pIdx)
		if _, ok := partitions[key]; !ok {
			order = append(order, key)
		}
		partitions[key] = append(partitions[key], r)
	}
	for _, key := range order {
		members := partitions[key]
		if err := orderMembers(members, rows, idx, w.OrderBy, env); err != nil {
			return nil, err
		}
		if err := applyWindow(w, members, rows, idx, env, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func partKey(row []any, pIdx []int) string {
	if len(pIdx) == 0 {
		return "__all__"
	}
	parts := make([]any, len(pIdx))
	for i, ci := range pIdx {
		parts[i] = row[ci]
	}
	return fmt.Sprintf("%v", parts)
}

// orderMembers stably sorts the partition's row indices by the window ORDER BY.
func orderMembers(members []int, rows [][]any, idx map[string]int, order []tds.OrderItem, env *Env) error {
	if len(order) == 0 {
		return nil
	}
	keys := make([][]any, len(members))
	for i, m := range members {
		k, err := windowOrderKey(rows[m], idx, order, env)
		if err != nil {
			return err
		}
		keys[i] = k
	}
	type pair struct {
		m   int
		key []any
	}
	pairs := make([]pair, len(members))
	for i := range members {
		pairs[i] = pair{members[i], keys[i]}
	}
	sort.SliceStable(pairs, func(a, b int) bool {
		for j, o := range order {
			c, ok := compare(pairs[a].key[j], pairs[b].key[j])
			if !ok || c == 0 {
				continue
			}
			if o.Desc {
				return c > 0
			}
			return c < 0
		}
		return false
	})
	for i := range pairs {
		members[i] = pairs[i].m
	}
	return nil
}

func windowOrderKey(row []any, idx map[string]int, order []tds.OrderItem, env *Env) ([]any, error) {
	k := make([]any, len(order))
	for j, o := range order {
		if o.Expr != nil {
			v, err := evalValue(idx, row, o.Expr, env)
			if err != nil {
				return nil, err
			}
			k[j] = v
			continue
		}
		i, ok := resolveCol(idx, o.Column)
		if !ok {
			return nil, fmt.Errorf("exec: unknown ORDER BY column %q in OVER", o.Column)
		}
		k[j] = row[i]
	}
	return k, nil
}

// applyWindow assigns the function value to out[origRowIndex] for each member of an ordered partition.
func applyWindow(w *tds.WindowSpec, members []int, rows [][]any, idx map[string]int, env *Env, out []any) error {
	switch w.Func {
	case "ROW_NUMBER":
		for i, m := range members {
			out[m] = int64(i + 1)
		}
	case "RANK", "DENSE_RANK":
		dense := w.Func == "DENSE_RANK"
		rank := int64(1)
		for i, m := range members {
			if i > 0 {
				ka, err := windowOrderKey(rows[members[i-1]], idx, w.OrderBy, env)
				if err != nil {
					return err
				}
				kb, err := windowOrderKey(rows[m], idx, w.OrderBy, env)
				if err != nil {
					return err
				}
				if !keysEqual(ka, kb) {
					if dense {
						rank++
					} else {
						rank = int64(i + 1)
					}
				}
			}
			out[m] = rank
		}
	case "LAG", "LEAD":
		offset := 1
		if len(w.Args) >= 2 {
			if n, ok := toInt(asLit(w.Args[1])); ok {
				offset = int(n)
			}
		}
		var def any
		if len(w.Args) >= 3 {
			def = asLit(w.Args[2])
		}
		for i, m := range members {
			j := i - offset
			if w.Func == "LEAD" {
				j = i + offset
			}
			if j < 0 || j >= len(members) {
				out[m] = def
				continue
			}
			v, err := evalValue(idx, rows[members[j]], w.Args[0], env)
			if err != nil {
				return err
			}
			out[m] = v
		}
	default:
		return fmt.Errorf("exec: unsupported window function %q", w.Func)
	}
	return nil
}

func keysEqual(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if c, ok := compare(a[i], b[i]); !ok || c != 0 {
			return false
		}
	}
	return true
}

func asLit(ve *tds.ValueExpr) any {
	if ve != nil && ve.Kind == tds.ValLit {
		return ve.Lit
	}
	return nil
}

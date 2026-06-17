// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package exec

import (
	"fmt"

	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

// Pivot rotates rows to columns: it groups the source by every column except the value and pivot
// columns, then emits one column per pivot value holding the aggregate of the value column.
func Pivot(cols []catalog.Column, data [][]any, spec *tds.PivotSpec) ([]catalog.Column, [][]any, error) {
	idx := indexCols(cols)
	pivIdx, ok := resolveCol(idx, spec.PivotCol)
	if !ok {
		return nil, nil, fmt.Errorf("exec: PIVOT column %q not found", spec.PivotCol)
	}
	valIdx := -1
	if spec.ValueCol != "*" {
		if i, ok := resolveCol(idx, spec.ValueCol); ok {
			valIdx = i
		} else {
			return nil, nil, fmt.Errorf("exec: PIVOT value column %q not found", spec.ValueCol)
		}
	}

	var groupIdx []int
	var outCols []catalog.Column
	for i, c := range cols {
		if i == pivIdx || i == valIdx {
			continue
		}
		groupIdx = append(groupIdx, i)
		outCols = append(outCols, c)
	}
	fn := aggFuncFromName(spec.Agg)
	pcType := pivotColType(fn, cols, valIdx)
	for _, v := range spec.Values {
		outCols = append(outCols, catalog.Column{Name: v, Type: pcType})
	}

	var order []string
	groups := map[string][][]any{}
	for _, row := range data {
		parts := make([]any, len(groupIdx))
		for j, gi := range groupIdx {
			parts[j] = row[gi]
		}
		key := fmt.Sprintf("%v", parts)
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], row)
	}

	var out [][]any
	for _, k := range order {
		grp := groups[k]
		outRow := make([]any, 0, len(outCols))
		for _, gi := range groupIdx {
			outRow = append(outRow, grp[0][gi])
		}
		for _, v := range spec.Values {
			var matched [][]any
			for _, r := range grp {
				if fmt.Sprintf("%v", r[pivIdx]) == v {
					matched = append(matched, r)
				}
			}
			agg, err := computeAgg(fn, spec.ValueCol, "", idx, matched)
			if err != nil {
				return nil, nil, err
			}
			if len(matched) == 0 && fn != tds.AggCount && fn != tds.AggCountBig {
				agg = nil // SQL Server PIVOT yields NULL (not 0) for a SUM/MIN/… over no rows
			}
			outRow = append(outRow, agg)
		}
		out = append(out, outRow)
	}
	return outCols, out, nil
}

// Unpivot rotates columns to rows: each listed column becomes a (NameCol, ValueCol) pair per source
// row, carrying the other columns through. NULL cells are dropped, matching T-SQL UNPIVOT.
func Unpivot(cols []catalog.Column, data [][]any, spec *tds.UnpivotSpec) ([]catalog.Column, [][]any, error) {
	idx := indexCols(cols)
	upMember := make(map[int]bool, len(spec.Columns))
	upIdx := make([]int, 0, len(spec.Columns))
	var valType types.Type
	for n, name := range spec.Columns {
		i, ok := resolveCol(idx, name)
		if !ok {
			return nil, nil, fmt.Errorf("exec: UNPIVOT column %q not found", name)
		}
		upMember[i] = true
		upIdx = append(upIdx, i)
		if n == 0 {
			valType = cols[i].Type
		}
	}

	var keepIdx []int
	var outCols []catalog.Column
	for i, c := range cols {
		if upMember[i] {
			continue
		}
		keepIdx = append(keepIdx, i)
		outCols = append(outCols, c)
	}
	outCols = append(outCols,
		catalog.Column{Name: spec.NameCol, Type: types.Type{Kind: types.String, MaxLen: 128}},
		catalog.Column{Name: spec.ValueCol, Type: valType})

	var out [][]any
	for _, row := range data {
		for n, ci := range upIdx {
			if row[ci] == nil {
				continue
			}
			outRow := make([]any, 0, len(outCols))
			for _, ki := range keepIdx {
				outRow = append(outRow, row[ki])
			}
			outRow = append(outRow, spec.Columns[n], row[ci])
			out = append(out, outRow)
		}
	}
	return outCols, out, nil
}

func pivotColType(fn tds.AggFunc, cols []catalog.Column, valIdx int) types.Type {
	switch fn {
	case tds.AggCount, tds.AggCountBig:
		return types.Type{Kind: types.Int64, Nullable: true}
	case tds.AggMin, tds.AggMax:
		if valIdx >= 0 {
			t := cols[valIdx].Type
			t.Nullable = true
			return t
		}
	}
	return types.Type{Kind: types.Float64, Nullable: true}
}

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package exec

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/functions"
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

// tableFuncs are the wired FROM-clause table-valued functions.
var tableFuncs = []string{"OPENJSON", "STRING_SPLIT"}

// TableFuncNames returns the wired table-valued-function names, sorted.
func TableFuncNames() []string {
	out := append([]string(nil), tableFuncs...)
	sort.Strings(out)
	return out
}

// EvalTableFunc evaluates a FROM-clause table-valued function (constant arguments) into a rowset.
func EvalTableFunc(name string, args []*tds.ValueExpr) ([]catalog.Column, [][]any, error) {
	switch strings.ToUpper(name) {
	case "STRING_SPLIT":
		if len(args) < 2 {
			return nil, nil, fmt.Errorf("exec: STRING_SPLIT requires (string, separator)")
		}
		s, err := constStr(args[0])
		if err != nil {
			return nil, nil, err
		}
		sep, err := constStr(args[1])
		if err != nil {
			return nil, nil, err
		}
		cols := []catalog.Column{{Name: "value", Type: types.Type{Kind: types.String, MaxLen: 4000}}}
		var data [][]any
		for _, part := range strings.Split(s, sep) {
			data = append(data, []any{part})
		}
		return cols, data, nil
	case "OPENJSON":
		if len(args) < 1 {
			return nil, nil, fmt.Errorf("exec: OPENJSON requires a json argument")
		}
		s, err := constStr(args[0])
		if err != nil {
			return nil, nil, err
		}
		path := ""
		if len(args) >= 2 {
			if path, err = constStr(args[1]); err != nil {
				return nil, nil, err
			}
		}
		cols := []catalog.Column{
			{Name: "key", Type: types.Type{Kind: types.String, MaxLen: 4000}},
			{Name: "value", Type: types.Type{Kind: types.String, MaxLen: 4000}},
			{Name: "type", Type: types.Type{Kind: types.Int32}},
		}
		entries, ok := functions.OpenJSONRows(s, path)
		if !ok {
			return cols, nil, nil // invalid JSON or scalar root: correct columns, zero rows
		}
		data := make([][]any, 0, len(entries))
		for _, e := range entries {
			data = append(data, []any{e.Key, e.Value, e.Type})
		}
		return cols, data, nil
	}
	return nil, nil, fmt.Errorf("exec: unknown table function %q", name)
}

// constStr evaluates a constant argument (literal or constant-function) to its string form.
func constStr(ve *tds.ValueExpr) (string, error) {
	v, err := evalValue(map[string]int{}, nil, ve, nil)
	if err != nil {
		return "", err
	}
	if v == nil {
		return "", nil
	}
	return fmt.Sprintf("%v", v), nil
}

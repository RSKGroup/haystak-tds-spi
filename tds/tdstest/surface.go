// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tdstest

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
)

// P is a predicate cell: a Case.Want entry of type P passes when it returns true for the actual cell.
type P func(any) bool

// checkFn is a whole-result-set assertion for catalog and shape cases.
type checkFn func(tb testing.TB, cols []catalog.Column, data [][]any)

// Case is one surface entry: a statement, plus either an exact single-row Want or a Check. Element is
// the namespaced wired element it proves (e.g. "func:UPPER"); the completeness gate keys off it.
type Case struct {
	Element string
	Name    string
	SQL     string
	Want    []any
	Check   checkFn
}

// RunSurface runs the cross-backend surface suite against b, one subtest per Case.
func RunSurface(t *testing.T, b tds.Backend) {
	t.Helper()
	for _, c := range allCases() {
		c := c
		name := c.Name
		if name == "" {
			name = c.Element
		}
		t.Run(name, func(t *testing.T) { c.run(t, b) })
	}
}

// allCases aggregates every per-area surface table. Each new function/view/proc adds its case here.
func allCases() []Case {
	var all []Case
	for _, group := range [][]Case{
		stringCases, mathCases, datetimeCases, logicalCases, jsonCases, cryptoCases,
		metadataCases, securityCases, systemCases, conversionCases, aggregateCases,
		formatCases, windowCases, tvfCases, sysviewCases, infoschemaCases, procCases, languageCases,
	} {
		all = append(all, group...)
	}
	return all
}

func (c Case) run(t *testing.T, b tds.Backend) {
	t.Helper()
	rs, err := engine.Query(context.Background(), b, c.SQL)
	if err != nil {
		t.Fatalf("%s: query error: %v", c.SQL, err)
	}
	cols, data := drainRows(t, c.SQL, rs)
	if c.Check != nil {
		c.Check(t, cols, data)
		return
	}
	if len(data) != 1 {
		t.Fatalf("%s: got %d rows, want exactly 1", c.SQL, len(data))
	}
	row := data[0]
	if len(row) != len(c.Want) {
		t.Fatalf("%s: got %d columns, want %d", c.SQL, len(row), len(c.Want))
	}
	for i, w := range c.Want {
		if !cellEq(row[i], w) {
			t.Errorf("%s: col %d = %#v, want %s", c.SQL, i, row[i], wantDesc(w))
		}
	}
}

func drainRows(t *testing.T, sql string, rs tds.Rows) ([]catalog.Column, [][]any) {
	t.Helper()
	if rs == nil {
		return nil, nil
	}
	defer rs.Close()
	cols := rs.Columns()
	var data [][]any
	for rs.Next() {
		v, err := rs.Values()
		if err != nil {
			t.Fatalf("%s: row values: %v", sql, err)
		}
		data = append(data, v)
	}
	if err := rs.Err(); err != nil {
		t.Fatalf("%s: rows err: %v", sql, err)
	}
	return cols, data
}

func cellEq(got, want any) bool {
	switch w := want.(type) {
	case P:
		return w(got)
	case []byte:
		gb, ok := got.([]byte)
		return ok && bytes.Equal(gb, w)
	default:
		return fmt.Sprintf("%v", got) == fmt.Sprintf("%v", want)
	}
}

func wantDesc(want any) string {
	if _, ok := want.(P); ok {
		return "(predicate)"
	}
	return fmt.Sprintf("%v", want)
}

// Predicate cells for non-deterministic or representation-sensitive results.

func notNull(v any) bool { return v != nil }

func isNull(v any) bool { return v == nil }

func isGUID(v any) bool {
	s, ok := v.(string)
	return ok && len(s) == 36 && strings.Count(s, "-") == 4
}

func contains(sub string) P {
	return func(v any) bool { return v != nil && strings.Contains(fmt.Sprintf("%v", v), sub) }
}

func approx(want float64) P {
	return func(v any) bool {
		f, ok := toFloat(v)
		return ok && math.Abs(f-want) < 1e-9
	}
}

func inRange(lo, hi float64) P {
	return func(v any) bool {
		f, ok := toFloat(v)
		return ok && f >= lo && f <= hi
	}
}

func bytesLen(n int) P {
	return func(v any) bool {
		b, ok := v.([]byte)
		return ok && len(b) == n
	}
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int64:
		return float64(n), true
	case int:
		return float64(n), true
	case string:
		f, err := strconv.ParseFloat(n, 64)
		return f, err == nil
	}
	return 0, false
}

// Check builders for catalog and shape cases.

func checks(fns ...checkFn) checkFn {
	return func(tb testing.TB, cols []catalog.Column, data [][]any) {
		for _, f := range fns {
			f(tb, cols, data)
		}
	}
}

// wantCols asserts every named column is present (case-insensitive, schema/alias-stripped).
func wantCols(names ...string) checkFn {
	return func(tb testing.TB, cols []catalog.Column, _ [][]any) {
		have := map[string]bool{}
		for _, c := range cols {
			have[strings.ToLower(bareName(c.Name))] = true
		}
		for _, n := range names {
			if !have[strings.ToLower(n)] {
				tb.Errorf("missing column %q; have %v", n, colNames(cols))
			}
		}
	}
}

func exactRows(n int) checkFn {
	return func(tb testing.TB, _ []catalog.Column, data [][]any) {
		if len(data) != n {
			tb.Errorf("got %d rows, want exactly %d", len(data), n)
		}
	}
}

func atLeastRows(n int) checkFn {
	return func(tb testing.TB, _ []catalog.Column, data [][]any) {
		if len(data) < n {
			tb.Errorf("got %d rows, want at least %d", len(data), n)
		}
	}
}

func bareName(col string) string {
	if i := strings.LastIndex(col, "."); i >= 0 {
		return col[i+1:]
	}
	return col
}

func colNames(cols []catalog.Column) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.Name
	}
	return out
}

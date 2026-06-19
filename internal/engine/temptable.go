// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/RSKGroup/haystak-tds-spi/internal/exec"
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
)

type tempStoreKey struct{}

type tempTable struct {
	cols []catalog.Column
	rows [][]any
}

// tempStore holds #temp tables for the life of a batch; it rides in ctx so every statement shares it.
type tempStore struct {
	mu     sync.Mutex
	tables map[string]*tempTable
}

func withTempStore(ctx context.Context) context.Context {
	if tempStoreFrom(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, tempStoreKey{}, &tempStore{tables: map[string]*tempTable{}})
}

func tempStoreFrom(ctx context.Context) *tempStore {
	s, _ := ctx.Value(tempStoreKey{}).(*tempStore)
	return s
}

func isTempName(name string) bool { return strings.HasPrefix(strings.Trim(name, "[]\""), "#") }

func normTemp(name string) string { return strings.ToLower(strings.Trim(name, "[]\"")) }

func (s *tempStore) create(t *catalog.Table) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tables[normTemp(t.Name)] = &tempTable{cols: append([]catalog.Column(nil), t.Columns...)}
}

func (s *tempStore) drop(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tables, normTemp(name))
}

func (s *tempStore) get(name string) (*tempTable, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tables[normTemp(name)]
	return t, ok
}

func (s *tempStore) insertRows(name string, cols []string, rows [][]any) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tables[normTemp(name)]
	if !ok {
		return 0, fmt.Errorf("invalid object name '%s'", name)
	}
	for _, row := range rows {
		t.rows = append(t.rows, t.mapRow(cols, row))
	}
	return int64(len(rows)), nil
}

// mapRow places a source row into the temp table's column order; positional when no column list is given.
func (t *tempTable) mapRow(cols []string, row []any) []any {
	out := make([]any, len(t.cols))
	if len(cols) == 0 {
		copy(out, row)
		return out
	}
	pos := map[string]int{}
	for i, c := range t.cols {
		pos[strings.ToLower(c.Name)] = i
	}
	for i, name := range cols {
		if i >= len(row) {
			break
		}
		if j, ok := pos[strings.ToLower(strings.Trim(name, "[]\""))]; ok {
			out[j] = row[i]
		}
	}
	return out
}

// execTempInsertSource handles `INSERT [INTO] #temp [(cols)] (EXEC|SELECT) …`: it runs the source and
// appends the rows to the temp table. Returns handled=false for non-temp or VALUES inserts.
func execTempInsertSource(ctx context.Context, b tds.Backend, sql string) (bool, int64, error) {
	s := tempStoreFrom(ctx)
	if s == nil {
		return false, 0, nil
	}
	rest := strings.TrimSpace(sql)
	if !hasWordPrefix(rest, "INSERT") {
		return false, 0, nil
	}
	rest = strings.TrimSpace(rest[len("INSERT"):])
	if hasWordPrefix(rest, "INTO") {
		rest = strings.TrimSpace(rest[len("INTO"):])
	}
	name, after := firstToken(rest)
	if !isTempName(name) {
		return false, 0, nil
	}
	after = strings.TrimSpace(after)
	var cols []string
	if strings.HasPrefix(after, "(") {
		if end := matchParen(after); end > 0 {
			cols = splitTopList(after[1:end])
			after = strings.TrimSpace(after[end+1:])
		}
	}
	if !hasWordPrefix(after, "EXEC") && !hasWordPrefix(after, "EXECUTE") && !hasWordPrefix(after, "SELECT") {
		return false, 0, nil
	}
	rs, _, err := queryOne(ctx, b, after)
	if err != nil {
		return true, 0, err
	}
	_, data, err := materialize(rs)
	if err != nil {
		return true, 0, err
	}
	n, err := s.insertRows(name, cols, data)
	return true, n, err
}

func hasWordPrefix(s, word string) bool {
	if len(s) < len(word) || !strings.EqualFold(s[:len(word)], word) {
		return false
	}
	return len(s) == len(word) || s[len(word)] == ' ' || s[len(word)] == '\t' || s[len(word)] == '\n' || s[len(word)] == '\r' || s[len(word)] == '('
}

func firstToken(s string) (string, string) {
	i := strings.IndexAny(s, " \t\r\n(")
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i:]
}

func matchParen(s string) int {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func splitTopList(s string) []string {
	var out []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	return append(out, strings.TrimSpace(s[start:]))
}

func runTempQuery(s *tempStore, q *tds.Query) (tds.Rows, error) {
	t, ok := s.get(q.Table)
	if !ok {
		return nil, fmt.Errorf("invalid object name '%s'", q.Table)
	}
	cols := append([]catalog.Column(nil), t.cols...)
	data := make([][]any, len(t.rows))
	for i, r := range t.rows {
		data[i] = append([]any(nil), r...)
	}
	return exec.Apply(cols, data, q)
}

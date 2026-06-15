// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package functions

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

// JSONEntry is one OPENJSON row: key, value (scalar as text, object/array as JSON text), type code.
type JSONEntry struct {
	Key   string
	Value any
	Type  int64
}

// OpenJSONRows enumerates a JSON array or object (optionally at path) into OPENJSON key/value/type rows.
func OpenJSONRows(s, path string) ([]JSONEntry, bool) {
	var root any
	if json.Unmarshal([]byte(s), &root) != nil {
		return nil, false
	}
	if p := strings.TrimSpace(path); p != "" && p != "$" {
		v, ok := jsonNav(root, p)
		if !ok {
			return nil, false
		}
		root = v
	}
	switch v := root.(type) {
	case []any:
		out := make([]JSONEntry, 0, len(v))
		for i, e := range v {
			out = append(out, JSONEntry{Key: strconv.Itoa(i), Value: jsonValueText(e), Type: jsonType(e)})
		}
		return out, true
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys) // document order is lost on decode; sort for determinism
		out := make([]JSONEntry, 0, len(v))
		for _, k := range keys {
			out = append(out, JSONEntry{Key: k, Value: jsonValueText(v[k]), Type: jsonType(v[k])})
		}
		return out, true
	}
	return nil, false
}

func jsonType(v any) int64 {
	switch v.(type) {
	case string:
		return 1
	case float64:
		return 2
	case bool:
		return 3
	case []any:
		return 4
	case map[string]any:
		return 5
	}
	return 0
}

func jsonValueText(v any) any {
	switch v.(type) {
	case map[string]any, []any:
		b, _ := json.Marshal(v)
		return string(b)
	case nil:
		return nil
	default:
		return jsonScalar(v)
	}
}

func init() {
	register("ISJSON", func(a []any) any {
		s, ok := jsonInput(a)
		if !ok {
			return nil
		}
		var v any
		if json.Unmarshal([]byte(s), &v) == nil {
			return int64(1)
		}
		return int64(0)
	})
	register("JSON_VALUE", func(a []any) any { return jsonExtract(a, false) })
	register("JSON_QUERY", func(a []any) any { return jsonExtract(a, true) })
	register("JSON_MODIFY", func(a []any) any {
		if len(a) < 3 || a[0] == nil || a[1] == nil {
			return nil
		}
		js, ok := a[0].(string)
		path, ok2 := a[1].(string)
		if !ok || !ok2 {
			return nil
		}
		out, ok := jsonModify(js, path, a[2])
		if !ok {
			return nil
		}
		return out
	})
}

type jsonSeg struct {
	key   string
	idx   int
	isIdx bool
}

// jsonPathSegs splits a JSON path ($, $.a.b[0]) into navigable segments.
func jsonPathSegs(path string) ([]jsonSeg, bool) {
	p := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(path), "lax")), "strict"))
	if !strings.HasPrefix(p, "$") {
		return nil, false
	}
	p = p[1:]
	var segs []jsonSeg
	for i := 0; i < len(p); {
		switch p[i] {
		case '.':
			i++
			start := i
			for i < len(p) && p[i] != '.' && p[i] != '[' {
				i++
			}
			segs = append(segs, jsonSeg{key: p[start:i]})
		case '[':
			j := strings.IndexByte(p[i:], ']')
			if j < 0 {
				return nil, false
			}
			n, err := strconv.Atoi(strings.TrimSpace(p[i+1 : i+j]))
			if err != nil {
				return nil, false
			}
			segs = append(segs, jsonSeg{idx: n, isIdx: true})
			i += j + 1
		default:
			return nil, false
		}
	}
	return segs, true
}

func jsonChild(parent any, seg jsonSeg) (any, bool) {
	if seg.isIdx {
		arr, ok := parent.([]any)
		if !ok || seg.idx < 0 || seg.idx >= len(arr) {
			return nil, false
		}
		return arr[seg.idx], true
	}
	m, ok := parent.(map[string]any)
	if !ok {
		return nil, false
	}
	v, ok := m[seg.key]
	return v, ok
}

// jsonModify sets, inserts, deletes (NULL value), or appends ("append $.path") a value at a JSON path.
func jsonModify(jsonStr, path string, val any) (string, bool) {
	var root any
	if json.Unmarshal([]byte(jsonStr), &root) != nil {
		return "", false
	}
	doAppend := false
	p := strings.TrimSpace(path)
	if strings.HasPrefix(strings.ToLower(p), "append ") {
		doAppend = true
		p = strings.TrimSpace(p[len("append "):])
	}
	segs, ok := jsonPathSegs(p)
	if !ok || len(segs) == 0 {
		return "", false
	}
	parent := root
	for i := 0; i < len(segs)-1; i++ {
		if parent, ok = jsonChild(parent, segs[i]); !ok {
			return "", false
		}
	}
	last := segs[len(segs)-1]
	switch {
	case doAppend:
		m, ok := parent.(map[string]any)
		if !ok || last.isIdx {
			return "", false
		}
		arr, ok := m[last.key].([]any)
		if !ok {
			return "", false
		}
		m[last.key] = append(arr, val)
	case val == nil:
		if m, ok := parent.(map[string]any); ok && !last.isIdx {
			delete(m, last.key)
		}
	case last.isIdx:
		arr, ok := parent.([]any)
		if !ok || last.idx < 0 || last.idx >= len(arr) {
			return "", false
		}
		arr[last.idx] = val
	default:
		m, ok := parent.(map[string]any)
		if !ok {
			return "", false
		}
		m[last.key] = val
	}
	b, err := json.Marshal(root)
	if err != nil {
		return "", false
	}
	return string(b), true
}

func jsonInput(a []any) (string, bool) {
	if len(a) == 0 || a[0] == nil {
		return "", false
	}
	s, ok := a[0].(string)
	return s, ok
}

// jsonExtract backs JSON_VALUE (container=false: scalars only) and JSON_QUERY (container=true: object/array only).
func jsonExtract(a []any, container bool) any {
	s, ok := jsonInput(a)
	if !ok {
		return nil
	}
	path := "$"
	if len(a) >= 2 {
		if p, ok := a[1].(string); ok {
			path = p
		}
	}
	var root any
	if json.Unmarshal([]byte(s), &root) != nil {
		return nil
	}
	v, ok := jsonNav(root, path)
	if !ok {
		return nil
	}
	switch v.(type) {
	case map[string]any, []any:
		if !container {
			return nil
		}
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		return string(b)
	default:
		if container {
			return nil
		}
		return jsonScalar(v)
	}
}

func jsonScalar(v any) any {
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	}
	return nil
}

// jsonNav walks a SQL Server JSON path ($, $.a.b, $.arr[0]) over a decoded value.
func jsonNav(root any, path string) (any, bool) {
	p := strings.TrimSpace(path)
	p = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(p, "lax")), "strict"))
	if !strings.HasPrefix(p, "$") {
		return nil, false
	}
	p = p[1:]
	cur := root
	for i := 0; i < len(p); {
		switch p[i] {
		case '.':
			i++
			start := i
			for i < len(p) && p[i] != '.' && p[i] != '[' {
				i++
			}
			m, ok := cur.(map[string]any)
			if !ok {
				return nil, false
			}
			if cur, ok = m[p[start:i]]; !ok {
				return nil, false
			}
		case '[':
			j := strings.IndexByte(p[i:], ']')
			if j < 0 {
				return nil, false
			}
			arr, ok := cur.([]any)
			if !ok {
				return nil, false
			}
			n, err := strconv.Atoi(strings.TrimSpace(p[i+1 : i+j]))
			if err != nil || n < 0 || n >= len(arr) {
				return nil, false
			}
			cur = arr[n]
			i += j + 1
		default:
			return nil, false
		}
	}
	return cur, true
}

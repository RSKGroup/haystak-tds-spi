// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package functions

import (
	"encoding/json"
	"strconv"
	"strings"
)

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

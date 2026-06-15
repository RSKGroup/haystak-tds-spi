// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package funcs

func init() {
	register("ABS", func(a []any) any {
		if len(a) < 1 {
			return nil
		}
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
		return nil
	})
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

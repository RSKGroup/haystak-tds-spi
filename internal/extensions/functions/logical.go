// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package functions

func init() {
	register("ISNULL", coalesce)
	register("COALESCE", coalesce)
	register("NULLIF", func(a []any) any {
		if len(a) < 1 {
			return nil
		}
		if len(a) >= 2 && eqVals(a[0], a[1]) {
			return nil
		}
		return a[0]
	})
	register("CHOOSE", func(a []any) any {
		i, ok := argInt(a, 0)
		if !ok || i < 1 || int(i) >= len(a) {
			return nil
		}
		return a[i] // a[0] is the index; choices start at a[1]
	})
}

// coalesce backs both ISNULL(a, b) and COALESCE(a, b, …): the first non-NULL argument.
func coalesce(a []any) any {
	for _, v := range a {
		if v != nil {
			return v
		}
	}
	return nil
}

func eqVals(x, y any) bool {
	if x == nil || y == nil {
		return false
	}
	switch xv := x.(type) {
	case int64:
		if yv, ok := y.(int64); ok {
			return xv == yv
		}
		if yv, ok := toFloatOk(y); ok {
			return float64(xv) == yv
		}
	case float64:
		if yv, ok := toFloatOk(y); ok {
			return xv == yv
		}
	case string:
		yv, ok := y.(string)
		return ok && xv == yv
	case bool:
		yv, ok := y.(bool)
		return ok && xv == yv
	}
	return x == y
}

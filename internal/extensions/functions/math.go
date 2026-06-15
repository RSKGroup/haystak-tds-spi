// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package functions

import (
	"math"
	"math/rand"
)

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
	register("CEILING", func(a []any) any { return mathFn1(a, math.Ceil) })
	register("FLOOR", func(a []any) any { return mathFn1(a, math.Floor) })
	register("SQRT", func(a []any) any { return mathFn1(a, math.Sqrt) })
	register("EXP", func(a []any) any { return mathFn1(a, math.Exp) })
	register("LOG10", func(a []any) any { return mathFn1(a, math.Log10) })
	register("SIN", func(a []any) any { return mathFn1(a, math.Sin) })
	register("COS", func(a []any) any { return mathFn1(a, math.Cos) })
	register("TAN", func(a []any) any { return mathFn1(a, math.Tan) })
	register("COT", func(a []any) any { return mathFn1(a, func(x float64) float64 { return 1 / math.Tan(x) }) })
	register("ASIN", func(a []any) any { return mathFn1(a, math.Asin) })
	register("ACOS", func(a []any) any { return mathFn1(a, math.Acos) })
	register("ATAN", func(a []any) any { return mathFn1(a, math.Atan) })
	register("DEGREES", func(a []any) any { return mathFn1(a, func(x float64) float64 { return x * 180 / math.Pi }) })
	register("RADIANS", func(a []any) any { return mathFn1(a, func(x float64) float64 { return x * math.Pi / 180 }) })
	register("SQUARE", func(a []any) any { return mathFn1(a, func(x float64) float64 { return x * x }) })
	register("PI", func([]any) any { return math.Pi })
	register("POWER", func(a []any) any { return mathFn2(a, math.Pow) })
	register("ATN2", func(a []any) any { return mathFn2(a, math.Atan2) })
	register("LOG", func(a []any) any {
		x, ok := toFloatOk(arg0(a))
		if !ok {
			return nil
		}
		if base, ok := toFloatOk(arg1(a)); ok {
			return math.Log(x) / math.Log(base)
		}
		return math.Log(x)
	})
	register("ROUND", func(a []any) any {
		x, ok := toFloatOk(arg0(a))
		if !ok {
			return nil
		}
		n := int64(0)
		if v, ok := argInt(a, 1); ok {
			n = v
		}
		f := math.Pow(10, float64(n))
		return math.Round(x*f) / f
	})
	register("SIGN", func(a []any) any {
		x, ok := toFloatOk(arg0(a))
		if !ok {
			return nil
		}
		switch {
		case x > 0:
			return int64(1)
		case x < 0:
			return int64(-1)
		}
		return int64(0)
	})
	register("RAND", func(a []any) any {
		if s, ok := argInt(a, 0); ok {
			return rand.New(rand.NewSource(s)).Float64()
		}
		return rand.Float64()
	})
}

func mathFn1(a []any, f func(float64) float64) any {
	if x, ok := toFloatOk(arg0(a)); ok {
		return f(x)
	}
	return nil
}

func mathFn2(a []any, f func(x, y float64) float64) any {
	x, ok := toFloatOk(arg0(a))
	y, ok2 := toFloatOk(arg1(a))
	if ok && ok2 {
		return f(x, y)
	}
	return nil
}

func arg0(a []any) any {
	if len(a) > 0 {
		return a[0]
	}
	return nil
}

func arg1(a []any) any {
	if len(a) > 1 {
		return a[1]
	}
	return nil
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

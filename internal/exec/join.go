// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package exec

import (
	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
)

func Join(lcols []catalog.Column, lrows [][]any, jt tds.JoinType, rcols []catalog.Column, rrows [][]any, on *tds.Expr) ([]catalog.Column, [][]any, error) {
	cols := make([]catalog.Column, 0, len(lcols)+len(rcols))
	cols = append(cols, lcols...)
	cols = append(cols, rcols...)
	idx := indexCols(cols)
	rightMatched := make([]bool, len(rrows))
	var out [][]any
	for _, lr := range lrows {
		matched := false
		for ri, rr := range rrows {
			combined := make([]any, 0, len(lr)+len(rr))
			combined = append(combined, lr...)
			combined = append(combined, rr...)
			keep := true
			if on != nil {
				ok, err := evalExpr(idx, combined, on, nil)
				if err != nil {
					return nil, nil, err
				}
				keep = ok
			}
			if keep {
				out = append(out, combined)
				matched = true
				rightMatched[ri] = true
			}
		}
		if !matched && (jt == tds.JoinLeft || jt == tds.JoinFull) {
			combined := append(append([]any{}, lr...), make([]any, len(rcols))...)
			out = append(out, combined)
		}
	}
	if jt == tds.JoinRight || jt == tds.JoinFull {
		for ri, rr := range rrows {
			if !rightMatched[ri] {
				combined := append(make([]any, len(lcols)), rr...)
				out = append(out, combined)
			}
		}
	}
	return cols, out, nil
}

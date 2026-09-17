// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package functions

import "testing"

func TestStringSearchFunctionsFoldASCIICase(t *testing.T) {
	if got := replaceFold("Smith and SMITH, smithy", "smith", "Jones"); got != "Jones and Jones, Jonesy" {
		t.Errorf("REPLACE = %q", got)
	}
	if got := replaceFold("Émile émile", "émile", "x"); got != "Émile x" {
		t.Errorf("REPLACE must fold A-Z only, got %q", got)
	}
	if got := replaceFold("abc", "", "x"); got != "abc" {
		t.Errorf("REPLACE with an empty pattern must return the input, got %q", got)
	}
	if got := registry["REPLACE"]([]any{"Mr SMITH", "smith", "Jones"}); got != "Mr Jones" {
		t.Errorf("REPLACE() = %v", got)
	}
	charindex := registry["CHARINDEX"]
	if got := charindex([]any{"SMITH", "mr smith"}); got != int64(4) {
		t.Errorf("CHARINDEX = %v, want 4", got)
	}
	if got := registry["PATINDEX"]([]any{"%SMI_H%", "mr smith"}); got != int64(4) {
		t.Errorf("PATINDEX = %v, want 4", got)
	}
}

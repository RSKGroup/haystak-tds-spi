// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package functions

import (
	"os"
	"regexp"
	"testing"
)

// Every function string.go registers must say which of its arguments are text, so none is guessed.
func TestEveryStringFunctionDeclaresItsTextArguments(t *testing.T) {
	src, err := os.ReadFile("string.go")
	if err != nil {
		t.Fatal(err)
	}
	names := regexp.MustCompile(`register\("([A-Z_]+)"`).FindAllStringSubmatch(string(src), -1)
	if len(names) < 20 {
		t.Fatalf("found only %d registrations in string.go", len(names))
	}
	for _, m := range names {
		if _, ok := textParams[m[1]]; !ok {
			t.Errorf("string function %s has no textParams entry", m[1])
		}
	}
	for name := range textParams {
		if _, ok := registry[name]; !ok {
			t.Errorf("textParams names %s, which is not a registered function", name)
		}
	}
}

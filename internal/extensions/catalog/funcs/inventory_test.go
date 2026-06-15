// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package funcs

import (
	"flag"
	"os"
	"strings"
	"testing"
)

// readmePath is internal/extensions/README.md, relative to this package (catalog/funcs).
const readmePath = "../../README.md"

const (
	genBegin = "<!-- BEGIN GENERATED: scalar-functions (go test ./internal/extensions/catalog/funcs -run Readme -update) -->"
	genEnd   = "<!-- END GENERATED: scalar-functions -->"
)

var update = flag.Bool("update", false, "rewrite the generated scalar-function section of the README")

// TestEveryImplementedFunctionIsDocumented is the chef: nothing ships out of the kitchen undocumented.
// Every registered function (and every implemented-elsewhere exception) must appear in the Manifest.
func TestEveryImplementedFunctionIsDocumented(t *testing.T) {
	inManifest := map[string]bool{}
	for _, fam := range Manifest {
		for _, fn := range fam.Funcs {
			inManifest[strings.ToUpper(fn)] = true
		}
	}
	for _, n := range Names() {
		if !inManifest[n] {
			t.Errorf("registered function %q is not in the README Manifest (inventory.go) — add it to its family", n)
		}
	}
	for n := range implementedElsewhere {
		if !inManifest[strings.ToUpper(n)] {
			t.Errorf("implemented-elsewhere function %q is not in the Manifest", n)
		}
	}
}

// TestFamilyFilesExist fails when a family with a registry-backed function names a file that is missing —
// catches a claimed ✓ in `string.go` when `string.go` was never created.
func TestFamilyFilesExist(t *testing.T) {
	for _, fam := range Manifest {
		hasRegistry := false
		for _, fn := range fam.Funcs {
			if _, ok := registry[strings.ToUpper(fn)]; ok {
				hasRegistry = true
				break
			}
		}
		if hasRegistry {
			if _, err := os.Stat(fam.File); err != nil {
				t.Errorf("family %q has registered functions but its file %q is missing", fam.Name, fam.File)
			}
		}
	}
}

// TestReadmeInventoryInSync fails when the README's generated section drifts from Render() (a fake ✓ or
// a stale row). Regenerate with the -update flag.
func TestReadmeInventoryInSync(t *testing.T) {
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	got := string(data)
	i, j := strings.Index(got, genBegin), strings.Index(got, genEnd)
	if i < 0 || j < 0 {
		t.Fatalf("README is missing the generated markers %q / %q", genBegin, genEnd)
	}
	current := got[i : j+len(genEnd)]
	want := genBegin + "\n\n" + Render() + "\n" + genEnd
	if *update {
		if err := os.WriteFile(readmePath, []byte(strings.Replace(got, current, want, 1)), 0o644); err != nil {
			t.Fatalf("update README: %v", err)
		}
		return
	}
	if current != want {
		t.Errorf("README scalar-function inventory is stale — run:\n  go test ./internal/extensions/catalog/funcs -run Readme -update")
	}
}

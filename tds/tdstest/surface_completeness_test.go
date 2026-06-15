// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tdstest

import (
	"sort"
	"strings"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
	"github.com/RSKGroup/haystak-tds-spi/internal/exec"
	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/functions"
	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/infoschema"
	"github.com/RSKGroup/haystak-tds-spi/internal/extensions/sysviews"
	"github.com/RSKGroup/haystak-tds-spi/internal/tsql"
)

// wiredElements is the union of every dispatch list, reflected from code (not the README). The keys are
// the namespaced wired elements the surface suite must cover.
func wiredElements() map[string]bool {
	w := map[string]bool{}
	add := func(ns string, names []string) {
		for _, n := range names {
			w[ns+":"+n] = true
		}
	}
	add("func", functions.RegisteredNames())
	add("func", exec.EnvScalarNames())
	add("agg", exec.AggregateNames())
	add("sys", sysviews.SupportedViews())
	add("infoschema", infoschema.SupportedViews())
	add("proc", engine.SupportedProcs())
	add("parser", tsql.SpecialFuncs())
	return w
}

// TestSurfaceCompleteness fails on any wired element lacking a surface case (gap) or any case naming an unwired element (orphan).
func TestSurfaceCompleteness(t *testing.T) {
	wired := wiredElements()
	covered := map[string]bool{}
	for _, c := range allCases() {
		if c.Element == "" {
			continue // language/feature cases prove no specific element
		}
		covered[c.Element] = true
	}

	var gaps, orphans []string
	for el := range wired {
		if !covered[el] {
			gaps = append(gaps, el)
		}
	}
	for el := range covered {
		if !wired[el] {
			orphans = append(orphans, el)
		}
	}
	sort.Strings(gaps)
	sort.Strings(orphans)

	if len(gaps) > 0 {
		t.Errorf("%d wired element(s) have no surface case:\n  %s", len(gaps), strings.Join(gaps, "\n  "))
	}
	if len(orphans) > 0 {
		t.Errorf("%d surface case(s) reference unwired elements:\n  %s", len(orphans), strings.Join(orphans, "\n  "))
	}
	if len(gaps) == 0 && len(orphans) == 0 {
		t.Logf("surface covers all %d wired elements", len(wired))
	}
}

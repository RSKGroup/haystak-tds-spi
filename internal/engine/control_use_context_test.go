// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
)

// SMO's Object Explorer "resetting the context" batch is a top-level USE bracketing a control-flow IF.
// That whole batch routes through the control-flow runner, which previously ran USE against a throwaway
// session -- so the connection's database never changed and no ENVCHANGE fired. SMO then issued the
// table enumeration believing it was in the target db while the gateway was still in master, yielding
// zero tables (the "dive from the top fails, but selecting the database on connect works" symptom).
func TestControlFlowUsePersistsAndEnvChanges(t *testing.T) {
	s := engine.NewSession(inmem.New(), "master")
	batch := "use [tenant];\nif 1 = 1 BEGIN SELECT 0 AS r END ELSE BEGIN SELECT 1 AS r END;\nuse [tenant]"
	_, envDB, err := s.ExecBatch(context.Background(), batch)
	if err != nil {
		t.Fatalf("control-flow batch with USE: %v", err)
	}
	if got := s.Database(); got != "tenant" {
		t.Fatalf("session db after control-flow USE = %q, want tenant (USE must persist)", got)
	}
	if envDB != "tenant" {
		t.Errorf("envDB = %q, want tenant (ENVCHANGE must fire so the client tracks the context)", envDB)
	}
}

// Only the taken branch's USE applies, and it persists.
func TestControlFlowConditionalUse(t *testing.T) {
	s := engine.NewSession(inmem.New(), "master")
	if _, _, err := s.ExecBatch(context.Background(), "IF 1 = 1 use [taken] ELSE use [skipped]"); err != nil {
		t.Fatalf("conditional USE: %v", err)
	}
	if got := s.Database(); got != "taken" {
		t.Errorf("conditional USE db = %q, want taken", got)
	}
}

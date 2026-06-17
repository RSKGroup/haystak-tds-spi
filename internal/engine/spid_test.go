// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
	"github.com/RSKGroup/haystak-tds-spi/tds"
)

func TestSpid(t *testing.T) {
	if g := qry(t, "SELECT @@SPID"); len(g) != 1 || g[0][0] != int64(1) {
		t.Errorf("no-session @@SPID = %v, want 1", g)
	}

	ctx := tds.WithCurrentSession(
		tds.WithSessions(context.Background(), []tds.SessionInfo{{SessionID: 53, LoginName: "sa"}}),
		tds.SessionInfo{SessionID: 53, LoginName: "sa"})

	bare, err := engine.Query(ctx, inmem.New(), "SELECT @@SPID")
	if err != nil {
		t.Fatal(err)
	}
	if g := collect(t, bare); len(g) != 1 || g[0][0] != int64(53) {
		t.Errorf("@@SPID = %v, want 53", g)
	}

	health, err := engine.Query(ctx, inmem.New(), "SELECT login_name FROM sys.dm_exec_sessions WHERE session_id = @@SPID")
	if err != nil {
		t.Fatal(err)
	}
	if g := collect(t, health); len(g) != 1 || cell(g[0][0]) != "sa" {
		t.Errorf("WHERE session_id = @@SPID = %v, want [[sa]]", g)
	}
}

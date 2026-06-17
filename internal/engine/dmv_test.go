// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"testing"
	"time"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
	"github.com/RSKGroup/haystak-tds-spi/tds"
)

func sessionsCtx(s ...tds.SessionInfo) context.Context {
	return tds.WithSessions(context.Background(), s)
}

// With no registry snapshot in ctx the runtime DMVs return the correct empty shape.
func TestDMVEmptyShape(t *testing.T) {
	for _, v := range []string{"dm_exec_sessions", "dm_exec_connections", "dm_exec_requests", "dm_exec_query_stats", "dm_os_waiting_tasks"} {
		if n := len(qry(t, "SELECT * FROM sys."+v)); n != 0 {
			t.Errorf("no-session sys.%s = %d rows, want 0", v, n)
		}
	}
	if n := len(qry(t, "EXEC sp_who")); n != 0 {
		t.Errorf("no-session sp_who = %d rows, want 0", n)
	}
}

// The DMVs and sp_who enumerate every live session in the registry snapshot.
func TestDMVAllSessions(t *testing.T) {
	ctx := sessionsCtx(
		tds.SessionInfo{SessionID: 51, LoginName: "ada", Host: "h1", Program: "app1", LoginTime: time.Unix(0, 0).UTC()},
		tds.SessionInfo{SessionID: 52, LoginName: "alan", Host: "h2", Program: "app2", LoginTime: time.Unix(0, 0).UTC()},
	)
	rs, err := engine.Query(ctx, inmem.New(), "SELECT session_id, login_name, host_name FROM sys.dm_exec_sessions ORDER BY session_id")
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, rs)
	if len(got) != 2 || got[0][0] != int64(51) || cell(got[0][1]) != "ada" || got[1][0] != int64(52) || cell(got[1][1]) != "alan" {
		t.Fatalf("dm_exec_sessions = %v, want sessions 51/ada and 52/alan", got)
	}

	// A health check filters to one session by id.
	one, err := engine.Query(ctx, inmem.New(), "SELECT login_name FROM sys.dm_exec_sessions WHERE session_id = 52")
	if err != nil {
		t.Fatal(err)
	}
	if g := collect(t, one); len(g) != 1 || cell(g[0][0]) != "alan" {
		t.Fatalf("WHERE session_id = 52 = %v, want [[alan]]", g)
	}

	rw, err := engine.Query(ctx, inmem.New(), "EXEC sp_who")
	if err != nil {
		t.Fatal(err)
	}
	if w := collect(t, rw); len(w) != 2 {
		t.Fatalf("sp_who = %d rows, want 2", len(w))
	}
}

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

func sessionCtx() context.Context {
	return tds.WithSessionInfo(context.Background(), tds.SessionInfo{
		SessionID: 7, LoginName: "sa", Host: "client01", Program: "app", LoginTime: time.Unix(0, 0).UTC(),
	})
}

// With no session in ctx the runtime DMVs return the correct empty shape.
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

// With a session in ctx the DMVs and sp_who project the current session (covers WHERE session_id = @@SPID).
func TestDMVCurrentSession(t *testing.T) {
	ctx := sessionCtx()
	rs, err := engine.Query(ctx, inmem.New(), "SELECT session_id, login_name, host_name FROM sys.dm_exec_sessions WHERE session_id = 7")
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, rs)
	if len(got) != 1 || got[0][0] != int64(7) || cell(got[0][1]) != "sa" || cell(got[0][2]) != "client01" {
		t.Fatalf("dm_exec_sessions = %v, want [[7 sa client01]]", got)
	}
	rw, err := engine.Query(ctx, inmem.New(), "EXEC sp_who")
	if err != nil {
		t.Fatal(err)
	}
	w := collect(t, rw)
	if len(w) != 1 || w[0][0] != int64(7) || cell(w[0][3]) != "sa" {
		t.Fatalf("sp_who = %v, want one row for spid 7 / sa", w)
	}
}

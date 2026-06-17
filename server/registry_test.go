// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"testing"
	"time"

	"github.com/RSKGroup/haystak-tds-spi/tds"
)

func TestSessionRegistry(t *testing.T) {
	r := newSessionRegistry()
	a := r.add("ada", "h1", "app1", time.Unix(0, 0))
	b := r.add("alan", "h2", "app2", time.Unix(0, 0))
	if a.SessionID != 51 || b.SessionID != 52 {
		t.Fatalf("spids = %d, %d, want 51, 52", a.SessionID, b.SessionID)
	}
	if a.LoginName != "ada" || b.Host != "h2" {
		t.Errorf("session fields not captured: %+v %+v", a, b)
	}
	if len(r.snapshot()) != 2 {
		t.Fatalf("snapshot = %d, want 2", len(r.snapshot()))
	}
	r.remove(a.SessionID)
	snap := r.snapshot()
	if len(snap) != 1 || snap[0].SessionID != 52 {
		t.Fatalf("after remove snapshot = %v, want only spid 52", snap)
	}
}

func TestOwnSession(t *testing.T) {
	r := newSessionRegistry()
	ada := r.add("ada", "h1", "app1", time.Unix(0, 0))
	r.add("alan", "h2", "app2", time.Unix(0, 0))
	all := r.snapshot()
	own := ownSession(ada, all)
	if len(own) != 1 || own[0].SessionID != ada.SessionID || own[0].LoginName != "ada" {
		t.Fatalf("ownSession = %v, want only ada's row", own)
	}
	if got := ownSession(tds.SessionInfo{SessionID: 999}, all); got != nil {
		t.Fatalf("ownSession for unknown spid = %v, want nil", got)
	}
}

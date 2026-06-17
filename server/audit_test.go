// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"testing"
	"time"

	"github.com/RSKGroup/haystak-tds-spi/tds"
)

func TestAuditHook(t *testing.T) {
	var got []tds.SessionEvent
	s := &Server{Audit: func(ev tds.SessionEvent) { got = append(got, ev) }}
	s.audit("login", true, tds.SessionInfo{SessionID: 51, LoginName: "ada"}, time.Unix(0, 0))
	s.audit("login", false, tds.SessionInfo{LoginName: "mallory"}, time.Unix(0, 0))
	s.audit("logout", true, tds.SessionInfo{SessionID: 51, LoginName: "ada"}, time.Unix(0, 0))

	if len(got) != 3 {
		t.Fatalf("got %d events, want 3", len(got))
	}
	if got[0].Kind != "login" || !got[0].Succeeded {
		t.Errorf("event 0 = %+v, want successful login", got[0])
	}
	if got[1].Succeeded || got[1].Session.LoginName != "mallory" {
		t.Errorf("event 1 = %+v, want failed login for mallory", got[1])
	}
	if got[2].Kind != "logout" || !got[2].Succeeded {
		t.Errorf("event 2 = %+v, want logout", got[2])
	}

	// With no hook set (logging disabled) the audit is a safe no-op.
	(&Server{}).audit("login", false, tds.SessionInfo{LoginName: "x"}, time.Unix(0, 0))
}

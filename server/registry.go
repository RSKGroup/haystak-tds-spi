// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"sync"
	"time"

	"github.com/RSKGroup/haystak-tds-spi/tds"
)

// sessionRegistry tracks every live connection so the runtime DMVs and sp_who can enumerate them.
type sessionRegistry struct {
	mu       sync.Mutex
	nextSPID int
	sessions map[int]tds.SessionInfo
}

func newSessionRegistry() *sessionRegistry {
	return &sessionRegistry{nextSPID: 51, sessions: map[int]tds.SessionInfo{}}
}

func (r *sessionRegistry) add(login, host, app string, at time.Time) tds.SessionInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	spid := r.nextSPID
	r.nextSPID++
	s := tds.SessionInfo{SessionID: spid, LoginName: login, Host: host, Program: app, LoginTime: at}
	r.sessions[spid] = s
	return s
}

func (r *sessionRegistry) remove(spid int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, spid)
}

func (r *sessionRegistry) snapshot() []tds.SessionInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]tds.SessionInfo, 0, len(r.sessions))
	for _, s := range r.sessions {
		out = append(out, s)
	}
	return out
}

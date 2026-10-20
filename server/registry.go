// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"net"
	"sync"
	"time"

	"github.com/RSKGroup/haystak-tds-spi/tds"
)

// sessionRegistry tracks every live connection so the runtime DMVs and sp_who can enumerate them.
type sessionRegistry struct {
	mu       sync.Mutex
	nextSPID int
	sessions map[int]tds.SessionInfo
	conns    map[int]liveSession
}

// liveSession is what ending a session needs: its connection and the principal it signed in as.
type liveSession struct {
	info  tds.SessionInfo
	conn  net.Conn
	princ tds.Principal
}

func newSessionRegistry() *sessionRegistry {
	return &sessionRegistry{nextSPID: 51, sessions: map[int]tds.SessionInfo{}, conns: map[int]liveSession{}}
}

func (r *sessionRegistry) attach(info tds.SessionInfo, conn net.Conn, princ tds.Principal) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.sessions[info.SessionID]; ok {
		r.conns[info.SessionID] = liveSession{info: info, conn: conn, princ: princ}
	}
}

func (r *sessionRegistry) live() []liveSession {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]liveSession, 0, len(r.conns))
	for _, l := range r.conns {
		out = append(out, l)
	}
	return out
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
	delete(r.conns, spid)
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

// ownSession narrows a snapshot to the caller's own row (the non-privileged DMV view).
func ownSession(cur tds.SessionInfo, all []tds.SessionInfo) []tds.SessionInfo {
	for _, s := range all {
		if s.SessionID == cur.SessionID {
			return []tds.SessionInfo{s}
		}
	}
	return nil
}

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package server runs the TDS wire protocol (PRELOGIN/LOGIN7, TLS-in-TDS, SQL_BATCH, RPC) on a tds.Backend.
// ListenAndServe is the one-liner; Server adds TLS, authentication, and server/database naming.
package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime/debug"
	"sync"
	"time"

	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
	"github.com/RSKGroup/haystak-tds-spi/internal/wire"
	"github.com/RSKGroup/haystak-tds-spi/tds"
)

// Server serves a tds.Backend over the TDS wire. The zero value needs only Backend set; the rest is optional.
type Server struct {
	Backend    tds.Backend
	Auth       tds.Authenticator // optional; falls back to a Backend that implements Authenticator
	ServerName string            // reported as @@SERVERNAME (default "haystak")
	Database   string            // reported as the current database (default "master")
	TLSConfig  *tls.Config       // non-nil enables TLS-in-TDS
	Logf       func(string, ...any)
	Audit      func(tds.SessionEvent) // optional: called on each login and logout
	// SessionVisibility decides whether a principal may enumerate every live session via the runtime
	// DMVs / sp_who (SQL Server's VIEW SERVER STATE gate). nil ⇒ a caller sees only its own session.
	SessionVisibility func(context.Context, tds.Principal) bool

	regOnce sync.Once
	reg     *sessionRegistry
}

func (s *Server) registry() *sessionRegistry {
	s.regOnce.Do(func() { s.reg = newSessionRegistry() })
	return s.reg
}

func (s *Server) audit(kind string, ok bool, info tds.SessionInfo, at time.Time) {
	s.logf("audit: %s ok=%v spid=%d user=%q host=%q app=%q", kind, ok, info.SessionID, info.LoginName, info.Host, info.Program)
	if s.Audit != nil {
		s.Audit(tds.SessionEvent{Kind: kind, Succeeded: ok, Session: info, At: at})
	}
}

// ListenAndServe serves b on addr (host:port) with default settings: no TLS, anonymous auth.
func ListenAndServe(addr string, b tds.Backend) error {
	return (&Server{Backend: b}).ListenAndServe(addr)
}

// ListenAndServe listens on addr (host:port) and serves until the listener fails.
func (s *Server) ListenAndServe(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	return s.Serve(ln)
}

// Serve accepts connections on ln and serves each in its own goroutine until Accept fails.
func (s *Server) Serve(ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.handle(conn)
	}
}

func (s *Server) logf(format string, args ...any) {
	if s.Logf != nil {
		s.Logf(format, args...)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	defer func() {
		if r := recover(); r != nil {
			s.logf("PANIC in connection: %v\n%s", r, debug.Stack())
		}
	}()
	s.logf("conn from %s", conn.RemoteAddr())
	sess, princ, db, info, err := s.handshake(conn)
	if err != nil {
		s.logf("handshake error: %v", err)
		return
	}
	defer func() {
		s.registry().remove(info.SessionID)
		s.audit("logout", true, info, time.Now())
	}()
	s.logf("handshake complete (user=%q db=%q spid=%d)", princ.Username, db, info.SessionID)
	s.serve(sess, princ, db, info)
}

func (s *Server) handshake(conn net.Conn) (net.Conn, tds.Principal, string, tds.SessionInfo, error) {
	var none tds.Principal
	var nosess tds.SessionInfo
	pre, err := wire.ReadMessage(conn)
	if err != nil {
		return nil, none, "", nosess, err
	}
	s.logf("recv PRELOGIN type=0x%02X len=%d", byte(pre.Type), len(pre.Payload))
	if pre.Type != wire.PacketPreLogin {
		return nil, none, "", nosess, errors.New("server: expected PRELOGIN")
	}

	useTLS := false
	if s.TLSConfig != nil {
		if pl, perr := wire.ParsePrelogin(pre.Payload); perr == nil {
			if enc, ok := pl.Encryption(); ok && enc != wire.EncryptNotSup {
				useTLS = true
			}
		}
	}
	respEnc := wire.EncryptNotSup
	if useTLS {
		respEnc = wire.EncryptOn
	}
	if err := s.send(conn, wire.ServerPreloginResponse(respEnc)); err != nil {
		return nil, none, "", nosess, err
	}
	s.logf("sent PRELOGIN response (enc=%d)", respEnc)

	if useTLS {
		tlsConn, terr := wire.ServerTLS(conn, s.TLSConfig)
		if terr != nil {
			return nil, none, "", nosess, fmt.Errorf("server: tls handshake: %w", terr)
		}
		s.logf("TLS established")
		conn = tlsConn
	}

	login, err := wire.ReadMessage(conn)
	if err != nil {
		return nil, none, "", nosess, err
	}
	s.logf("recv LOGIN7 type=0x%02X len=%d", byte(login.Type), len(login.Payload))
	if login.Type != wire.PacketLogin7 {
		return nil, none, "", nosess, errors.New("server: expected LOGIN7")
	}
	l, err := wire.ParseLogin7(login.Payload)
	if err != nil {
		return nil, none, "", nosess, err
	}
	s.logf("login user=%q db=%q app=%q tls=%v", l.UserName, l.Database, l.AppName, useTLS)

	princ, autherr := s.authenticate(context.Background(), tds.Login{
		Username: l.UserName, Password: l.Password, Database: l.Database,
		AppName: l.AppName, Host: l.HostName,
	})
	if autherr != nil {
		s.logf("auth rejected user=%q: %v", l.UserName, autherr)
		s.audit("login", false, tds.SessionInfo{LoginName: l.UserName, Host: l.HostName, Program: l.AppName, LoginTime: time.Now()}, time.Now())
		_ = s.send(conn, wire.LoginError("Login failed for user '"+l.UserName+"'."))
		return nil, none, "", nosess, autherr
	}

	loginDB := s.database()
	if l.Database != "" {
		loginDB = l.Database
	}
	if err := s.send(conn, wire.BuildLoginResponse(s.serverName(), loginDB)); err != nil {
		return nil, none, "", nosess, err
	}
	s.logf("sent LOGIN response")
	info := s.registry().add(l.UserName, l.HostName, l.AppName, time.Now())
	s.audit("login", true, info, info.LoginTime)
	return conn, princ, loginDB, info, nil
}

// authenticate runs the configured Authenticator, else a Backend that implements one; with neither
// configured it allows the connection anonymously (trusting the presented username).
func (s *Server) authenticate(ctx context.Context, l tds.Login) (tds.Principal, error) {
	a := s.Auth
	if a == nil {
		if ba, ok := s.Backend.(tds.Authenticator); ok {
			a = ba
		}
	}
	if a == nil {
		return tds.Principal{Username: l.Username}, nil
	}
	return a.Authenticate(ctx, l)
}

// StaticAuth is a convenience Authenticator backed by a username→password map (for demos/tests).
func StaticAuth(creds map[string]string) tds.Authenticator {
	return tds.AuthFunc(func(ctx context.Context, l tds.Login) (tds.Principal, error) {
		if pw, ok := creds[l.Username]; ok && pw == l.Password {
			return tds.Principal{Username: l.Username}, nil
		}
		return tds.Principal{}, fmt.Errorf("login failed for user %q", l.Username)
	})
}

func (s *Server) serve(conn net.Conn, princ tds.Principal, initialDB string, info tds.SessionInfo) {
	ctx := tds.WithCurrentSession(tds.WithPrincipal(context.Background(), princ), info)
	seeAll := s.SessionVisibility != nil && s.SessionVisibility(ctx, princ)
	sess := engine.NewSession(s.Backend, initialDB)
	for {
		msg, err := wire.ReadMessage(conn)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				s.logf("read error: %v", err)
			}
			return
		}
		if !s.handleMessage(conn, sess, ctx, info, seeAll, msg) {
			return
		}
	}
}

// handleMessage processes one client message (false ⇒ close the connection); a panic is recovered and returned as a SQL error.
func (s *Server) handleMessage(conn net.Conn, sess *engine.Session, ctx context.Context, info tds.SessionInfo, seeAll bool, msg wire.Message) (keepGoing bool) {
	defer func() {
		if r := recover(); r != nil {
			s.logf("PANIC handling message: %v\n%s", r, debug.Stack())
			_ = s.send(conn, wire.BuildError(fmt.Sprintf("internal error: %v", r)))
			keepGoing = true
		}
	}()
	s.logf("recv type=0x%02X len=%d", byte(msg.Type), len(msg.Payload))
	var sql string
	switch msg.Type {
	case wire.PacketSQLBatch:
		sql = wire.DecodeSQLBatch(msg.Payload)
	case wire.PacketRPC:
		expanded, ok := wire.DecodeRPC(msg.Payload)
		if !ok {
			s.logf("rpc decode declined: %s", wire.RPCDiag(msg.Payload))
			return s.send(conn, wire.EmptyDone()) == nil
		}
		sql = expanded
	case wire.PacketAttention:
		_ = s.send(conn, wire.AttentionAck()) // ack the cancel so the client doesn't wait forever
		return true
	default:
		return true
	}
	s.logf("stmt: %q", sql)
	visible := s.registry().snapshot()
	if !seeAll {
		visible = ownSession(info, visible)
	}
	qctx := tds.WithSessions(ctx, visible)
	results, envDB, err := sess.ExecBatch(qctx, sql)
	if err != nil {
		s.logf("query error: %v", err)
		_ = s.send(conn, wire.BuildError(err.Error()))
		return true
	}
	resp, err := buildResponse(results, envDB)
	if err != nil {
		s.logf("response error: %v", err)
		_ = s.send(conn, wire.BuildError(err.Error()))
		return true
	}
	return s.send(conn, resp) == nil
}

// buildResponse renders every result set in order (DONE_MORE between them), led by an ENVCHANGE on USE.
func buildResponse(results []engine.Result, envDB string) ([]byte, error) {
	var out []byte
	if envDB != "" {
		out = wire.EnvChangeDatabase(envDB)
	}
	if len(results) == 0 {
		return append(out, wire.EmptyDone()...), nil
	}
	for i, r := range results {
		more := i < len(results)-1
		if r.Rows == nil {
			n := uint64(0)
			if r.Affected > 0 {
				n = uint64(r.Affected)
			}
			out = append(out, wire.DoneRows(n, more)...)
			continue
		}
		cols := r.Rows.Columns()
		var data [][]any
		for r.Rows.Next() {
			v, err := r.Rows.Values()
			if err != nil {
				r.Rows.Close()
				return nil, err
			}
			data = append(data, v)
		}
		err := r.Rows.Err()
		r.Rows.Close()
		if err != nil {
			return nil, err
		}
		out = append(out, wire.BuildResultSet(cols, data, more)...)
	}
	return out, nil
}

func (s *Server) send(conn net.Conn, payload []byte) error {
	return wire.WriteMessage(conn, wire.Message{Type: wire.PacketResponse, Payload: payload}, wire.DefaultPacketSize)
}

func (s *Server) serverName() string {
	if s.ServerName != "" {
		return s.ServerName
	}
	return "haystak"
}

func (s *Server) database() string {
	if s.Database != "" {
		return s.Database
	}
	return "master"
}

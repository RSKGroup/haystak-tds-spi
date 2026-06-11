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
	"net"

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
	s.logf("conn from %s", conn.RemoteAddr())
	sess, princ, db, err := s.handshake(conn)
	if err != nil {
		s.logf("handshake error: %v", err)
		return
	}
	s.logf("handshake complete (user=%q db=%q)", princ.Username, db)
	s.serve(sess, princ, db)
}

func (s *Server) handshake(conn net.Conn) (net.Conn, tds.Principal, string, error) {
	var none tds.Principal
	pre, err := wire.ReadMessage(conn)
	if err != nil {
		return nil, none, "", err
	}
	s.logf("recv PRELOGIN type=0x%02X len=%d", byte(pre.Type), len(pre.Payload))
	if pre.Type != wire.PacketPreLogin {
		return nil, none, "", errors.New("server: expected PRELOGIN")
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
		return nil, none, "", err
	}
	s.logf("sent PRELOGIN response (enc=%d)", respEnc)

	if useTLS {
		tlsConn, terr := wire.ServerTLS(conn, s.TLSConfig)
		if terr != nil {
			return nil, none, "", fmt.Errorf("server: tls handshake: %w", terr)
		}
		s.logf("TLS established")
		conn = tlsConn
	}

	login, err := wire.ReadMessage(conn)
	if err != nil {
		return nil, none, "", err
	}
	s.logf("recv LOGIN7 type=0x%02X len=%d", byte(login.Type), len(login.Payload))
	if login.Type != wire.PacketLogin7 {
		return nil, none, "", errors.New("server: expected LOGIN7")
	}
	l, err := wire.ParseLogin7(login.Payload)
	if err != nil {
		return nil, none, "", err
	}
	s.logf("login user=%q db=%q app=%q tls=%v", l.UserName, l.Database, l.AppName, useTLS)

	princ, autherr := s.authenticate(context.Background(), tds.Login{
		Username: l.UserName, Password: l.Password, Database: l.Database,
		AppName: l.AppName, Host: l.HostName,
	})
	if autherr != nil {
		s.logf("auth rejected user=%q: %v", l.UserName, autherr)
		_ = s.send(conn, wire.LoginError("Login failed for user '"+l.UserName+"'."))
		return nil, none, "", autherr
	}

	loginDB := s.database()
	if l.Database != "" {
		loginDB = l.Database
	}
	if err := s.send(conn, wire.BuildLoginResponse(s.serverName(), loginDB)); err != nil {
		return nil, none, "", err
	}
	s.logf("sent LOGIN response")
	return conn, princ, loginDB, nil
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

func (s *Server) serve(conn net.Conn, princ tds.Principal, initialDB string) {
	ctx := tds.WithPrincipal(context.Background(), princ)
	sess := engine.NewSession(s.Backend, initialDB)
	for {
		msg, err := wire.ReadMessage(conn)
		if err != nil {
			return
		}
		s.logf("recv type=0x%02X len=%d", byte(msg.Type), len(msg.Payload))
		var sql string
		switch msg.Type {
		case wire.PacketSQLBatch:
			sql = wire.DecodeSQLBatch(msg.Payload)
		case wire.PacketRPC:
			expanded, ok := wire.DecodeRPC(msg.Payload)
			if !ok {
				if err := s.send(conn, wire.EmptyDone()); err != nil {
					return
				}
				continue
			}
			sql = expanded
		default:
			continue
		}
		s.logf("stmt: %q", sql)
		rows, affected, envDB, err := sess.Exec(ctx, sql)
		if err != nil {
			s.logf("query error: %v", err)
			_ = s.send(conn, wire.BuildError(err.Error()))
			continue
		}
		resp, err := buildResponse(rows, affected, envDB)
		if err != nil {
			s.logf("response error: %v", err)
			_ = s.send(conn, wire.BuildError(err.Error()))
			continue
		}
		if err := s.send(conn, resp); err != nil {
			return
		}
	}
}

// buildResponse renders a result set or rows-affected, led by an ENVCHANGE when USE changed the db.
func buildResponse(rows tds.Rows, affected int64, envDB string) ([]byte, error) {
	var lead []byte
	if envDB != "" {
		lead = wire.EnvChangeDatabase(envDB)
	}
	if rows == nil {
		if affected >= 0 {
			return append(lead, wire.DoneWithCount(uint64(affected))...), nil
		}
		return append(lead, wire.EmptyDone()...), nil
	}
	defer rows.Close()
	cols := rows.Columns()
	var data [][]any
	for rows.Next() {
		v, err := rows.Values()
		if err != nil {
			return nil, err
		}
		data = append(data, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	body, err := wire.BuildResultResponse(cols, data)
	if err != nil {
		return nil, err
	}
	return append(lead, body...), nil
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

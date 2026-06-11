// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"crypto/tls"
	"net"
)

// ServerTLS performs the TDS TLS handshake (handshake records are wrapped in PRELOGIN
// packets) and returns a net.Conn that carries TLS for the rest of the session.
func ServerTLS(conn net.Conn, config *tls.Config) (net.Conn, error) {
	cfg := config.Clone()
	if cfg.MaxVersion == 0 || cfg.MaxVersion > tls.VersionTLS12 {
		cfg.MaxVersion = tls.VersionTLS12 // TDS-in-TLS packet wrapping needs the 1.2 handshake flight
	}
	hc := &handshakeConn{Conn: conn}
	tlsConn := tls.Server(hc, cfg)
	if err := tlsConn.Handshake(); err != nil {
		return nil, err
	}
	hc.done = true
	return tlsConn, nil
}

// handshakeConn carries the TLS handshake inside TDS packets, then passes through raw.
type handshakeConn struct {
	net.Conn
	done bool
	rbuf []byte
}

func (c *handshakeConn) Read(p []byte) (int, error) {
	if len(c.rbuf) > 0 {
		n := copy(p, c.rbuf)
		c.rbuf = c.rbuf[n:]
		return n, nil
	}
	if c.done {
		return c.Conn.Read(p)
	}
	msg, err := ReadMessage(c.Conn)
	if err != nil {
		return 0, err
	}
	c.rbuf = msg.Payload
	n := copy(p, c.rbuf)
	c.rbuf = c.rbuf[n:]
	return n, nil
}

func (c *handshakeConn) Write(p []byte) (int, error) {
	if c.done {
		return c.Conn.Write(p)
	}
	if err := WriteMessage(c.Conn, Message{Type: PacketPreLogin, Payload: p}, DefaultPacketSize); err != nil {
		return 0, err
	}
	return len(p), nil
}

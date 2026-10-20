// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
	"unicode/utf16"

	"github.com/RSKGroup/haystak-tds-spi/internal/wire"
	"github.com/RSKGroup/haystak-tds-spi/tds"
)

func testLogin7(user, pass string) []byte {
	const hdrLen = 94
	hdr := make([]byte, hdrLen)
	var data []byte
	put := func(pos int, s string, obfuscate bool) {
		u := utf16.Encode([]rune(s))
		b := make([]byte, len(u)*2)
		for i, c := range u {
			binary.LittleEndian.PutUint16(b[i*2:], c)
		}
		if obfuscate {
			for i, c := range b {
				b[i] = ((c << 4) | (c >> 4)) ^ 0xA5
			}
		}
		binary.LittleEndian.PutUint16(hdr[pos:], uint16(hdrLen+len(data)))
		binary.LittleEndian.PutUint16(hdr[pos+2:], uint16(len(u)))
		data = append(data, b...)
	}
	put(36, "host", false)
	put(40, user, false)
	put(44, pass, true)
	put(48, "test", false)
	put(52, "haystak", false)
	put(68, "", false)
	payload := append(hdr, data...)
	binary.LittleEndian.PutUint32(payload[0:4], uint32(len(payload)))
	return payload
}

func signIn(t *testing.T, addr, user, pass string) net.Conn {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	if err := wire.WriteMessage(c, wire.Message{Type: wire.PacketPreLogin, Payload: wire.BuildPrelogin(nil)}, wire.DefaultPacketSize); err != nil {
		t.Fatal(err)
	}
	if _, err := wire.ReadMessage(c); err != nil {
		t.Fatal(err)
	}
	if err := wire.WriteMessage(c, wire.Message{Type: wire.PacketLogin7, Payload: testLogin7(user, pass)}, wire.DefaultPacketSize); err != nil {
		t.Fatal(err)
	}
	if _, err := wire.ReadMessage(c); err != nil {
		t.Fatal(err)
	}
	return c
}

// A host that disables a login must be able to end the sessions that login already holds.
func TestCloseSessionsEndsOnlyTheMatchingLogins(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	events := make(chan tds.SessionEvent, 8)
	s := &Server{Auth: tds.AuthFunc(func(_ context.Context, l tds.Login) (tds.Principal, error) {
		return tds.Principal{Username: l.Username, Attributes: map[string]string{"k": "v"}}, nil
	}), Audit: func(ev tds.SessionEvent) { events <- ev }}
	go func() { _ = s.Serve(ln) }()

	ada := signIn(t, ln.Addr().String(), "ada", "pw")
	defer ada.Close()
	alan := signIn(t, ln.Addr().String(), "alan", "pw")
	defer alan.Close()
	<-events
	<-events

	ended := s.CloseSessions(func(info tds.SessionInfo, p tds.Principal) bool {
		return p.Username == "ada" && p.Attributes["k"] == "v" && info.LoginName == "ada"
	})
	if len(ended) != 1 || ended[0].LoginName != "ada" {
		t.Fatalf("ended %v, want ada's one session", ended)
	}
	if ev := <-events; ev.Kind != "logout" || ev.Session.LoginName != "ada" {
		t.Fatalf("next event %+v, want ada's logout", ev)
	}
	if _, err := ada.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Fatalf("ada's connection still open after CloseSessions: %v", err)
	}
	if snap := s.registry().snapshot(); len(snap) != 1 || snap[0].LoginName != "alan" {
		t.Fatalf("registry after the close = %v, want only alan", snap)
	}
}

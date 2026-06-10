// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"bytes"
	"testing"
)

func TestRoundTripSingle(t *testing.T) {
	var buf bytes.Buffer
	in := Message{Type: PacketSQLBatch, Payload: []byte("SELECT 1")}
	if err := WriteMessage(&buf, in, DefaultPacketSize); err != nil {
		t.Fatal(err)
	}
	out, err := ReadMessage(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if out.Type != in.Type || !bytes.Equal(out.Payload, in.Payload) {
		t.Fatalf("got %+v, want %+v", out, in)
	}
}

func TestRoundTripMultiPacket(t *testing.T) {
	payload := bytes.Repeat([]byte("ab"), 100) // 200 bytes
	var buf bytes.Buffer
	in := Message{Type: PacketResponse, Payload: payload}
	if err := WriteMessage(&buf, in, headerSize+32); err != nil { // 32-byte bodies force splitting
		t.Fatal(err)
	}
	out, err := ReadMessage(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if out.Type != in.Type || !bytes.Equal(out.Payload, payload) {
		t.Fatalf("payload mismatch: got %d bytes", len(out.Payload))
	}
}

func TestEmptyPayload(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteMessage(&buf, Message{Type: PacketAttention}, DefaultPacketSize); err != nil {
		t.Fatal(err)
	}
	out, err := ReadMessage(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if out.Type != PacketAttention || len(out.Payload) != 0 {
		t.Fatalf("got %+v", out)
	}
}

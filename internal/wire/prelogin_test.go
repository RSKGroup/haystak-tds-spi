// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"bytes"
	"testing"
)

func TestPreloginRoundTrip(t *testing.T) {
	entries := []PreloginEntry{
		{PreloginVersion, []byte{1, 2, 3, 4, 5, 6}},
		{PreloginEncryption, []byte{byte(EncryptOff)}},
		{PreloginMARS, []byte{0}},
	}
	pl, err := ParsePrelogin(BuildPrelogin(entries))
	if err != nil {
		t.Fatal(err)
	}
	enc, ok := pl.Encryption()
	if !ok || enc != EncryptOff {
		t.Fatalf("encryption = %v ok=%v, want EncryptOff", enc, ok)
	}
	if !bytes.Equal(pl.Options[PreloginVersion], []byte{1, 2, 3, 4, 5, 6}) {
		t.Fatalf("version = %v", pl.Options[PreloginVersion])
	}
}

func TestServerPreloginResponse(t *testing.T) {
	for _, want := range []Encryption{EncryptNotSup, EncryptOn} {
		pl, err := ParsePrelogin(ServerPreloginResponse(want))
		if err != nil {
			t.Fatal(err)
		}
		if enc, ok := pl.Encryption(); !ok || enc != want {
			t.Fatalf("encryption = %v, want %v", enc, want)
		}
	}
}

func TestParsePreloginOutOfRange(t *testing.T) {
	bad := []byte{byte(PreloginEncryption), 0x00, 0xFF, 0x00, 0x01, byte(0xFF)}
	if _, err := ParsePrelogin(bad); err == nil {
		t.Fatal("expected out-of-range error")
	}
}

func TestParsePreloginNoTerminator(t *testing.T) {
	if _, err := ParsePrelogin([]byte{byte(PreloginEncryption), 0x00, 0x06, 0x00, 0x01}); err == nil {
		t.Fatal("expected missing-terminator error")
	}
}

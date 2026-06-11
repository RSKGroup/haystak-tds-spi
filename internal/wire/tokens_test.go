// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package wire

import "testing"

func TestLoginResponseStructure(t *testing.T) {
	toks, err := ParseTokens(BuildLoginResponse("haystak", "master"))
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 4 {
		t.Fatalf("tokens = %d, want 4 (ENVCHANGE, ENVCHANGE, LOGINACK, DONE)", len(toks))
	}
	if toks[0].Type != tokenEnvChange || toks[1].Type != tokenEnvChange {
		t.Errorf("want two ENVCHANGE, got %#x %#x", toks[0].Type, toks[1].Type)
	}
	if toks[2].Type != tokenLoginAck {
		t.Errorf("want LOGINACK, got %#x", toks[2].Type)
	}
	if toks[3].Type != tokenDone || len(toks[3].Data) != 12 {
		t.Errorf("want DONE with 12-byte body, got %#x len=%d", toks[3].Type, len(toks[3].Data))
	}
}

// TestLoginAckVersion locks the LOGINACK TDS version to real SQL Server's 74 00 00 04 (go-mssqldb reads it big-endian as verTDS74; FreeTDS accepts it).
func TestLoginAckVersion(t *testing.T) {
	toks, err := ParseTokens(BuildLoginResponse("haystak", "master"))
	if err != nil {
		t.Fatal(err)
	}
	v := toks[2].Data[1:5]
	if want := []byte{0x74, 0x00, 0x00, 0x04}; string(v) != string(want) {
		t.Fatalf("LOGINACK version = % x, want % x", v, want)
	}
}

func TestParseTokensUnknown(t *testing.T) {
	if _, err := ParseTokens([]byte{0x99}); err == nil {
		t.Fatal("expected unknown-token error")
	}
}

func TestParseTokensTruncated(t *testing.T) {
	if _, err := ParseTokens([]byte{tokenLoginAck, 0x0A, 0x00}); err == nil {
		t.Fatal("expected truncated-body error")
	}
}

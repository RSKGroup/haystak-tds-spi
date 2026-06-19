// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package wire

import "testing"

func TestLoginResponseStructure(t *testing.T) {
	toks, err := ParseTokens(BuildLoginResponse("haystak", "master"))
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 5 {
		t.Fatalf("tokens = %d, want 5 (ENVCHANGE db, ENVCHANGE collation, ENVCHANGE packet, LOGINACK, DONE)", len(toks))
	}
	for i := 0; i < 3; i++ {
		if toks[i].Type != tokenEnvChange {
			t.Errorf("token[%d] = %#x, want ENVCHANGE", i, toks[i].Type)
		}
	}
	if toks[1].Data[0] != envSQLCollation {
		t.Errorf("want collation ENVCHANGE at [1], got env type %d", toks[1].Data[0])
	}
	if toks[3].Type != tokenLoginAck {
		t.Errorf("want LOGINACK, got %#x", toks[3].Type)
	}
	if toks[4].Type != tokenDone || len(toks[4].Data) != 12 {
		t.Errorf("want DONE with 12-byte body, got %#x len=%d", toks[4].Type, len(toks[4].Data))
	}
}

// TestLoginAckVersion locks the LOGINACK TDS version to real SQL Server's 74 00 00 04 (go-mssqldb reads it big-endian as verTDS74; FreeTDS accepts it).
func TestLoginAckVersion(t *testing.T) {
	toks, err := ParseTokens(BuildLoginResponse("haystak", "master"))
	if err != nil {
		t.Fatal(err)
	}
	var ack *Token
	for i := range toks {
		if toks[i].Type == tokenLoginAck {
			ack = &toks[i]
			break
		}
	}
	if ack == nil {
		t.Fatal("no LOGINACK token")
	}
	v := ack.Data[1:5]
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

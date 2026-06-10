// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"encoding/binary"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

func TestBuildResultResponse(t *testing.T) {
	cols := []catalog.Column{
		{Name: "id", Type: types.Type{Kind: types.Int64}},
		{Name: "name", Type: types.Type{Kind: types.String, MaxLen: 128}},
	}
	rows := [][]any{{int64(1), "ada"}, {int64(2), "alan"}}
	out, err := BuildResultResponse(cols, rows)
	if err != nil {
		t.Fatal(err)
	}
	if out[0] != tokenColMetadata {
		t.Fatalf("first byte = %#x, want COLMETADATA", out[0])
	}
	tail := out[len(out)-13:]
	if tail[0] != tokenDone {
		t.Fatalf("stream should end with DONE, got %#x", tail[0])
	}
	if rc := binary.LittleEndian.Uint64(tail[5:13]); rc != 2 {
		t.Fatalf("DONE rowcount = %d, want 2", rc)
	}
}

func TestDecodeSQLBatch(t *testing.T) {
	body, _ := ucs2bytes("SELECT name FROM users")
	if got := DecodeSQLBatch(body); got != "SELECT name FROM users" {
		t.Fatalf("no-header: got %q", got)
	}
	hdr := make([]byte, 8)
	binary.LittleEndian.PutUint32(hdr[0:4], 8)
	if got := DecodeSQLBatch(append(hdr, body...)); got != "SELECT name FROM users" {
		t.Fatalf("with-header: got %q", got)
	}
}

func TestEncodeNullNVarchar(t *testing.T) {
	if got := encodeNVarchar(nil); len(got) != 2 || got[0] != 0xFF || got[1] != 0xFF {
		t.Fatalf("NULL nvarchar = %v, want FFFF", got)
	}
}

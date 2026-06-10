// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"encoding/binary"
	"testing"
)

func bExecSQL(stmt, decls string, args []any) []byte {
	names := declaredNames(decls)
	var b []byte
	b = append(b, 0xFF, 0xFF) // NameLenProcID = 0xFFFF → ProcID form
	b = append(b, 0x0A, 0x00) // ProcID 10 = sp_executesql
	b = append(b, 0x00, 0x00) // OptionFlags
	b = append(b, bNVarParam("", stmt)...)
	b = append(b, bNVarParam("", decls)...)
	for i, a := range args {
		name := ""
		if i < len(names) {
			name = names[i]
		}
		switch v := a.(type) {
		case int64:
			b = append(b, bIntParam(name, v)...)
		case string:
			b = append(b, bNVarParam(name, v)...)
		}
	}
	hdr := []byte{4, 0, 0, 0} // ALL_HEADERS total = 4 (no headers)
	return append(hdr, b...)
}

func bNVarParam(name, val string) []byte {
	var b []byte
	b = append(b, bVarchar(name)...)
	b = append(b, 0x00)                                    // status
	b = append(b, typeNVARCHAR, 0x40, 0x1F, 0, 0, 0, 0, 0) // NVARCHAR maxlen 8000 + collation
	return append(b, encodeNVarchar(val)...)
}

func bIntParam(name string, v int64) []byte {
	var b []byte
	b = append(b, bVarchar(name)...)
	b = append(b, 0x00)                 // status
	b = append(b, typeINTN, 0x08, 0x08) // INTN max 8, actual 8
	var v8 [8]byte
	binary.LittleEndian.PutUint64(v8[:], uint64(v))
	return append(b, v8[:]...)
}

func TestDecodeExecuteSQLInt(t *testing.T) {
	sql, ok := DecodeRPC(bExecSQL("SELECT name FROM users WHERE id = @p1", "@p1 int", []any{int64(2)}))
	if !ok {
		t.Fatal("DecodeRPC not ok")
	}
	if want := "SELECT name FROM users WHERE id = 2"; sql != want {
		t.Fatalf("got %q, want %q", sql, want)
	}
}

func TestDecodeExecuteSQLString(t *testing.T) {
	sql, ok := DecodeRPC(bExecSQL("SELECT id FROM users WHERE name = @n", "@n nvarchar(50)", []any{"alan"}))
	if !ok {
		t.Fatal("DecodeRPC not ok")
	}
	if want := "SELECT id FROM users WHERE name = 'alan'"; sql != want {
		t.Fatalf("got %q, want %q", sql, want)
	}
}

func TestDecodeRPCDeclines(t *testing.T) {
	if _, ok := DecodeRPC([]byte{0x01, 0x00}); ok {
		t.Fatal("expected ok=false for malformed/non-sp_executesql RPC")
	}
}

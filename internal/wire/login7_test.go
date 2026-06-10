// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

const login7HeaderLen = 94

func ucs2bytes(s string) ([]byte, int) {
	u := utf16.Encode([]rune(s))
	b := make([]byte, len(u)*2)
	for i, c := range u {
		binary.LittleEndian.PutUint16(b[i*2:], c)
	}
	return b, len(u)
}

func buildLogin7(host, user, pass, app, server, db string) []byte {
	hdr := make([]byte, login7HeaderLen)
	var data []byte
	put := func(pos int, s string, obfuscate bool) {
		b, cch := ucs2bytes(s)
		if obfuscate {
			for i, c := range b {
				b[i] = ((c << 4) | (c >> 4)) ^ 0xA5
			}
		}
		ib := login7HeaderLen + len(data)
		binary.LittleEndian.PutUint16(hdr[pos:], uint16(ib))
		binary.LittleEndian.PutUint16(hdr[pos+2:], uint16(cch))
		data = append(data, b...)
	}
	put(36, host, false)
	put(40, user, false)
	put(44, pass, true)
	put(48, app, false)
	put(52, server, false)
	put(68, db, false)
	payload := append(hdr, data...)
	binary.LittleEndian.PutUint32(payload[0:4], uint32(len(payload)))
	return payload
}

func TestParseLogin7(t *testing.T) {
	l, err := ParseLogin7(buildLogin7("myhost", "sa", "p@ss-W0rd", "SQLPro", "haystak", "master"))
	if err != nil {
		t.Fatal(err)
	}
	if l.HostName != "myhost" || l.UserName != "sa" || l.Password != "p@ss-W0rd" ||
		l.AppName != "SQLPro" || l.ServerName != "haystak" || l.Database != "master" {
		t.Fatalf("got %+v", l)
	}
}

func TestParseLogin7EmptyFields(t *testing.T) {
	l, err := ParseLogin7(buildLogin7("", "sa", "", "", "", ""))
	if err != nil {
		t.Fatal(err)
	}
	if l.UserName != "sa" || l.Password != "" || l.HostName != "" {
		t.Fatalf("got %+v", l)
	}
}

func TestParseLogin7TooShort(t *testing.T) {
	if _, err := ParseLogin7([]byte{1, 2, 3, 4}); err == nil {
		t.Fatal("expected too-short error")
	}
}

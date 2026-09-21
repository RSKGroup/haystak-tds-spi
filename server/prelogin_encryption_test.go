// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/internal/wire"
)

func preloginWith(enc wire.Encryption) []byte {
	return wire.BuildPrelogin([]wire.PreloginEntry{
		{Option: wire.PreloginVersion, Data: []byte{0x10, 0x00, 0x03, 0xE8, 0x00, 0x06}},
		{Option: wire.PreloginEncryption, Data: []byte{byte(enc)}},
	})
}

func TestNegotiateEncryption(t *testing.T) {
	cases := []struct {
		name           string
		tlsConfigured  bool
		allowPlaintext bool
		payload        []byte
		wantResp       wire.Encryption
		wantTLS        bool
		wantErr        bool
	}{
		// Defect 202: this exact shape sent a cleartext password on a TLS-configured listener.
		{"required refuses NOT_SUP", true, false, preloginWith(wire.EncryptNotSup), wire.EncryptReq, false, true},
		{"required accepts ON", true, false, preloginWith(wire.EncryptOn), wire.EncryptReq, true, false},
		{"required accepts OFF", true, false, preloginWith(wire.EncryptOff), wire.EncryptReq, true, false},
		{"required fails closed on garbage", true, false, []byte{0xFF}, wire.EncryptReq, false, true},

		// The opt-out keeps the old behaviour for a deployment that cannot move yet.
		{"allowed downgrades on NOT_SUP", true, true, preloginWith(wire.EncryptNotSup), wire.EncryptNotSup, false, false},
		{"allowed encrypts on ON", true, true, preloginWith(wire.EncryptOn), wire.EncryptOn, true, false},
		{"allowed tolerates garbage", true, true, []byte{0xFF}, wire.EncryptNotSup, false, false},

		// No TLS configured: nothing is promised, so nothing is refused.
		{"no tls ignores ON", false, false, preloginWith(wire.EncryptOn), wire.EncryptNotSup, false, false},
		{"no tls ignores NOT_SUP", false, false, preloginWith(wire.EncryptNotSup), wire.EncryptNotSup, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, useTLS, err := negotiateEncryption(tc.tlsConfigured, tc.allowPlaintext, tc.payload)
			if resp != tc.wantResp {
				t.Errorf("resp = 0x%02X, want 0x%02X", byte(resp), byte(tc.wantResp))
			}
			if useTLS != tc.wantTLS {
				t.Errorf("useTLS = %v, want %v", useTLS, tc.wantTLS)
			}
			if (err != nil) != tc.wantErr {
				t.Errorf("err = %v, want error: %v", err, tc.wantErr)
			}
		})
	}
}

// The zero-value Server must be the safe one: a struct literal that sets only TLSConfig requires
// encryption, which is the whole point of defect 202.
func TestZeroValueServerRequiresEncryption(t *testing.T) {
	s := &Server{}
	if s.AllowPlaintext {
		t.Fatal("AllowPlaintext defaults true; the zero value must refuse a downgrade")
	}
	resp, useTLS, err := negotiateEncryption(true, s.AllowPlaintext, preloginWith(wire.EncryptNotSup))
	if err == nil {
		t.Fatal("a TLS-configured zero-value Server accepted a cleartext session")
	}
	if useTLS {
		t.Error("refused session still reported useTLS")
	}
	if resp != wire.EncryptReq {
		t.Errorf("resp = 0x%02X, want ENCRYPT_REQ so the client learns why", byte(resp))
	}
}

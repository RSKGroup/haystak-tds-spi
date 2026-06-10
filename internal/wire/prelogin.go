// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"encoding/binary"
	"fmt"
)

type PreloginOption byte

const (
	PreloginVersion    PreloginOption = 0x00
	PreloginEncryption PreloginOption = 0x01
	PreloginInstOpt    PreloginOption = 0x02
	PreloginThreadID   PreloginOption = 0x03
	PreloginMARS       PreloginOption = 0x04
	preloginTerminator PreloginOption = 0xFF
)

type Encryption byte

const (
	EncryptOff    Encryption = 0x00
	EncryptOn     Encryption = 0x01
	EncryptNotSup Encryption = 0x02
	EncryptReq    Encryption = 0x03
)

type PreloginEntry struct {
	Option PreloginOption
	Data   []byte
}

type Prelogin struct {
	Options map[PreloginOption][]byte
}

func (p Prelogin) Encryption() (Encryption, bool) {
	v, ok := p.Options[PreloginEncryption]
	if !ok || len(v) < 1 {
		return 0, false
	}
	return Encryption(v[0]), true
}

// ParsePrelogin decodes a PRELOGIN payload: a table of 5-byte option entries
// (token, offset[BE], length[BE]) ended by a terminator, then the option data.
func ParsePrelogin(payload []byte) (Prelogin, error) {
	opts := map[PreloginOption][]byte{}
	i := 0
	for {
		if i >= len(payload) {
			return Prelogin{}, fmt.Errorf("wire: prelogin: missing terminator")
		}
		if PreloginOption(payload[i]) == preloginTerminator {
			return Prelogin{Options: opts}, nil
		}
		if i+5 > len(payload) {
			return Prelogin{}, fmt.Errorf("wire: prelogin: truncated option entry")
		}
		off := binary.BigEndian.Uint16(payload[i+1 : i+3])
		length := binary.BigEndian.Uint16(payload[i+3 : i+5])
		end := int(off) + int(length)
		if int(off) > len(payload) || end > len(payload) {
			return Prelogin{}, fmt.Errorf("wire: prelogin: option data out of range")
		}
		opts[PreloginOption(payload[i])] = payload[off:end]
		i += 5
	}
}

// BuildPrelogin encodes a PRELOGIN payload from entries in the given order.
func BuildPrelogin(entries []PreloginEntry) []byte {
	headerLen := len(entries)*5 + 1
	table := make([]byte, 0, headerLen)
	var data []byte
	for _, e := range entries {
		var ob [5]byte
		ob[0] = byte(e.Option)
		binary.BigEndian.PutUint16(ob[1:3], uint16(headerLen+len(data)))
		binary.BigEndian.PutUint16(ob[3:5], uint16(len(e.Data)))
		table = append(table, ob[:]...)
		data = append(data, e.Data...)
	}
	table = append(table, byte(preloginTerminator))
	return append(table, data...)
}

// ServerPreloginResponse builds a server PRELOGIN reply advertising the given encryption
// level (EncryptNotSup = plaintext; EncryptOn triggers the TLS handshake).
func ServerPreloginResponse(enc Encryption) []byte {
	return BuildPrelogin([]PreloginEntry{
		{PreloginVersion, []byte{0, 0, 0, 0, 0, 0}},
		{PreloginEncryption, []byte{byte(enc)}},
		{PreloginMARS, []byte{0}},
	})
}

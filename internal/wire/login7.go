// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

type Login7 struct {
	HostName   string
	UserName   string
	Password   string
	AppName    string
	ServerName string
	Database   string
}

// ParseLogin7 extracts the identity fields from a LOGIN7 payload. Strings are
// UCS-2/UTF-16LE; the password is de-obfuscated (XOR 0xA5 then nibble-swap).
func ParseLogin7(payload []byte) (Login7, error) {
	if len(payload) < 72 {
		return Login7{}, fmt.Errorf("wire: login7: payload too short (%d)", len(payload))
	}
	str := func(ibPos int) (string, error) {
		ib := int(binary.LittleEndian.Uint16(payload[ibPos : ibPos+2]))
		cch := int(binary.LittleEndian.Uint16(payload[ibPos+2 : ibPos+4]))
		if cch == 0 {
			return "", nil
		}
		end := ib + cch*2
		if end > len(payload) {
			return "", fmt.Errorf("wire: login7: field at %d out of range", ibPos)
		}
		return ucs2(payload[ib:end]), nil
	}

	var (
		l   Login7
		err error
	)
	if l.HostName, err = str(36); err != nil {
		return l, err
	}
	if l.UserName, err = str(40); err != nil {
		return l, err
	}
	pwIb := int(binary.LittleEndian.Uint16(payload[44:46]))
	pwCch := int(binary.LittleEndian.Uint16(payload[46:48]))
	if pwCch > 0 {
		end := pwIb + pwCch*2
		if end > len(payload) {
			return l, fmt.Errorf("wire: login7: password out of range")
		}
		l.Password = decodePassword(payload[pwIb:end])
	}
	if l.AppName, err = str(48); err != nil {
		return l, err
	}
	if l.ServerName, err = str(52); err != nil {
		return l, err
	}
	if l.Database, err = str(68); err != nil {
		return l, err
	}
	return l, nil
}

func ucs2(b []byte) string {
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	return string(utf16.Decode(u))
}

func decodePassword(b []byte) string {
	dec := make([]byte, len(b))
	for i, c := range b {
		x := c ^ 0xA5
		dec[i] = (x << 4) | (x >> 4)
	}
	return ucs2(dec)
}

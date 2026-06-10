// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

const (
	tokenColMetadata = 0x81
	tokenRow         = 0xD1

	typeINTN     = 0x26
	typeBITN     = 0x68
	typeFLTN     = 0x6D
	typeNVARCHAR = 0xE7

	typeGUID         = 0x24
	typeDECIMALN     = 0x6A
	typeDATETIME2    = 0x2A
	typeBIGVARBINARY = 0xA5
)

// DecodeSQLBatch extracts the UCS-2 query text from a SQL_BATCH payload, skipping the
// optional ALL_HEADERS block (TDS 7.2+).
func DecodeSQLBatch(payload []byte) string {
	body := payload
	if len(payload) >= 4 {
		total := int(binary.LittleEndian.Uint32(payload[0:4]))
		if total >= 4 && total <= len(payload) {
			body = payload[total:]
		}
	}
	return ucs2(body)
}

// BuildResultResponse builds COLMETADATA + ROWs + DONE(count) for a result set.
func BuildResultResponse(cols []catalog.Column, rows [][]any) ([]byte, error) {
	out := colMetadata(cols)
	for _, row := range rows {
		out = append(out, tokenRow)
		for i, c := range cols {
			out = append(out, encodeValue(c.Type, row[i])...)
		}
	}
	return append(out, done(DoneFinal|DoneCount, 0, uint64(len(rows)))...), nil
}

// BuildError builds an ERROR token + DONE(error) for a failed batch.
func BuildError(msg string) []byte {
	var data []byte
	data = append(data, 0, 0, 0, 0) // error number
	data = append(data, 1)          // state
	data = append(data, 16)         // severity/class
	data = append(data, usVarchar(msg)...)
	data = append(data, bVarchar("haystak")...) // server name
	data = append(data, bVarchar("")...)        // proc name
	data = append(data, 0, 0, 0, 0)             // line number
	out := mkToken(tokenError, data)
	return append(out, done(DoneFinal|0x0002, 0, 0)...)
}

// LoginError is a LOGIN7-phase rejection: a login-failed (18456, severity 14) ERROR token + DONE.
func LoginError(msg string) []byte {
	var data []byte
	data = append(data, 0x18, 0x48, 0x00, 0x00) // error number 18456
	data = append(data, 1)                      // state
	data = append(data, 14)                     // severity (login failed)
	data = append(data, usVarchar(msg)...)
	data = append(data, bVarchar("haystak")...) // server name
	data = append(data, bVarchar("")...)        // proc name
	data = append(data, 0, 0, 0, 0)             // line number
	out := mkToken(tokenError, data)
	return append(out, done(DoneFinal|0x0002, 0, 0)...)
}

func colMetadata(cols []catalog.Column) []byte {
	out := []byte{tokenColMetadata, 0, 0}
	binary.LittleEndian.PutUint16(out[1:3], uint16(len(cols)))
	for _, c := range cols {
		out = append(out, 0, 0, 0, 0) // UserType
		out = append(out, 0x01, 0x00) // Flags (nullable)
		out = append(out, typeInfo(c.Type)...)
		out = append(out, bVarchar(c.Name)...)
	}
	return out
}

func typeInfo(t types.Type) []byte {
	switch t.Kind {
	case types.Int64:
		return []byte{typeINTN, 8}
	case types.Int32:
		return []byte{typeINTN, 4}
	case types.Bool:
		return []byte{typeBITN, 1}
	case types.Float64:
		return []byte{typeFLTN, 8}
	case types.Decimal:
		prec := t.Precision
		if prec == 0 {
			prec = 18
		}
		return []byte{typeDECIMALN, decStorage(prec), byte(prec), byte(t.Scale)}
	case types.UUID:
		return []byte{typeGUID, 16}
	case types.Time:
		return []byte{typeDATETIME2, 0}
	case types.Bytes:
		maxBytes := 8000
		if t.MaxLen > 0 && t.MaxLen < maxBytes {
			maxBytes = t.MaxLen
		}
		out := []byte{typeBIGVARBINARY, 0, 0}
		binary.LittleEndian.PutUint16(out[1:3], uint16(maxBytes))
		return out
	default:
		maxBytes := 8000
		if t.MaxLen > 0 && t.MaxLen*2 < maxBytes {
			maxBytes = t.MaxLen * 2
		}
		out := []byte{typeNVARCHAR, 0, 0}
		binary.LittleEndian.PutUint16(out[1:3], uint16(maxBytes))
		return append(out, 0, 0, 0, 0, 0) // collation
	}
}

func encodeValue(t types.Type, v any) []byte {
	switch t.Kind {
	case types.Int64:
		return encodeIntN(v, 8)
	case types.Int32:
		return encodeIntN(v, 4)
	case types.Bool:
		if v == nil {
			return []byte{0}
		}
		b := byte(0)
		if x, ok := v.(bool); ok && x {
			b = 1
		}
		return []byte{1, b}
	case types.Float64:
		if v == nil {
			return []byte{0}
		}
		out := make([]byte, 9)
		out[0] = 8
		binary.LittleEndian.PutUint64(out[1:], math.Float64bits(toFloat(v)))
		return out
	case types.Decimal:
		return encodeDecimal(t, v)
	case types.UUID:
		return encodeGUID(v)
	case types.Time:
		return encodeDateTime2(v)
	case types.Bytes:
		return encodeVarbinary(v)
	default:
		return encodeNVarchar(v)
	}
}

func encodeVarbinary(v any) []byte {
	b, ok := v.([]byte)
	if !ok {
		return []byte{0xFF, 0xFF}
	}
	out := make([]byte, 2, 2+len(b))
	binary.LittleEndian.PutUint16(out[0:2], uint16(len(b)))
	return append(out, b...)
}

func decStorage(prec int) byte {
	switch {
	case prec <= 9:
		return 5
	case prec <= 19:
		return 9
	case prec <= 28:
		return 13
	default:
		return 17
	}
}

func encodeDecimal(t types.Type, v any) []byte {
	if v == nil {
		return []byte{0}
	}
	prec := t.Precision
	if prec == 0 {
		prec = 18
	}
	storage := decStorage(prec)
	magSize := int(storage) - 1
	f := toFloat(v)
	sign := byte(1)
	if f < 0 {
		sign = 0
		f = -f
	}
	mag := uint64(math.Round(f * math.Pow(10, float64(t.Scale))))
	out := make([]byte, 0, 1+1+magSize)
	out = append(out, storage, sign)
	var mb [16]byte
	binary.LittleEndian.PutUint64(mb[:8], mag)
	return append(out, mb[:magSize]...)
}

func encodeGUID(v any) []byte {
	if v == nil {
		return []byte{0}
	}
	g := guidBytes(fmt.Sprintf("%v", v))
	if g == nil {
		return []byte{0}
	}
	return append([]byte{16}, g...)
}

func guidBytes(s string) []byte {
	s = strings.ReplaceAll(s, "-", "")
	if len(s) != 32 {
		return nil
	}
	var b [16]byte
	for i := 0; i < 16; i++ {
		hi, ok1 := hexVal(s[i*2])
		lo, ok2 := hexVal(s[i*2+1])
		if !ok1 || !ok2 {
			return nil
		}
		b[i] = hi<<4 | lo
	}
	b[0], b[3] = b[3], b[0]
	b[1], b[2] = b[2], b[1]
	b[4], b[5] = b[5], b[4]
	b[6], b[7] = b[7], b[6]
	return b[:]
}

func hexVal(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

func encodeDateTime2(v any) []byte {
	t, ok := v.(time.Time)
	if !ok {
		return []byte{0}
	}
	t = t.UTC()
	secs := t.Hour()*3600 + t.Minute()*60 + t.Second()
	days := daysSince1(t.Year(), int(t.Month()), t.Day())
	return []byte{
		6,
		byte(secs), byte(secs >> 8), byte(secs >> 16),
		byte(days), byte(days >> 8), byte(days >> 16),
	}
}

func daysSince1(y, m, d int) int {
	a := (14 - m) / 12
	yy := y + 4800 - a
	mm := m + 12*a - 3
	jdn := d + (153*mm+2)/5 + 365*yy + yy/4 - yy/100 + yy/400 - 32045
	return jdn - 1721426
}

func encodeIntN(v any, size byte) []byte {
	if v == nil {
		return []byte{0}
	}
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], uint64(toInt64(v)))
	return append([]byte{size}, b[:size]...)
}

func encodeNVarchar(v any) []byte {
	if v == nil {
		return []byte{0xFF, 0xFF}
	}
	u := utf16.Encode([]rune(fmt.Sprintf("%v", v)))
	out := make([]byte, 2, 2+len(u)*2)
	binary.LittleEndian.PutUint16(out[0:2], uint16(len(u)*2))
	for _, c := range u {
		out = append(out, byte(c), byte(c>>8))
	}
	return out
}

func usVarchar(s string) []byte {
	u := utf16.Encode([]rune(s))
	out := make([]byte, 2, 2+len(u)*2)
	binary.LittleEndian.PutUint16(out[0:2], uint16(len(u)))
	for _, c := range u {
		out = append(out, byte(c), byte(c>>8))
	}
	return out
}

func toInt64(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case int32:
		return int64(x)
	case bool:
		if x {
			return 1
		}
	}
	return 0
}

func toFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int64:
		return float64(x)
	}
	return 0
}

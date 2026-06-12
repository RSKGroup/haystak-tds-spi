// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

const (
	tokenError      = 0xAA
	tokenInfo       = 0xAB
	tokenLoginAck   = 0xAD
	tokenEnvChange  = 0xE3
	tokenDone       = 0xFD
	tokenDoneProc   = 0xFE
	tokenDoneInProc = 0xFF
)

const (
	DoneFinal uint16 = 0x0000
	DoneMore  uint16 = 0x0001
	DoneCount uint16 = 0x0010
)

const (
	envDatabase   = 1
	envPacketSize = 4
)

// LOGINACK TDS 7.4: the bytes real SQL Server sends; go-mssqldb reads them big-endian as verTDS74 (0x74000004).
var tdsVersion74 = [4]byte{0x74, 0x00, 0x00, 0x04}

// LOGINACK server build 16.0.1000 (SQL 2022); 0.0.0.0 makes ODBC Driver 18 reject as "SQL 2000 or earlier".
var progVersion = [4]byte{0x10, 0x00, 0x03, 0xE8}

type Token struct {
	Type byte
	Data []byte
}

// BuildLoginResponse assembles the server's login-acceptance token stream:
// ENVCHANGE(database) + ENVCHANGE(packet size) + LOGINACK + DONE(final).
func BuildLoginResponse(serverName, database string) []byte {
	var b []byte
	b = append(b, envChange(envDatabase, database, "")...)
	b = append(b, envChange(envPacketSize, "4096", "4096")...)
	b = append(b, loginAck(serverName)...)
	b = append(b, done(DoneFinal, 0, 0)...)
	return b
}

func loginAck(serverName string) []byte {
	var data []byte
	data = append(data, 0x01)
	data = append(data, tdsVersion74[:]...)
	data = append(data, bVarchar(serverName)...)
	data = append(data, progVersion[:]...)
	return mkToken(tokenLoginAck, data)
}

func envChange(envType byte, newVal, oldVal string) []byte {
	var data []byte
	data = append(data, envType)
	data = append(data, bVarchar(newVal)...)
	data = append(data, bVarchar(oldVal)...)
	return mkToken(tokenEnvChange, data)
}

func done(status, curCmd uint16, rowCount uint64) []byte {
	out := make([]byte, 13)
	out[0] = tokenDone
	binary.LittleEndian.PutUint16(out[1:3], status)
	binary.LittleEndian.PutUint16(out[3:5], curCmd)
	binary.LittleEndian.PutUint64(out[5:13], rowCount)
	return out
}

// EmptyDone is a standalone DONE(final) response for a command with no result set.
func EmptyDone() []byte { return done(DoneFinal, 0, 0) }

// DoneWithCount is the DONE(final, count) response to a write/DDL with rows-affected set.
func DoneWithCount(n uint64) []byte { return done(DoneFinal|DoneCount, 0, n) }

// EnvChangeDatabase is the ENVCHANGE(database) token, to lead a response after USE changes the db.
func EnvChangeDatabase(db string) []byte { return envChange(envDatabase, db, "") }

// DatabaseChange is the standalone USE response: ENVCHANGE(database) + DONE(final).
func DatabaseChange(db string) []byte { return append(EnvChangeDatabase(db), done(DoneFinal, 0, 0)...) }

func mkToken(typ byte, data []byte) []byte {
	out := make([]byte, 3, 3+len(data))
	out[0] = typ
	binary.LittleEndian.PutUint16(out[1:3], uint16(len(data)))
	return append(out, data...)
}

func bVarchar(s string) []byte {
	u := utf16.Encode([]rune(s))
	out := make([]byte, 1+len(u)*2)
	out[0] = byte(len(u))
	for i, c := range u {
		binary.LittleEndian.PutUint16(out[1+i*2:], c)
	}
	return out
}

// ParseTokens walks a response token stream (the length-prefixed tokens we emit + DONE).
func ParseTokens(b []byte) ([]Token, error) {
	var toks []Token
	i := 0
	for i < len(b) {
		typ := b[i]
		i++
		switch typ {
		case tokenDone, tokenDoneProc, tokenDoneInProc:
			if i+12 > len(b) {
				return nil, fmt.Errorf("wire: token: truncated DONE")
			}
			toks = append(toks, Token{typ, b[i : i+12]})
			i += 12
		case tokenLoginAck, tokenEnvChange, tokenInfo, tokenError:
			if i+2 > len(b) {
				return nil, fmt.Errorf("wire: token: truncated length")
			}
			n := int(binary.LittleEndian.Uint16(b[i : i+2]))
			i += 2
			if i+n > len(b) {
				return nil, fmt.Errorf("wire: token: truncated body")
			}
			toks = append(toks, Token{typ, b[i : i+n]})
			i += n
		default:
			return nil, fmt.Errorf("wire: token: unknown token 0x%02X", typ)
		}
	}
	return toks, nil
}

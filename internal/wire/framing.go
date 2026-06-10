// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"encoding/binary"
	"fmt"
	"io"
)

type PacketType byte

const (
	PacketSQLBatch  PacketType = 0x01
	PacketRPC       PacketType = 0x03
	PacketResponse  PacketType = 0x04
	PacketAttention PacketType = 0x06
	PacketLogin7    PacketType = 0x10
	PacketPreLogin  PacketType = 0x12
)

const (
	headerSize        = 8
	statusEOM         = 0x01
	DefaultPacketSize = 4096
)

type Message struct {
	Type    PacketType
	Payload []byte
}

// ReadMessage reads TDS packets until the EOM status bit and returns the assembled
// message. A message may span multiple packets; the type is taken from the first.
func ReadMessage(r io.Reader) (Message, error) {
	var (
		typ     PacketType
		payload []byte
		first   = true
	)
	for {
		var hdr [headerSize]byte
		if _, err := io.ReadFull(r, hdr[:]); err != nil {
			return Message{}, err
		}
		length := binary.BigEndian.Uint16(hdr[2:4])
		if length < headerSize {
			return Message{}, fmt.Errorf("wire: bad packet length %d", length)
		}
		if first {
			typ = PacketType(hdr[0])
			first = false
		}
		body := make([]byte, int(length)-headerSize)
		if _, err := io.ReadFull(r, body); err != nil {
			return Message{}, err
		}
		payload = append(payload, body...)
		if hdr[1]&statusEOM != 0 {
			return Message{Type: typ, Payload: payload}, nil
		}
	}
}

// WriteMessage writes a message, splitting the payload across packets of packetSize
// (header + body). The final packet carries the EOM status bit.
func WriteMessage(w io.Writer, m Message, packetSize int) error {
	if packetSize <= headerSize {
		packetSize = DefaultPacketSize
	}
	maxBody := packetSize - headerSize
	payload := m.Payload
	pid := byte(1)
	for {
		chunk := payload
		last := true
		if len(chunk) > maxBody {
			chunk, last = chunk[:maxBody], false
		}
		var hdr [headerSize]byte
		hdr[0] = byte(m.Type)
		if last {
			hdr[1] = statusEOM
		}
		binary.BigEndian.PutUint16(hdr[2:4], uint16(headerSize+len(chunk)))
		hdr[6] = pid
		if _, err := w.Write(hdr[:]); err != nil {
			return err
		}
		if _, err := w.Write(chunk); err != nil {
			return err
		}
		if last {
			return nil
		}
		payload = payload[maxBody:]
		pid++
	}
}

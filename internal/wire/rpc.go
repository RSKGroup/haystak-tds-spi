// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"encoding/binary"
	"math"
	"sort"
	"strconv"
	"strings"
)

const procIDExecuteSQL = 10

type rpcParam struct {
	name string
	val  any
}

// DecodeRPC parses an RPC request and, for sp_executesql, returns the parameter-expanded SQL
// (values substituted as literals). Returns ok=false for any RPC form we don't expand.
func DecodeRPC(payload []byte) (string, bool) {
	body := payload
	if len(payload) >= 4 {
		total := int(binary.LittleEndian.Uint32(payload[0:4]))
		if total >= 4 && total <= len(payload) {
			body = payload[total:]
		}
	}
	r := &cur{b: body}
	nameLen, ok := r.u16()
	if !ok {
		return "", false
	}
	proc := ""
	if nameLen == 0xFFFF {
		pid, ok := r.u16()
		if !ok {
			return "", false
		}
		if pid == procIDExecuteSQL {
			proc = "sp_executesql"
		}
	} else {
		proc = r.ucs2n(int(nameLen))
	}
	if !strings.EqualFold(proc, "sp_executesql") {
		return "", false
	}
	if _, ok := r.u16(); !ok { // OptionFlags
		return "", false
	}
	var params []rpcParam
	for r.remaining() > 0 {
		p, ok := r.param()
		if !ok {
			break
		}
		params = append(params, p)
	}
	if len(params) == 0 {
		return "", false
	}
	stmt, _ := params[0].val.(string)
	if stmt == "" {
		return "", false
	}
	decls := ""
	if len(params) >= 2 {
		decls, _ = params[1].val.(string)
	}
	return expandParams(stmt, decls, params[2:]), true
}

type cur struct {
	b   []byte
	pos int
}

func (c *cur) remaining() int { return len(c.b) - c.pos }

func (c *cur) byte() (byte, bool) {
	if c.pos >= len(c.b) {
		return 0, false
	}
	v := c.b[c.pos]
	c.pos++
	return v, true
}

func (c *cur) u16() (uint16, bool) {
	if c.pos+2 > len(c.b) {
		return 0, false
	}
	v := binary.LittleEndian.Uint16(c.b[c.pos:])
	c.pos += 2
	return v, true
}

func (c *cur) take(n int) ([]byte, bool) {
	if n < 0 || c.pos+n > len(c.b) {
		return nil, false
	}
	v := c.b[c.pos : c.pos+n]
	c.pos += n
	return v, true
}

func (c *cur) ucs2n(chars int) string {
	b, ok := c.take(chars * 2)
	if !ok {
		return ""
	}
	return ucs2(b)
}

func (c *cur) param() (rpcParam, bool) {
	nlen, ok := c.byte()
	if !ok {
		return rpcParam{}, false
	}
	name := c.ucs2n(int(nlen))
	if _, ok := c.byte(); !ok { // status flags
		return rpcParam{}, false
	}
	val, ok := c.typeValue()
	if !ok {
		return rpcParam{}, false
	}
	return rpcParam{name: name, val: val}, true
}

// typeValue reads a TYPE_INFO + value, returning a Go value (nil for NULL).
func (c *cur) typeValue() (any, bool) {
	tok, ok := c.byte()
	if !ok {
		return nil, false
	}
	switch tok {
	case typeINTN:
		return c.varInt()
	case typeBITN:
		if _, ok := c.byte(); !ok {
			return nil, false
		}
		raw, ok := c.lenPrefixed1()
		if !ok {
			return nil, false
		}
		if raw == nil {
			return nil, true
		}
		return raw[0] != 0, true
	case typeFLTN:
		if _, ok := c.byte(); !ok {
			return nil, false
		}
		raw, ok := c.lenPrefixed1()
		if !ok {
			return nil, false
		}
		switch len(raw) {
		case 8:
			return math.Float64frombits(binary.LittleEndian.Uint64(raw)), true
		case 4:
			return float64(math.Float32frombits(binary.LittleEndian.Uint32(raw))), true
		}
		return nil, true
	case typeNVARCHAR:
		maxlen, ok := c.u16()
		if !ok {
			return nil, false
		}
		if _, ok := c.take(5); !ok { // collation
			return nil, false
		}
		if maxlen == 0xFFFF {
			return c.plpString()
		}
		vlen, ok := c.u16()
		if !ok {
			return nil, false
		}
		if vlen == 0xFFFF {
			return nil, true
		}
		raw, ok := c.take(int(vlen))
		if !ok {
			return nil, false
		}
		return ucs2(raw), true
	}
	return nil, false
}

func (c *cur) varInt() (any, bool) {
	if _, ok := c.byte(); !ok { // declared max len
		return nil, false
	}
	raw, ok := c.lenPrefixed1()
	if !ok {
		return nil, false
	}
	if raw == nil {
		return nil, true
	}
	return leInt(raw), true
}

// lenPrefixed1 reads a 1-byte length then that many bytes; len 0 → nil (NULL).
func (c *cur) lenPrefixed1() ([]byte, bool) {
	n, ok := c.byte()
	if !ok {
		return nil, false
	}
	if n == 0 {
		return nil, true
	}
	return c.take(int(n))
}

func (c *cur) plpString() (any, bool) {
	hdr, ok := c.take(8)
	if !ok {
		return nil, false
	}
	if binary.LittleEndian.Uint64(hdr) == 0xFFFFFFFFFFFFFFFF { // PLP_NULL
		return nil, true
	}
	var buf []byte
	for {
		lb, ok := c.take(4)
		if !ok {
			return nil, false
		}
		n := binary.LittleEndian.Uint32(lb)
		if n == 0 {
			break
		}
		chunk, ok := c.take(int(n))
		if !ok {
			return nil, false
		}
		buf = append(buf, chunk...)
	}
	return ucs2(buf), true
}

func leInt(b []byte) int64 {
	var u uint64
	for i := len(b) - 1; i >= 0; i-- {
		u = u<<8 | uint64(b[i])
	}
	bits := len(b) * 8
	if bits < 64 && u&(1<<uint(bits-1)) != 0 {
		u |= ^uint64(0) << uint(bits)
	}
	return int64(u)
}

func expandParams(stmt, decls string, values []rpcParam) string {
	names := declaredNames(decls)
	repl := map[string]any{}
	for i, p := range values {
		key := p.name
		if key == "" && i < len(names) {
			key = names[i]
		}
		if key != "" {
			repl[key] = p.val
		}
	}
	keys := make([]string, 0, len(repl))
	for k := range repl {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	out := stmt
	for _, k := range keys {
		out = strings.ReplaceAll(out, k, sqlLiteral(repl[k]))
	}
	return out
}

func declaredNames(decls string) []string {
	var names []string
	for _, part := range strings.Split(decls, ",") {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) > 0 && strings.HasPrefix(fields[0], "@") {
			names = append(names, fields[0])
		}
	}
	return names
}

func sqlLiteral(v any) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case string:
		return "'" + strings.ReplaceAll(x, "'", "''") + "'"
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case bool:
		if x {
			return "1"
		}
		return "0"
	}
	return "NULL"
}

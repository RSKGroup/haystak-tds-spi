// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package functions

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash/fnv"
	"io"
	"strings"
)

func init() {
	register("HASHBYTES", func(a []any) any {
		if len(a) < 2 {
			return nil
		}
		data := hashBytesArg(a[1])
		switch strings.ToUpper(argStr(a, 0)) {
		case "MD5":
			h := md5.Sum(data)
			return h[:]
		case "SHA1", "SHA":
			h := sha1.Sum(data)
			return h[:]
		case "SHA2_256":
			h := sha256.Sum256(data)
			return h[:]
		case "SHA2_512":
			h := sha512.Sum512(data)
			return h[:]
		}
		return nil
	})
	// CHECKSUM/BINARY_CHECKSUM are a deterministic 32-bit hash for change detection -- stable, not
	// bit-identical to SQL Server's undocumented algorithm.
	register("CHECKSUM", checksum)
	register("BINARY_CHECKSUM", checksum)
	register("COMPRESS", func(a []any) any {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		if _, err := w.Write(hashBytesArg(a[0])); err != nil {
			return nil
		}
		if err := w.Close(); err != nil {
			return nil
		}
		return buf.Bytes()
	})
	register("DECOMPRESS", func(a []any) any {
		r, err := gzip.NewReader(bytes.NewReader(hashBytesArg(a[0])))
		if err != nil {
			return nil
		}
		out, err := io.ReadAll(r)
		if err != nil {
			return nil
		}
		return out
	})
	// PWDENCRYPT/PWDCOMPARE are a self-consistent SHA-256 hash, not SQL Server's internal salted format.
	register("PWDENCRYPT", func(a []any) any {
		h := sha256.Sum256([]byte(argStr(a, 0)))
		return h[:]
	})
	register("PWDCOMPARE", func(a []any) any {
		h := sha256.Sum256([]byte(argStr(a, 0)))
		if hb, ok := a[1].([]byte); ok && bytes.Equal(h[:], hb) {
			return int64(1)
		}
		return int64(0)
	})
}

func hashBytesArg(v any) []byte {
	switch x := v.(type) {
	case []byte:
		return x
	case string:
		return []byte(x)
	case nil:
		return nil
	}
	return []byte(fmt.Sprint(v))
}

func checksum(a []any) any {
	h := fnv.New32a()
	for _, v := range a {
		if v == nil {
			continue
		}
		_, _ = h.Write([]byte(fmt.Sprint(v)))
	}
	return int64(int32(h.Sum32()))
}

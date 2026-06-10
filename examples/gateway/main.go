// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/server"
)

func main() {
	addr := "127.0.0.1:1433"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}
	gw := &server.Server{Backend: inmem.New(), Logf: log.Printf}
	if creds := os.Getenv("HAYSTAK_AUTH"); creds != "" {
		gw.Auth = server.StaticAuth(parseCreds(creds))
		log.Printf("authentication enabled")
	}
	if os.Getenv("HAYSTAK_TLS") != "" {
		cfg, err := selfSignedTLS()
		if err != nil {
			log.Fatal(err)
		}
		gw.TLSConfig = cfg
		log.Printf("TLS enabled (self-signed)")
	}
	log.Printf("haystak-tds-spi gateway (in-mem demo) on %s", addr)
	log.Fatal(gw.ListenAndServe(addr))
}

// parseCreds reads "user:pass,user2:pass2" from HAYSTAK_AUTH into a credential map.
func parseCreds(s string) map[string]string {
	m := map[string]string{}
	for _, pair := range strings.Split(s, ",") {
		if u, p, ok := strings.Cut(strings.TrimSpace(pair), ":"); ok {
			m[u] = p
		}
	}
	return m
}

func selfSignedTLS() (*tls.Config, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "haystak-tds-spi"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}
	cert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: priv}
	return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
}

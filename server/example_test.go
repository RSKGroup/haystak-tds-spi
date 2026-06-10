// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package server_test

import (
	"context"
	"fmt"
	"log"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/server"
	"github.com/RSKGroup/haystak-tds-spi/tds"
)

// Server adds TLS, authentication, and naming on top of a backend. Set TLSConfig to enable
// TLS-in-TDS and Auth (e.g. StaticAuth) to require credentials.
func ExampleServer() {
	gw := &server.Server{
		Backend:    inmem.New(),
		Auth:       server.StaticAuth(map[string]string{"sa": "secret"}),
		ServerName: "haystak",
		Database:   "master",
		Logf:       log.Printf,
	}
	log.Fatal(gw.ListenAndServe("127.0.0.1:1433"))
}

// StaticAuth is a username→password Authenticator for demos and tests. A wrong password is rejected
// with a login-failed error, which the gateway turns into TDS error 18456 on the wire.
func ExampleStaticAuth() {
	auth := server.StaticAuth(map[string]string{"sa": "secret"})

	ok, _ := auth.Authenticate(context.Background(), tds.Login{Username: "sa", Password: "secret"})
	_, err := auth.Authenticate(context.Background(), tds.Login{Username: "sa", Password: "wrong"})
	fmt.Println(ok.Username, "/", err)
	// Output: sa / login failed for user "sa"
}

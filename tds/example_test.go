// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tds_test

import (
	"context"
	"fmt"
	"log"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/server"
	"github.com/RSKGroup/haystak-tds-spi/tds"
)

// Serve any tds.Backend over the SQL Server wire. inmem is the reference Scanner backend; swap it
// for your own, then connect with sqlcmd, SSMS, Power BI, or any go-mssqldb/.NET/JDBC client.
func Example() {
	if err := server.ListenAndServe("127.0.0.1:1433", inmem.New()); err != nil {
		log.Fatal(err)
	}
}

// A backend owns authentication via tds.Authenticator (here AuthFunc): the gateway hands it each
// LOGIN7 login and enforces the go/no-go, then rides the returned Principal in ctx for per-user authz.
func ExampleAuthFunc() {
	auth := tds.AuthFunc(func(ctx context.Context, l tds.Login) (tds.Principal, error) {
		if l.Username == "ada" && l.Password == "lovelace" {
			return tds.Principal{Username: "ada", Roles: []string{"reader"}}, nil
		}
		return tds.Principal{}, fmt.Errorf("login failed for user %q", l.Username)
	})

	p, err := auth.Authenticate(context.Background(), tds.Login{Username: "ada", Password: "lovelace"})
	fmt.Println(p.Username, p.Roles, err)
	// Output: ada [reader] <nil>
}

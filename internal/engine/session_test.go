// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/tds"
)

func TestSessionUseScopesDatabase(t *testing.T) {
	sess := NewSession(nil, "master")
	rs, _, env, err := sess.Exec(context.Background(), "USE [content]; SELECT DB_NAME()")
	if err != nil {
		t.Fatal(err)
	}
	if env != "content" || sess.Database() != "content" {
		t.Fatalf("env=%q sessDB=%q, want content", env, sess.Database())
	}
	rs.Next()
	v, _ := rs.Values()
	rs.Close()
	if len(v) == 0 || v[0] != "content" {
		t.Fatalf("DB_NAME() = %v, want content", v)
	}
}

func TestApplyDefaultDB(t *testing.T) {
	cases := []struct {
		q    tds.Query
		want string
	}{
		{tds.Query{Table: "patients"}, "syn"},                          // unqualified → defaulted
		{tds.Query{Table: "patients", Database: "other"}, "other"},     // already qualified → untouched
		{tds.Query{Table: "tables", Schema: "INFORMATION_SCHEMA"}, ""}, // system schema → untouched
		{tds.Query{Table: "databases", Schema: "sys"}, ""},             // system schema → untouched
	}
	for _, c := range cases {
		q := c.q
		applyDefaultDB(&q, "syn")
		if q.Database != c.want {
			t.Errorf("table=%q schema=%q: db=%q, want %q", c.q.Table, c.q.Schema, q.Database, c.want)
		}
	}
}

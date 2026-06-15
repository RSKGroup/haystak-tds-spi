// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

func TestSelectAssign(t *testing.T) {
	if got := cell(qry(t, "DECLARE @n VARCHAR(50); SELECT @n = name FROM users WHERE id = 2; SELECT @n AS r")[0][0]); got != "alan" {
		t.Errorf("SELECT @n = name = %q, want alan", got)
	}
}

func TestSelectAssignMulti(t *testing.T) {
	r := qry(t, "DECLARE @a INT; DECLARE @c INT; SELECT @a = id, @c = amount FROM orders WHERE id = 11; SELECT @a AS a, @c AS c")[0]
	if cell(r[0]) != "11" || cell(r[1]) != "200" {
		t.Errorf("multi-assign = %v, want 11/200", r)
	}
}

func TestSelectAssignAggregate(t *testing.T) {
	if got := cell(qry(t, "DECLARE @t INT; SELECT @t = COUNT(*) FROM orders; SELECT @t AS total")[0][0]); got != "3" {
		t.Errorf("SELECT @t = COUNT(*) = %q, want 3", got)
	}
}

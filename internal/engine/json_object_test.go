// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

func TestJSONObject(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"SELECT JSON_OBJECT('a':1, 'b':'x')", `{"a":1,"b":"x"}`},
		{"SELECT JSON_OBJECT('n':2)", `{"n":2}`},
		// categories(category_id=1) has parent_id NULL -> dropped (ABSENT ON NULL).
		{"SELECT JSON_OBJECT('id':category_id, 'p':parent_id) FROM categories WHERE category_id = 1", `{"id":1}`},
	}
	for _, c := range cases {
		got := qry(t, c.sql)
		if len(got) != 1 || cell(got[0][0]) != c.want {
			t.Errorf("%s = %v, want %s", c.sql, got, c.want)
		}
	}
}

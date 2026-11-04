// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/examples/inmem"
	"github.com/RSKGroup/haystak-tds-spi/internal/engine"
	"github.com/RSKGroup/haystak-tds-spi/tds"
)

func TestLikeEscape(t *testing.T) {
	cases := []struct {
		pred string
		want int
	}{
		{"'50%' LIKE '50!%' ESCAPE '!'", 1},
		{"'500' LIKE '50!%' ESCAPE '!'", 0},
		{"'a_b' LIKE 'a!_b' ESCAPE '!'", 1},
		{"'axb' LIKE 'a!_b' ESCAPE '!'", 0},
		{"'a[b' LIKE 'a![b' ESCAPE '!'", 1},
		{"'a!b' LIKE 'a!!b' ESCAPE '!'", 1},
		{"'50%' NOT LIKE '50!%' ESCAPE '!'", 0},
		{"'500' NOT LIKE '50!%' ESCAPE '!'", 1},
		{"'^' LIKE '[!^a]' ESCAPE '!'", 1},
		{"'b' LIKE '[!^a]' ESCAPE '!'", 0},
		{"'-' LIKE '[a!-c]' ESCAPE '!'", 1},
		{"'b' LIKE '[a!-c]' ESCAPE '!'", 0},
		{"']' LIKE '[!]]' ESCAPE '!'", 1},
		{"'ab' LIKE 'ab!' ESCAPE '!'", 0},
		{"'AB%' LIKE 'ab#%' ESCAPE '#'", 1},
		{"'x10%y' LIKE '%10\\%%' ESCAPE '\\'", 1},
	}
	for _, c := range cases {
		sql := "SELECT CASE WHEN " + c.pred + " THEN 1 ELSE 0 END"
		if got := fmt.Sprint(qry(t, sql)); got != fmt.Sprintf("[[%d]]", c.want) {
			t.Errorf("%s = %s, want %d", c.pred, got, c.want)
		}
	}
	if got := fmt.Sprint(qry(t, "SELECT name FROM users WHERE name + '%' LIKE LEFT(name, 3) + '!%' ESCAPE '!'")); got != "[[ada]]" {
		t.Errorf("expression pattern with ESCAPE = %s, want [[ada]]", got)
	}
	if _, err := engine.Query(context.Background(), inmem.New(), "SELECT 1 WHERE 'a' LIKE 'a' ESCAPE 'ab'"); err == nil {
		t.Errorf("a two-character ESCAPE was accepted")
	}
}

func TestLikeEscapeIsNotPushedAsText(t *testing.T) {
	b := &recordingBackend{Backend: inmem.New()}
	rs, err := engine.Query(context.Background(), b, "SELECT name FROM users WHERE name LIKE 'a!%' ESCAPE '!'")
	if err != nil {
		t.Fatal(err)
	}
	if got := collect(t, rs); len(got) != 0 {
		t.Errorf("rows = %v, want none (no name is a%%)", got)
	}
	for _, w := range b.wheres {
		if w != nil && w.Pred != nil && w.Pred.Op == tds.OpLike {
			if _, isText := w.Pred.Value.(string); isText {
				t.Errorf("an ESCAPE pattern reached the backend as pushable text %q", w.Pred.Value)
			}
		}
	}
}

package tsql

import (
	"strings"
	"testing"
)

// A DELETE or UPDATE whose WHERE was not parsed became a whole-table write: tds/dml.go documents
// an empty Where as "all rows". An unsupported clause must ERROR, never widen the write.

func writeCases() []struct {
	sql        string
	wantErr    bool
	wantPreds  int
	whyItBites string
} {
	return []struct {
		sql        string
		wantErr    bool
		wantPreds  int
		whyItBites string
	}{
		{"DELETE FROM orders o WHERE o.id = 10", true, 0, "a table alias is idiomatic T-SQL and wiped the table"},
		{"DELETE FROM orders AS o WHERE o.id = 10", true, 0, "same, with AS"},
		{"DELETE FROM orders OUTPUT deleted.id WHERE id = 10", true, 0, "OUTPUT is not in the keyword table"},
		{"DELETE FROM orders WHERE id = 1 OR id = 2", true, 0, "a top-level OR was truncated to its first predicate"},
		{"UPDATE orders SET amount = 1 OUTPUT inserted.id WHERE id = 10", true, 0, "OUTPUT on an update"},
		{"UPDATE orders SET amount = 1 FROM orders o JOIN users u ON u.id = o.user_id WHERE u.id = 9", true, 0, "UPDATE...FROM"},

		// Hints are documented as supported no-ops, so these must PARSE and keep their predicate.
		{"DELETE FROM orders WITH (ROWLOCK) WHERE id = 10", false, 1, ""},
		{"UPDATE orders WITH (ROWLOCK) SET amount = 1 WHERE id = 10", false, 1, ""},

		// Controls: these worked before and must still work.
		{"DELETE FROM orders WHERE id = 10", false, 1, ""},
		{"DELETE FROM orders WHERE a = 1 AND b = 2", false, 2, ""},
		{"UPDATE orders SET amount = 1 WHERE id = 10", false, 1, ""},
		// A deliberate whole-table delete is still legal.
		{"DELETE FROM orders", false, 0, ""},
	}
}

func TestUnsupportedWriteClausesErrorRatherThanWidening(t *testing.T) {
	for _, c := range writeCases() {
		st, ok, err := ParseWrite(c.sql)
		if !ok {
			t.Errorf("%q was not recognised as a write", c.sql)
			continue
		}
		if c.wantErr {
			if err == nil {
				t.Errorf("%q parsed with no error and would run as a whole-table write — %s", c.sql, c.whyItBites)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q errored unexpectedly: %v", c.sql, err)
			continue
		}
		var got int
		switch {
		case st.Delete != nil:
			got = len(st.Delete.Where)
		case st.Update != nil:
			got = len(st.Update.Where)
		}
		if got != c.wantPreds {
			t.Errorf("%q kept %d predicates, want %d", c.sql, got, c.wantPreds)
		}
	}
}

// The refusal must say the form is unsupported, not leave the caller guessing.
func TestWideningRefusalIsLegible(t *testing.T) {
	_, _, err := ParseWrite("DELETE FROM orders o WHERE o.id = 10")
	if err == nil {
		t.Fatal("no error")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("refusal does not say the form is unsupported: %v", err)
	}
}

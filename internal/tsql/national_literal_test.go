package tsql

import "testing"

// N'...' is SQL Server's national (Unicode) string literal. SSMS uses it
// throughout its connect-time metadata queries (SERVERPROPERTY(N'Edition'), ...);
// the lexer must treat the N prefix as part of the string, not a separate ident.
func TestParseNationalStringLiteral(t *testing.T) {
	for _, sql := range []string{
		"SELECT SERVERPROPERTY(N'Edition')",
		"SELECT SERVERPROPERTY(N'EngineEdition')",
		"SELECT name FROM t WHERE x = N'abc'",
		"SELECT n'lower'",        // lowercase prefix
		"SELECT 'N' FROM t",      // bare 'N' string is unaffected
		"SELECT Name FROM Nodes", // identifier starting with N is unaffected
	} {
		if _, err := Parse(sql); err != nil {
			t.Errorf("Parse(%q) = %v", sql, err)
		}
	}
}

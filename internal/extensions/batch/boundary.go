// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package batch

import "strings"

// leadWords open a statement; at paren depth 0 none of them can continue an expression.
var leadWords = map[string]bool{
	"SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true, "MERGE": true, "DECLARE": true,
	"SET": true, "WITH": true, "EXEC": true, "EXECUTE": true, "IF": true, "WHILE": true, "BEGIN": true,
	"PRINT": true, "RETURN": true, "USE": true, "CREATE": true, "ALTER": true, "DROP": true,
	"TRUNCATE": true, "RAISERROR": true, "THROW": true, "GOTO": true, "WAITFOR": true,
	"BREAK": true, "CONTINUE": true,
}

type word struct {
	text       string
	at, end    int
	depth      int
	parenAfter bool
}

// words lists s's identifiers and keywords with their paren depth, skipping strings and comments.
func words(s string) []word {
	var out []word
	depth := 0
	for i := 0; i < len(s); {
		if j := SkipComment(s, i); j > i {
			i = j
			continue
		}
		c := s[i]
		j := i + 1
		switch {
		case c == '\'':
			j = skipString(s, i)
		case c == '[':
			j = skipBracket(s, i)
			out = append(out, word{text: s[i:j], at: i, end: j, depth: depth})
		case c == '(':
			depth++
			if n := len(out); n > 0 && strings.TrimSpace(s[out[n-1].end:i]) == "" {
				out[n-1].parenAfter = true
			}
		case c == ')':
			if depth > 0 {
				depth--
			}
		case isWordByte(c):
			for j < len(s) && isWordByte(s[j]) {
				j++
			}
			out = append(out, word{text: s[i:j], at: i, end: j, depth: depth})
		}
		i = j
	}
	return out
}

func isWordByte(c byte) bool { return isNamePart(c) || c == '@' || c == '#' || c == '$' || c == '.' }

// cutStatements splits text that T-SQL reads as several statements with no ';' between them.
func cutStatements(s string) []string {
	var out []string
	for s = TrimLeadingComments(s); s != ""; s = TrimLeadingComments(s) {
		end := statementEnd(s)
		out = append(out, strings.TrimSpace(s[:end]))
		s = s[end:]
	}
	return out
}

// statementEnd is where the statement that opens s stops, judged by the first word of the next one.
func statementEnd(s string) int {
	ws := words(s)
	if len(ws) == 0 {
		return len(s)
	}
	lead := strings.ToUpper(ws[0].text)
	values, ownSet := false, false
	for k := 1; k < len(ws); k++ {
		w := ws[k]
		if w.depth != 0 {
			continue
		}
		u := strings.ToUpper(w.text)
		hint := u == "WITH" && w.parenAfter // a table hint, WITH (NOLOCK), not a new statement
		switch lead {
		case "CREATE", "ALTER":
			return len(s) // a routine body runs to the end of its batch
		case "USE":
			return w.end
		case "DECLARE", "SET", "EXEC", "EXECUTE", "PRINT", "DROP", "TRUNCATE", "RAISERROR", "THROW", "WAITFOR":
			if u == "CURSOR" && lead == "DECLARE" {
				return len(s)
			}
			if leadWords[u] {
				return w.at
			}
		case "WITH":
			if u == "SELECT" || u == "INSERT" || u == "UPDATE" || u == "DELETE" || u == "MERGE" {
				lead = u // the CTE's statement: its words follow that statement's rules
			}
		case "SELECT":
			switch {
			case u == "SELECT":
				if !setOpWord(ws[k-1].text) {
					return w.at
				}
			case u == "WITH":
				if !hint && (k+1 >= len(ws) || !strings.EqualFold(ws[k+1].text, "TIES")) {
					return w.at
				}
			case leadWords[u]:
				return w.at
			}
		case "UPDATE", "DELETE":
			switch {
			case u == "SET" && lead == "UPDATE" && !ownSet:
				ownSet = true // an UPDATE has one SET clause; a second SET starts a statement
			case hint:
			case leadWords[u]:
				return w.at
			}
		case "MERGE":
			switch {
			case u == "SET" && strings.EqualFold(ws[k-1].text, "UPDATE"):
			case u == "INSERT" || u == "UPDATE" || u == "DELETE" || hint: // its WHEN ... THEN actions
			case leadWords[u]:
				return w.at
			}
		case "INSERT":
			values = values || u == "VALUES"
			switch {
			case u == "SELECT" && !values: // INSERT ... SELECT is one statement, INSERT ... VALUES (...) then SELECT two
			case u == "EXEC" || u == "EXECUTE" || hint:
			case leadWords[u]:
				return w.at
			}
		default:
			if u == "DECLARE" || u == "USE" || u == "SET" {
				return w.at
			}
		}
	}
	return len(s)
}

func setOpWord(s string) bool {
	switch strings.ToUpper(s) {
	case "UNION", "ALL", "EXCEPT", "INTERSECT":
		return true
	}
	return false
}

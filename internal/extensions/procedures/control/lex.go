// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package control

import "strings"

// tokKind classifies a control-batch token. Raw SQL is reassembled from token text, so the lexer keeps
// strings and [bracketed] names intact and never splits inside them.
type tokKind int

const (
	tWord   tokKind = iota // identifier or keyword
	tString                // '…' literal (text includes the quotes)
	tParenL
	tParenR
	tSemi
	tOther // any other run of punctuation/operators (=, >, +, etc.)
)

type tok struct {
	kind tokKind
	text string
}

func lex(s string) []tok {
	var out []tok
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			i++
		case c == '\'' || ((c == 'N' || c == 'n') && i+1 < len(s) && s[i+1] == '\''):
			start := i
			if c != '\'' {
				i++ // keep the national-literal N prefix with the string
			}
			j := i + 1
			for j < len(s) {
				if s[j] == '\'' {
					if j+1 < len(s) && s[j+1] == '\'' {
						j += 2
						continue
					}
					j++
					break
				}
				j++
			}
			out = append(out, tok{tString, s[start:j]})
			i = j
		case c == '[':
			j := i + 1
			for j < len(s) && s[j] != ']' {
				j++
			}
			if j < len(s) {
				j++
			}
			out = append(out, tok{tWord, s[i:j]})
			i = j
		case c == '(':
			out = append(out, tok{tParenL, "("})
			i++
		case c == ')':
			out = append(out, tok{tParenR, ")"})
			i++
		case c == ';':
			out = append(out, tok{tSemi, ";"})
			i++
		case isNamePart(c) || c == '@' || c == '#':
			j := i
			for j < len(s) && (isNamePart(s[j]) || s[j] == '@' || s[j] == '#' || s[j] == '.') {
				j++
			}
			out = append(out, tok{tWord, s[i:j]})
			i = j
		default:
			j := i
			for j < len(s) && isOther(s[j]) {
				j++
			}
			if j == i {
				j++
			}
			out = append(out, tok{tOther, s[i:j]})
			i = j
		}
	}
	return out
}

func isNamePart(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isOther(c byte) bool {
	switch c {
	case ' ', '\t', '\r', '\n', '\'', '[', '(', ')', ';', '@', '#':
		return false
	}
	return !isNamePart(c)
}

// stmtKeywords lead a statement; at paren depth 0 they bound a preceding raw-SQL run or condition.
var stmtKeywords = map[string]bool{
	"BEGIN": true, "END": true, "IF": true, "ELSE": true, "WHILE": true,
	"BREAK": true, "CONTINUE": true, "RETURN": true, "PRINT": true, "THROW": true,
	"RAISERROR": true, "GOTO": true, "WAITFOR": true, "DECLARE": true, "SET": true,
	"SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true, "EXEC": true,
	"EXECUTE": true, "WITH": true, "MERGE": true, "TRUNCATE": true, "CREATE": true,
	"DROP": true, "ALTER": true, "USE": true, "TRY": true, "CATCH": true,
}

func isStmtKeyword(t tok) bool {
	return t.kind == tWord && stmtKeywords[strings.ToUpper(t.text)]
}

func eqKw(t tok, kw string) bool {
	return t.kind == tWord && strings.EqualFold(t.text, kw)
}

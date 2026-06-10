// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tsql

import (
	"fmt"
	"strings"
)

type tokKind int

const (
	tEOF tokKind = iota
	tIdent
	tKeyword
	tNumber
	tString
	tStar
	tComma
	tDot
	tLParen
	tRParen
	tOp
)

type token struct {
	kind tokKind
	text string
}

var keywords = map[string]bool{
	"SELECT": true, "WITH": true, "DISTINCT": true, "TOP": true, "PERCENT": true, "FROM": true, "WHERE": true,
	"GROUP": true, "HAVING": true, "ORDER": true, "BY": true, "ASC": true, "DESC": true, "AS": true,
	"AND": true, "OR": true, "NOT": true, "IN": true, "LIKE": true, "EXISTS": true,
	"IS": true, "NULL": true, "BETWEEN": true,
	"JOIN": true, "INNER": true, "LEFT": true, "RIGHT": true, "FULL": true, "OUTER": true, "CROSS": true, "ON": true,
	"OFFSET": true, "FETCH": true, "NEXT": true, "FIRST": true, "ROWS": true, "ROW": true, "ONLY": true,
	"CASE": true, "WHEN": true, "THEN": true, "ELSE": true, "END": true,
	"UNION": true, "ALL": true, "INTERSECT": true, "EXCEPT": true,
	"INSERT": true, "INTO": true, "VALUES": true, "UPDATE": true, "SET": true,
	"DELETE": true, "CREATE": true, "DROP": true, "TABLE": true, "DATABASE": true,
	"ALTER": true, "ADD": true, "COLUMN": true,
}

func lex(s string) ([]token, error) {
	var toks []token
	i, n := 0, len(s)
	for i < n {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '*':
			toks = append(toks, token{tStar, "*"})
			i++
		case c == ',':
			toks = append(toks, token{tComma, ","})
			i++
		case c == '.':
			toks = append(toks, token{tDot, "."})
			i++
		case c == '(':
			toks = append(toks, token{tLParen, "("})
			i++
		case c == ')':
			toks = append(toks, token{tRParen, ")"})
			i++
		case c == '=':
			toks = append(toks, token{tOp, "="})
			i++
		case c == '+' || c == '-' || c == '/' || c == '%':
			toks = append(toks, token{tOp, string(c)})
			i++
		case c == '<':
			switch {
			case i+1 < n && s[i+1] == '=':
				toks = append(toks, token{tOp, "<="})
				i += 2
			case i+1 < n && s[i+1] == '>':
				toks = append(toks, token{tOp, "<>"})
				i += 2
			default:
				toks = append(toks, token{tOp, "<"})
				i++
			}
		case c == '>':
			if i+1 < n && s[i+1] == '=' {
				toks = append(toks, token{tOp, ">="})
				i += 2
			} else {
				toks = append(toks, token{tOp, ">"})
				i++
			}
		case c == '\'':
			j := i + 1
			var b strings.Builder
			for j < n {
				if s[j] == '\'' {
					if j+1 < n && s[j+1] == '\'' {
						b.WriteByte('\'')
						j += 2
						continue
					}
					break
				}
				b.WriteByte(s[j])
				j++
			}
			if j >= n {
				return nil, fmt.Errorf("tsql: unterminated string literal")
			}
			toks = append(toks, token{tString, b.String()})
			i = j + 1
		case c == '[':
			j := i + 1
			for j < n && s[j] != ']' {
				j++
			}
			if j >= n {
				return nil, fmt.Errorf("tsql: unterminated [identifier]")
			}
			toks = append(toks, token{tIdent, s[i+1 : j]})
			i = j + 1
		case c == '"':
			j := i + 1
			for j < n && s[j] != '"' {
				j++
			}
			if j >= n {
				return nil, fmt.Errorf("tsql: unterminated quoted identifier")
			}
			toks = append(toks, token{tIdent, s[i+1 : j]})
			i = j + 1
		case isDigit(c):
			j := i
			for j < n && (isDigit(s[j]) || s[j] == '.') {
				j++
			}
			toks = append(toks, token{tNumber, s[i:j]})
			i = j
		case isIdentStart(c):
			j := i
			for j < n && isIdentPart(s[j]) {
				j++
			}
			word := s[i:j]
			if keywords[strings.ToUpper(word)] {
				toks = append(toks, token{tKeyword, strings.ToUpper(word)})
			} else {
				toks = append(toks, token{tIdent, word})
			}
			i = j
		default:
			return nil, fmt.Errorf("tsql: unexpected character %q", string(c))
		}
	}
	toks = append(toks, token{tEOF, ""})
	return toks, nil
}

func isDigit(c byte) bool      { return c >= '0' && c <= '9' }
func isIdentStart(c byte) bool { return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }
func isIdentPart(c byte) bool  { return isIdentStart(c) || isDigit(c) }

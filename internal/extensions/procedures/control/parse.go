// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package control

import (
	"fmt"
	"strings"
)

// Stmt is one node of a parsed control batch.
type Stmt interface{ isStmt() }

type Block struct{ Stmts []Stmt }
type Raw struct{ SQL string }
type Declare struct{ Vars []declVar }
type Set struct{ Name, Expr string }
type Print struct{ Expr string }
type Return struct{}
type Break struct{}
type Continue struct{}
type If struct {
	Cond       string
	Then, Else Stmt
}
type While struct {
	Cond string
	Body Stmt
}

// Try is BEGIN TRY … END TRY BEGIN CATCH … END CATCH (wired in the error-model wave).
type Try struct{ Body, Catch Stmt }

// Throw is THROW [n, msg, state]; Raiserror is RAISERROR(msg, sev, state) (error-model wave).
type Throw struct{ Args string }
type Raiserror struct{ Args string }

type declVar struct{ name, init string }

func (*Block) isStmt()     {}
func (*Raw) isStmt()       {}
func (*Declare) isStmt()   {}
func (*Set) isStmt()       {}
func (*Print) isStmt()     {}
func (*Return) isStmt()    {}
func (*Break) isStmt()     {}
func (*Continue) isStmt()  {}
func (*If) isStmt()        {}
func (*While) isStmt()     {}
func (*Try) isStmt()       {}
func (*Throw) isStmt()     {}
func (*Raiserror) isStmt() {}

type parser struct {
	toks []tok
	pos  int
}

func parse(sql string) (*Block, error) {
	p := &parser{toks: lex(sql)}
	stmts, err := p.stmtList(nil)
	if err != nil {
		return nil, err
	}
	if !p.eof() {
		return nil, fmt.Errorf("control: unexpected %q", p.toks[p.pos].text)
	}
	return &Block{stmts}, nil
}

func (p *parser) eof() bool { return p.pos >= len(p.toks) }
func (p *parser) peek() tok { return p.toks[p.pos] }
func (p *parser) advance() {
	p.pos++
}

func (p *parser) skipSemis() {
	for !p.eof() && p.peek().kind == tSemi {
		p.advance()
	}
}

// stmtList reads statements until EOF or one of the terminator keywords (END / ELSE / CATCH / TRY).
func (p *parser) stmtList(terms map[string]bool) ([]Stmt, error) {
	var out []Stmt
	for {
		p.skipSemis()
		if p.eof() {
			break
		}
		t := p.peek()
		if terms != nil && t.kind == tWord && terms[strings.ToUpper(t.text)] {
			break
		}
		s, err := p.stmt()
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (p *parser) stmt() (Stmt, error) {
	t := p.peek()
	if t.kind == tWord {
		switch strings.ToUpper(t.text) {
		case "BEGIN":
			if p.pos+1 < len(p.toks) && eqKw(p.toks[p.pos+1], "TRY") {
				return p.tryStmt()
			}
			return p.block()
		case "IF":
			return p.ifStmt()
		case "WHILE":
			return p.whileStmt()
		case "DECLARE":
			return p.declare()
		case "SET":
			return p.setStmt()
		case "PRINT":
			p.advance()
			return &Print{Expr: p.readToStmtBound()}, nil
		case "THROW":
			p.advance()
			return &Throw{Args: p.readToStmtBound()}, nil
		case "RAISERROR":
			p.advance()
			return &Raiserror{Args: p.readToStmtBound()}, nil
		case "RETURN":
			p.advance()
			p.readToStmtBound()
			return &Return{}, nil
		case "BREAK":
			p.advance()
			p.readToStmtBound()
			return &Break{}, nil
		case "CONTINUE":
			p.advance()
			p.readToStmtBound()
			return &Continue{}, nil
		}
	}
	return &Raw{SQL: p.rawStmt()}, nil
}

func (p *parser) block() (Stmt, error) {
	p.advance() // BEGIN
	stmts, err := p.stmtList(map[string]bool{"END": true})
	if err != nil {
		return nil, err
	}
	if p.eof() || !eqKw(p.peek(), "END") {
		return nil, fmt.Errorf("control: expected END to close BEGIN")
	}
	p.advance() // END
	return &Block{Stmts: stmts}, nil
}

func (p *parser) ifStmt() (Stmt, error) {
	p.advance() // IF
	cond := p.readCond()
	then, err := p.stmt()
	if err != nil {
		return nil, err
	}
	out := &If{Cond: cond, Then: then}
	save := p.pos
	p.skipSemis()
	if !p.eof() && eqKw(p.peek(), "ELSE") {
		p.advance()
		els, err := p.stmt()
		if err != nil {
			return nil, err
		}
		out.Else = els
	} else {
		p.pos = save
	}
	return out, nil
}

func (p *parser) whileStmt() (Stmt, error) {
	p.advance() // WHILE
	cond := p.readCond()
	body, err := p.stmt()
	if err != nil {
		return nil, err
	}
	return &While{Cond: cond, Body: body}, nil
}

// tryStmt parses BEGIN TRY … END TRY BEGIN CATCH … END CATCH.
func (p *parser) tryStmt() (Stmt, error) {
	p.advance() // BEGIN
	p.advance() // TRY
	body, err := p.stmtList(map[string]bool{"END": true})
	if err != nil {
		return nil, err
	}
	if err := p.expectEndKw("TRY"); err != nil {
		return nil, err
	}
	p.skipSemis()
	if p.eof() || !eqKw(p.peek(), "BEGIN") || p.pos+1 >= len(p.toks) || !eqKw(p.toks[p.pos+1], "CATCH") {
		return nil, fmt.Errorf("control: expected BEGIN CATCH after END TRY")
	}
	p.advance() // BEGIN
	p.advance() // CATCH
	catch, err := p.stmtList(map[string]bool{"END": true})
	if err != nil {
		return nil, err
	}
	if err := p.expectEndKw("CATCH"); err != nil {
		return nil, err
	}
	return &Try{Body: &Block{body}, Catch: &Block{catch}}, nil
}

func (p *parser) expectEndKw(kw string) error {
	if p.eof() || !eqKw(p.peek(), "END") {
		return fmt.Errorf("control: expected END %s", kw)
	}
	p.advance() // END
	if p.eof() || !eqKw(p.peek(), kw) {
		return fmt.Errorf("control: expected END %s", kw)
	}
	p.advance() // TRY/CATCH
	return nil
}

func (p *parser) declare() (Stmt, error) {
	p.advance() // DECLARE
	body := p.readToStmtBound()
	var vars []declVar
	for _, part := range splitTopCommas(body) {
		part = strings.TrimSpace(part)
		name := firstWord(part)
		if !strings.HasPrefix(name, "@") {
			return nil, fmt.Errorf("control: DECLARE expects @name, got %q", part)
		}
		v := declVar{name: strings.ToLower(strings.TrimPrefix(name, "@"))}
		if eq := topEq(part); eq >= 0 {
			v.init = strings.TrimSpace(part[eq+1:])
		}
		vars = append(vars, v)
	}
	return &Declare{Vars: vars}, nil
}

func (p *parser) setStmt() (Stmt, error) {
	p.advance() // SET
	body := p.readToStmtBound()
	eq := topEq(body)
	name := strings.TrimSpace(body)
	if eq >= 0 {
		name = strings.TrimSpace(body[:eq])
	}
	if !strings.HasPrefix(name, "@") {
		return &Raw{SQL: "SET " + body}, nil // SET NOCOUNT ON etc.: pass through to the engine
	}
	if eq < 0 {
		return nil, fmt.Errorf("control: SET %s expects '='", name)
	}
	return &Set{Name: strings.ToLower(strings.TrimPrefix(name, "@")), Expr: strings.TrimSpace(body[eq+1:])}, nil
}

// readCond reads a condition: tokens until a statement keyword at paren depth 0.
func (p *parser) readCond() string {
	var ts []tok
	depth := 0
	for !p.eof() {
		t := p.peek()
		switch t.kind {
		case tParenL:
			depth++
		case tParenR:
			if depth > 0 {
				depth--
			}
		case tSemi:
			p.advance()
			return render(ts)
		}
		if depth == 0 && isStmtKeyword(t) {
			break
		}
		ts = append(ts, t)
		p.advance()
	}
	return render(ts)
}

// readToStmtBound reads the rest of a simple statement: until ';' (consumed) or a statement keyword
// at paren depth 0 (left for the next statement).
func (p *parser) readToStmtBound() string {
	var ts []tok
	depth := 0
	for !p.eof() {
		t := p.peek()
		if t.kind == tSemi {
			p.advance()
			break
		}
		if t.kind == tParenL {
			depth++
		}
		if t.kind == tParenR && depth > 0 {
			depth--
		}
		if depth == 0 && isStmtKeyword(t) {
			break
		}
		ts = append(ts, t)
		p.advance()
	}
	return render(ts)
}

// rawStmt reads a plain SQL statement: its leading keyword, then until ';' or the next statement keyword.
func (p *parser) rawStmt() string {
	var ts []tok
	depth := 0
	first := true
	for !p.eof() {
		t := p.peek()
		if t.kind == tSemi {
			p.advance()
			break
		}
		if t.kind == tParenL {
			depth++
		}
		if t.kind == tParenR && depth > 0 {
			depth--
		}
		if !first && depth == 0 && isStmtKeyword(t) {
			break
		}
		ts = append(ts, t)
		p.advance()
		first = false
	}
	return render(ts)
}

func render(ts []tok) string {
	parts := make([]string, len(ts))
	for i, t := range ts {
		parts[i] = t.text
	}
	return strings.Join(parts, " ")
}

func firstWord(s string) string {
	s = strings.TrimSpace(s)
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			return s[:i]
		}
	}
	return s
}

// topEq / splitTopCommas operate at paren depth 0, skipping '…' strings.
func topEq(s string) int {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'':
			i = skipStr(s, i)
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case '=':
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func splitTopCommas(s string) []string {
	var out []string
	start, depth := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'':
			i = skipStr(s, i)
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	return append(out, s[start:])
}

func skipStr(s string, i int) int {
	for i++; i < len(s); i++ {
		if s[i] == '\'' {
			return i
		}
	}
	return i
}

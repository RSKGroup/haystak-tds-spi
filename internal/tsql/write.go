// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tsql

import (
	"fmt"
	"strings"

	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

// ParseWrite parses INSERT/UPDATE/DELETE/CREATE/DROP. ok=false means it is not a write statement
// (the caller should try the SELECT path); ok=true with a non-nil error means a malformed write.
func ParseWrite(sql string) (*tds.WriteStmt, bool, error) {
	toks, err := lex(sql)
	if err != nil {
		return nil, false, nil
	}
	p := &parser{toks: toks}
	switch {
	case p.isKeyword("INSERT"):
		ins, err := p.parseInsert()
		return &tds.WriteStmt{Insert: ins}, true, err
	case p.isKeyword("UPDATE"):
		up, err := p.parseUpdate()
		return &tds.WriteStmt{Update: up}, true, err
	case p.isKeyword("DELETE"):
		del, err := p.parseDelete()
		return &tds.WriteStmt{Delete: del}, true, err
	case p.isKeyword("CREATE"):
		p.next()
		if p.isKeyword("TABLE") {
			p.next()
			t, err := p.parseTableDef()
			return &tds.WriteStmt{CreateTable: t}, true, err
		}
		if p.isKeyword("DATABASE") {
			p.next()
			name, _ := p.qualifiedName()
			return &tds.WriteStmt{CreateDB: name}, true, nil
		}
		return nil, true, fmt.Errorf("tsql: expected TABLE or DATABASE after CREATE")
	case p.isKeyword("DROP"):
		p.next()
		if p.isKeyword("TABLE") {
			p.next()
			_, _, tbl, err := p.tableName()
			return &tds.WriteStmt{DropTable: tbl}, true, err
		}
		if p.isKeyword("DATABASE") {
			p.next()
			name, _ := p.qualifiedName()
			return &tds.WriteStmt{DropDB: name}, true, nil
		}
		return nil, true, fmt.Errorf("tsql: expected TABLE or DATABASE after DROP")
	case p.isKeyword("ALTER"):
		p.next()
		if err := p.expectKeyword("TABLE"); err != nil {
			return nil, true, err
		}
		a, err := p.parseAlter()
		return &tds.WriteStmt{Alter: a}, true, err
	}
	return nil, false, nil
}

func (p *parser) parseInsert() (*tds.Insert, error) {
	p.next() // INSERT
	if err := p.expectKeyword("INTO"); err != nil {
		return nil, err
	}
	db, sch, tbl, err := p.tableName()
	if err != nil {
		return nil, err
	}
	ins := &tds.Insert{Database: db, Schema: sch, Table: tbl}
	if p.peek().kind == tLParen {
		p.next()
		cols, err := p.identList()
		if err != nil {
			return nil, err
		}
		ins.Columns = cols
		if p.peek().kind != tRParen {
			return nil, fmt.Errorf("tsql: expected ')' after INSERT columns, got %q", p.peek().text)
		}
		p.next()
	}
	if err := p.expectKeyword("VALUES"); err != nil {
		return nil, err
	}
	for {
		if p.peek().kind != tLParen {
			return nil, fmt.Errorf("tsql: expected '(' in VALUES, got %q", p.peek().text)
		}
		p.next()
		vals, err := p.literalList()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tRParen {
			return nil, fmt.Errorf("tsql: expected ')' after VALUES row, got %q", p.peek().text)
		}
		p.next()
		ins.Rows = append(ins.Rows, vals)
		if p.peek().kind == tComma {
			p.next()
			continue
		}
		break
	}
	return ins, nil
}

func (p *parser) parseUpdate() (*tds.Update, error) {
	p.next() // UPDATE
	db, sch, tbl, err := p.tableName()
	if err != nil {
		return nil, err
	}
	up := &tds.Update{Database: db, Schema: sch, Table: tbl}
	if err := p.expectKeyword("SET"); err != nil {
		return nil, err
	}
	for {
		col, ok := p.qualifiedName()
		if !ok {
			return nil, fmt.Errorf("tsql: expected column in SET, got %q", p.peek().text)
		}
		if p.peek().kind != tOp || p.peek().text != "=" {
			return nil, fmt.Errorf("tsql: expected '=' in SET, got %q", p.peek().text)
		}
		p.next()
		val, err := p.literal()
		if err != nil {
			return nil, err
		}
		up.Assignments = append(up.Assignments, tds.Assignment{Column: col, Value: val})
		if p.peek().kind == tComma {
			p.next()
			continue
		}
		break
	}
	if p.isKeyword("WHERE") {
		p.next()
		preds, err := p.simpleWhere()
		if err != nil {
			return nil, err
		}
		up.Where = preds
	}
	return up, nil
}

func (p *parser) parseDelete() (*tds.Delete, error) {
	p.next() // DELETE
	if err := p.expectKeyword("FROM"); err != nil {
		return nil, err
	}
	db, sch, tbl, err := p.tableName()
	if err != nil {
		return nil, err
	}
	del := &tds.Delete{Database: db, Schema: sch, Table: tbl}
	if p.isKeyword("WHERE") {
		p.next()
		preds, err := p.simpleWhere()
		if err != nil {
			return nil, err
		}
		del.Where = preds
	}
	return del, nil
}

func (p *parser) parseTableDef() (*catalog.Table, error) {
	_, _, tbl, err := p.tableName()
	if err != nil {
		return nil, err
	}
	t := &catalog.Table{Name: tbl}
	if p.peek().kind != tLParen {
		return nil, fmt.Errorf("tsql: expected '(' in CREATE TABLE, got %q", p.peek().text)
	}
	p.next()
	for {
		col, ok := p.qualifiedName()
		if !ok {
			return nil, fmt.Errorf("tsql: expected column name, got %q", p.peek().text)
		}
		typ, err := p.typeName()
		if err != nil {
			return nil, err
		}
		t.Columns = append(t.Columns, catalog.Column{Name: col, Type: sqlTypeToKind(typ)})
		if p.peek().kind == tComma {
			p.next()
			continue
		}
		break
	}
	if p.peek().kind != tRParen {
		return nil, fmt.Errorf("tsql: expected ')' after columns, got %q", p.peek().text)
	}
	p.next()
	return t, nil
}

func (p *parser) parseAlter() (*tds.AlterTable, error) {
	_, _, tbl, err := p.tableName()
	if err != nil {
		return nil, err
	}
	a := &tds.AlterTable{Table: tbl}
	switch {
	case p.isKeyword("ADD"):
		p.next()
		if p.isKeyword("COLUMN") {
			p.next()
		}
		for {
			col, ok := p.qualifiedName()
			if !ok {
				return nil, fmt.Errorf("tsql: expected column name in ALTER TABLE ADD, got %q", p.peek().text)
			}
			typ, err := p.typeName()
			if err != nil {
				return nil, err
			}
			a.AddColumns = append(a.AddColumns, catalog.Column{Name: col, Type: sqlTypeToKind(typ)})
			if p.peek().kind == tComma {
				p.next()
				continue
			}
			break
		}
	case p.isKeyword("DROP"):
		p.next()
		if p.isKeyword("COLUMN") {
			p.next()
		}
		for {
			col, ok := p.qualifiedName()
			if !ok {
				return nil, fmt.Errorf("tsql: expected column name in ALTER TABLE DROP, got %q", p.peek().text)
			}
			a.DropColumns = append(a.DropColumns, col)
			if p.peek().kind == tComma {
				p.next()
				continue
			}
			break
		}
	default:
		return nil, fmt.Errorf("tsql: expected ADD or DROP after ALTER TABLE, got %q", p.peek().text)
	}
	return a, nil
}

func (p *parser) simpleWhere() ([]tds.Predicate, error) {
	var preds []tds.Predicate
	for {
		col, ok := p.qualifiedName()
		if !ok {
			return nil, fmt.Errorf("tsql: expected column in WHERE, got %q", p.peek().text)
		}
		opTok := p.peek()
		if opTok.kind != tOp {
			return nil, fmt.Errorf("tsql: expected operator in WHERE, got %q", opTok.text)
		}
		p.next()
		op, err := mapOp(opTok.text)
		if err != nil {
			return nil, err
		}
		val, err := p.literal()
		if err != nil {
			return nil, err
		}
		preds = append(preds, tds.Predicate{Column: col, Op: op, Value: val})
		if p.isKeyword("AND") {
			p.next()
			continue
		}
		break
	}
	return preds, nil
}

func sqlTypeToKind(t string) types.Type {
	switch strings.ToUpper(t) {
	case "INT", "INTEGER", "SMALLINT", "TINYINT":
		return types.Type{Kind: types.Int32}
	case "BIGINT":
		return types.Type{Kind: types.Int64}
	case "BIT":
		return types.Type{Kind: types.Bool}
	case "FLOAT", "REAL":
		return types.Type{Kind: types.Float64}
	case "DECIMAL", "NUMERIC", "MONEY":
		return types.Type{Kind: types.Decimal, Precision: 18, Scale: 2}
	case "DATE", "DATETIME", "DATETIME2", "TIME":
		return types.Type{Kind: types.Time}
	case "UNIQUEIDENTIFIER":
		return types.Type{Kind: types.UUID}
	case "VARBINARY", "BINARY":
		return types.Type{Kind: types.Bytes}
	}
	return types.Type{Kind: types.String, MaxLen: 255}
}

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tds

// Op is a predicate comparison or membership operator.
type Op int

const (
	OpEq Op = iota
	OpNe
	OpLt
	OpLe
	OpGt
	OpGe
	OpIn   // Value is []any
	OpLike // Value is a string pattern (% and _)
	OpIsNull
	OpIsNotNull
	OpExists // Sub returns ≥1 row
)

// Predicate is one comparison in a WHERE expression: Column (or LeftExpr) Op Value.
type Predicate struct {
	Column   string
	LeftExpr *ValueExpr // optional: left side is an expression (nil → use Column)
	Op       Op
	Value    any
	Sub      *Query // IN (subquery): resolved to a value list before exec
}

// ColRef marks a Predicate value as a reference to another column (col = col).
type ColRef struct{ Name string }

// Expr is a boolean WHERE expression tree; exactly one field is set.
type Expr struct {
	And   []*Expr
	Or    []*Expr
	Not   *Expr
	Pred  *Predicate
	Const *bool // resolved constant (e.g. EXISTS folded before exec)
}

// AggFunc identifies an aggregate in a SelectItem (AggNone for a plain column).
type AggFunc int

const (
	AggNone AggFunc = iota
	AggCount
	AggSum
	AggAvg
	AggMin
	AggMax
)

// SelectItem is one entry in the select list: a plain column, an aggregate, or an expression.
type SelectItem struct {
	Column string     // plain column reference
	Agg    AggFunc    // aggregate function (AggNone for a plain column)
	Arg    string     // aggregate argument column ("*" for COUNT(*))
	Expr   *ValueExpr // scalar expression (nil unless this item is computed)
	Alias  string     // output column name (optional)
}

// ValueKind tags the variant of a ValueExpr scalar expression.
type ValueKind int

const (
	ValCol ValueKind = iota
	ValLit
	ValBinary
	ValFunc
	ValCase
	ValCast
	ValSubquery
)

// ValueExpr is a scalar select-list expression: column, literal, arithmetic, function,
// CASE, or CAST.
type ValueExpr struct {
	Kind    ValueKind
	Col     string
	Lit     any
	Op      string // ValBinary: + - * / %
	Left    *ValueExpr
	Right   *ValueExpr
	Func    string // ValFunc: UPPER, LEN, ISNULL, ...
	Args    []*ValueExpr
	Operand *ValueExpr // ValCase: simple-CASE operand (nil for searched CASE)
	Whens   []CaseWhen // ValCase
	Else    *ValueExpr // ValCase
	Cast    string     // ValCast: target type name (INT, VARCHAR, ...)
	Sub     *Query     // ValSubquery: scalar subquery, resolved to a literal before exec
}

// CaseWhen is one WHEN arm of a CASE expression.
type CaseWhen struct {
	Cond   *Expr      // searched CASE: boolean condition
	Match  *ValueExpr // simple CASE: value compared to the operand
	Result *ValueExpr
}

// OrderItem is one ORDER BY term: a column, a 1-based select-list ordinal, or an expression
// (e.g. an aggregate like COUNT(*)), ascending unless Desc.
type OrderItem struct {
	Column  string
	Ordinal int        // 1-based select-list position (ORDER BY n); 0 = use Column/Expr
	Expr    *ValueExpr // set when the term is an expression rather than a bare column
	Desc    bool
}

// JoinType is the kind of join in a multi-table FROM.
type JoinType int

const (
	JoinInner JoinType = iota
	JoinLeft
	JoinRight
	JoinFull
	JoinCross
)

// Join is one joined table in a multi-table FROM.
type Join struct {
	Type     JoinType
	Database string
	Schema   string
	Table    string
	Alias    string
	On       *Expr // nil for CROSS JOIN
}

// SetOp is the junction between SELECTs in a UNION/INTERSECT/EXCEPT chain.
type SetOp int

const (
	SetUnion     SetOp = iota // UNION (distinct)
	SetUnionAll               // UNION ALL
	SetIntersect              // INTERSECT
	SetExcept                 // EXCEPT
)

// Query is the logical request handed to a backend; M1's parser produces it from T-SQL.
type Query struct {
	Database     string // db qualifier from db.schema.table (empty = current/default)
	Schema       string
	Table        string
	FromSub      *Query // derived table: FROM (SELECT …) (Table empty when set)
	FromAlias    string
	Joins        []Join
	Distinct     bool
	Select       []SelectItem // empty = all columns (*)
	Where        *Expr
	GroupBy      []string
	Having       *Expr
	OrderBy      []OrderItem
	Limit        int
	LimitPercent bool
	Offset       int
	Union        *Query            // next SELECT in a UNION/INTERSECT/EXCEPT chain (nil if none)
	SetOp        SetOp             // junction operation to Union
	CTEs         map[string]*Query // WITH-clause named queries, resolved at FROM
}

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
	AggCountBig
	AggStdev
	AggStdevp
	AggVar
	AggVarp
	AggStringAgg
	AggChecksumAgg
	AggApproxCountDistinct
)

// SelectItem is one entry in the select list: a plain column, an aggregate, an expression, or a window.
type SelectItem struct {
	Column  string      // plain column reference
	Agg     AggFunc     // aggregate function (AggNone for a plain column)
	Arg     string      // aggregate argument column ("*" for COUNT(*))
	ArgExpr *ValueExpr  // aggregate argument expression (e.g. MAX(CASE …)); nil for a plain column arg
	Sep     string      // STRING_AGG separator literal
	Expr    *ValueExpr  // scalar expression (nil unless this item is computed)
	Window  *WindowSpec // window function over an OVER clause (nil otherwise)
	Alias   string      // output column name (optional)
}

// TableFunc is a table-valued function in a FROM clause: STRING_SPLIT(…), OPENJSON(…).
type TableFunc struct {
	Name string
	Args []*ValueExpr
}

// WindowSpec is a window function call: FUNC(args) OVER (PARTITION BY … ORDER BY …).
type WindowSpec struct {
	Func        string
	Args        []*ValueExpr
	PartitionBy []string
	OrderBy     []OrderItem
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
	JoinCrossApply // per-left-row lateral join, inner (drops left rows with no right rows)
	JoinOuterApply // per-left-row lateral join, left-outer (keeps left rows, right NULLs)
)

// Join is one joined table in a multi-table FROM.
type Join struct {
	Type     JoinType
	Database string
	Schema   string
	Table    string
	FromSub  *Query     // derived table on the right side: JOIN/APPLY (SELECT …) (Table empty when set)
	FromFunc *TableFunc // table-valued function on the right side: APPLY STRING_SPLIT(…)/OPENJSON(…)
	Alias    string
	On       *Expr // nil for CROSS JOIN / APPLY
}

// PivotSpec rotates rows to columns: PIVOT (Agg(ValueCol) FOR PivotCol IN (Values…)) AS Alias.
type PivotSpec struct {
	Agg      string // SUM, COUNT, MIN, MAX, …
	ValueCol string // aggregated column ("*" for COUNT(*))
	PivotCol string // column whose distinct values become output columns
	Values   []string
	Alias    string
}

// UnpivotSpec rotates columns to rows: UNPIVOT (ValueCol FOR NameCol IN (Columns…)) AS Alias.
type UnpivotSpec struct {
	ValueCol string // new column holding each source cell's value
	NameCol  string // new column holding the source column's name
	Columns  []string
	Alias    string
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
	FromSub      *Query     // derived table: FROM (SELECT …) (Table empty when set)
	FromFunc     *TableFunc // table-valued function: FROM STRING_SPLIT(…)/OPENJSON(…)
	Pivot        *PivotSpec // PIVOT crosstab over the FROM source (nil if none)
	Unpivot      *UnpivotSpec
	FromAlias    string
	Joins        []Join
	Distinct     bool
	Select       []SelectItem // empty = all columns (*)
	Where        *Expr
	GroupBy      []string   // grouping-column universe (all columns across GroupingSets when those are set)
	GroupingSets [][]string // GROUP BY ROLLUP/CUBE/GROUPING SETS: aggregate once per set, union; nil = single GroupBy
	Having       *Expr
	OrderBy      []OrderItem
	Limit        int
	LimitPercent bool
	Offset       int
	Union        *Query            // next SELECT in a UNION/INTERSECT/EXCEPT chain (nil if none)
	SetOp        SetOp             // junction operation to Union
	CTEs         map[string]*Query // WITH-clause named queries, resolved at FROM
}

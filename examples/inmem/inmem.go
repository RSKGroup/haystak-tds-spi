// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package inmem

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

type table struct {
	def  catalog.Table
	rows [][]any
}

type Backend struct {
	mu           sync.Mutex
	tables       map[string]*table
	order        []string
	routines     map[string]*tds.Routine
	routineOrder []string
}

func New() *Backend {
	b := &Backend{tables: map[string]*table{}, routines: map[string]*tds.Routine{}}
	b.add(catalog.Table{
		Name: "users",
		Columns: []catalog.Column{
			{Name: "id", Type: types.Type{Kind: types.Int64}},
			{Name: "name", Type: types.Type{Kind: types.String, MaxLen: 128}},
		},
		PrimaryKey: []string{"id"},
	}, [][]any{
		{int64(1), "ada"},
		{int64(2), "alan"},
	})
	b.add(catalog.Table{
		Name: "orders",
		Columns: []catalog.Column{
			{Name: "id", Type: types.Type{Kind: types.Int64}},
			{Name: "user_id", Type: types.Type{Kind: types.Int64}},
			{Name: "amount", Type: types.Type{Kind: types.Int64}},
		},
		PrimaryKey:  []string{"id"},
		ForeignKeys: []catalog.ForeignKey{{Columns: []string{"user_id"}, RefTable: "users", RefColumns: []string{"id"}}},
	}, [][]any{
		{int64(10), int64(1), int64(100)},
		{int64(11), int64(2), int64(200)},
		{int64(12), int64(2), int64(50)},
	})
	b.add(catalog.Table{
		Name: "items",
		Columns: []catalog.Column{
			{Name: "id", Type: types.Type{Kind: types.Int64}},
			{Name: "price", Type: types.Type{Kind: types.Decimal, Precision: 10, Scale: 2}},
			{Name: "ref", Type: types.Type{Kind: types.UUID}},
			{Name: "made", Type: types.Type{Kind: types.Time}},
			{Name: "data", Type: types.Type{Kind: types.Bytes, MaxLen: 16}},
		},
		PrimaryKey: []string{"id"},
	}, [][]any{
		{int64(1), 19.99, "12345678-1234-1234-1234-123456789abc", time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC), []byte{0xDE, 0xAD, 0xBE, 0xEF}},
	})
	b.add(catalog.Table{
		Name: "depts",
		Columns: []catalog.Column{
			{Name: "id", Type: types.Type{Kind: types.Int64}},
			{Name: "name", Type: types.Type{Kind: types.String, MaxLen: 64}},
		},
		PrimaryKey: []string{"id"},
	}, [][]any{
		{int64(1), "eng"},
		{int64(2), "sales"},
		{int64(3), "ops"},
	})
	b.add(catalog.Table{
		Name: "emps",
		Columns: []catalog.Column{
			{Name: "id", Type: types.Type{Kind: types.Int64}},
			{Name: "dept_id", Type: types.Type{Kind: types.Int64}},
			{Name: "name", Type: types.Type{Kind: types.String, MaxLen: 64}},
		},
		PrimaryKey:  []string{"id"},
		ForeignKeys: []catalog.ForeignKey{{Columns: []string{"dept_id"}, RefTable: "depts", RefColumns: []string{"id"}}},
	}, [][]any{
		{int64(10), int64(1), "amy"},
		{int64(11), int64(2), "bob"},
		{int64(12), int64(99), "orphan"},
	})
	b.seedRich()
	return b
}

// seedRich adds the reference objects that exercise the rest of the "be SQL Server" surface — every
// value type, IDENTITY / computed / default columns, a composite PK, a self-referencing FK, indexes,
// CHECK constraints, and one of each routine kind (view, procedure, function, trigger). Most of this
// metadata has no sys.* view reading it yet; it is seeded ahead of those views deliberately.
func (b *Backend) seedRich() {
	b.add(catalog.Table{
		Name: "products",
		Columns: []catalog.Column{
			{Name: "product_id", Type: types.Type{Kind: types.Int32}, Identity: true},
			{Name: "sku", Type: types.Type{Kind: types.String, MaxLen: 32}},
			{Name: "name", Type: types.Type{Kind: types.String, MaxLen: 100}},
			{Name: "price", Type: types.Type{Kind: types.Decimal, Precision: 10, Scale: 2}},
			{Name: "in_stock", Type: types.Type{Kind: types.Bool}},
			{Name: "weight", Type: types.Type{Kind: types.Float64, Nullable: true}},
			{Name: "created", Type: types.Type{Kind: types.Time}, Default: "GETDATE()"},
			{Name: "margin", Type: types.Type{Kind: types.Decimal, Precision: 12, Scale: 2}, Computed: "([price]*(0.3))"},
		},
		PrimaryKey: []string{"product_id"},
		Indexes: []catalog.Index{
			{Name: "PK_products", Columns: []string{"product_id"}, Unique: true, Primary: true, Clustered: true},
			{Name: "UX_products_sku", Columns: []string{"sku"}, Unique: true},
		},
		Checks: []catalog.Check{{Name: "CK_products_price", Expression: "([price]>=(0))"}},
	}, [][]any{
		{int64(1), "SKU-1", "Widget", 9.99, true, 1.5, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), 3.00},
		{int64(2), "SKU-2", "Gadget", 19.99, false, 2.0, time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC), 6.00},
	})
	b.add(catalog.Table{
		Name: "product_tags",
		Columns: []catalog.Column{
			{Name: "product_id", Type: types.Type{Kind: types.Int32}},
			{Name: "tag", Type: types.Type{Kind: types.String, MaxLen: 40}},
		},
		PrimaryKey:  []string{"product_id", "tag"},
		ForeignKeys: []catalog.ForeignKey{{Columns: []string{"product_id"}, RefTable: "products", RefColumns: []string{"product_id"}}},
	}, [][]any{
		{int64(1), "new"}, {int64(1), "sale"}, {int64(2), "new"},
	})
	b.add(catalog.Table{
		Name: "categories",
		Columns: []catalog.Column{
			{Name: "category_id", Type: types.Type{Kind: types.Int32}, Identity: true},
			{Name: "name", Type: types.Type{Kind: types.String, MaxLen: 60}},
			{Name: "parent_id", Type: types.Type{Kind: types.Int32, Nullable: true}},
		},
		PrimaryKey:  []string{"category_id"},
		ForeignKeys: []catalog.ForeignKey{{Columns: []string{"parent_id"}, RefTable: "categories", RefColumns: []string{"category_id"}}},
	}, [][]any{
		{int64(1), "All", nil}, {int64(2), "Electronics", int64(1)}, {int64(3), "Phones", int64(2)},
	})

	// typed declares Type.Name so the catalog reports the exact T-SQL type, not the broader Kind default.
	b.add(catalog.Table{
		Name: "typed",
		Columns: []catalog.Column{
			{Name: "id", Type: types.Type{Kind: types.Int64}},
			{Name: "code", Type: types.Type{Kind: types.String, MaxLen: 10, Name: "varchar"}},
			{Name: "grade", Type: types.Type{Kind: types.String, MaxLen: 2, Name: "char"}},
			{Name: "born", Type: types.Type{Kind: types.Time, Name: "date"}},
			{Name: "level", Type: types.Type{Kind: types.Int32, Name: "smallint"}},
		},
		PrimaryKey: []string{"id"},
	}, [][]any{
		{int64(1), "ABC", "A", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), int64(5)},
	})

	b.addRoutine(&tds.Routine{Schema: "dbo", Name: "vActiveProducts", Kind: tds.RoutineView,
		Body: "SELECT product_id, name, price FROM products WHERE in_stock = 1"})
	b.addRoutine(&tds.Routine{Schema: "dbo", Name: "vOrderTotals", Kind: tds.RoutineView,
		Body: "SELECT user_id, SUM(amount) AS total FROM orders GROUP BY user_id"})
	b.addRoutine(&tds.Routine{Schema: "dbo", Name: "vPremiumProducts", Kind: tds.RoutineView,
		Body: "SELECT name, price FROM vActiveProducts WHERE price > 10"}) // view-on-view dependency chain
	b.addRoutine(&tds.Routine{Schema: "dbo", Name: "uspGetUser", Kind: tds.RoutineProc,
		Params: []tds.RoutineParam{{Name: "@id", Type: "int"}},
		Body:   "SELECT name FROM users WHERE id = @id"})
	b.addRoutine(&tds.Routine{Schema: "dbo", Name: "uspAddOrder", Kind: tds.RoutineProc,
		Params: []tds.RoutineParam{{Name: "@user_id", Type: "int"}, {Name: "@amount", Type: "int"}},
		Body:   "INSERT INTO orders (user_id, amount) VALUES (@user_id, @amount)"})
	b.addRoutine(&tds.Routine{Schema: "dbo", Name: "ufnPriceWithTax", Kind: tds.RoutineFunc,
		Params: []tds.RoutineParam{{Name: "@price", Type: "decimal(10,2)"}},
		Body:   "RETURNS decimal(10,2)\nAS\nBEGIN\n  RETURN @price * 1.08\nEND"})
	b.addRoutine(&tds.Routine{Schema: "dbo", Name: "trgOrdersAudit", Kind: tds.RoutineTrigger,
		Body: "ON orders AFTER INSERT\nAS\nBEGIN\n  /* audit hook */\nEND"})
}

func (b *Backend) addRoutine(r *tds.Routine) {
	key := routineKey(r.Name)
	b.routines[key] = r
	b.routineOrder = append(b.routineOrder, key)
}

func (b *Backend) add(def catalog.Table, rows [][]any) {
	b.tables[def.Name] = &table{def: def, rows: rows}
	b.order = append(b.order, def.Name)
}

func (b *Backend) Describe(ctx context.Context) (catalog.Schema, error) {
	var s catalog.Schema
	for _, n := range b.order {
		s.Tables = append(s.Tables, b.tables[n].def)
	}
	s.TableTypes = []catalog.TableType{{
		Name: "OrderLineType",
		Columns: []catalog.Column{
			{Name: "product_id", Type: types.Type{Kind: types.Int64}},
			{Name: "qty", Type: types.Type{Kind: types.Int32}},
		},
	}}
	return s, nil
}

func (b *Backend) Capabilities() tds.Caps {
	return tds.Caps{Pushdown: true, Writable: true, DDL: true, Routines: true}
}

// PutRoutine / GetRoutine / DropRoutine / ListRoutines implement tds.RoutineStore. inmem is a single
// database, so the database argument is ignored; names match case-insensitively, schema-stripped.
func (b *Backend) PutRoutine(ctx context.Context, r *tds.Routine) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	key := routineKey(r.Name)
	if _, ok := b.routines[key]; !ok {
		b.routineOrder = append(b.routineOrder, key)
	}
	b.routines[key] = r
	return nil
}

func (b *Backend) GetRoutine(ctx context.Context, database, name string) (*tds.Routine, bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	r, ok := b.routines[routineKey(name)]
	return r, ok, nil
}

func (b *Backend) DropRoutine(ctx context.Context, database, name string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	key := routineKey(name)
	if _, ok := b.routines[key]; !ok {
		return fmt.Errorf("inmem: unknown routine %q", name)
	}
	delete(b.routines, key)
	for i, n := range b.routineOrder {
		if n == key {
			b.routineOrder = append(b.routineOrder[:i], b.routineOrder[i+1:]...)
			break
		}
	}
	return nil
}

func (b *Backend) ListRoutines(ctx context.Context, database string) ([]*tds.Routine, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]*tds.Routine, 0, len(b.routineOrder))
	for _, n := range b.routineOrder {
		out = append(out, b.routines[n])
	}
	return out, nil
}

func routineKey(name string) string {
	name = strings.Trim(name, "[]\"`")
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		name = strings.Trim(name[i+1:], "[]\"`")
	}
	return strings.ToLower(name)
}

// Scan returns a snapshot of the whole table; the gateway core applies WHERE / projection /
// ORDER BY / LIMIT. A thin backend like this carries no query logic of its own.
func (b *Backend) Scan(ctx context.Context, q *tds.Query) (tds.Rows, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	t, ok := b.tables[q.Table]
	if !ok {
		return nil, fmt.Errorf("inmem: unknown table %q", q.Table)
	}
	snap := make([][]any, len(t.rows))
	copy(snap, t.rows)
	return &rows{cols: t.def.Columns, data: snap}, nil
}

func (b *Backend) Insert(ctx context.Context, in *tds.Insert) (tds.Result, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	t, ok := b.tables[in.Table]
	if !ok {
		return tds.Result{}, fmt.Errorf("inmem: unknown table %q", in.Table)
	}
	for _, vals := range in.Rows {
		full := make([]any, len(t.def.Columns))
		if len(in.Columns) == 0 {
			copy(full, vals)
		} else {
			for i, cn := range in.Columns {
				if ci := colIndex(t.def.Columns, cn); ci >= 0 && i < len(vals) {
					full[ci] = vals[i]
				}
			}
		}
		t.rows = append(t.rows, full)
	}
	return tds.Result{RowsAffected: int64(len(in.Rows))}, nil
}

func (b *Backend) Update(ctx context.Context, up *tds.Update) (tds.Result, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	t, ok := b.tables[up.Table]
	if !ok {
		return tds.Result{}, fmt.Errorf("inmem: unknown table %q", up.Table)
	}
	var n int64
	for _, row := range t.rows {
		if !matchPreds(t.def.Columns, row, up.Where) {
			continue
		}
		for _, a := range up.Assignments {
			if ci := colIndex(t.def.Columns, a.Column); ci >= 0 {
				row[ci] = a.Value
			}
		}
		n++
	}
	return tds.Result{RowsAffected: n}, nil
}

func (b *Backend) Delete(ctx context.Context, del *tds.Delete) (tds.Result, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	t, ok := b.tables[del.Table]
	if !ok {
		return tds.Result{}, fmt.Errorf("inmem: unknown table %q", del.Table)
	}
	kept := t.rows[:0]
	var n int64
	for _, row := range t.rows {
		if matchPreds(t.def.Columns, row, del.Where) {
			n++
			continue
		}
		kept = append(kept, row)
	}
	t.rows = kept
	return tds.Result{RowsAffected: n}, nil
}

func (b *Backend) CreateTable(ctx context.Context, t *catalog.Table) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.tables[t.Name]; ok {
		return fmt.Errorf("inmem: table %q already exists", t.Name)
	}
	b.add(*t, nil)
	return nil
}

func (b *Backend) AlterTable(ctx context.Context, a *tds.AlterTable) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	t, ok := b.tables[a.Table]
	if !ok {
		return fmt.Errorf("inmem: unknown table %q", a.Table)
	}
	for _, c := range a.AddColumns {
		t.def.Columns = append(t.def.Columns, c)
		for i := range t.rows {
			t.rows[i] = append(t.rows[i], nil)
		}
	}
	for _, name := range a.DropColumns {
		idx := -1
		for i, c := range t.def.Columns {
			if c.Name == name {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fmt.Errorf("inmem: unknown column %q on %q", name, a.Table)
		}
		t.def.Columns = append(t.def.Columns[:idx], t.def.Columns[idx+1:]...)
		for i := range t.rows {
			if idx < len(t.rows[i]) {
				t.rows[i] = append(t.rows[i][:idx], t.rows[i][idx+1:]...)
			}
		}
	}
	return nil
}

func (b *Backend) DropTable(ctx context.Context, table string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.tables[table]; !ok {
		return fmt.Errorf("inmem: unknown table %q", table)
	}
	delete(b.tables, table)
	for i, n := range b.order {
		if n == table {
			b.order = append(b.order[:i], b.order[i+1:]...)
			break
		}
	}
	return nil
}

func colIndex(cols []catalog.Column, name string) int {
	for i, c := range cols {
		if c.Name == name {
			return i
		}
	}
	return -1
}

func matchPreds(cols []catalog.Column, row []any, preds []tds.Predicate) bool {
	for _, p := range preds {
		ci := colIndex(cols, p.Column)
		if ci < 0 || !predTrue(row[ci], p) {
			return false
		}
	}
	return true
}

func predTrue(v any, p tds.Predicate) bool {
	c, ok := cmp(v, p.Value)
	if !ok {
		return false
	}
	switch p.Op {
	case tds.OpEq:
		return c == 0
	case tds.OpNe:
		return c != 0
	case tds.OpLt:
		return c < 0
	case tds.OpLe:
		return c <= 0
	case tds.OpGt:
		return c > 0
	case tds.OpGe:
		return c >= 0
	}
	return false
}

func cmp(a, b any) (int, bool) {
	switch av := a.(type) {
	case int64:
		if bv, ok := b.(int64); ok {
			switch {
			case av < bv:
				return -1, true
			case av > bv:
				return 1, true
			default:
				return 0, true
			}
		}
	case string:
		if bv, ok := b.(string); ok {
			return strings.Compare(av, bv), true
		}
	}
	return 0, false
}

type rows struct {
	cols []catalog.Column
	data [][]any
	pos  int
}

func (r *rows) Columns() []catalog.Column { return r.cols }

func (r *rows) Next() bool {
	if r.pos >= len(r.data) {
		return false
	}
	r.pos++
	return true
}

func (r *rows) Values() ([]any, error) { return r.data[r.pos-1], nil }
func (r *rows) Err() error             { return nil }
func (r *rows) Close() error           { return nil }

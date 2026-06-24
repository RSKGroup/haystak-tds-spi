// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"sync"

	"github.com/RSKGroup/haystak-tds-spi/tds"
	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
)

// schemaCache memoizes backend catalog introspection for the lifetime of a single top-level statement.
// The catalog cannot change while one read query executes, so a query that derives a system-catalog view
// many times -- e.g. an APPLY or correlated subquery whose right side reads sys.*/INFORMATION_SCHEMA once
// per outer row -- reuses a single Describe/ListRoutines call instead of re-deriving the whole schema for
// every inner evaluation. It is installed per statement, after DDL/write handling, so a DDL-then-SELECT
// batch still introspects fresh schema for the SELECT.
type schemaCache struct {
	mu       sync.Mutex
	dbs      []string
	schemas  map[string]catalog.Schema
	routines map[string][]*tds.Routine
}

type schemaCacheKey struct{}

// withSchemaCache returns a child context carrying a fresh, empty cache.
func withSchemaCache(ctx context.Context) context.Context {
	return context.WithValue(ctx, schemaCacheKey{}, &schemaCache{
		schemas:  map[string]catalog.Schema{},
		routines: map[string][]*tds.Routine{},
	})
}

func schemaCacheFrom(ctx context.Context) *schemaCache {
	c, _ := ctx.Value(schemaCacheKey{}).(*schemaCache)
	return c
}

func (c *schemaCache) getSchema(db string) (catalog.Schema, []string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, ok := c.schemas[db]
	return s, c.dbs, ok
}

func (c *schemaCache) putSchema(db string, s catalog.Schema, dbs []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.schemas[db] = s
	if dbs != nil {
		c.dbs = dbs
	}
}

func (c *schemaCache) getRoutines(db string) ([]*tds.Routine, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	r, ok := c.routines[db]
	return r, ok
}

func (c *schemaCache) putRoutines(db string, r []*tds.Routine) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.routines[db] = r
}

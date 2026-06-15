// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package engine_test

import "testing"

// sys.partitions projects one row per table; database_files/filegroups report the single-DB topology.
func TestPartitionsAndFiles(t *testing.T) {
	if n := len(qry(t, "SELECT object_id FROM sys.partitions")); n == 0 {
		t.Error("sys.partitions returned no rows; expected one per table")
	}
	if n := len(qry(t, "SELECT name FROM sys.database_files")); n != 1 {
		t.Errorf("sys.database_files = %d rows, want 1", n)
	}
	if name := cell(qry(t, "SELECT name FROM sys.filegroups")[0][0]); name != "PRIMARY" {
		t.Errorf("sys.filegroups name = %q, want PRIMARY", name)
	}
}

// sp_helptrigger and sp_depends project real routine data (the inmem trigger on orders, view deps).
func TestSpHelptriggerAndDepends(t *testing.T) {
	if name := cell(qry(t, "EXEC sp_helptrigger 'orders'")[0][0]); name != "trgOrdersAudit" {
		t.Errorf("sp_helptrigger 'orders' = %q, want trgOrdersAudit", name)
	}
	if ref := cell(qry(t, "EXEC sp_depends 'vActiveProducts'")[0][0]); ref != "products" {
		t.Errorf("sp_depends 'vActiveProducts' = %q, want products", ref)
	}
}

// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package tdstest

// FROM-clause table-valued functions, asserted via constant inputs so they are backend-independent.
var tvfCases = []Case{
	{Element: "tvf:STRING_SPLIT", SQL: "SELECT STRING_AGG(value, '-') AS s FROM STRING_SPLIT('x,y,z', ',')", Want: []any{"x-y-z"}},
	{Element: "tvf:OPENJSON", SQL: "SELECT COUNT(*) AS c FROM OPENJSON('[1,2,3,4]')", Want: []any{4}},
}

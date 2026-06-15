// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package functions

import (
	"crypto/rand"
	"fmt"
	"strings"
)

func init() {
	register("SERVERPROPERTY", func(a []any) any { return serverProperty(argStr(a, 0)) })
	register("DATABASEPROPERTYEX", func([]any) any { return "ON" })
	register("NEWID", func([]any) any { return newID() })
	// no identity tracking and no TRY/CATCH context: these are NULL, matching SQL Server outside scope.
	register("SCOPE_IDENTITY", nilFn)
	register("IDENT_CURRENT", nilFn)
	register("ERROR_MESSAGE", nilFn)
	register("ERROR_NUMBER", nilFn)
	register("ERROR_SEVERITY", nilFn)
	register("ERROR_STATE", nilFn)
	register("ERROR_LINE", nilFn)
	register("ERROR_PROCEDURE", nilFn)
	register("NEWSEQUENTIALID", func([]any) any { return newID() })
	register("XACT_STATE", func([]any) any { return int64(0) }) // no active transaction
	register("CURSOR_STATUS", func([]any) any { return int64(-3) })
	register("CONNECTIONPROPERTY", func(a []any) any {
		switch strings.ToLower(argStr(a, 0)) {
		case "net_transport":
			return "TCP"
		case "protocol_type":
			return "TSQL"
		case "local_net_address":
			return "127.0.0.1"
		case "local_tcp_port":
			return int64(1433)
		case "client_net_address":
			return "127.0.0.1"
		}
		return nil
	})
}

func nilFn([]any) any { return nil }

// newID is NEWID(): a random v4 GUID in SQL Server's uppercase form.
func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return strings.ToUpper(fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]))
}

// serverProperty answers SERVERPROPERTY(name) with the values native GUIs read on connect.
func serverProperty(name string) any {
	switch strings.ToUpper(name) {
	case "PRODUCTVERSION":
		return "16.0.1000.6"
	case "PRODUCTLEVEL":
		return "RTM"
	case "EDITION":
		return "Developer Edition (64-bit)"
	case "ENGINEEDITION":
		return int64(3)
	case "COLLATION":
		return "SQL_Latin1_General_CP1_CI_AS"
	case "ISCLUSTERED", "ISINTEGRATEDSECURITYONLY", "ISFULLTEXTINSTALLED", "ISHADRENABLED":
		return int64(0)
	case "MACHINENAME", "SERVERNAME", "COMPUTERNAMEPHYSICALNETBIOS":
		return "haystak-tds-spi"
	case "INSTANCENAME":
		return nil
	case "BUILDCLRVERSION":
		return "v4.0.30319"
	}
	return ""
}

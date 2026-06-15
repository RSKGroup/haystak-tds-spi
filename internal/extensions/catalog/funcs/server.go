// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package funcs

import "strings"

// serverVersion is the @@VERSION banner — a real SQL Server build string so native drivers accept it.
const serverVersion = "Microsoft SQL Server 2022 (haystak-tds-spi gateway) - TDS 7.4"

func init() {
	register("@@VERSION", func([]any) any { return serverVersion })
	register("@@SPID", func([]any) any { return int64(1) })
	register("@@SERVERNAME", func([]any) any { return "haystak-tds-spi" })
	register("@@LANGUAGE", func([]any) any { return "us_english" })
	register("@@ROWCOUNT", zeroInt)
	register("@@ERROR", zeroInt)
	register("@@TRANCOUNT", zeroInt)
	register("@@FETCH_STATUS", zeroInt)
	register("SERVERPROPERTY", func(a []any) any { return serverProperty(argStr(a, 0)) })
	register("DATABASEPROPERTYEX", func([]any) any { return "ON" })
}

func zeroInt([]any) any { return int64(0) }

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

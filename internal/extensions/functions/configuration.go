// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package functions

// serverVersion is the @@VERSION banner — a real SQL Server build string so native drivers accept it.
const serverVersion = "Microsoft SQL Server 2022 (haystak-tds-spi gateway) - TDS 7.4"

func init() {
	register("@@VERSION", func([]any) any { return serverVersion })
	register("@@MICROSOFTVERSION", func([]any) any { return int64(268436456) }) // 16.0.1000
	register("@@SPID", func([]any) any { return int64(1) })
	register("@@SERVERNAME", func([]any) any { return "haystak-tds-spi" })
	register("@@LANGUAGE", func([]any) any { return "us_english" })
	register("@@ROWCOUNT", zeroInt)
	register("@@ERROR", zeroInt)
	register("@@TRANCOUNT", zeroInt)
	register("@@FETCH_STATUS", zeroInt)
	register("@@IDENTITY", func([]any) any { return nil })
	register("@@PROCID", func([]any) any { return nil })
	register("@@SERVICENAME", func([]any) any { return "MSSQLSERVER" })
	register("@@NESTLEVEL", zeroInt)
	register("@@CURSOR_ROWS", zeroInt)
	register("@@MAX_PRECISION", func([]any) any { return int64(38) })
	register("@@DATEFIRST", func([]any) any { return int64(7) })
	register("@@LOCK_TIMEOUT", func([]any) any { return int64(-1) })
	register("@@OPTIONS", func([]any) any { return int64(5496) })
}

func zeroInt([]any) any { return int64(0) }

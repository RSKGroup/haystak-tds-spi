// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package functions

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
}

func zeroInt([]any) any { return int64(0) }

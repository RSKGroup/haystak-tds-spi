// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package funcs

func init() {
	register("SYSTEM_USER", haystakUser)
	register("CURRENT_USER", haystakUser)
	register("SESSION_USER", haystakUser)
	register("USER", haystakUser)
	register("USER_NAME", haystakUser)
	register("SUSER_NAME", haystakUser)
	register("SUSER_SNAME", haystakUser)
	register("HOST_NAME", func([]any) any { return "haystak-tds-spi" })
	register("APP_NAME", func([]any) any { return "haystak-tds-spi" })
}

func haystakUser([]any) any { return "haystak" }

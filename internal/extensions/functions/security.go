// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package functions

import "strings"

func init() {
	register("SYSTEM_USER", haystakUser)
	register("CURRENT_USER", haystakUser)
	register("SESSION_USER", haystakUser)
	register("USER", haystakUser)
	register("USER_NAME", haystakUser)
	register("SUSER_NAME", haystakUser)
	register("SUSER_SNAME", haystakUser)
	register("ORIGINAL_LOGIN", haystakUser)
	register("HOST_NAME", func([]any) any { return "haystak-tds-spi" })
	register("APP_NAME", func([]any) any { return "haystak-tds-spi" })
	register("CONTEXT_INFO", func([]any) any { return nil })    // no session context_info set
	register("SESSION_CONTEXT", func([]any) any { return nil }) // no session keys set
	register("HAS_PERMS_BY_NAME", func([]any) any { return int64(1) })
	register("USER_ID", func(a []any) any {
		if len(a) == 0 || a[0] == nil {
			return int64(1)
		}
		if n := argStr(a, 0); strings.EqualFold(n, "dbo") || strings.EqualFold(n, "haystak") {
			return int64(1)
		}
		return nil
	})
	register("SUSER_ID", func(a []any) any {
		if len(a) == 0 || a[0] == nil {
			return int64(1)
		}
		return nil
	})
	// roles aren't modeled: everyone is in public, every other role is unknown -> NULL.
	register("IS_MEMBER", isPublicMember)
	register("IS_SRVROLEMEMBER", isPublicMember)
	register("IS_ROLEMEMBER", isPublicMember)
}

func haystakUser([]any) any { return "haystak" }

func isPublicMember(a []any) any {
	if strings.EqualFold(argStr(a, 0), "public") {
		return int64(1)
	}
	return nil
}

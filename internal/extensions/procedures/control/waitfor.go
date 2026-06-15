// Copyright 2026 RSKGroup, LLC.
// SPDX-License-Identifier: Apache-2.0

package control

import (
	"context"
	"strings"
	"time"
)

const maxWaitfor = 30 * time.Second // cap so WAITFOR cannot hold a connection indefinitely

type Waitfor struct {
	isDelay bool
	val     string
}

func (*Waitfor) isStmt() {}

func execWaitfor(ctx context.Context, n *Waitfor) error {
	if !n.isDelay { // WAITFOR TIME: blocking until a wall-clock time is not meaningful for a gateway
		return nil
	}
	d := parseDelay(n.val)
	if d <= 0 {
		return nil
	}
	if d > maxWaitfor {
		d = maxWaitfor
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// parseDelay reads an 'hh:mm:ss' delay into a duration.
func parseDelay(s string) time.Duration {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) != 3 {
		return 0
	}
	return time.Duration(atoiOr(parts[0]))*time.Hour +
		time.Duration(atoiOr(parts[1]))*time.Minute +
		time.Duration(atoiOr(parts[2]))*time.Second
}

func atoiOr(s string) int {
	n := 0
	for _, c := range strings.TrimSpace(s) {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}

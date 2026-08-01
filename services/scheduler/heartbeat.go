package main

import (
	"fmt"
	"time"
)

// evaluateHeartbeat decides whether a heartbeat monitor is healthy right now.
// There is no network call: the monitor is healthy while a ping has arrived
// within period + grace. A heartbeat that has never been pinged is given one
// full window from when it was created before it counts as late, so creating
// one doesn't immediately page you.
//
// Pure and clock-injected so the windows are unit-testable.
func evaluateHeartbeat(now time.Time, lastPing *time.Time, createdAt time.Time, periodSeconds, graceSeconds int) checkResult {
	window := time.Duration(periodSeconds)*time.Second + time.Duration(graceSeconds)*time.Second

	since := createdAt
	everPinged := lastPing != nil
	if everPinged {
		since = *lastPing
	}
	late := now.Sub(since)

	if late <= window {
		return checkResult{OK: true}
	}
	if !everPinged {
		return checkResult{
			Error: fmt.Sprintf("no heartbeat received since the monitor was created %s ago (expected every %s)",
				humanDur(late), humanDur(time.Duration(periodSeconds)*time.Second)),
			FailurePhase: "heartbeat",
		}
	}
	return checkResult{
		Error: fmt.Sprintf("heartbeat is %s late (last ping %s ago, expected every %s)",
			humanDur(late-window), humanDur(late), humanDur(time.Duration(periodSeconds)*time.Second)),
		FailurePhase: "heartbeat",
	}
}

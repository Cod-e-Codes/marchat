// Unit tests for the same window/burst/cooldown logic as client.readPump rate limiting.
// Keeps the algorithm checkable without spinning up WebSockets.

package server

import (
	"testing"
	"time"
)

// loadverifyRateLimitStep matches the gate in client.readPump before a message
// would be forwarded (uses the same package-level constants as client.go).
func loadverifyRateLimitStep(now time.Time, timestamps *[]time.Time, cooldown *time.Time) bool {
	if now.Before(*cooldown) {
		return false
	}
	cutoff := now.Add(-rateLimitWindow)
	filtered := (*timestamps)[:0]
	for _, ts := range *timestamps {
		if ts.After(cutoff) {
			filtered = append(filtered, ts)
		}
	}
	*timestamps = filtered
	if len(*timestamps) >= rateLimitMessages {
		*cooldown = now.Add(rateLimitCooldown)
		return false
	}
	*timestamps = append(*timestamps, now)
	return true
}

func TestLoadverify_RateLimitAllowsTwentyThenCooldown(t *testing.T) {
	t0 := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	var ts []time.Time
	var cd time.Time

	var accepted int
	for i := 0; i < 25; i++ {
		if loadverifyRateLimitStep(t0, &ts, &cd) {
			accepted++
		}
	}
	if accepted != 20 {
		t.Fatalf("expected 20 accepts in same-window burst, got %d (timestamps=%d)", accepted, len(ts))
	}
	if !cd.After(t0) {
		t.Fatalf("expected cooldown deadline after burst, got cd=%v t0=%v", cd, t0)
	}
}

func TestLoadverify_RateLimitCooldownSuppressesUntilExpiry(t *testing.T) {
	t0 := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	var ts []time.Time
	var cd time.Time

	for i := 0; i < 20; i++ {
		if !loadverifyRateLimitStep(t0, &ts, &cd) {
			t.Fatalf("message %d should be accepted", i+1)
		}
	}
	if loadverifyRateLimitStep(t0, &ts, &cd) {
		t.Fatal("21st same-timestamp message should be dropped")
	}

	during := t0.Add(rateLimitCooldown / 2)
	if loadverifyRateLimitStep(during, &ts, &cd) {
		t.Fatal("message during cooldown should be dropped")
	}

	after := cd.Add(time.Nanosecond)
	if !loadverifyRateLimitStep(after, &ts, &cd) {
		t.Fatal("first message after cooldown should be accepted")
	}
}

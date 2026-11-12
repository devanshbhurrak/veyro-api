package throttle

import (
	"testing"
	"time"
)

func TestLimiterAllowsUpToMaxThenBlocks(t *testing.T) {
	l := New(time.Minute, 2)
	if !l.Allow("k") || !l.Allow("k") {
		t.Fatal("first attempts within the limit must be allowed")
	}
	if l.Allow("k") {
		t.Fatal("attempt above the limit must be blocked")
	}
	l.Reset("k")
	if !l.Allow("k") {
		t.Fatal("reset must clear the limit")
	}
}

func TestLimiterExpiresEntriesAndBoundsMemory(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	l := New(time.Minute, 5)
	l.now = func() time.Time { return now }

	l.Allow("old")
	now = now.Add(2 * time.Minute)
	l.Allow("new")

	l.mu.Lock()
	_, oldExists := l.entries["old"]
	_, newExists := l.entries["new"]
	l.mu.Unlock()
	if oldExists {
		t.Fatal("expired entries must be removed by periodic cleanup")
	}
	if !newExists {
		t.Fatal("current entry must be retained")
	}
}

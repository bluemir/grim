package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLimiter(t *testing.T, limit int, window time.Duration, maxKeys int) (*rateLimiter, *testClock) {
	t.Helper()
	clock := &testClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	r, err := newRateLimiter(limit, window, maxKeys, clock.now)
	require.NoError(t, err)
	return r, clock
}

func allow(t *testing.T, r *rateLimiter, ip string) bool {
	t.Helper()
	ok, err := r.Allow(context.Background(), ip)
	require.NoError(t, err)
	return ok
}

func TestRateLimitRefillsGradually(t *testing.T) {
	r, clock := newTestLimiter(t, 4, time.Hour, 100)

	for range 4 {
		assert.True(t, allow(t, r, "192.0.2.1"), "the whole allowance may be used at once")
	}
	assert.False(t, allow(t, r, "192.0.2.1"))

	// One request comes back every window/limit (15m), not all at a boundary.
	clock.advance(15 * time.Minute)
	assert.True(t, allow(t, r, "192.0.2.1"))
	assert.False(t, allow(t, r, "192.0.2.1"))

	clock.advance(time.Hour)
	for range 4 {
		assert.True(t, allow(t, r, "192.0.2.1"))
	}
	assert.False(t, allow(t, r, "192.0.2.1"), "idling longer doesn't bank more than the allowance")
}

func TestRateLimitGroupsIPv6By64(t *testing.T) {
	r, _ := newTestLimiter(t, 1, time.Hour, 100)

	assert.True(t, allow(t, r, "2001:db8:1:2::1"))
	assert.False(t, allow(t, r, "2001:db8:1:2::ffff"), "same /64 shares the limit")
	assert.True(t, allow(t, r, "2001:db8:1:3::1"), "another /64 is another client")

	assert.True(t, allow(t, r, "192.0.2.1"))
	assert.False(t, allow(t, r, "::ffff:192.0.2.1"), "IPv4-mapped IPv6 is the same IPv4 client")
	assert.True(t, allow(t, r, "192.0.2.2"))
}

func TestRateLimitForgetsLeastRecentClientPastMaxKeys(t *testing.T) {
	r, _ := newTestLimiter(t, 1, time.Hour, 2)

	assert.True(t, allow(t, r, "192.0.2.1"))
	assert.True(t, allow(t, r, "192.0.2.2"))
	assert.False(t, allow(t, r, "192.0.2.2"))
	assert.True(t, allow(t, r, "192.0.2.3")) // evicts .1, the least recently seen

	assert.True(t, allow(t, r, "192.0.2.1"), "an evicted client starts over")
	assert.False(t, allow(t, r, "192.0.2.3"), "a client still tracked keeps its count")
}

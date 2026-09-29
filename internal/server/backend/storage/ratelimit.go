package storage

import (
	"context"
	"net/netip"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/memstore"
)

// rateLimitMaxKeys caps how many clients the limiter tracks. Past this, the
// least recently seen client is forgotten (its count starts over), so memory
// stays bounded even under a flood from many addresses.
const rateLimitMaxKeys = 100_000

// rateLimiter allows each client `limit` creates per `window`, using GCRA:
// a client may use the whole allowance at once, after which it refills
// evenly over the window instead of all at once at a window boundary.
type rateLimiter struct {
	gcra *throttled.GCRARateLimiterCtx
}

func newRateLimiter(limit int, window time.Duration, maxKeys int, now func() time.Time) (*rateLimiter, error) {
	if window <= 0 {
		window = time.Hour
	}
	st, err := memstore.New(maxKeys)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	st.SetTimeNow(now)

	gcra, err := throttled.NewGCRARateLimiterCtx(throttled.WrapStoreWithContext(st), throttled.RateQuota{
		MaxRate:  throttled.PerDuration(limit, window),
		MaxBurst: limit - 1,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &rateLimiter{gcra: gcra}, nil
}

func (r *rateLimiter) Allow(ctx context.Context, clientIP string) (bool, error) {
	limited, _, err := r.gcra.RateLimitCtx(ctx, clientKey(clientIP), 1)
	if err != nil {
		return false, errors.WithStack(err)
	}
	return !limited, nil
}

// clientKey groups IPv6 clients by /64, the block a single user or host
// usually gets; counting single IPv6 addresses would let one client rotate
// through addresses and never hit the limit.
func clientKey(ip string) string {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ip
	}
	addr = addr.Unmap()
	if addr.Is4() {
		return addr.String()
	}
	prefix, _ := addr.Prefix(64)
	return prefix.String()
}

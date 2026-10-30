//go:build testutil

package ratelimit_test

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/66gu1/easygodocs/internal/infrastructure/ratelimit"
	"github.com/66gu1/easygodocs/internal/infrastructure/redistest"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

var redisAddr string

func TestMain(m *testing.M) {
	addr, cleanup := redistest.StartRedis()
	redisAddr = addr
	code := m.Run()
	cleanup()
	os.Exit(code)
}

// newLimiter creates a limiter with a prefix unique to the test, so parallel tests don't share keys.
func newLimiter(t *testing.T, client redis.Scripter, prefix string, limit int, window time.Duration) *ratelimit.Limiter {
	t.Helper()

	l, err := ratelimit.NewLimiter(client, ratelimit.Config{
		Prefix:  prefix,
		Limit:   limit,
		Window:  window,
		Timeout: 5 * time.Second,
	})
	require.NoError(t, err)
	return l
}

func uniquePrefix() string {
	return "test:" + uuid.NewString() + ":"
}

func TestLimiter_AllowsUpToLimit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	l := newLimiter(t, redistest.NewClient(t, redisAddr), uniquePrefix(), 3, time.Minute)

	for i := range 3 {
		res, err := l.Allow(ctx, "ip")
		require.NoError(t, err)
		require.True(t, res.Allowed)
		require.Equal(t, 3, res.Limit)
		require.Equal(t, 2-i, res.Remaining)
		require.Zero(t, res.RetryAfter)
	}

	res, err := l.Allow(ctx, "ip")
	require.NoError(t, err)
	require.False(t, res.Allowed)
	require.Zero(t, res.Remaining)
	require.Positive(t, res.RetryAfter)
	require.LessOrEqual(t, res.RetryAfter, time.Minute)
}

func TestLimiter_KeysAreIndependent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	l := newLimiter(t, redistest.NewClient(t, redisAddr), uniquePrefix(), 1, time.Minute)

	res, err := l.Allow(ctx, "ip-1")
	require.NoError(t, err)
	require.True(t, res.Allowed)

	res, err = l.Allow(ctx, "ip-2")
	require.NoError(t, err)
	require.True(t, res.Allowed)

	res, err = l.Allow(ctx, "ip-1")
	require.NoError(t, err)
	require.False(t, res.Allowed)
}

func TestLimiter_WindowSlides(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	window := 500 * time.Millisecond
	l := newLimiter(t, redistest.NewClient(t, redisAddr), uniquePrefix(), 2, window)

	res, err := l.Allow(ctx, "ip")
	require.NoError(t, err)
	require.True(t, res.Allowed)

	time.Sleep(window / 2)
	res, err = l.Allow(ctx, "ip")
	require.NoError(t, err)
	require.True(t, res.Allowed)

	res, err = l.Allow(ctx, "ip")
	require.NoError(t, err)
	require.False(t, res.Allowed)

	// After RetryAfter only the first request has left the window: one slot is free, not two.
	// A fixed window would reset both here and allow a burst of 2*limit around the boundary.
	time.Sleep(res.RetryAfter + 20*time.Millisecond)
	res, err = l.Allow(ctx, "ip")
	require.NoError(t, err)
	require.True(t, res.Allowed)

	res, err = l.Allow(ctx, "ip")
	require.NoError(t, err)
	require.False(t, res.Allowed)
}

func TestLimiter_RejectedRequestsAreNotRecorded(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := redistest.NewClient(t, redisAddr)
	prefix := uniquePrefix()
	l := newLimiter(t, client, prefix, 2, time.Minute)

	for range 10 {
		_, err := l.Allow(ctx, "ip")
		require.NoError(t, err)
	}

	// Memory per key is bounded by the limit, however many requests are rejected.
	count, err := client.ZCard(ctx, prefix+"ip").Result()
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
}

func TestLimiter_KeyExpires(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := redistest.NewClient(t, redisAddr)
	prefix := uniquePrefix()
	l := newLimiter(t, client, prefix, 2, time.Minute)

	_, err := l.Allow(ctx, "ip")
	require.NoError(t, err)

	ttl, err := client.PTTL(ctx, prefix+"ip").Result()
	require.NoError(t, err)
	require.Positive(t, ttl)
	require.LessOrEqual(t, ttl, time.Minute)
}

// TestLimiter_ConcurrentInstances simulates several app instances, each with its own Redis client,
// sending requests for one key at the same time. With a non-atomic "get count, then increment"
// many of them would see a count below the limit and pass; the script lets exactly limit through.
func TestLimiter_ConcurrentInstances(t *testing.T) {
	t.Parallel()

	const (
		instances           = 4
		requestsPerInstance = 50
		limit               = 30
	)

	ctx := context.Background()
	client := redistest.NewClient(t, redisAddr)
	prefix := uniquePrefix()
	limiters := make([]*ratelimit.Limiter, instances)
	for i := range limiters {
		limiters[i] = newLimiter(t, redistest.NewClient(t, redisAddr), prefix, limit, time.Minute)
	}

	var allowed atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, l := range limiters {
		for range requestsPerInstance {
			wg.Go(func() {
				<-start
				res, err := l.Allow(ctx, "ip")
				if err != nil {
					t.Errorf("Allow: %v", err)
					return
				}
				if res.Allowed {
					allowed.Add(1)
				}
			})
		}
	}
	close(start)
	wg.Wait()

	require.EqualValues(t, limit, allowed.Load())
	// Many requests land in the same millisecond, so they have equal scores. Each must still be
	// a separate member: if the member were the time, they would overwrite each other and be undercounted.
	count, err := client.ZCard(ctx, prefix+"ip").Result()
	require.NoError(t, err)
	require.EqualValues(t, limit, count)
}

func TestLimiter_RedisUnavailable(t *testing.T) {
	t.Parallel()

	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", ContextTimeoutEnabled: true, MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() }) //nolint:errcheck
	l := newLimiter(t, client, uniquePrefix(), 1, time.Minute)

	_, err := l.Allow(context.Background(), "ip")
	require.Error(t, err)
}

func TestNewLimiter_Validation(t *testing.T) {
	t.Parallel()

	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	t.Cleanup(func() { _ = client.Close() }) //nolint:errcheck
	valid := ratelimit.Config{Prefix: "p:", Limit: 1, Window: time.Second, Timeout: time.Second}

	tests := []struct {
		name   string
		client redis.Scripter
		modify func(cfg *ratelimit.Config)
	}{
		{name: "nil client", modify: func(*ratelimit.Config) {}},
		{name: "empty prefix", client: client, modify: func(cfg *ratelimit.Config) { cfg.Prefix = "" }},
		{name: "zero limit", client: client, modify: func(cfg *ratelimit.Config) { cfg.Limit = 0 }},
		{name: "window under 1ms", client: client, modify: func(cfg *ratelimit.Config) { cfg.Window = time.Microsecond }},
		{name: "zero timeout", client: client, modify: func(cfg *ratelimit.Config) { cfg.Timeout = 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := valid
			tt.modify(&cfg)
			_, err := ratelimit.NewLimiter(tt.client, cfg)
			require.Error(t, err)
		})
	}

	_, err := ratelimit.NewLimiter(client, valid)
	require.NoError(t, err)
}

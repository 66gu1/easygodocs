//go:build testutil

package ratelimit_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
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

func newLoginAttempts(t *testing.T, client redis.Scripter, perEmailAndIP, perEmail int, window time.Duration) *ratelimit.LoginAttempts {
	t.Helper()

	l, err := ratelimit.NewLoginAttempts(client, ratelimit.LoginAttemptsConfig{
		PerEmailAndIP: perEmailAndIP,
		PerEmail:      perEmail,
		Window:        window,
		Timeout:       5 * time.Second,
	})
	require.NoError(t, err)
	return l
}

// uniqueEmail keeps parallel tests from sharing counters: the keys have no test prefix.
func uniqueEmail() string {
	return uuid.NewString() + "@example.com"
}

func reserve(t *testing.T, l *ratelimit.LoginAttempts, email, ip string) ratelimit.AttemptResult {
	t.Helper()

	res, err := l.Reserve(context.Background(), email, ip)
	require.NoError(t, err)
	return res
}

func TestLoginAttempts_PerEmailAndIP(t *testing.T) {
	t.Parallel()

	l := newLoginAttempts(t, redistest.NewClient(t, redisAddr), 3, 100, time.Minute)
	email := uniqueEmail()

	for range 3 {
		require.True(t, reserve(t, l, email, "attacker").Allowed)
	}
	res := reserve(t, l, email, "attacker")
	require.False(t, res.Allowed)
	require.Positive(t, res.RetryAfter)
	require.LessOrEqual(t, res.RetryAfter, time.Minute)

	// The attacker can't lock out the owner of the account, who logs in from another IP.
	require.True(t, reserve(t, l, email, "owner").Allowed)
	// Other accounts aren't affected either.
	require.True(t, reserve(t, l, uniqueEmail(), "attacker").Allowed)
}

func TestLoginAttempts_PerEmail(t *testing.T) {
	t.Parallel()

	l := newLoginAttempts(t, redistest.NewClient(t, redisAddr), 2, 5, time.Minute)
	email := uniqueEmail()

	// A distributed attack: each IP stays under its own limit, the total for the email is bounded.
	for i := range 5 {
		require.True(t, reserve(t, l, email, "ip-"+string(rune('a'+i))).Allowed)
	}
	res := reserve(t, l, email, "ip-new")
	require.False(t, res.Allowed)
	require.Positive(t, res.RetryAfter)
}

func TestLoginAttempts_RejectedAttemptsAreNotCounted(t *testing.T) {
	t.Parallel()

	l := newLoginAttempts(t, redistest.NewClient(t, redisAddr), 2, 3, time.Minute)
	email := uniqueEmail()

	// The email+IP limit rejects these, so they must not use up the email limit.
	for range 10 {
		reserve(t, l, email, "attacker")
	}
	require.True(t, reserve(t, l, email, "owner").Allowed)
}

func TestLoginAttempts_EmailIsCaseInsensitive(t *testing.T) {
	t.Parallel()

	l := newLoginAttempts(t, redistest.NewClient(t, redisAddr), 1, 100, time.Minute)
	email := uniqueEmail()

	require.True(t, reserve(t, l, email, "ip").Allowed)
	require.False(t, reserve(t, l, " "+strings.ToUpper(email)+" ", "ip").Allowed)
}

func TestLoginAttempts_Release(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	l := newLoginAttempts(t, redistest.NewClient(t, redisAddr), 2, 3, time.Minute)
	email := uniqueEmail()

	require.True(t, reserve(t, l, email, "ip-1").Allowed)
	require.True(t, reserve(t, l, email, "owner").Allowed)

	// The owner logs in: their failures are forgotten and their attempt is given back.
	require.NoError(t, l.Release(ctx, email, "owner"))

	// The email limit has 2 free attempts again (ip-1 used 1 of 3), not 1.
	require.True(t, reserve(t, l, email, "ip-2").Allowed)
	require.True(t, reserve(t, l, email, "ip-3").Allowed)
	require.False(t, reserve(t, l, email, "ip-4").Allowed)
}

func TestLoginAttempts_ReleaseWithoutReserve(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	l := newLoginAttempts(t, redistest.NewClient(t, redisAddr), 1, 1, time.Minute)
	email := uniqueEmail()

	// The counters may already have expired when a login succeeds.
	// Release must not leave a negative counter, that would raise the limit.
	require.NoError(t, l.Release(ctx, email, "ip"))
	require.True(t, reserve(t, l, email, "ip").Allowed)
	require.False(t, reserve(t, l, email, "other-ip").Allowed)
}

func TestLoginAttempts_Refund(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	l := newLoginAttempts(t, redistest.NewClient(t, redisAddr), 2, 100, time.Minute)
	email := uniqueEmail()

	require.True(t, reserve(t, l, email, "ip").Allowed)
	require.True(t, reserve(t, l, email, "ip").Allowed)

	// Only the refunded attempt is given back, the earlier failure stays: one slot, not two.
	require.NoError(t, l.Refund(ctx, email, "ip"))
	require.True(t, reserve(t, l, email, "ip").Allowed)
	require.False(t, reserve(t, l, email, "ip").Allowed)
}

func TestLoginAttempts_RefundKeepsTTL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := redistest.NewClient(t, redisAddr)
	l := newLoginAttempts(t, client, 2, 100, time.Minute)
	email := uniqueEmail()

	reserve(t, l, email, "ip")
	reserve(t, l, email, "ip")
	require.NoError(t, l.Refund(ctx, email, "ip"))

	sum := sha256.Sum256([]byte(email))
	keys, err := client.Keys(ctx, "login_attempts:{"+hex.EncodeToString(sum[:])+"}*").Result()
	require.NoError(t, err)
	require.Len(t, keys, 2)
	for _, key := range keys {
		ttl, err := client.PTTL(ctx, key).Result()
		require.NoError(t, err)
		require.Positive(t, ttl, "key %s must keep its TTL", key)
	}
}

func TestLoginAttempts_RefundWithoutReserve(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	l := newLoginAttempts(t, redistest.NewClient(t, redisAddr), 1, 1, time.Minute)
	email := uniqueEmail()

	// A refund after the counters expired must not leave negative counters.
	require.NoError(t, l.Refund(ctx, email, "ip"))
	require.True(t, reserve(t, l, email, "ip").Allowed)
	require.False(t, reserve(t, l, email, "ip").Allowed)
}

func TestLoginAttempts_WindowExpires(t *testing.T) {
	t.Parallel()

	l := newLoginAttempts(t, redistest.NewClient(t, redisAddr), 1, 100, 300*time.Millisecond)
	email := uniqueEmail()

	require.True(t, reserve(t, l, email, "ip").Allowed)
	res := reserve(t, l, email, "ip")
	require.False(t, res.Allowed)

	time.Sleep(res.RetryAfter + 20*time.Millisecond)
	require.True(t, reserve(t, l, email, "ip").Allowed)
}

// TestLoginAttempts_Concurrent sends many attempts at once, as an attacker would to get more guesses
// checked before the failures are recorded. The attempt is taken atomically before the password check,
// so exactly the limit passes.
func TestLoginAttempts_Concurrent(t *testing.T) {
	t.Parallel()

	const (
		instances           = 4
		attemptsPerInstance = 25
		limit               = 5
	)

	email := uniqueEmail()
	limiters := make([]*ratelimit.LoginAttempts, instances)
	for i := range limiters {
		limiters[i] = newLoginAttempts(t, redistest.NewClient(t, redisAddr), limit, 100, time.Minute)
	}

	var allowed atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, l := range limiters {
		for range attemptsPerInstance {
			wg.Go(func() {
				<-start
				res, err := l.Reserve(context.Background(), email, "attacker")
				if err != nil {
					t.Errorf("Reserve: %v", err)
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
}

func TestNewLoginAttempts_Validation(t *testing.T) {
	t.Parallel()

	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	t.Cleanup(func() { _ = client.Close() }) //nolint:errcheck
	valid := ratelimit.LoginAttemptsConfig{PerEmailAndIP: 5, PerEmail: 100, Window: time.Minute, Timeout: time.Second}

	tests := []struct {
		name   string
		client redis.Scripter
		modify func(cfg *ratelimit.LoginAttemptsConfig)
	}{
		{name: "nil client", modify: func(*ratelimit.LoginAttemptsConfig) {}},
		{name: "zero per email and IP", client: client, modify: func(cfg *ratelimit.LoginAttemptsConfig) { cfg.PerEmailAndIP = 0 }},
		{name: "zero per email", client: client, modify: func(cfg *ratelimit.LoginAttemptsConfig) { cfg.PerEmail = 0 }},
		{name: "per email below per email and IP", client: client, modify: func(cfg *ratelimit.LoginAttemptsConfig) { cfg.PerEmail = 4 }},
		{name: "window under 1ms", client: client, modify: func(cfg *ratelimit.LoginAttemptsConfig) { cfg.Window = time.Microsecond }},
		{name: "zero timeout", client: client, modify: func(cfg *ratelimit.LoginAttemptsConfig) { cfg.Timeout = 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := valid
			tt.modify(&cfg)
			_, err := ratelimit.NewLoginAttempts(tt.client, cfg)
			require.Error(t, err)
		})
	}

	_, err := ratelimit.NewLoginAttempts(client, valid)
	require.NoError(t, err)
}

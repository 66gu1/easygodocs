package ratelimit

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// slidingWindowScript counts requests in a sliding window log: a sorted set per key
// where every allowed request is a member scored by its time in milliseconds.
//
// Checking the count and recording the request must be one atomic step: with a separate
// "get count" and "increment", concurrent requests (also from different app instances)
// can all see a count below the limit and all pass. Redis runs a script without interleaving
// other commands, so the check and the write can't be split by another request.
//
// The time is taken from the Redis server, not from the app, so app instances with skewed
// clocks still share one consistent window.
//
// KEYS[1] - the key, ARGV[1] - window in ms, ARGV[2] - limit, ARGV[3] - unique member.
// Returns {allowed (0/1), remaining, retry after in ms}.
var slidingWindowScript = redis.NewScript(`
local key = KEYS[1]
local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])

local t = redis.call('TIME')
local now = tonumber(t[1]) * 1000 + math.floor(tonumber(t[2]) / 1000)

redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)
local count = redis.call('ZCARD', key)

if count < limit then
	redis.call('ZADD', key, now, ARGV[3])
	redis.call('PEXPIRE', key, window)
	return {1, limit - count - 1, 0}
end

local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
return {0, 0, tonumber(oldest[2]) + window - now}
`)

// Result is the outcome of a rate limit check.
type Result struct {
	Allowed   bool
	Limit     int
	Remaining int
	// RetryAfter is the time until the oldest request in the window expires. Zero if allowed.
	RetryAfter time.Duration
}

// Limiter allows at most limit requests per key within any window of the given length.
// Its state is kept in Redis, so all app instances share the same limit.
type Limiter struct {
	client  redis.Scripter
	prefix  string
	limit   int
	window  time.Duration
	timeout time.Duration
}

type Config struct {
	// Prefix is prepended to every key, so different limiters don't share counters.
	Prefix string
	Limit  int
	Window time.Duration
	// Timeout bounds a single check, so a slow Redis can't stall requests.
	// The Redis client must have ContextTimeoutEnabled for it to apply to network I/O.
	Timeout time.Duration
}

func NewLimiter(client redis.Scripter, cfg Config) (*Limiter, error) {
	if client == nil {
		return nil, errors.New("ratelimit.NewLimiter: client is nil")
	}
	if cfg.Prefix == "" {
		return nil, errors.New("ratelimit.NewLimiter: prefix is empty")
	}
	if cfg.Limit <= 0 {
		return nil, fmt.Errorf("ratelimit.NewLimiter: limit must be positive, got %d", cfg.Limit)
	}
	if cfg.Window < time.Millisecond {
		return nil, fmt.Errorf("ratelimit.NewLimiter: window must be at least 1ms, got %s", cfg.Window)
	}
	if cfg.Timeout <= 0 {
		return nil, fmt.Errorf("ratelimit.NewLimiter: timeout must be positive, got %s", cfg.Timeout)
	}

	return &Limiter{
		client:  client,
		prefix:  cfg.Prefix,
		limit:   cfg.Limit,
		window:  cfg.Window,
		timeout: cfg.Timeout,
	}, nil
}

// Allow records a request for the key if it is within the limit.
// Rejected requests are not recorded, so the set never holds more than limit members per key.
func (l *Limiter) Allow(ctx context.Context, key string) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()

	// Requests within the same millisecond have the same score, so members must be unique.
	member := rand.Text()
	res, err := slidingWindowScript.Run(ctx, l.client,
		[]string{l.prefix + key},
		l.window.Milliseconds(), l.limit, member,
	).Int64Slice()
	if err != nil {
		return Result{}, fmt.Errorf("ratelimit.Limiter.Allow: %w", err)
	}
	if len(res) != 3 {
		return Result{}, fmt.Errorf("ratelimit.Limiter.Allow: unexpected script result %v", res)
	}

	return Result{
		Allowed:    res[0] == 1,
		Limit:      l.limit,
		Remaining:  int(res[1]),
		RetryAfter: time.Duration(res[2]) * time.Millisecond,
	}, nil
}

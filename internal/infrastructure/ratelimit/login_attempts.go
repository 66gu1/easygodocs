package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// reserveScript takes one login attempt if both counters are below their limits.
// It increments both counters or neither, and the check and the increment are one atomic step,
// so concurrent attempts can't all pass the check before any of them is counted.
// A counter's window starts at its first attempt and the key expires with the window.
//
// KEYS[1] - email+IP counter, KEYS[2] - email counter.
// ARGV[1] - email+IP limit, ARGV[2] - email limit, ARGV[3] - window in ms.
// Returns {allowed (0/1), retry after in ms}.
var reserveScript = redis.NewScript(`
local limits = {tonumber(ARGV[1]), tonumber(ARGV[2])}
local window = tonumber(ARGV[3])

local retry = -1
for i, key in ipairs(KEYS) do
	local count = tonumber(redis.call('GET', key) or '0')
	if count >= limits[i] then
		retry = math.max(retry, redis.call('PTTL', key))
	end
end
if retry >= 0 then
	return {0, retry}
end

for _, key in ipairs(KEYS) do
	if redis.call('INCR', key) == 1 then
		redis.call('PEXPIRE', key, window)
	end
end
return {1, 0}
`)

// releaseScript is run after a successful login: it clears the email+IP counter and gives back
// the attempt taken from the email counter. DECR of a missing key would create -1 without
// a TTL, so a counter that drops to zero is deleted instead.
//
// KEYS[1] - email+IP counter, KEYS[2] - email counter.
var releaseScript = redis.NewScript(`
redis.call('DEL', KEYS[1])
if redis.call('DECR', KEYS[2]) <= 0 then
	redis.call('DEL', KEYS[2])
end
return 1
`)

// refundScript gives back one attempt to both counters, keeping the earlier failures.
// It is run when an attempt ended without the password being checked (e.g. a database error).
// DECR keeps the key's TTL; a counter that drops to zero is deleted, see releaseScript.
//
// KEYS[1] - email+IP counter, KEYS[2] - email counter.
var refundScript = redis.NewScript(`
for _, key in ipairs(KEYS) do
	if redis.call('DECR', key) <= 0 then
		redis.call('DEL', key)
	end
end
return 1
`)

// LoginAttempts limits password guessing for an account.
//
// Locking an account after N failures would let anyone lock out a victim by entering wrong passwords.
// Instead there are two counters:
//   - email+IP, strict: a single client gets a few attempts per window, other clients aren't affected;
//   - email, soft: bounds the total guesses for the account from all IPs (a distributed attack).
//     Reaching it requires many IPs and does block the account for everyone until the window ends.
//
// Every attempt is counted when it starts, before the password is checked, and a successful login
// gives it back. Counting only failures after the check would let concurrent requests all pass.
type LoginAttempts struct {
	client  redis.Scripter
	cfg     LoginAttemptsConfig
	timeout time.Duration
}

type LoginAttemptsConfig struct {
	PerEmailAndIP int
	PerEmail      int
	Window        time.Duration
	// Timeout bounds a single call, see Config.Timeout.
	Timeout time.Duration
}

type AttemptResult struct {
	Allowed bool
	// RetryAfter is the time until the exceeded counter expires. Zero if allowed.
	RetryAfter time.Duration
}

func NewLoginAttempts(client redis.Scripter, cfg LoginAttemptsConfig) (*LoginAttempts, error) {
	if client == nil {
		return nil, errors.New("ratelimit.NewLoginAttempts: client is nil")
	}
	if cfg.PerEmailAndIP <= 0 || cfg.PerEmail <= 0 {
		return nil, fmt.Errorf("ratelimit.NewLoginAttempts: limits must be positive, got %d and %d", cfg.PerEmailAndIP, cfg.PerEmail)
	}
	if cfg.PerEmail < cfg.PerEmailAndIP {
		return nil, fmt.Errorf("ratelimit.NewLoginAttempts: per email limit %d is less than per email and IP limit %d",
			cfg.PerEmail, cfg.PerEmailAndIP)
	}
	if cfg.Window < time.Millisecond {
		return nil, fmt.Errorf("ratelimit.NewLoginAttempts: window must be at least 1ms, got %s", cfg.Window)
	}
	if cfg.Timeout <= 0 {
		return nil, fmt.Errorf("ratelimit.NewLoginAttempts: timeout must be positive, got %s", cfg.Timeout)
	}

	return &LoginAttempts{client: client, cfg: cfg, timeout: cfg.Timeout}, nil
}

// Reserve takes an attempt for the email from the IP, if neither limit is reached.
func (l *LoginAttempts) Reserve(ctx context.Context, email, ip string) (AttemptResult, error) {
	ctx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()

	res, err := reserveScript.Run(ctx, l.client, l.keys(email, ip),
		l.cfg.PerEmailAndIP, l.cfg.PerEmail, l.cfg.Window.Milliseconds(),
	).Int64Slice()
	if err != nil {
		return AttemptResult{}, fmt.Errorf("ratelimit.LoginAttempts.Reserve: %w", err)
	}
	if len(res) != 2 {
		return AttemptResult{}, fmt.Errorf("ratelimit.LoginAttempts.Reserve: unexpected script result %v", res)
	}

	return AttemptResult{
		Allowed:    res[0] == 1,
		RetryAfter: time.Duration(res[1]) * time.Millisecond,
	}, nil
}

// Release is called after a successful login: the client's failures are forgotten
// and the attempt is given back to the email counter.
func (l *LoginAttempts) Release(ctx context.Context, email, ip string) error {
	ctx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()

	if err := releaseScript.Run(ctx, l.client, l.keys(email, ip)).Err(); err != nil {
		return fmt.Errorf("ratelimit.LoginAttempts.Release: %w", err)
	}
	return nil
}

// Refund gives back an attempt that ended without the password being checked.
// Unlike Release it doesn't clear the client's earlier failures: otherwise a client could reset
// its counter by making its own attempt fail, e.g. by sending an invalid email.
func (l *LoginAttempts) Refund(ctx context.Context, email, ip string) error {
	ctx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()

	if err := refundScript.Run(ctx, l.client, l.keys(email, ip)).Err(); err != nil {
		return fmt.Errorf("ratelimit.LoginAttempts.Refund: %w", err)
	}
	return nil
}

// keys hashes the email: the key length doesn't depend on the input (the email isn't validated yet),
// and Redis doesn't hold users' addresses. The email is normalized like on registration,
// so "User@Mail.com" and "user@mail.com" share a counter.
//
// The hash is a hash tag ({...}): both keys are used in one script, and in Redis Cluster
// keys of one script must be in the same slot.
func (l *LoginAttempts) keys(email, ip string) []string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(strings.ToLower(email))))
	base := "login_attempts:{" + hex.EncodeToString(sum[:]) + "}"
	return []string{base + ":ip:" + ip, base}
}

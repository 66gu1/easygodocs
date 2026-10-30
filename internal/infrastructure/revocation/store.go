package revocation

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// revokeUserScript stores the time before which the user's tokens and sessions are revoked.
// It only moves the time forward: with concurrent revocations (or instances with skewed clocks)
// an earlier time must not overwrite a later one.
//
// KEYS[1] - the user key, ARGV[1] - revocation time in ms, ARGV[2] - TTL in ms.
var revokeUserScript = redis.NewScript(`
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
local at = tonumber(ARGV[1])
if at > current then
	redis.call('SET', KEYS[1], at, 'PX', ARGV[2])
else
	redis.call('PEXPIRE', KEYS[1], ARGV[2])
end
return 1
`)

// Store keeps revoked sessions and users in Redis, so access tokens can be checked
// without a database query on every request. PostgreSQL stays the source of truth:
// the store only records revocations, and callers write it before the database.
//
// Entries live as long as a session can (the session TTL), not only as long as an access token:
// if the database step of a revocation fails, the session is still alive there, and the entry
// is what stops its refresh token from issuing new access tokens.
type Store struct {
	client  redis.Cmdable
	ttl     time.Duration
	timeout time.Duration
}

func NewStore(client redis.Cmdable, ttl, timeout time.Duration) (*Store, error) {
	if client == nil {
		return nil, errors.New("revocation.NewStore: client is nil")
	}
	if ttl < time.Millisecond {
		return nil, fmt.Errorf("revocation.NewStore: ttl must be at least 1ms, got %s", ttl)
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("revocation.NewStore: timeout must be positive, got %s", timeout)
	}
	return &Store{client: client, ttl: ttl, timeout: timeout}, nil
}

// RevokeSession revokes one session (logout).
func (s *Store) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := s.client.Set(ctx, sessionKey(userID, sessionID), 1, s.ttl).Err(); err != nil {
		return fmt.Errorf("revocation.Store.RevokeSession: %w", err)
	}
	return nil
}

// RevokeUser revokes every token and session of the user issued at or before the given time
// (logout everywhere, password change, user deletion).
func (s *Store) RevokeUser(ctx context.Context, userID uuid.UUID, at time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	err := revokeUserScript.Run(ctx, s.client, []string{userKey(userID)}, at.UnixMilli(), s.ttl.Milliseconds()).Err()
	if err != nil {
		return fmt.Errorf("revocation.Store.RevokeUser: %w", err)
	}
	return nil
}

// IsRevoked reports whether a token or session issued at issuedAt has been revoked.
// Times are compared in milliseconds, and a token issued in the same millisecond
// as the revocation counts as revoked.
func (s *Store) IsRevoked(ctx context.Context, userID, sessionID uuid.UUID, issuedAt time.Time) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Both keys share the user's hash tag, so one MGET works in Redis Cluster too.
	vals, err := s.client.MGet(ctx, sessionKey(userID, sessionID), userKey(userID)).Result()
	if err != nil {
		return false, fmt.Errorf("revocation.Store.IsRevoked: %w", err)
	}
	if len(vals) != 2 {
		return false, fmt.Errorf("revocation.Store.IsRevoked: unexpected MGET result %v", vals)
	}

	if vals[0] != nil {
		return true, nil
	}
	if vals[1] == nil {
		return false, nil
	}
	str, ok := vals[1].(string)
	if !ok {
		return false, fmt.Errorf("revocation.Store.IsRevoked: unexpected value type %T", vals[1])
	}
	revokedBefore, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return false, fmt.Errorf("revocation.Store.IsRevoked: %w", err)
	}
	return issuedAt.UnixMilli() <= revokedBefore, nil
}

// The user ID is a hash tag ({...}): keys of one user are in one Redis Cluster slot.
// It also scopes session keys to their owner, so a session ID can't be revoked for another user.
func sessionKey(userID, sessionID uuid.UUID) string {
	return "revocation:{" + userID.String() + "}:session:" + sessionID.String()
}

func userKey(userID uuid.UUID) string {
	return "revocation:{" + userID.String() + "}:user"
}

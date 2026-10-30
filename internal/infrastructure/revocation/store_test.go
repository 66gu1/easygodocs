//go:build testutil

package revocation_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/66gu1/easygodocs/internal/infrastructure/redistest"
	"github.com/66gu1/easygodocs/internal/infrastructure/revocation"
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

func newStore(t *testing.T, client redis.Cmdable, ttl time.Duration) *revocation.Store {
	t.Helper()

	s, err := revocation.NewStore(client, ttl, 5*time.Second)
	require.NoError(t, err)
	return s
}

func isRevoked(t *testing.T, s *revocation.Store, userID, sessionID uuid.UUID, issuedAt time.Time) bool {
	t.Helper()

	revoked, err := s.IsRevoked(context.Background(), userID, sessionID, issuedAt)
	require.NoError(t, err)
	return revoked
}

func TestStore_NothingRevoked(t *testing.T) {
	t.Parallel()

	s := newStore(t, redistest.NewClient(t, redisAddr), time.Hour)
	require.False(t, isRevoked(t, s, uuid.New(), uuid.New(), time.Now()))
}

func TestStore_RevokeSession(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	s := newStore(t, redistest.NewClient(t, redisAddr), time.Hour)
	userID, sessionID, otherSessionID := uuid.New(), uuid.New(), uuid.New()

	require.NoError(t, s.RevokeSession(ctx, userID, sessionID))

	require.True(t, isRevoked(t, s, userID, sessionID, time.Now()))
	require.False(t, isRevoked(t, s, userID, otherSessionID, time.Now()), "other sessions of the user stay valid")
	// Keys are scoped to the owner: revoking with another user ID doesn't touch the session.
	require.False(t, isRevoked(t, s, uuid.New(), sessionID, time.Now()))
}

func TestStore_RevokeUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	s := newStore(t, redistest.NewClient(t, redisAddr), time.Hour)
	userID := uuid.New()
	at := time.UnixMilli(time.Now().UnixMilli())

	require.NoError(t, s.RevokeUser(ctx, userID, at))

	require.True(t, isRevoked(t, s, userID, uuid.New(), at.Add(-time.Minute)), "issued before")
	require.True(t, isRevoked(t, s, userID, uuid.New(), at), "issued in the same millisecond")
	require.True(t, isRevoked(t, s, userID, uuid.New(), at.Add(999*time.Microsecond)), "same millisecond, sub-ms later")
	require.False(t, isRevoked(t, s, userID, uuid.New(), at.Add(time.Millisecond)), "issued a millisecond later")
	require.False(t, isRevoked(t, s, uuid.New(), uuid.New(), at.Add(-time.Minute)), "other users aren't affected")
}

func TestStore_RevokeUserOnlyMovesForward(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	s := newStore(t, redistest.NewClient(t, redisAddr), time.Hour)
	userID := uuid.New()
	later := time.Now()
	earlier := later.Add(-time.Minute)

	require.NoError(t, s.RevokeUser(ctx, userID, later))
	// A concurrent revocation with an earlier time (or a lagging clock) must not shrink the revoked range.
	require.NoError(t, s.RevokeUser(ctx, userID, earlier))

	require.True(t, isRevoked(t, s, userID, uuid.New(), later.Add(-time.Second)))
}

func TestStore_EntriesExpire(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := redistest.NewClient(t, redisAddr)
	ttl := 300 * time.Millisecond
	s := newStore(t, client, ttl)
	userID, sessionID := uuid.New(), uuid.New()

	require.NoError(t, s.RevokeSession(ctx, userID, sessionID))
	require.NoError(t, s.RevokeUser(ctx, userID, time.Now()))

	keys, err := client.Keys(ctx, "revocation:{"+userID.String()+"}*").Result()
	require.NoError(t, err)
	require.Len(t, keys, 2)
	for _, key := range keys {
		pttl, err := client.PTTL(ctx, key).Result()
		require.NoError(t, err)
		require.Positive(t, pttl, key)
		require.LessOrEqual(t, pttl, ttl, key)
	}

	time.Sleep(ttl + 50*time.Millisecond)
	require.False(t, isRevoked(t, s, userID, sessionID, time.Now().Add(-time.Hour)))
}

func TestStore_RedisUnavailable(t *testing.T) {
	t.Parallel()

	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", ContextTimeoutEnabled: true, MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() }) //nolint:errcheck
	s := newStore(t, client, time.Hour)
	ctx := context.Background()

	_, err := s.IsRevoked(ctx, uuid.New(), uuid.New(), time.Now())
	require.Error(t, err)
	require.Error(t, s.RevokeSession(ctx, uuid.New(), uuid.New()))
	require.Error(t, s.RevokeUser(ctx, uuid.New(), time.Now()))
}

func TestNewStore_Validation(t *testing.T) {
	t.Parallel()

	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	t.Cleanup(func() { _ = client.Close() }) //nolint:errcheck

	_, err := revocation.NewStore(nil, time.Hour, time.Second)
	require.Error(t, err)
	_, err = revocation.NewStore(client, 0, time.Second)
	require.Error(t, err)
	_, err = revocation.NewStore(client, time.Hour, 0)
	require.Error(t, err)
	_, err = revocation.NewStore(client, time.Hour, time.Second)
	require.NoError(t, err)
}

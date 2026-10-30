//go:build testutil

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/66gu1/easygodocs/config"
	"github.com/66gu1/easygodocs/internal/app/auth"
	"github.com/66gu1/easygodocs/internal/app/entity"
	"github.com/66gu1/easygodocs/internal/app/user"
	dbinfra "github.com/66gu1/easygodocs/internal/infrastructure/db"
	"github.com/66gu1/easygodocs/internal/infrastructure/redistest"
	"github.com/66gu1/easygodocs/internal/infrastructure/revocation"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// These tests run the real handler (routes, middleware, services) on PostgreSQL and Redis
// and check that every way of revoking access takes effect for the access token at once
// and for the refresh token.

var (
	testDB    *dbinfra.TestDB
	redisAddr string
)

func TestMain(m *testing.M) {
	td, stopPostgres := dbinfra.StartPostgres()
	testDB = td
	addr, stopRedis := redistest.StartRedis()
	redisAddr = addr

	code := m.Run()

	stopRedis()
	stopPostgres()
	os.Exit(code)
}

func testConfig() appConfig {
	return appConfig{
		App: config.Config{
			MaxBodySize: 1 << 20,
			RateLimit: config.RateLimitConfig{
				// All test requests come from one IP, the limits aren't under test here.
				PublicRequestsPerMinute:    100000,
				TimeoutMilliseconds:        1000,
				LoginAttemptsPerEmailAndIP: 1000,
				LoginAttemptsPerEmail:      1000,
				LoginWindowMinutes:         15,
			},
		},
		User:             user.Config{PasswordHashCost: 4}, // the bcrypt minimum, to keep tests fast
		UserValidation:   user.ValidationConfig{MaxEmailLength: 254, MaxNameLength: 30, MinPasswordLength: 8, MaxPasswordLength: 50},
		Auth:             auth.Config{SessionTTLMinutes: 6000, AccessTokenTTLMinutes: 15},
		Entity:           entity.Config{MaxHierarchyDepth: 15},
		EntityValidation: entity.ValidationConfig{MaxNameLength: 100},
		JWTSecret:        []byte("test-secret-that-is-at-least-32-bytes-long"),
	}
}

type testApp struct {
	t       *testing.T
	handler http.Handler
	db      *gorm.DB
}

func newTestApp(t *testing.T, db *gorm.DB, redisClient *redis.Client) *testApp {
	t.Helper()

	h, err := newHandler(db, redisClient, testConfig())
	require.NoError(t, err)
	return &testApp{t: t, handler: h, db: db}
}

func (a *testApp) do(method, path, token string, body any) *httptest.ResponseRecorder {
	a.t.Helper()

	var buf bytes.Buffer
	if body != nil {
		require.NoError(a.t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, "/api/v1"+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	a.handler.ServeHTTP(rr, req)
	return rr
}

const testPassword = "password123"

// register creates a user with testPassword and returns its ID.
func (a *testApp) register(email string) uuid.UUID {
	a.t.Helper()

	rr := a.do(http.MethodPost, "/register", "", map[string]string{"email": email, "name": "Test", "password": testPassword})
	require.Equal(a.t, http.StatusCreated, rr.Code, rr.Body.String())

	var id string
	require.NoError(a.t, a.db.Raw("SELECT id FROM users WHERE email = ?", email).Row().Scan(&id))
	return uuid.MustParse(id)
}

func (a *testApp) login(email, password string) auth.Tokens {
	a.t.Helper()

	rr := a.do(http.MethodPost, "/login", "", map[string]string{"email": email, "password": password})
	require.Equal(a.t, http.StatusOK, rr.Code, rr.Body.String())
	var tokens auth.Tokens
	require.NoError(a.t, json.NewDecoder(rr.Body).Decode(&tokens))
	return tokens
}

// authorized makes an authenticated request and returns its status.
func (a *testApp) authorized(token string, userID uuid.UUID) int {
	a.t.Helper()
	return a.do(http.MethodGet, "/users/"+userID.String(), token, nil).Code
}

func (a *testApp) refresh(rt auth.RefreshToken) int {
	a.t.Helper()
	return a.do(http.MethodPost, "/refresh", "", rt).Code
}

// requireRefreshRejected checks that the refresh token issues no new tokens. The response is 401
// whatever the reason (session deleted, user deleted, revoked), so it doesn't reveal whether the session existed.
func (a *testApp) requireRefreshRejected(rt auth.RefreshToken) {
	a.t.Helper()
	require.Equal(a.t, http.StatusUnauthorized, a.refresh(rt))
}

func newDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, _, cleanup := testDB.CreateIsolatedDB(t)
	t.Cleanup(cleanup)
	return db
}

func uniqueEmail() string {
	return uuid.NewString() + "@example.com"
}

func TestRevocation_Logout(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, newDB(t), redistest.NewClient(t, redisAddr))
	email := uniqueEmail()
	userID := app.register(email)
	first := app.login(email, testPassword)
	second := app.login(email, testPassword)
	require.Equal(t, http.StatusOK, app.authorized(first.AccessToken, userID))

	rr := app.do(http.MethodDelete, fmt.Sprintf("/sessions/%s?user_id=%s", first.RefreshToken.SessionID, userID), first.AccessToken, nil)
	require.Equal(t, http.StatusNoContent, rr.Code, rr.Body.String())

	require.Equal(t, http.StatusUnauthorized, app.authorized(first.AccessToken, userID), "the access token is rejected at once")
	app.requireRefreshRejected(first.RefreshToken)
	require.Equal(t, http.StatusOK, app.authorized(second.AccessToken, userID), "other sessions stay valid")
}

func TestRevocation_LogoutOfAnotherUsersSession(t *testing.T) {
	t.Parallel()

	redisClient := redistest.NewClient(t, redisAddr)
	app := newTestApp(t, newDB(t), redisClient)
	attackerEmail, victimEmail := uniqueEmail(), uniqueEmail()
	attackerID := app.register(attackerEmail)
	victimID := app.register(victimEmail)
	attacker := app.login(attackerEmail, testPassword)
	victim := app.login(victimEmail, testPassword)

	// The attacker passes their own user ID to satisfy the permission check.
	rr := app.do(http.MethodDelete, fmt.Sprintf("/sessions/%s?user_id=%s", victim.RefreshToken.SessionID, attackerID),
		attacker.AccessToken, nil)
	require.Equal(t, http.StatusNotFound, rr.Code, rr.Body.String())

	require.Equal(t, http.StatusOK, app.authorized(victim.AccessToken, victimID))
	require.Equal(t, http.StatusOK, app.refresh(victim.RefreshToken))
	// The owner is checked before writing, so foreign or random session IDs leave no entries in Redis.
	keys, err := redisClient.Keys(context.Background(), "revocation:{"+attackerID.String()+"}*").Result()
	require.NoError(t, err)
	require.Empty(t, keys)
}

func TestRevocation_LogoutEverywhere(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, newDB(t), redistest.NewClient(t, redisAddr))
	email := uniqueEmail()
	userID := app.register(email)
	first := app.login(email, testPassword)
	second := app.login(email, testPassword)

	rr := app.do(http.MethodDelete, "/sessions?user_id="+userID.String(), first.AccessToken, nil)
	require.Equal(t, http.StatusNoContent, rr.Code, rr.Body.String())

	for _, tokens := range []auth.Tokens{first, second} {
		require.Equal(t, http.StatusUnauthorized, app.authorized(tokens.AccessToken, userID))
		app.requireRefreshRejected(tokens.RefreshToken)
	}

	// A login after the revocation is a new session and works.
	fresh := app.login(email, testPassword)
	require.Equal(t, http.StatusOK, app.authorized(fresh.AccessToken, userID))
	require.Equal(t, http.StatusOK, app.refresh(fresh.RefreshToken))
}

func TestRevocation_PasswordChange(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, newDB(t), redistest.NewClient(t, redisAddr))
	email := uniqueEmail()
	userID := app.register(email)
	tokens := app.login(email, testPassword)

	rr := app.do(http.MethodPost, "/users/"+userID.String()+"/password", tokens.AccessToken,
		map[string]string{"old_password": testPassword, "new_password": "newpassword123"})
	require.Equal(t, http.StatusNoContent, rr.Code, rr.Body.String())

	require.Equal(t, http.StatusUnauthorized, app.authorized(tokens.AccessToken, userID))
	app.requireRefreshRejected(tokens.RefreshToken)

	fresh := app.login(email, "newpassword123")
	require.Equal(t, http.StatusOK, app.authorized(fresh.AccessToken, userID))
}

func TestRevocation_UserDeletion(t *testing.T) {
	t.Parallel()

	db := newDB(t)
	app := newTestApp(t, db, redistest.NewClient(t, redisAddr))
	adminEmail, email := uniqueEmail(), uniqueEmail()
	adminID := app.register(adminEmail)
	require.NoError(t, db.Exec("INSERT INTO user_roles (user_id, role) VALUES (?, ?)", adminID, auth.RoleAdmin).Error)
	admin := app.login(adminEmail, testPassword)
	userID := app.register(email)
	tokens := app.login(email, testPassword)

	rr := app.do(http.MethodDelete, "/users/"+userID.String(), admin.AccessToken, nil)
	require.Equal(t, http.StatusNoContent, rr.Code, rr.Body.String())

	require.Equal(t, http.StatusUnauthorized, app.authorized(tokens.AccessToken, userID))
	app.requireRefreshRejected(tokens.RefreshToken)
}

// TestRevocation_DatabaseStepFailed covers a revocation written to Redis whose deletion in PostgreSQL failed:
// the session is still in the database, and its refresh token must not issue a new access token.
func TestRevocation_DatabaseStepFailed(t *testing.T) {
	t.Parallel()

	redisClient := redistest.NewClient(t, redisAddr)
	app := newTestApp(t, newDB(t), redisClient)
	store, err := revocation.NewStore(redisClient, time.Hour, time.Second)
	require.NoError(t, err)

	t.Run("session", func(t *testing.T) {
		email := uniqueEmail()
		userID := app.register(email)
		tokens := app.login(email, testPassword)

		require.NoError(t, store.RevokeSession(context.Background(), userID, tokens.RefreshToken.SessionID))

		require.Equal(t, http.StatusUnauthorized, app.authorized(tokens.AccessToken, userID))
		app.requireRefreshRejected(tokens.RefreshToken)
	})
	t.Run("user", func(t *testing.T) {
		email := uniqueEmail()
		userID := app.register(email)
		tokens := app.login(email, testPassword)

		require.NoError(t, store.RevokeUser(context.Background(), userID, time.Now()))

		require.Equal(t, http.StatusUnauthorized, app.authorized(tokens.AccessToken, userID))
		app.requireRefreshRejected(tokens.RefreshToken)
	})
}

// TestRevocation_RedisUnavailable checks the fallback to PostgreSQL: with Redis down,
// valid tokens keep working and tokens of sessions deleted in the database are rejected.
func TestRevocation_RedisUnavailable(t *testing.T) {
	t.Parallel()

	db := newDB(t)
	app := newTestApp(t, db, redistest.NewClient(t, redisAddr))
	deadRedis := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", ContextTimeoutEnabled: true, MaxRetries: -1})
	t.Cleanup(func() { _ = deadRedis.Close() }) //nolint:errcheck
	appWithoutRedis := newTestApp(t, db, deadRedis)

	email := uniqueEmail()
	userID := app.register(email)
	first := app.login(email, testPassword)
	second := app.login(email, testPassword)

	require.Equal(t, http.StatusOK, appWithoutRedis.authorized(first.AccessToken, userID))

	// A revocation can't be written without Redis: it fails and changes nothing.
	rr := appWithoutRedis.do(http.MethodDelete,
		fmt.Sprintf("/sessions/%s?user_id=%s", first.RefreshToken.SessionID, userID), first.AccessToken, nil)
	require.Equal(t, http.StatusInternalServerError, rr.Code, rr.Body.String())
	require.Equal(t, http.StatusOK, appWithoutRedis.authorized(first.AccessToken, userID))

	// Revoked while Redis was up: without Redis the database check still rejects the token.
	rr = app.do(http.MethodDelete,
		fmt.Sprintf("/sessions/%s?user_id=%s", first.RefreshToken.SessionID, userID), first.AccessToken, nil)
	require.Equal(t, http.StatusNoContent, rr.Code, rr.Body.String())
	require.Equal(t, http.StatusUnauthorized, appWithoutRedis.authorized(first.AccessToken, userID))
	require.Equal(t, http.StatusOK, appWithoutRedis.authorized(second.AccessToken, userID))
}

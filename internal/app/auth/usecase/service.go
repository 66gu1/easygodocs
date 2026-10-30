package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/66gu1/easygodocs/internal/app/auth"
	"github.com/66gu1/easygodocs/internal/app/user"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/66gu1/easygodocs/internal/infrastructure/ratelimit"
	"github.com/66gu1/easygodocs/internal/infrastructure/secure"
	"github.com/google/uuid"
)

const (
	CodeInvalidCredentials   apperr.Code = "user/invalid_credentials" //nolint:gosec
	CodeTooManyLoginAttempts apperr.Code = "auth/too_many_login_attempts"
)

func ErrInvalidPasswordOrEmail() error {
	return apperr.New("invalid password or email", CodeInvalidCredentials, apperr.ClassUnauthorized, apperr.LogLevelWarn)
}

func ErrTooManyLoginAttempts() *apperr.AppError {
	return apperr.New("too many login attempts, try again later", CodeTooManyLoginAttempts,
		apperr.ClassTooManyRequests, apperr.LogLevelWarn)
}

type Core interface {
	GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]auth.Session, error)
	GetSessionByID(ctx context.Context, id uuid.UUID) (auth.Session, string, error)
	DeleteSession(ctx context.Context, id, userID uuid.UUID) error
	DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error
	RefreshTokens(ctx context.Context, session auth.Session, refreshToken, rtHash string) (auth.Tokens, error)
	CheckSessionActive(session auth.Session) error
	IssueTokens(ctx context.Context, userID uuid.UUID, sessionVersion int) (auth.Tokens, error)
	AddUserRole(ctx context.Context, role auth.UserRole) error
	ListUserRoles(ctx context.Context, userID uuid.UUID) ([]auth.UserRole, error)
	DeleteUserRole(ctx context.Context, role auth.UserRole) error
	CheckSelfOrAdmin(ctx context.Context, targetUserID uuid.UUID) error
	CheckIsAdmin(ctx context.Context) error
	IsAdmin(ctx context.Context) (bool, error)
}

type PasswordHasher interface {
	CheckPasswordHash(hash, password []byte) error
}

type UserCore interface {
	GetUser(ctx context.Context, id uuid.UUID) (user.User, string, error)
	GetUserByEmail(ctx context.Context, email string) (user.User, string, error)
}

// LoginAttempts limits password guesses per account, see ratelimit.LoginAttempts.
type LoginAttempts interface {
	Reserve(ctx context.Context, email, ip string) (ratelimit.AttemptResult, error)
	Release(ctx context.Context, email, ip string) error
	Refund(ctx context.Context, email, ip string) error
}

// Revocations records revoked sessions and users, see revocation.Store.
type Revocations interface {
	RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error
	RevokeUser(ctx context.Context, userID uuid.UUID, at time.Time) error
	IsRevoked(ctx context.Context, userID, sessionID uuid.UUID, issuedAt time.Time) (bool, error)
}

type TimeGenerator interface {
	Now() time.Time
}

type LoginCmd struct {
	Email    string
	Password []byte `json:"-"`
	ClientIP string
}

type Service struct {
	core              Core
	userCore          UserCore
	passwordHasher    PasswordHasher
	loginAttempts     LoginAttempts
	revocations       Revocations
	timeGenerator     TimeGenerator
	dummyPasswordHash []byte
}

type Deps struct {
	Core           Core
	UserCore       UserCore
	PasswordHasher PasswordHasher
	LoginAttempts  LoginAttempts
	Revocations    Revocations
	TimeGenerator  TimeGenerator
}

// NewService creates the auth service. dummyPasswordHash must be a hash produced with the same
// cost as real password hashes: Login checks it when the email is unknown, so that response time
// doesn't reveal whether a user exists.
func NewService(deps Deps, dummyPasswordHash []byte) *Service {
	if deps.Core == nil || deps.UserCore == nil || deps.PasswordHasher == nil || deps.LoginAttempts == nil ||
		deps.Revocations == nil || deps.TimeGenerator == nil {
		panic("nil dependency")
	}
	if len(dummyPasswordHash) == 0 {
		panic("empty dummy password hash")
	}
	return &Service{
		core:              deps.Core,
		userCore:          deps.UserCore,
		passwordHasher:    deps.PasswordHasher,
		loginAttempts:     deps.LoginAttempts,
		revocations:       deps.Revocations,
		timeGenerator:     deps.TimeGenerator,
		dummyPasswordHash: dummyPasswordHash,
	}
}

func (s *Service) GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]auth.Session, error) {
	err := s.core.CheckSelfOrAdmin(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.service.GetSessionsByUserID: %w", err)
	}

	sessions, err := s.core.GetSessionsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.service.GetSessionsByUserID: %w", err)
	}
	return sessions, nil
}

// DeleteSession revokes a session (logout).
//
// Revocation writes Redis before PostgreSQL. If Redis fails, nothing has changed and the error is returned.
// If PostgreSQL fails, the access tokens are already rejected, and the session left in the database
// can't be refreshed either: RefreshTokens checks the revocation too.
func (s *Service) DeleteSession(ctx context.Context, userID, id uuid.UUID) error {
	if err := s.core.CheckSelfOrAdmin(ctx, userID); err != nil {
		return fmt.Errorf("auth.service.DeleteSession: %w", err)
	}

	// Check the owner before writing the revocation, so random IDs don't fill Redis with entries.
	session, _, err := s.core.GetSessionByID(ctx, id)
	if err != nil {
		return fmt.Errorf("auth.service.DeleteSession: %w", err)
	}
	if session.UserID != userID {
		return fmt.Errorf("auth.service.DeleteSession: %w", auth.ErrSessionNotFound())
	}

	if err = s.revocations.RevokeSession(ctx, userID, id); err != nil {
		return fmt.Errorf("auth.service.DeleteSession: %w", err)
	}
	if err = s.core.DeleteSession(ctx, id, userID); err != nil {
		return fmt.Errorf("auth.service.DeleteSession: %w", err)
	}
	return nil
}

// DeleteSessionsByUserID revokes all sessions of the user (logout everywhere). See DeleteSession for the order.
func (s *Service) DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error {
	if err := s.core.CheckSelfOrAdmin(ctx, userID); err != nil {
		return fmt.Errorf("auth.service.DeleteSessionsByUserID: %w", err)
	}

	if err := s.RevokeUser(ctx, userID); err != nil {
		return fmt.Errorf("auth.service.DeleteSessionsByUserID: %w", err)
	}
	if err := s.core.DeleteSessionsByUserID(ctx, userID); err != nil {
		return fmt.Errorf("auth.service.DeleteSessionsByUserID: %w", err)
	}
	return nil
}

// RevokeUser rejects all tokens and sessions of the user issued until now. It doesn't check permissions:
// it is called by operations that have checked them, before they change the database
// (password change, user deletion), so that a failure leaves nothing changed.
func (s *Service) RevokeUser(ctx context.Context, userID uuid.UUID) error {
	if err := s.revocations.RevokeUser(ctx, userID, s.timeGenerator.Now()); err != nil {
		return fmt.Errorf("auth.service.RevokeUser: %w", err)
	}
	return nil
}

func (s *Service) AddUserRole(ctx context.Context, userRole auth.UserRole) error {
	if err := s.core.CheckIsAdmin(ctx); err != nil {
		return fmt.Errorf("auth.service.AddUserRole: %w", err)
	}

	if err := s.core.AddUserRole(ctx, userRole); err != nil {
		return fmt.Errorf("auth.service.AddUserRole: %w", err)
	}
	return nil
}

func (s *Service) DeleteUserRole(ctx context.Context, role auth.UserRole) error {
	if err := s.core.CheckIsAdmin(ctx); err != nil {
		return fmt.Errorf("auth.service.DeleteUserRole: %w", err)
	}

	if err := s.core.DeleteUserRole(ctx, role); err != nil {
		return fmt.Errorf("auth.service.DeleteUserRole.DeleteUserRole: %w", err)
	}
	return nil
}

func (s *Service) ListUserRoles(ctx context.Context, userID uuid.UUID) ([]auth.UserRole, error) {
	if err := s.core.CheckSelfOrAdmin(ctx, userID); err != nil {
		return nil, fmt.Errorf("auth.service.ListUserRoles: %w", err)
	}

	roles, err := s.core.ListUserRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.service.ListUserRoles: %w", err)
	}
	return roles, nil
}

func (s *Service) RefreshTokens(ctx context.Context, refreshToken auth.RefreshToken) (auth.Tokens, error) {
	if refreshToken.Token == "" {
		err := apperr.ErrBadRequest()
		return auth.Tokens{}, fmt.Errorf("auth.service.RefreshTokens: %w", err)
	}

	session, rtHash, err := s.core.GetSessionByID(ctx, refreshToken.SessionID)
	if err != nil {
		return auth.Tokens{}, fmt.Errorf("auth.service.RefreshTokens: %w", notFoundAsUnauthorized(err))
	}

	if err = s.checkSessionUser(ctx, session); err != nil {
		return auth.Tokens{}, fmt.Errorf("auth.service.RefreshTokens: %w", notFoundAsUnauthorized(err))
	}
	if err = s.checkSessionNotRevoked(ctx, session); err != nil {
		return auth.Tokens{}, fmt.Errorf("auth.service.RefreshTokens: %w", err)
	}

	// The rotation is a compare-and-swap: a refresh token already used by a concurrent request
	// fails it with "not found".
	tokens, err := s.core.RefreshTokens(ctx, session, refreshToken.Token, rtHash)
	if err != nil {
		return auth.Tokens{}, fmt.Errorf("auth.service.RefreshTokens: %w", notFoundAsUnauthorized(err))
	}

	return tokens, nil
}

// Login checks the credentials and issues tokens. The attempt is reserved before the user is looked up,
// so it is counted the same way for unknown emails and the limit doesn't reveal which emails exist.
// A locked out client gets 429 without the password being checked.
//
// The attempt is spent only on a wrong password or an unknown email. If the login fails before
// the password is checked (a database error, a timeout), the attempt is refunded, so server errors
// don't lock users out.
func (s *Service) Login(ctx context.Context, req LoginCmd) (auth.Tokens, error) {
	defer secure.ZeroBytes(req.Password)

	reserved, err := s.reserveLoginAttempt(ctx, req)
	if err != nil {
		return auth.Tokens{}, fmt.Errorf("auth.service.Login: %w", err)
	}

	tokens, err := s.authenticate(ctx, req)
	if err != nil {
		if reserved && !errors.Is(err, ErrInvalidPasswordOrEmail()) {
			s.finishLoginAttempt(ctx, req, s.loginAttempts.Refund)
		}
		return auth.Tokens{}, fmt.Errorf("auth.service.Login: %w", err)
	}

	if reserved {
		s.finishLoginAttempt(ctx, req, s.loginAttempts.Release)
	}

	return tokens, nil
}

// reserveLoginAttempt fails open: if Redis is unavailable, the attempt is allowed without being reserved
// and the error is logged, so a Redis outage doesn't stop everyone from logging in. The per-IP limit
// of public endpoints is in Redis too and fails open as well, so during an outage guessing is bounded
// only by the cost of bcrypt.
func (s *Service) reserveLoginAttempt(ctx context.Context, req LoginCmd) (reserved bool, err error) {
	res, err := s.loginAttempts.Reserve(ctx, req.Email, req.ClientIP)
	if err != nil {
		logger.Error(ctx, err).Msg("auth.service.Login: login attempts limiter is unavailable, attempt is allowed")
		return false, nil
	}
	if !res.Allowed {
		return false, ErrTooManyLoginAttempts().WithRetryAfter(res.RetryAfter)
	}
	return true, nil
}

// finishLoginAttempt releases or refunds a reserved attempt. Its failure is only logged:
// it must not change the result of the login.
func (s *Service) finishLoginAttempt(ctx context.Context, req LoginCmd, finish func(ctx context.Context, email, ip string) error) {
	if err := finish(context.WithoutCancel(ctx), req.Email, req.ClientIP); err != nil {
		logger.Error(ctx, err).Msg("auth.service.Login: failed to finish login attempt")
	}
}

func (s *Service) authenticate(ctx context.Context, req LoginCmd) (auth.Tokens, error) {
	usr, passwordHash, err := s.userCore.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound()) {
			// Spend the same time as for an existing user to prevent user enumeration.
			_ = s.passwordHasher.CheckPasswordHash(s.dummyPasswordHash, req.Password) //nolint:errcheck
			err = ErrInvalidPasswordOrEmail()
		}
		return auth.Tokens{}, fmt.Errorf("authenticate: %w", err)
	}

	if err = s.passwordHasher.CheckPasswordHash([]byte(passwordHash), req.Password); err != nil {
		if errors.Is(err, secure.ErrMismatchedHashAndPassword) {
			err = ErrInvalidPasswordOrEmail()
		}
		return auth.Tokens{}, fmt.Errorf("authenticate: %w", err)
	}

	tokens, err := s.core.IssueTokens(ctx, usr.ID, usr.SessionVersion)
	if err != nil {
		return auth.Tokens{}, fmt.Errorf("authenticate: %w", err)
	}

	return tokens, nil
}

// ValidateSession checks that the session behind an access token is still active: it exists,
// belongs to the user, hasn't expired, and wasn't invalidated by a password change or user deletion.
func (s *Service) ValidateSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	session, _, err := s.core.GetSessionByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("auth.service.ValidateSession: %w", err)
	}
	if session.UserID != userID {
		err = apperr.ErrUnauthorized().WithDetail("session belongs to another user")
		return fmt.Errorf("auth.service.ValidateSession: %w", err)
	}
	if err = s.core.CheckSessionActive(session); err != nil {
		return fmt.Errorf("auth.service.ValidateSession: %w", err)
	}
	if err = s.checkSessionUser(ctx, session); err != nil {
		return fmt.Errorf("auth.service.ValidateSession: %w", err)
	}

	return nil
}

// notFoundAsUnauthorized hides why a refresh was rejected: a missing session or user gets the same 401
// as an invalid token, so the response doesn't reveal whether a session existed.
// The reason stays in the error detail for the logs.
func notFoundAsUnauthorized(err error) error {
	if apperr.ClassOf(err) == apperr.ClassNotFound {
		return apperr.ErrUnauthorized().WithDetail(err.Error())
	}
	return err
}

// checkSessionNotRevoked catches a session that was revoked in Redis, but whose deletion
// in the database failed: without it, its refresh token would issue access tokens newer than the revocation.
//
// It fails open: the database is the source of truth and has already been checked, the revocation entry
// only matters when the database step of a revocation has failed.
func (s *Service) checkSessionNotRevoked(ctx context.Context, session auth.Session) error {
	revoked, err := s.revocations.IsRevoked(ctx, session.UserID, session.ID, session.CreatedAt)
	if err != nil {
		logger.Error(ctx, err).Msg("auth.service.RefreshTokens: revocations are unavailable, refresh is allowed")
		return nil
	}
	if revoked {
		return fmt.Errorf("checkSessionNotRevoked: %w", apperr.ErrUnauthorized().WithDetail("session is revoked"))
	}
	return nil
}

// checkSessionUser checks that the session owner still exists and the session
// wasn't invalidated by a session version bump (e.g. after a password change).
func (s *Service) checkSessionUser(ctx context.Context, session auth.Session) error {
	usr, _, err := s.userCore.GetUser(ctx, session.UserID)
	if err != nil {
		return fmt.Errorf("checkSessionUser: %w", err)
	}
	if usr.SessionVersion != session.SessionVersion {
		return fmt.Errorf("checkSessionUser: %w", apperr.ErrUnauthorized().WithDetail("session version mismatch"))
	}

	return nil
}

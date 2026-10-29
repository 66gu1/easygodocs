package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/66gu1/easygodocs/internal/app/auth"
	"github.com/66gu1/easygodocs/internal/app/user"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/secure"
	"github.com/google/uuid"
)

const CodeInvalidCredentials apperr.Code = "user/invalid_credentials" //nolint:gosec

func ErrInvalidPasswordOrEmail() error {
	return apperr.New("invalid password or email", CodeInvalidCredentials, apperr.ClassUnauthorized, apperr.LogLevelWarn)
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

type LoginCmd struct {
	Email    string
	Password []byte `json:"-"`
}

type Service struct {
	core              Core
	userCore          UserCore
	passwordHasher    PasswordHasher
	dummyPasswordHash []byte
}

// NewService creates the auth service. dummyPasswordHash must be a hash produced with the same
// cost as real password hashes: Login checks it when the email is unknown, so that response time
// doesn't reveal whether a user exists.
func NewService(core Core, userCore UserCore, passwordHasher PasswordHasher, dummyPasswordHash []byte) *Service {
	if core == nil || userCore == nil || passwordHasher == nil {
		panic("nil dependency")
	}
	if len(dummyPasswordHash) == 0 {
		panic("empty dummy password hash")
	}
	return &Service{
		core:              core,
		userCore:          userCore,
		passwordHasher:    passwordHasher,
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

func (s *Service) DeleteSession(ctx context.Context, userID, id uuid.UUID) error {
	if err := s.core.CheckSelfOrAdmin(ctx, userID); err != nil {
		return fmt.Errorf("auth.service.DeleteSession: %w", err)
	}

	if err := s.core.DeleteSession(ctx, id, userID); err != nil {
		return fmt.Errorf("auth.service.DeleteSession: %w", err)
	}
	return nil
}

func (s *Service) DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error {
	if err := s.core.CheckSelfOrAdmin(ctx, userID); err != nil {
		return fmt.Errorf("auth.service.DeleteSessionsByUserID: %w", err)
	}

	if err := s.core.DeleteSessionsByUserID(ctx, userID); err != nil {
		return fmt.Errorf("auth.service.DeleteSessionsByUserID: %w", err)
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
		return auth.Tokens{}, fmt.Errorf("auth.service.RefreshTokens: %w", err)
	}

	if err = s.checkSessionUser(ctx, session); err != nil {
		return auth.Tokens{}, fmt.Errorf("auth.service.RefreshTokens: %w", err)
	}

	tokens, err := s.core.RefreshTokens(ctx, session, refreshToken.Token, rtHash)
	if err != nil {
		return auth.Tokens{}, fmt.Errorf("auth.service.RefreshTokens: %w", err)
	}

	return tokens, nil
}

func (s *Service) Login(ctx context.Context, req LoginCmd) (auth.Tokens, error) {
	defer secure.ZeroBytes(req.Password)

	usr, passwordHash, err := s.userCore.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound()) {
			// Spend the same time as for an existing user to prevent user enumeration.
			_ = s.passwordHasher.CheckPasswordHash(s.dummyPasswordHash, req.Password) //nolint:errcheck
			err = ErrInvalidPasswordOrEmail()
		}
		return auth.Tokens{}, fmt.Errorf("auth.service.Login: %w", err)
	}

	if err = s.passwordHasher.CheckPasswordHash([]byte(passwordHash), req.Password); err != nil {
		if errors.Is(err, secure.ErrMismatchedHashAndPassword) {
			err = ErrInvalidPasswordOrEmail()
		}
		return auth.Tokens{}, fmt.Errorf("auth.service.Login: %w", err)
	}

	tokens, err := s.core.IssueTokens(ctx, usr.ID, usr.SessionVersion)
	if err != nil {
		return auth.Tokens{}, fmt.Errorf("auth.service.Login: %w", err)
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

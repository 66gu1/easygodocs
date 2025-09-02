package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/66gu1/easygodocs/internal/app/auth"
	"github.com/66gu1/easygodocs/internal/app/user"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/contextx"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/66gu1/easygodocs/internal/infrastructure/secure"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type LoginCmd struct {
	Email    string
	Password []byte `json:"-"`
}

const CodeInvalidCredentials apperr.Code = "user/invalid_credentials" //nolint:gosec

func ErrInvalidPasswordOrEmail() error {
	return apperr.New("invalid password or email", CodeInvalidCredentials, apperr.ClassUnauthorized, apperr.LogLevelWarn)
}

type Service struct {
	core     Core
	userCore UserCore
}

type Core interface {
	GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]auth.Session, error)
	GetSessionByID(ctx context.Context, id uuid.UUID) (auth.Session, string, error)
	DeleteSession(ctx context.Context, id, userID uuid.UUID, isAdmin bool) error
	DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error
	RefreshTokens(ctx context.Context, session auth.Session, refreshToken, rtHash string) (auth.Tokens, error)
	IssueTokens(ctx context.Context, userID uuid.UUID, sessionVersion int) (auth.Tokens, error)
	AddUserRole(ctx context.Context, role auth.UserRole) error
	ListUserRoles(ctx context.Context, userID uuid.UUID) ([]auth.UserRole, error)
	DeleteUserRole(ctx context.Context, role auth.UserRole) error
	CheckSelfOrAdmin(ctx context.Context, targetUserID uuid.UUID) error
	CheckIsAdmin(ctx context.Context) error
	IsAdmin(ctx context.Context) (bool, error)
}

type UserCore interface {
	GetUser(ctx context.Context, id uuid.UUID) (user.User, string, error)
	GetUserByEmail(ctx context.Context, email string) (user.User, string, error)
}

func NewService(core Core, userCore UserCore) *Service {
	if core == nil || userCore == nil {
		panic("nil core")
	}
	return &Service{
		core:     core,
		userCore: userCore,
	}
}

func (s *Service) GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]auth.Session, error) {
	err := s.core.CheckSelfOrAdmin(ctx, userID)
	if err != nil {
		logger.Error(ctx, err).
			Str(auth.FieldUserID.String(), userID.String()).
			Msg("auth.service.GetSessionsByUserID.core.CheckSelfOrAdmin")
		return nil, fmt.Errorf("auth.service.GetSessionsByUserID: %w", err)
	}

	sessions, err := s.core.GetSessionsByUserID(ctx, userID)
	if err != nil {
		logger.Error(ctx, err).
			Str(auth.FieldUserID.String(), userID.String()).
			Msg("auth.service.GetSessionsByUserID.core.GetSessionsByUserID")
		return nil, fmt.Errorf("auth.service.GetSessionsByUserID: %w", err)
	}
	return sessions, nil
}

func (s *Service) DeleteSession(ctx context.Context, id uuid.UUID) error {
	currentUserID, err := contextx.GetUserID(ctx)
	if err != nil {
		logger.Error(ctx, err).
			Str(auth.FieldSessionID.String(), id.String()).
			Msg("auth.service.DeleteSession.contextx.GetUserID")
		return fmt.Errorf("auth.service.DeleteSession: %w", err)
	}
	isAdmin, err := s.core.IsAdmin(ctx)
	if err != nil {
		logger.Error(ctx, err).
			Str(auth.FieldSessionID.String(), id.String()).
			Msg("auth.service.DeleteSession.core.IsAdmin")
		return fmt.Errorf("auth.service.DeleteSession: %w", err)
	}

	if err = s.core.DeleteSession(ctx, id, currentUserID, isAdmin); err != nil {
		logger.Error(ctx, err).
			Str(auth.FieldSessionID.String(), id.String()).
			Msg("auth.service.DeleteSession.core.DeleteSession")
		return fmt.Errorf("auth.service.DeleteSession: %w", err)
	}
	return nil
}

func (s *Service) DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error {
	if err := s.core.CheckSelfOrAdmin(ctx, userID); err != nil {
		logger.Error(ctx, err).
			Str(auth.FieldUserID.String(), userID.String()).
			Msg("auth.service.DeleteSessionsByUserID.core.CheckSelfOrAdmin")
		return fmt.Errorf("auth.service.DeleteSessionsByUserID: %w", err)
	}

	if err := s.core.DeleteSessionsByUserID(ctx, userID); err != nil {
		logger.Error(ctx, err).
			Str(auth.FieldUserID.String(), userID.String()).
			Msg("auth.service.DeleteSessionsByUserID.core.DeleteSessionsByUserID")
		return fmt.Errorf("auth.service.DeleteSessionsByUserID: %w", err)
	}
	return nil
}

func (s *Service) AddUserRole(ctx context.Context, userRole auth.UserRole) error {
	if err := s.core.CheckIsAdmin(ctx); err != nil {
		logger.Error(ctx, err).
			Interface(auth.FieldUserRole.String(), userRole).
			Msg("auth.service.AddUserRole.core.CheckIsAdmin")
		return fmt.Errorf("auth.service.AddUserRole: %w", err)
	}

	if err := s.core.AddUserRole(ctx, userRole); err != nil {
		if errors.Is(err, auth.ErrInvalidRole) {
			err = apperr.New("invalid role", auth.CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
				WithViolation(apperr.Violation{Field: auth.FieldRole, Rule: apperr.RuleInvalidFormat})
		}
		logger.Error(ctx, err).
			Interface(auth.FieldUserRole.String(), userRole).
			Msg("auth.service.AddUserRole.core.AddUserRole")
		return fmt.Errorf("auth.service.AddUserRole.AddUserRole: %w", err)
	}
	return nil
}

func (s *Service) DeleteUserRole(ctx context.Context, role auth.UserRole) error {
	if err := s.core.CheckIsAdmin(ctx); err != nil {
		logger.Error(ctx, err).
			Interface(auth.FieldUserRole.String(), role).
			Msg("auth.service.DeleteUserRole.core.CheckIsAdmin")
		return fmt.Errorf("auth.service.DeleteUserRole: %w", err)
	}

	if err := s.core.DeleteUserRole(ctx, role); err != nil {
		logger.Error(ctx, err).
			Interface(auth.FieldUserRole.String(), role).
			Msg("auth.service.DeleteUserRole.core.DeleteUserRole")
		return fmt.Errorf("auth.service.DeleteUserRole.DeleteUserRole: %w", err)
	}
	return nil
}

func (s *Service) ListUserRoles(ctx context.Context, userID uuid.UUID) ([]auth.UserRole, error) {
	if err := s.core.CheckSelfOrAdmin(ctx, userID); err != nil {
		logger.Error(ctx, err).
			Str(auth.FieldUserID.String(), userID.String()).
			Msg("auth.service.ListUserRoles.core.CheckSelfOrAdmin")
		return nil, fmt.Errorf("auth.service.ListUserRoles: %w", err)
	}

	roles, err := s.core.ListUserRoles(ctx, userID)
	if err != nil {
		logger.Error(ctx, err).
			Str(auth.FieldUserID.String(), userID.String()).
			Msg("auth.service.ListUserRoles.core.ListUserRoles")
		return nil, fmt.Errorf("auth.service.ListUserRoles: %w", err)
	}
	return roles, nil
}

func (s *Service) RefreshTokens(ctx context.Context, refreshToken auth.RefreshToken) (auth.Tokens, error) {
	if refreshToken.Token == "" {
		err := apperr.ErrBadRequest()
		logger.Error(ctx, err).
			Str(auth.FieldSessionID.String(), refreshToken.SessionID.String()).
			Msg("auth.service.RefreshToken: empty refresh token")
		return auth.Tokens{}, fmt.Errorf("auth.service.RefreshTokens: %w", err)
	}

	session, rtHash, err := s.core.GetSessionByID(ctx, refreshToken.SessionID)
	if err != nil {
		logger.Error(ctx, err).
			Str(auth.FieldSessionID.String(), refreshToken.SessionID.String()).
			Msg("auth.service.RefreshTokens.core.GetSessionByID")
		return auth.Tokens{}, fmt.Errorf("auth.service.RefreshTokens: %w", apperr.ErrUnauthorized())
	}

	usr, _, err := s.userCore.GetUser(ctx, session.UserID)
	if err != nil {
		logger.Error(ctx, err).
			Str(auth.FieldUserID.String(), session.UserID.String()).
			Interface(auth.FieldSession.String(), session).
			Msg("auth.service.RefreshTokens.userCore.GetUser")
		return auth.Tokens{}, fmt.Errorf("auth.service.RefreshTokens: %w", apperr.ErrUnauthorized())
	}

	if usr.SessionVersion != session.SessionVersion {
		err = apperr.ErrUnauthorized()
		logger.Error(ctx, err).
			Str(auth.FieldUserID.String(), session.UserID.String()).
			Interface(auth.FieldSession.String(), session).
			Msg("auth.service.RefreshTokens: session version mismatch")
		return auth.Tokens{}, fmt.Errorf("auth.service.RefreshTokens: %w", err)
	}

	tokens, err := s.core.RefreshTokens(ctx, session, refreshToken.Token, rtHash)
	if err != nil {
		logger.Error(ctx, err).
			Str(auth.FieldSessionID.String(), refreshToken.SessionID.String()).
			Interface(auth.FieldSession.String(), session).
			Msg("auth.service.RefreshTokens.core.RefreshTokens")
		return auth.Tokens{}, fmt.Errorf("auth.service.RefreshTokens: %w", apperr.ErrUnauthorized())
	}

	return tokens, nil
}

func (s *Service) Login(ctx context.Context, req LoginCmd) (auth.Tokens, error) {
	defer secure.ZeroBytes(req.Password)

	usr, passwordHash, err := s.userCore.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound()) {
			err = ErrInvalidPasswordOrEmail()
		}
		logger.Error(ctx, err).
			Str(user.FieldEmail.String(), req.Email).
			Msg("auth.service.Login.userCore.GetAuthByEmail")
		return auth.Tokens{}, fmt.Errorf("auth.service.Login: %w", err)
	}

	if !checkPassword(req.Password, passwordHash) {
		err = ErrInvalidPasswordOrEmail()
		logger.Error(ctx, err).
			Str(user.FieldEmail.String(), req.Email).
			Interface(user.FieldUser.String(), usr).
			Msg("auth.service.Login: invalid password")
		return auth.Tokens{}, fmt.Errorf("auth.service.Login: %w", err)
	}

	tokens, err := s.core.IssueTokens(ctx, usr.ID, usr.SessionVersion)
	if err != nil {
		logger.Error(ctx, err).
			Str(user.FieldEmail.String(), req.Email).
			Interface(user.FieldUser.String(), usr).
			Msg("auth.service.Login.core.IssueTokens")
		return auth.Tokens{}, fmt.Errorf("auth.service.Login: %w", err)
	}

	return tokens, nil
}

func checkPassword(password []byte, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), password)
	return err == nil
}

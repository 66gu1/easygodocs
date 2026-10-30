package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/66gu1/easygodocs/internal/app/user"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/66gu1/easygodocs/internal/infrastructure/secure"
	"github.com/google/uuid"
)

func ErrNewPasswordSameAsOld() error {
	return apperr.New("New password must differ from the old one", user.CodeSamePassword, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{Field: user.FieldPassword, Rule: apperr.RuleDuplicate})
}

func ErrOldPasswordIncorrect() error {
	return apperr.New("Old password does not match", user.CodePasswordMismatch, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{Field: user.FieldPassword, Rule: apperr.RuleMismatch})
}

func ErrOldPasswordRequired() error {
	return apperr.New("Old password is required", user.CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{Field: user.FieldPassword, Rule: apperr.RuleRequired})
}

type Core interface {
	CreateUser(ctx context.Context, req user.CreateUserReq) (uuid.UUID, error)
	GetUser(ctx context.Context, id uuid.UUID) (user.User, string, error)
	GetAllUsers(ctx context.Context) ([]user.User, error)
	UpdateUser(ctx context.Context, req user.UpdateUserReq) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	ChangePassword(ctx context.Context, id uuid.UUID, newPassword []byte) error
}

type AuthService interface {
	CheckSelfOrAdmin(ctx context.Context, targetUserID uuid.UUID) error
	CheckIsAdmin(ctx context.Context) error
	DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteUserRolesByUserID(ctx context.Context, userID uuid.UUID) error
	CheckSelf(ctx context.Context, targetUserID uuid.UUID) error
	IsAdmin(ctx context.Context) (bool, error)
}

// TokenRevoker rejects all tokens and sessions of a user issued until now.
type TokenRevoker interface {
	RevokeUser(ctx context.Context, userID uuid.UUID) error
}

type PasswordHasher interface {
	CheckPasswordHash(hash, password []byte) error
}

type ChangePasswordCmd struct {
	ID          uuid.UUID
	OldPassword []byte
	NewPassword []byte
}

type Service struct {
	core           Core
	authService    AuthService
	tokenRevoker   TokenRevoker
	passwordHasher PasswordHasher
}

func NewService(core Core, authService AuthService, tokenRevoker TokenRevoker, passwordHasher PasswordHasher) *Service {
	if core == nil || authService == nil || tokenRevoker == nil || passwordHasher == nil {
		panic("user.NewService: nil dependency")
	}
	return &Service{
		core:           core,
		authService:    authService,
		tokenRevoker:   tokenRevoker,
		passwordHasher: passwordHasher,
	}
}

func (s *Service) CreateUser(ctx context.Context, req user.CreateUserReq) error {
	if _, err := s.core.CreateUser(ctx, req); err != nil {
		return fmt.Errorf("user.service.CreateUser: %w", err)
	}

	return nil
}

func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (user.User, error) {
	if err := s.authService.CheckSelfOrAdmin(ctx, id); err != nil {
		return user.User{}, fmt.Errorf("user.service.GetUser: %w", err)
	}

	u, _, err := s.core.GetUser(ctx, id)
	if err != nil {
		return user.User{}, fmt.Errorf("user.service.GetUser: %w", err)
	}
	return u, nil
}

func (s *Service) GetAllUsers(ctx context.Context) ([]user.User, error) {
	if err := s.authService.CheckIsAdmin(ctx); err != nil {
		return nil, fmt.Errorf("user.service.GetAllUsers: %w", err)
	}
	users, err := s.core.GetAllUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("user.service.GetAllUsers: %w", err)
	}
	return users, nil
}

func (s *Service) UpdateUser(ctx context.Context, req user.UpdateUserReq) error {
	if err := s.authService.CheckSelfOrAdmin(ctx, req.UserID); err != nil {
		return fmt.Errorf("user.service.UpdateUser: %w", err)
	}

	if err := s.core.UpdateUser(ctx, req); err != nil {
		return fmt.Errorf("user.service.UpdateUser: %w", err)
	}
	return nil
}

func (s *Service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if err := s.authService.CheckIsAdmin(ctx); err != nil {
		return fmt.Errorf("user.service.DeleteUser: %w", err)
	}

	// Revoke the user's tokens before deleting: if it fails, nothing has changed.
	// After the deletion, refresh is rejected anyway, because the user doesn't exist.
	if err := s.tokenRevoker.RevokeUser(ctx, id); err != nil {
		return fmt.Errorf("user.service.DeleteUser: %w", err)
	}
	if err := s.core.DeleteUser(ctx, id); err != nil {
		return fmt.Errorf("user.service.DeleteUser: %w", err)
	}

	// Best-effort cleanup: failures are only logged, because a deleted user
	// can't authenticate anyway: the tokens are revoked and refresh checks that the user exists.
	cleanupCtx := context.WithoutCancel(ctx)
	if err := s.authService.DeleteSessionsByUserID(cleanupCtx, id); err != nil {
		logger.Error(ctx, err).
			Str(user.FieldUserID.String(), id.String()).
			Msg("user.service.DeleteUser: failed to delete sessions")
	}
	if err := s.authService.DeleteUserRolesByUserID(cleanupCtx, id); err != nil {
		logger.Error(ctx, err).
			Str(user.FieldUserID.String(), id.String()).
			Msg("user.service.DeleteUser: failed to delete roles")
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, req ChangePasswordCmd) error {
	isAdmin, err := s.authService.IsAdmin(ctx)
	if err != nil {
		return fmt.Errorf("user.service.ChangePassword: %w", err)
	}

	if !isAdmin {
		if len(req.OldPassword) == 0 {
			err = ErrOldPasswordRequired()
			return fmt.Errorf("user.service.ChangePassword: %w", err)
		}

		err = s.authService.CheckSelf(ctx, req.ID)
		if err != nil {
			return fmt.Errorf("user.service.ChangePassword: %w", err)
		}
	}

	_, oldPasswordHash, err := s.core.GetUser(ctx, req.ID)
	if err != nil {
		return fmt.Errorf("user.service.ChangePassword: %w", err)
	}

	if !isAdmin {
		if err = s.passwordHasher.CheckPasswordHash([]byte(oldPasswordHash), req.OldPassword); err != nil {
			if errors.Is(err, secure.ErrMismatchedHashAndPassword) {
				err = ErrOldPasswordIncorrect()
			}
			return fmt.Errorf("user.service.ChangePassword: %w", err)
		}
	}

	err = s.passwordHasher.CheckPasswordHash([]byte(oldPasswordHash), req.NewPassword)
	if err != nil && !errors.Is(err, secure.ErrMismatchedHashAndPassword) {
		return fmt.Errorf("user.service.ChangePassword: %w", err)
	}
	if err == nil {
		err = ErrNewPasswordSameAsOld()
		return fmt.Errorf("user.service.ChangePassword: %w", err)
	}

	// Revoke the user's tokens before changing the password: if it fails, nothing has changed.
	// After the change, refresh is rejected anyway, because the session version no longer matches.
	if err = s.tokenRevoker.RevokeUser(ctx, req.ID); err != nil {
		return fmt.Errorf("user.service.ChangePassword: %w", err)
	}
	err = s.core.ChangePassword(ctx, req.ID, req.NewPassword)
	if err != nil {
		return fmt.Errorf("user.service.ChangePassword: %w", err)
	}

	// Best-effort: session cleanup is attempted, but failures are ignored
	// since the tokens are revoked, session_version prevents refresh, and the user
	// should not see an error if this step fails.
	if err = s.authService.DeleteSessionsByUserID(context.WithoutCancel(ctx), req.ID); err != nil {
		logger.Error(ctx, err).
			Str(user.FieldUserID.String(), req.ID.String()).
			Msg("user.service.ChangePassword: failed to delete sessions")
	}

	return nil
}

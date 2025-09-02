package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/66gu1/easygodocs/internal/app/user"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type ChangePasswordCmd struct {
	ID          uuid.UUID
	OldPassword []byte
	NewPassword []byte
}

type service struct {
	core        Core
	authService AuthService
}

type Core interface {
	CreateUser(ctx context.Context, req user.CreateUserReq) error
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
	CheckSelf(ctx context.Context, targetUserID uuid.UUID) error
	IsAdmin(ctx context.Context) (bool, error)
}

func NewService(core Core, authService AuthService) *service {
	return &service{
		core:        core,
		authService: authService,
	}
}

func (s *service) CreateUser(ctx context.Context, req user.CreateUserReq) error {
	if err := s.core.CreateUser(ctx, req); err != nil {
		logger.Error(ctx, err).
			Str(user.FieldEmail.String(), req.Email).
			Str(user.FieldName.String(), req.Name).
			Msg("user.Service.CreateUser: failed to create user")
		return fmt.Errorf("user.Service.CreateUser: %w", err)
	}

	return nil
}

func (s *service) GetUser(ctx context.Context, id uuid.UUID) (user.User, error) {
	if err := s.authService.CheckSelfOrAdmin(ctx, id); err != nil {
		logger.Error(ctx, err).
			Str(user.FieldUserID.String(), id.String()).
			Msg("user.Service.GetUser: failed to check self or admin")
		return user.User{}, fmt.Errorf("user.Service.GetUser: %w", err)
	}

	u, _, err := s.core.GetUser(ctx, id)
	if err != nil {
		logger.Error(ctx, err).
			Str(user.FieldUserID.String(), id.String()).
			Msg("user.Service.GetUser: failed to get user")
		return user.User{}, fmt.Errorf("user.Service.GetUser: %w", err)
	}
	return u, nil
}

func (s *service) GetAllUsers(ctx context.Context) ([]user.User, error) {
	if err := s.authService.CheckIsAdmin(ctx); err != nil {
		logger.Error(ctx, err).Msg("user.Service.GetAllUsers: failed to check admin")
		return nil, fmt.Errorf("user.Service.GetAllUsers: %w", err)
	}
	users, err := s.core.GetAllUsers(ctx)
	if err != nil {
		logger.Error(ctx, err).Msg("user.Service.GetAllUsers: failed to get all users")
		return nil, fmt.Errorf("user.Service.GetAllUsers: %w", err)
	}
	return users, nil
}

func (s *service) UpdateUser(ctx context.Context, req user.UpdateUserReq) error {
	if err := s.authService.CheckSelfOrAdmin(ctx, req.UserID); err != nil {
		logger.Error(ctx, err).
			Interface(apperr.FieldRequest.String(), req).
			Msg("user.Service.UpdateUser: failed to check self or admin")
		return fmt.Errorf("user.Service.UpdateUser: %w", err)
	}

	if err := s.core.UpdateUser(ctx, req); err != nil {
		logger.Error(ctx, err).
			Interface(apperr.FieldRequest.String(), req).
			Msg("user.Service.UpdateUser: failed to update user")
		return fmt.Errorf("user.Service.UpdateUser: %w", err)
	}
	return nil
}

func (s *service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if err := s.authService.CheckIsAdmin(ctx); err != nil {
		logger.Error(ctx, err).
			Str(user.FieldUserID.String(), id.String()).
			Msg("user.Service.DeleteUser: failed to check admin")
		return fmt.Errorf("user.Service.DeleteUser: %w", err)
	}

	if err := s.core.DeleteUser(ctx, id); err != nil {
		logger.Error(ctx, err).
			Str(user.FieldUserID.String(), id.String()).
			Msg("user.Service.DeleteUser: failed to delete user")
		return fmt.Errorf("user.Service.DeleteUser: %w", err)
	}

	if err := s.authService.DeleteSessionsByUserID(ctx, id); err != nil {
		logger.Error(ctx, err).
			Str(user.FieldUserID.String(), id.String()).
			Msg("user.Service.DeleteUser: failed to delete sessions")
		return fmt.Errorf("user.Service.DeleteUser: %w", err)
	}
	return nil
}

func (s *service) ChangePassword(ctx context.Context, req ChangePasswordCmd) error {
	isAdmin, err := s.authService.IsAdmin(ctx)
	if err != nil {
		logger.Error(ctx, err).
			Str(user.FieldUserID.String(), req.ID.String()).
			Msg("user.Service.ChangePassword: failed to check admin")
		return fmt.Errorf("user.Service.ChangePassword: %w", err)
	}

	if !isAdmin {
		if len(req.OldPassword) == 0 {
			err = apperr.New("Old password is required", user.CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
				WithViolation(apperr.Violation{
					Field: user.FieldPassword, Rule: apperr.RuleRequired,
				})
			logger.Error(ctx, err).
				Str(user.FieldUserID.String(), req.ID.String()).
				Msg("user.Service.ChangePassword: old password is required")
			return fmt.Errorf("user.Service.ChangePassword: %w", err)
		}

		err = s.authService.CheckSelf(ctx, req.ID)
		if err != nil {
			logger.Error(ctx, err).
				Str(user.FieldUserID.String(), req.ID.String()).
				Msg("user.Service.ChangePassword: failed to check self")
			return fmt.Errorf("user.Service.ChangePassword: %w", err)
		}
	}

	_, oldPasswordHash, err := s.core.GetUser(ctx, req.ID)
	if err != nil {
		logger.Error(ctx, err).
			Str(user.FieldUserID.String(), req.ID.String()).
			Msg("user.Service.ChangePassword: failed to get user")
		return fmt.Errorf("user.Service.ChangePassword: %w", err)
	}

	if !isAdmin {
		if err = bcrypt.CompareHashAndPassword([]byte(oldPasswordHash), req.OldPassword); err != nil {
			if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
				err = apperr.New("Old password does not match", user.CodePasswordMismatch, apperr.ClassBadRequest, apperr.LogLevelWarn).
					WithViolation(apperr.Violation{
						Field: user.FieldPassword, Rule: apperr.RuleMismatch,
					})
			}
			logger.Error(ctx, err).
				Str(user.FieldUserID.String(), req.ID.String()).
				Msg("user.Service.ChangePassword: old password does not match")
			return fmt.Errorf("user.Service.ChangePassword: %w", err)
		}
	}

	err = bcrypt.CompareHashAndPassword([]byte(oldPasswordHash), req.NewPassword)
	if err != nil && !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		logger.Error(ctx, err).
			Str(user.FieldUserID.String(), req.ID.String()).
			Msg("user.Service.ChangePassword: bcrypt compare failed for new password")
		return fmt.Errorf("user.Service.ChangePassword: %w", err)
	}
	if err == nil {
		err = apperr.New("New password must differ from the old one", user.CodeSamePassword, apperr.ClassBadRequest, apperr.LogLevelWarn).
			WithViolation(apperr.Violation{
				Field: user.FieldPassword, Rule: apperr.RuleDuplicate,
			})
		logger.Error(ctx, err).
			Str(user.FieldUserID.String(), req.ID.String()).
			Msg("user.Service.ChangePassword: new password must differ from old one")
		return fmt.Errorf("user.Service.ChangePassword: %w", err)
	}

	err = s.core.ChangePassword(ctx, req.ID, req.NewPassword)
	if err != nil {
		logger.Error(ctx, err).
			Str(user.FieldUserID.String(), req.ID.String()).
			Msg("user.Service.ChangePassword: failed to change password")
		return fmt.Errorf("user.Service.ChangePassword: %w", err)
	}

	// Best-effort: session cleanup is attempted, but failures are ignored
	// since session_version already invalidates old tokens and the user
	// should not see an error if this step fails.
	if err = s.authService.DeleteSessionsByUserID(context.WithoutCancel(ctx), req.ID); err != nil {
		logger.Error(ctx, err).
			Str(user.FieldUserID.String(), req.ID.String()).
			Msg("user.Service.ChangePassword: failed to delete sessions")
	}

	return nil
}

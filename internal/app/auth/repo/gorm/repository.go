package gorm

import (
	"context"
	"errors"
	"fmt"

	"github.com/66gu1/easygodocs/internal/app/auth"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

var (
	ErrSessionNotFound = apperr.New("session not found", auth.CodeSessionNotFound, apperr.ClassNotFound, apperr.LogLevelWarn)
	ErrRoleNotFound    = apperr.New("role not found", auth.CodeRoleNotFound, apperr.ClassNotFound, apperr.LogLevelWarn)
)

type gormRepo struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *gormRepo {
	return &gormRepo{db: db}
}

func (r *gormRepo) CreateSession(ctx context.Context, req auth.Session, rtHash string) error {
	model := &userSession{
		ID:               req.ID,
		UserID:           req.UserID,
		RefreshTokenHash: rtHash,
		CreatedAt:        req.CreatedAt,
		ExpiresAt:        req.ExpiresAt,
		SessionVersion:   req.SessionVersion,
	}

	err := r.db.WithContext(ctx).Create(model).Error
	if err != nil {
		return fmt.Errorf("gormRepo.CreateSession: %w", err)
	}

	return nil
}

func (r *gormRepo) GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]auth.Session, error) {
	models := make([]userSession, 0)

	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetSessionsByUserID: %w", err)
	}

	return lo.Map(models, func(s userSession, _ int) auth.Session { return s.toDTO() }), nil
}

func (r *gormRepo) GetSessionByID(ctx context.Context, id uuid.UUID) (auth.Session, string, error) {
	var model userSession
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = ErrSessionNotFound
		}
		return auth.Session{}, "", fmt.Errorf("gormRepo.GetSessionByID: %w", err)
	}

	return model.toDTO(), model.RefreshTokenHash, nil
}

func (r *gormRepo) DeleteSessionByID(ctx context.Context, id uuid.UUID) error {
	model := &userSession{ID: id}

	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(model)
	if result.Error != nil {
		return fmt.Errorf("gormRepo.DeleteSessionByID: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

func (r *gormRepo) DeleteSessionByIDAndUser(ctx context.Context, id, userID uuid.UUID) error {
	model := &userSession{ID: id}

	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?",
		id, userID).Delete(model)
	if result.Error != nil {
		return fmt.Errorf("gormRepo.DeleteSessionByIDAndUser: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

func (r *gormRepo) DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error {
	model := &userSession{}

	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(model)
	if result.Error != nil {
		return fmt.Errorf("gormRepo.DeleteSessionsByUserID: %w", result.Error)
	}

	return nil
}

func (r *gormRepo) UpdateRefreshToken(ctx context.Context, req auth.UpdateTokenReq) error {
	model := &userSession{}

	result := r.db.WithContext(ctx).Model(model).Where("id = ? AND refresh_token_hash = ? AND user_id = ?",
		req.SessionID, req.OldRefreshTokenHash, req.UserID).
		Updates(map[string]interface{}{"refresh_token_hash": req.RefreshTokenHash, "expires_at": req.ExpiresAt})
	if result.Error != nil {
		return fmt.Errorf("gormRepo.UpdateRefreshToken: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

func (r *gormRepo) AddUserRole(ctx context.Context, req auth.UserRole) error {
	if err := r.db.WithContext(ctx).Create(userRoleFromDTO(req)).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == db.DuplicateCode {
			return apperr.New("role already assigned to user",
				auth.CodeRoleDuplicate, apperr.ClassConflict, apperr.LogLevelWarn)
		}
		return fmt.Errorf("gormRepo.AddUserRole: %w", err)
	}

	return nil
}

func (r *gormRepo) GetUserRoles(ctx context.Context, userID uuid.UUID, roles []auth.Role) ([]auth.UserRole, error) {
	models := make([]userRole, 0)

	err := r.db.WithContext(ctx).Where("user_id = ? AND role IN ?", userID, roles).Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetUserRoles: %w", err)
	}

	return lo.Map(models, func(ur userRole, _ int) auth.UserRole { return ur.toDTO() }), nil
}

func (r *gormRepo) ListUserRoles(ctx context.Context, userID uuid.UUID) ([]auth.UserRole, error) {
	models := make([]userRole, 0)

	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.ListUserRoles: %w", err)
	}

	return lo.Map(models, func(ur userRole, _ int) auth.UserRole { return ur.toDTO() }), nil
}

func (r *gormRepo) DeleteUserRole(ctx context.Context, req auth.UserRole) error {
	var result *gorm.DB
	if req.EntityID == nil {
		result = r.db.WithContext(ctx).Where("user_id = ? AND role = ? AND entity_id IS NULL",
			req.UserID, req.Role).Delete(&userRole{})
	} else {
		result = r.db.WithContext(ctx).Where("user_id = ? AND role = ? AND entity_id = ?",
			req.UserID, req.Role, req.EntityID).Delete(&userRole{})
	}
	if result.Error != nil {
		return fmt.Errorf("gormRepo.DeleteUserRole: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrRoleNotFound
	}

	return nil
}

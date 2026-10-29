package gorm

import (
	"context"
	"errors"
	"fmt"

	"github.com/66gu1/easygodocs/internal/app/auth"
	"github.com/66gu1/easygodocs/internal/infrastructure/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("auth.repo.NewRepository: %w", fmt.Errorf("nil db"))
	}
	return &Repository{db: db}, nil
}

func (r *Repository) CreateSession(ctx context.Context, req auth.Session, rtHash string) error {
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
		return fmt.Errorf("auth.repo.CreateSession: %w", err)
	}

	return nil
}

func (r *Repository) GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]auth.Session, error) {
	models := make([]userSession, 0)

	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("auth.repo.GetSessionsByUserID: %w", err)
	}

	return lo.Map(models, func(s userSession, _ int) auth.Session { return s.toDTO() }), nil
}

func (r *Repository) GetSessionByID(ctx context.Context, id uuid.UUID) (auth.Session, string, error) {
	var model userSession
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = auth.ErrSessionNotFound()
		}
		return auth.Session{}, "", fmt.Errorf("auth.repo.GetSessionByID: %w", err)
	}

	return model.toDTO(), model.RefreshTokenHash, nil
}

func (r *Repository) DeleteSessionByIDAndUser(ctx context.Context, id, userID uuid.UUID) error {
	model := &userSession{ID: id}

	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?",
		id, userID).Delete(model)
	if result.Error != nil {
		return fmt.Errorf("auth.repo.DeleteSessionByIDAndUser: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("auth.repo.DeleteSessionByIDAndUser: %w", auth.ErrSessionNotFound())
	}

	return nil
}

func (r *Repository) DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error {
	model := &userSession{}

	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(model)
	if result.Error != nil {
		return fmt.Errorf("auth.repo.DeleteSessionsByUserID: %w", result.Error)
	}

	return nil
}

func (r *Repository) UpdateRefreshToken(ctx context.Context, req auth.UpdateTokenReq) error {
	model := &userSession{}

	result := r.db.WithContext(ctx).Model(model).Where("id = ? AND refresh_token_hash = ? AND user_id = ?",
		req.SessionID, req.OldRefreshTokenHash, req.UserID).
		Updates(map[string]interface{}{"refresh_token_hash": req.RefreshTokenHash, "expires_at": req.ExpiresAt})
	if result.Error != nil {
		return fmt.Errorf("auth.repo.UpdateRefreshToken: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("auth.repo.UpdateRefreshToken: %w", auth.ErrSessionNotFound())
	}

	return nil
}

func (r *Repository) AddUserRole(ctx context.Context, req auth.UserRole) error {
	if err := r.db.WithContext(ctx).Create(userRoleFromDTO(req)).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == db.DuplicateCode {
			return fmt.Errorf("auth.repo.AddUserRole: %w", auth.ErrDuplicateUserRole())
		}
		return fmt.Errorf("auth.repo.AddUserRole: %w", err)
	}

	return nil
}

func (r *Repository) GetUserRoles(ctx context.Context, userID uuid.UUID, roles []auth.Role) ([]auth.UserRole, error) {
	models := make([]userRole, 0)

	err := r.db.WithContext(ctx).Where("user_id = ? AND role IN ?", userID, roles).Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("auth.repo.GetUserRoles: %w", err)
	}

	return lo.Map(models, func(ur userRole, _ int) auth.UserRole { return ur.toDTO() }), nil
}

func (r *Repository) ListUserRoles(ctx context.Context, userID uuid.UUID) ([]auth.UserRole, error) {
	models := make([]userRole, 0)

	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("auth.repo.ListUserRoles: %w", err)
	}

	return lo.Map(models, func(ur userRole, _ int) auth.UserRole { return ur.toDTO() }), nil
}

func (r *Repository) DeleteUserRole(ctx context.Context, req auth.UserRole) error {
	var result *gorm.DB
	if req.EntityID == nil {
		result = r.db.WithContext(ctx).Where("user_id = ? AND role = ? AND entity_id IS NULL",
			req.UserID, req.Role).Delete(&userRole{})
	} else {
		result = r.db.WithContext(ctx).Where("user_id = ? AND role = ? AND entity_id = ?",
			req.UserID, req.Role, req.EntityID).Delete(&userRole{})
	}
	if result.Error != nil {
		return fmt.Errorf("auth.repo.DeleteUserRole: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("auth.repo.DeleteUserRole: %w", auth.ErrRoleNotFound())
	}

	return nil
}

func (r *Repository) DeleteUserRolesByUserID(ctx context.Context, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&userRole{})
	if result.Error != nil {
		return fmt.Errorf("auth.repo.DeleteUserRolesByUserID: %w", result.Error)
	}

	return nil
}

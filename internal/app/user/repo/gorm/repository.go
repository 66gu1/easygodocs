package gorm

import (
	"context"
	"errors"
	"fmt"

	"github.com/66gu1/easygodocs/internal/app/user"
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
		return nil, fmt.Errorf("user.repo.NewRepository: %w", fmt.Errorf("nil db"))
	}
	return &Repository{db: db}, nil
}

func (r *Repository) CreateUser(ctx context.Context, req user.CreateUserReq, id uuid.UUID, passwordHash string) error {
	model := &userModel{
		ID:           id,
		Email:        req.Email,
		PasswordHash: passwordHash,
		Name:         req.Name,
	}

	err := r.db.WithContext(ctx).Create(model).Error
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == db.DuplicateCode {
			err = user.ErrUserWithEmailAlreadyExists()
		}
		return fmt.Errorf("user.repo.CreateUser: %w", err)
	}

	return nil
}

func (r *Repository) GetUser(ctx context.Context, id uuid.UUID) (user.User, string, error) {
	model := userModel{}

	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = user.ErrUserNotFound()
		}
		return user.User{}, "", fmt.Errorf("user.repo.GetUser: %w", err)
	}

	return model.toDTO(), model.PasswordHash, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (user.User, string, error) {
	model := userModel{}

	err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = user.ErrUserNotFound()
		}
		return user.User{}, "", fmt.Errorf("user.repo.GetUserByEmail: %w", err)
	}

	return model.toDTO(), model.PasswordHash, nil
}

func (r *Repository) GetAllUsers(ctx context.Context) ([]user.User, error) {
	models := make([]userModel, 0)

	err := r.db.WithContext(ctx).
		Select("id", "email", "name", "created_at", "updated_at", "deleted_at", "session_version").
		Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("user.repo.GetAllUsers: %w", err)
	}

	return lo.Map(models, func(u userModel, _ int) user.User { return u.toDTO() }), nil
}

func (r *Repository) UpdateUser(ctx context.Context, req user.UpdateUserReq) error {
	model := &userModel{}

	result := r.db.WithContext(ctx).Model(model).Where("id = ?", req.UserID).
		Updates(map[string]interface{}{"name": req.Name, "email": req.Email})
	if result.Error != nil {
		err := result.Error
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == db.DuplicateCode {
			err = user.ErrUserWithEmailAlreadyExists()
		}
		return fmt.Errorf("user.repo.UpdateUser: %w", err)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user.repo.UpdateUser: %w", user.ErrUserNotFound())
	}

	return nil
}

func (r *Repository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&userModel{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("user.repo.DeleteUser: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user.repo.DeleteUser: %w", user.ErrUserNotFound())
	}

	return nil
}

func (r *Repository) ChangePassword(ctx context.Context, id uuid.UUID, newPasswordHash string) error {
	result := r.db.WithContext(ctx).
		Model(&userModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"password_hash":   newPasswordHash,
			"session_version": gorm.Expr("session_version + 1"),
		})
	if result.Error != nil {
		return fmt.Errorf("user.repo.ChangePassword: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user.repo.ChangePassword: %w", user.ErrUserNotFound())
	}

	return nil
}

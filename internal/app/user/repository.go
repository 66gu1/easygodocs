package user

import (
	"context"
	"errors"
	"fmt"
	hierarchy "github.com/66gu1/easygodocs/internal/app/hierarchy/dto"
	"github.com/66gu1/easygodocs/internal/app/user/dto"
	"github.com/66gu1/easygodocs/internal/infrastructure/appslices"
	"github.com/66gu1/easygodocs/internal/infrastructure/auth"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type gormRepo struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *gormRepo {
	return &gormRepo{db: db}
}

func (r *gormRepo) CreateUser(ctx context.Context, req dto.CreateUserReq) error {
	model := &user{
		ID:           req.ID,
		Email:        req.Email,
		PasswordHash: req.PasswordHash,
		Name:         req.Name,
	}

	err := r.db.WithContext(ctx).Create(model).Error
	if err != nil {
		return fmt.Errorf("gormRepo.Create: %w", err)
	}

	return nil
}

func (r *gormRepo) GetUser(ctx context.Context, id uuid.UUID) (dto.User, error) {
	model := user{ID: id}

	err := r.db.WithContext(ctx).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = ErrUserNotFound
		}
		return dto.User{}, fmt.Errorf("gormRepo.Get: %w", err)
	}

	return model.toDTO(), nil
}

func (r *gormRepo) GetUserByEmail(ctx context.Context, email string) (dto.User, error) {
	model := user{}

	err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = ErrUserNotFound
		}
		return dto.User{}, fmt.Errorf("gormRepo.GetUserByEmail: %w", err)
	}

	return model.toDTO(), nil
}

func (r *gormRepo) GetAllUsers(ctx context.Context) ([]dto.User, error) {
	models := make([]user, 0)

	err := r.db.WithContext(ctx).Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetAll: %w", err)
	}

	return toDTOs(models, func(u user) dto.User { return u.toDTO() }), nil
}

func (r *gormRepo) UpdateUser(ctx context.Context, req dto.UpdateUserReq) error {
	model := &user{}

	result := r.db.WithContext(ctx).Model(model).Where("id = ?", req.ID).
		Updates(map[string]interface{}{"name": req.Name, "email": req.Email})
	if result.Error != nil {
		return fmt.Errorf("gormRepo.Update: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *gormRepo) UpdateUserPassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	model := &user{}

	result := r.db.WithContext(ctx).Model(model).Where("id = ?", id).
		Update("password_hash", passwordHash)
	if result.Error != nil {
		return fmt.Errorf("gormRepo.UpdateUserPassword: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *gormRepo) DeleteUser(ctx context.Context, id uuid.UUID) error {
	model := &user{ID: id}

	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(model)
	if result.Error != nil {
		return fmt.Errorf("gormRepo.Delete: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *gormRepo) CreateSession(ctx context.Context, req dto.Session) error {
	model := &session{
		ID:           req.ID,
		UserID:       req.UserID,
		UserAgent:    req.UserAgent,
		RefreshToken: req.RefreshToken,
		CreatedAt:    req.CreatedAt,
		ExpiresAt:    req.ExpiresAt,
	}

	err := r.db.WithContext(ctx).Create(model).Error
	if err != nil {
		return fmt.Errorf("gormRepo.CreateSession: %w", err)
	}

	return nil
}

func (r *gormRepo) GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]dto.Session, error) {
	models := make([]session, 0)

	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetSessions: %w", err)
	}

	return toDTOs(models, func(s session) dto.Session { return s.toDTO() }), nil
}

func (r *gormRepo) GetSessionByID(ctx context.Context, id uuid.UUID) (dto.Session, error) {
	model := session{ID: id}

	err := r.db.WithContext(ctx).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.Session{}, ErrSessionNotFound
		}
		return dto.Session{}, fmt.Errorf("gormRepo.GetSessionByID: %w", err)
	}

	return model.toDTO(), nil
}

func (r *gormRepo) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (dto.Session, error) {
	model := session{RefreshToken: refreshToken}

	err := r.db.WithContext(ctx).Where("refresh_token = ?", refreshToken).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.Session{}, ErrSessionNotFound
		}
		return dto.Session{}, fmt.Errorf("gormRepo.GetSessionByRefreshToken: %w", err)
	}

	return model.toDTO(), nil
}

func (r *gormRepo) DeleteSession(ctx context.Context, id uuid.UUID) error {
	model := &session{ID: id}

	result := r.db.WithContext(ctx).Where("id = ? AND expires_at > ?", id, time.Now().UTC()).Delete(model)
	if result.Error != nil {
		return fmt.Errorf("gormRepo.DeleteSession: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

func (r *gormRepo) DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error {
	model := &session{}

	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(model)
	if result.Error != nil {
		return fmt.Errorf("gormRepo.DeleteSessionsByUserID: %w", result.Error)
	}

	return nil
}

func (r *gormRepo) UpdateRefreshToken(ctx context.Context, req updateTokenReq) error {
	model := &session{}

	result := r.db.WithContext(ctx).Model(model).Where("id = ?", req.ID).
		Updates(map[string]interface{}{"refresh_token": req.RefreshToken, "expires_at": req.ExpiresAt})
	if result.Error != nil {
		return fmt.Errorf("gormRepo.UpdateRefreshToken: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

func (r *gormRepo) AddUserRole(ctx context.Context, req dto.UserRole) error {
	result := r.db.WithContext(ctx).Create(userRoleFromDTO(req))
	if result.Error != nil {
		return fmt.Errorf("gormRepo.AddUserRole: %w", result.Error)
	}

	return nil
}

func (r *gormRepo) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]dto.UserRole, error) {
	models := make([]userRole, 0)

	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetUserRoles: %w", err)
	}

	return appslices.Map(models, func(u userRole) dto.UserRole { return u.toDTO() }), nil
}

func (r *gormRepo) DeleteUserRole(ctx context.Context, req dto.UserRole) error {
	result := r.db.WithContext(ctx)
	if req.Entity != nil {
		result = result.Where("user_id = ? AND role = ? AND entity_id = ? AND entity_type = ?",
			req.UserID, req.Role, req.Entity.ID, req.Entity.Type)
	} else {
		result = result.Where("user_id = ? AND role = ?", req.UserID, req.Role)
	}

	result = result.Delete(&userRole{})
	if result.Error != nil {
		return fmt.Errorf("gormRepo.DeleteUserRole: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrRoleNotFound
	}

	return nil
}

func (r *gormRepo) GetPermissionsByUserAndRole(ctx context.Context, userID uuid.UUID, role auth.Role) (dto.Permissions, error) {
	models := make([]accessScope, 0)

	err := r.db.WithContext(ctx).Table("user_roles").Where("user_id = ? AND role = ?", userID, role).
		Find(&models).Error

	if err != nil {
		return dto.Permissions{}, fmt.Errorf("gormRepo.GetAccessScopeByUserAndRole: %w", err)
	}

	var (
		departmentIDs = make([]uuid.UUID, 0, len(models))
		articleIDs    = make([]uuid.UUID, 0, len(models))
	)

	for _, model := range models {
		switch model.EntityType {
		case hierarchy.EntityTypeDepartment:
			departmentIDs = append(departmentIDs, model.EntityID)
		case hierarchy.EntityTypeArticle:
			articleIDs = append(articleIDs, model.EntityID)
		default:
			return dto.Permissions{}, fmt.Errorf("gormRepo.GetPermissionsByUserAndRole: invalid entity type %s", model.EntityType)
		}
	}

	return dto.Permissions{
		DepartmentIDs: departmentIDs,
		ArticleIDs:    articleIDs,
	}, nil
}

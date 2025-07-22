package user

import (
	"context"
	"errors"
	"fmt"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound = &apperror.Error{
		Message:  "user not found",
		Code:     apperror.NotFound,
		LogLevel: apperror.LogLevelWarn,
	}
	ErrSessionNotFound = &apperror.Error{
		Message:  "session not found",
		Code:     apperror.NotFound,
		LogLevel: apperror.LogLevelWarn,
	}
)

type gormRepo struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *gormRepo {
	return &gormRepo{db: db}
}

func (r *gormRepo) CreateUser(ctx context.Context, req CreateUserReq) error {
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

func (r *gormRepo) GetUser(ctx context.Context, id uuid.UUID) (User, error) {
	model := user{ID: id}

	err := r.db.WithContext(ctx).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = ErrUserNotFound
		}
		return User{}, fmt.Errorf("gormRepo.Get: %w", err)
	}

	return model.toDTO(), nil
}

func (r *gormRepo) GetUserByEmail(ctx context.Context, email string) (User, error) {
	model := user{}

	err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = ErrUserNotFound
		}
		return User{}, fmt.Errorf("gormRepo.GetUserByEmail: %w", err)
	}

	return model.toDTO(), nil
}

func (r *gormRepo) GetAllUsers(ctx context.Context) ([]User, error) {
	models := make([]user, 0)

	err := r.db.WithContext(ctx).Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetAll: %w", err)
	}

	return toDTOs(models, func(u user) User { return u.toDTO() }), nil
}

func (r *gormRepo) UpdateUser(ctx context.Context, req UpdateUserReq) error {
	model := &user{ID: req.ID}

	result := r.db.WithContext(ctx).Model(model).Where("id = ?", model.ID).
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
	model := &user{ID: id, PasswordHash: passwordHash}

	result := r.db.WithContext(ctx).Model(model).Where("id = ?", model.ID).
		Update("password_hash", model.PasswordHash)
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

func (r *gormRepo) CreateSession(ctx context.Context, req Session) error {
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

func (r *gormRepo) GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	models := make([]session, 0)

	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetSessions: %w", err)
	}

	return toDTOs(models, func(s session) Session { return s.toDTO() }), nil
}

func (r *gormRepo) GetSessionByID(ctx context.Context, id uuid.UUID) (Session, error) {
	model := session{ID: id}

	err := r.db.WithContext(ctx).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Session{}, ErrSessionNotFound
		}
		return Session{}, fmt.Errorf("gormRepo.GetSessionByID: %w", err)
	}

	return model.toDTO(), nil
}

func (r *gormRepo) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (Session, error) {
	model := session{RefreshToken: refreshToken}

	err := r.db.WithContext(ctx).Where("refresh_token = ?", refreshToken).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Session{}, ErrSessionNotFound
		}
		return Session{}, fmt.Errorf("gormRepo.GetSessionByRefreshToken: %w", err)
	}

	return model.toDTO(), nil
}

func (r *gormRepo) DeleteSession(ctx context.Context, id uuid.UUID) error {
	model := &session{ID: id}

	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(model)
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
	if result.RowsAffected == 0 {
		return ErrSessionNotFound
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

func (r *gormRepo) AddUserRole(ctx context.Context, role UserRole) error {
	model := &userRole{}
	model.fromDTO(role)
	result := r.db.WithContext(ctx).Create(&model)
	if result.Error != nil {
		return fmt.Errorf("gormRepo.AddUserRole: %w", result.Error)
	}

	return nil
}

func (r *gormRepo) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]UserRole, error) {
	models := make([]userRole, 0)

	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetUserRoles: %w", err)
	}

	return toDTOs(models, func(u userRole) UserRole { return u.toDTO() }), nil
}

func (r *gormRepo) DeleteUserRole(ctx context.Context, role UserRole) error {
	model := &userRole{}
	model.fromDTO(role)

	result := r.db.WithContext(ctx).Where("user_id = ? AND role = ?", model.UserID, model.Role).Delete(model)
	if result.Error != nil {
		return fmt.Errorf("gormRepo.DeleteUserRole: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *gormRepo) GetAccessScopeByUserAndRole(ctx context.Context, userID uuid.UUID, role role) ([]uuid.UUID, []uuid.UUID, error) {
	models := make([]accessScope, 0)

	query := r.db.WithContext(ctx).Table("user_roles").Select("department_id, article_id").
		Where("user_id = ? AND role = ?", userID, role)

	err := query.Scan(&models).Error
	if err != nil {
		return nil, nil, fmt.Errorf("gormRepo.GetAccessScopeByUserAndRole: %w", err)
	}

	var (
		departmentIDs = make([]uuid.UUID, 0, len(models))
		articleIDs    = make([]uuid.UUID, 0, len(models))
	)

	for _, model := range models {
		if model.DepartmentID != nil {
			departmentIDs = append(departmentIDs, *model.DepartmentID)
			// Only one of DepartmentID or ArticleID can be non-nil per record
			continue
		}
		if model.ArticleID != nil {
			articleIDs = append(articleIDs, *model.ArticleID)
		}
	}

	return departmentIDs, articleIDs, nil
}

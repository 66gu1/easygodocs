package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/contextx"
	"github.com/66gu1/easygodocs/internal/infrastructure/secure"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	FieldSessionID apperr.Field = "session_id"
	FieldUserID    apperr.Field = "user_id"
	FieldSession   apperr.Field = "session"
	FieldUserRole  apperr.Field = "user_role"
	FieldRole      apperr.Field = "role"
	FieldEntity    apperr.Field = "entity"
)

const (
	CodeSessionNotFound  apperr.Code = "auth/session_not_found"
	CodeRoleNotFound     apperr.Code = "auth/role_not_found"
	CodeValidationFailed apperr.Code = "auth/validation_failed"
	CodeRoleDuplicate    apperr.Code = "auth/role_duplicate"
)

type Repository interface {
	CreateSession(ctx context.Context, req Session, rtHash string) error
	GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]Session, error)
	GetSessionByID(ctx context.Context, id uuid.UUID) (Session, string, error)
	DeleteSessionByID(ctx context.Context, id uuid.UUID) error
	DeleteSessionByIDAndUser(ctx context.Context, id, userID uuid.UUID) error
	DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error
	UpdateRefreshToken(ctx context.Context, req UpdateTokenReq) error
	AddUserRole(ctx context.Context, role UserRole) error
	GetUserRoles(ctx context.Context, userID uuid.UUID, roles []Role) ([]UserRole, error)
	DeleteUserRole(ctx context.Context, role UserRole) error
	ListUserRoles(ctx context.Context, userID uuid.UUID) ([]UserRole, error)
}

type TokenCodec interface {
	GenerateToken(claims jwt.Claims) (string, error)
}

type UUIDGenerator interface {
	New() (uuid.UUID, error)
}

type RNDGenerator interface {
	New(n int) (string, error)
}

type TimeGenerator interface {
	Now() time.Time
}

type generators struct {
	idGenerator   UUIDGenerator
	rndGenerator  RNDGenerator
	timeGenerator TimeGenerator
}

type Config struct {
	SessionTTLMinutes     int `mapstructure:"session_ttl_minutes" json:"session_ttl_minutes"`
	AccessTokenTTLMinutes int `mapstructure:"access_token_ttl_minutes" json:"access_token_ttl_minutes"`
}

type core struct {
	repo       Repository
	codec      TokenCodec
	generators generators
	cfg        Config
}

func NewCore(repo Repository, codec TokenCodec, idGenerator UUIDGenerator, rndGenerator RNDGenerator, timeGenerator TimeGenerator, cfg Config) *core {
	if cfg.SessionTTLMinutes <= 0 || cfg.AccessTokenTTLMinutes <= 0 {
		panic("auth.core: invalid config")
	}
	if rndGenerator == nil || idGenerator == nil || timeGenerator == nil || repo == nil || codec == nil {
		panic("auth.core: nil dependency")
	}

	return &core{
		repo:       repo,
		codec:      codec,
		generators: generators{idGenerator, rndGenerator, timeGenerator},
		cfg:        cfg,
	}
}

func (c *core) IssueTokens(ctx context.Context, userID uuid.UUID, sessionVersion int) (Tokens, error) {
	if userID == uuid.Nil {
		return Tokens{}, fmt.Errorf("auth.core.IssueTokens: user ID cannot be nil")
	}

	sessionID, err := c.generators.idGenerator.New()
	if err != nil {
		return Tokens{}, fmt.Errorf("auth.core.IssueTokens: %w", err)
	}

	now := c.generators.timeGenerator.Now()
	accessToken, refreshToken, rtHash, err := c.generateTokens(userID, sessionID, now)
	if err != nil {
		return Tokens{}, fmt.Errorf("auth.core.IssueTokens: %w", err)
	}

	session := Session{
		ID:             sessionID,
		UserID:         userID,
		CreatedAt:      now,
		ExpiresAt:      now.Add(time.Duration(c.cfg.SessionTTLMinutes) * time.Minute),
		SessionVersion: sessionVersion,
	}
	err = c.repo.CreateSession(ctx, session, string(rtHash))
	if err != nil {
		return Tokens{}, fmt.Errorf("auth.core.IssueTokens: %w", err)
	}

	return Tokens{
		AccessToken: accessToken,
		RefreshToken: RefreshToken{
			SessionID: sessionID,
			Token:     refreshToken,
		},
	}, nil
}

func (c *core) RefreshTokens(ctx context.Context, session Session, refreshToken, rtHash string) (Tokens, error) {
	now := c.generators.timeGenerator.Now()
	if !session.ExpiresAt.After(now) {
		err := apperr.ErrUnauthorized().WithDetail("session has expired")
		return Tokens{}, fmt.Errorf("auth.core.RefreshTokens: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(rtHash), []byte(refreshToken)); err != nil {
		err = apperr.ErrUnauthorized().WithDetail("invalid refresh token")
		return Tokens{}, fmt.Errorf("auth.core.RefreshTokens: %w", err)
	}

	accessToken, newRefreshToken, newRTHash, err := c.generateTokens(session.UserID, session.ID, now)
	if err != nil {
		return Tokens{}, fmt.Errorf("auth.core.RefreshTokens: %w", err)
	}

	if err = c.repo.UpdateRefreshToken(ctx, UpdateTokenReq{
		SessionID:           session.ID,
		UserID:              session.UserID,
		RefreshTokenHash:    string(newRTHash),
		ExpiresAt:           now.Add(time.Duration(c.cfg.SessionTTLMinutes) * time.Minute),
		OldRefreshTokenHash: rtHash,
	}); err != nil {
		return Tokens{}, fmt.Errorf("auth.core.RefreshTokens: %w", err)
	}

	return Tokens{
		AccessToken: accessToken,
		RefreshToken: RefreshToken{
			SessionID: session.ID,
			Token:     newRefreshToken,
		},
	}, nil
}

func (c *core) GetSessionByID(ctx context.Context, id uuid.UUID) (Session, string, error) {
	if id == uuid.Nil {
		return Session{}, "", fmt.Errorf("auth.core.GetSessionByID: %w", apperr.ErrNilUUID(FieldSessionID))
	}
	session, rtHash, err := c.repo.GetSessionByID(ctx, id)
	if err != nil {
		return Session{}, "", fmt.Errorf("auth.core.GetSessionByID: %w", err)
	}

	return session, rtHash, nil
}

func (c *core) GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("auth.core.GetSessionsByUserID: %w", apperr.ErrNilUUID(FieldUserID))
	}
	sessions, err := c.repo.GetSessionsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.core.GetSessionsByUserID: %w", err)
	}

	return sessions, nil
}

func (c *core) DeleteSession(ctx context.Context, id, userID uuid.UUID, isAdmin bool) error {
	if isAdmin {
		if err := c.repo.DeleteSessionByID(ctx, id); err != nil {
			return fmt.Errorf("auth.core.DeleteSession: %w", err)
		}

		return nil
	}

	if err := c.repo.DeleteSessionByIDAndUser(ctx, id, userID); err != nil {
		return fmt.Errorf("auth.core.DeleteSession: %w", err)
	}

	return nil
}

func (c *core) DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error {
	if err := c.repo.DeleteSessionsByUserID(ctx, userID); err != nil {
		return fmt.Errorf("auth.core.DeleteSessionsByUserID: %w", err)
	}

	return nil
}

func (c *core) AddUserRole(ctx context.Context, userRole UserRole) error {
	if userRole.UserID == uuid.Nil {
		return fmt.Errorf("auth.core.AddUserRole: %w", apperr.ErrNilUUID(FieldUserID))
	}
	if err := userRole.Role.Validate(); err != nil {
		return fmt.Errorf("auth.core.AddUserRole: %w", err)
	}
	if err := userRole.Role.ValidateEntity(userRole.EntityID); err != nil {
		return fmt.Errorf("auth.core.AddUserRole: %w", err)
	}
	if err := c.repo.AddUserRole(ctx, userRole); err != nil {
		return fmt.Errorf("auth.core.AddUserRole: %w", err)
	}

	return nil
}

func (c *core) DeleteUserRole(ctx context.Context, role UserRole) error {
	if err := c.repo.DeleteUserRole(ctx, role); err != nil {
		return fmt.Errorf("auth.core.DeleteUserRole: %w", err)
	}

	return nil
}

// ListUserRoles returns all roles assigned to the specified user.
// This method is intended for display purposes only (e.g., in an admin UI).
func (c *core) ListUserRoles(ctx context.Context, userID uuid.UUID) ([]UserRole, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("auth.core.ListUserRoles: %w", apperr.ErrNilUUID(FieldUserID))
	}

	userRoles, err := c.repo.ListUserRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.core.ListUserRoles: %w", err)
	}

	return userRoles, nil
}

// Permission check helpers.
// These methods are intended for internal authorization logic.

// GetCurrentUserDirectPermissions doesn't return ids if isAdmin is true.
func (c *core) GetCurrentUserDirectPermissions(ctx context.Context, role Role) (ids []uuid.UUID, isAdmin bool, err error) {
	currentUserID, err := contextx.GetUserID(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("auth.core.GetCurrentUserDirectPermissions: %w", err)
	}
	if err = role.Validate(); err != nil {
		return nil, false, fmt.Errorf("auth.core.GetCurrentUserDirectPermissions: %w", err)
	}

	roles := role.GetHierarchy()
	userRoles, err := c.repo.GetUserRoles(ctx, currentUserID, roles)
	if err != nil {
		return nil, false, fmt.Errorf("auth.core.GetCurrentUserDirectPermissions: %w", err)
	}

	for _, ur := range userRoles {
		if ur.Role == roleAdmin {
			return nil, true, nil
		}
		if ur.EntityID != nil {
			ids = append(ids, *ur.EntityID)
		}
	}

	return ids, false, nil
}

func (c *core) CheckSelfOrAdmin(ctx context.Context, targetUserID uuid.UUID) error {
	self, err := c.IsSelf(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("auth.core.CheckSelfOrAdmin.IsSelf: %w", err)
	}
	if self {
		return nil
	}

	err = c.CheckIsAdmin(ctx)
	if err != nil {
		return fmt.Errorf("auth.core.CheckSelfOrAdmin: %w", err)
	}

	return nil
}

func (c *core) CheckIsAdmin(ctx context.Context) error {
	isAdmin, err := c.IsAdmin(ctx)
	if err != nil {
		return fmt.Errorf("auth.core.CheckIsAdmin: %w", err)
	}
	if !isAdmin {
		return fmt.Errorf("auth.core.CheckIsAdmin: %w", apperr.ErrForbidden())
	}
	return nil
}

func (c *core) CheckSelf(ctx context.Context, targetUserID uuid.UUID) error {
	self, err := c.IsSelf(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("auth.core.CheckSelf: %w", err)
	}
	if !self {
		return fmt.Errorf("auth.core.CheckSelf: %w", apperr.ErrForbidden())
	}

	return nil
}

func (c *core) IsAdmin(ctx context.Context) (bool, error) {
	isAdmin, err := c.checkAdminRights(ctx)
	if err != nil {
		return false, fmt.Errorf("auth.core.IsAdmin: %w", err)
	}

	return isAdmin, nil
}

func (c *core) IsSelf(ctx context.Context, targetUserID uuid.UUID) (bool, error) {
	if targetUserID == uuid.Nil {
		return false, fmt.Errorf("auth.core.IsSelf: %w", apperr.ErrNilUUID(FieldUserID))
	}
	currentUserID, err := contextx.GetUserID(ctx)
	if err != nil {
		return false, fmt.Errorf("auth.core.IsSelf: %w", err)
	}
	return currentUserID == targetUserID, nil
}

func (c *core) checkAdminRights(ctx context.Context) (bool, error) {
	_, isAdmin, err := c.GetCurrentUserDirectPermissions(ctx, roleAdmin)
	if err != nil {
		return false, fmt.Errorf("checkAdminRights: %w", err)
	}

	return isAdmin, nil
}

func (c *core) generateTokens(userID, sessionID uuid.UUID, now time.Time) (string, string, []byte, error) {
	refreshToken, err := c.generators.rndGenerator.New(32) // 32 bytes = 256 bits of entropy
	if err != nil {
		return "", "", nil, fmt.Errorf("generateTokens: %w", err)
	}
	rtHash, err := secure.HashRefreshToken([]byte(refreshToken))
	if err != nil {
		return "", "", nil, fmt.Errorf("generateTokens: %w", err)
	}

	accessToken, err := c.codec.GenerateToken(AccessTokenClaims{
		SID: sessionID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(c.cfg.AccessTokenTTLMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	})
	if err != nil {
		return "", "", nil, fmt.Errorf("generateTokens: %w", err)
	}

	return accessToken, refreshToken, rtHash, nil
}

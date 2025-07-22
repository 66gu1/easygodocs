package user

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/66gu1/easygodocs/internal/infrastructure/auth"
	"github.com/66gu1/easygodocs/internal/infrastructure/contextutil"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"
)

type UserService struct {
	repo Repository
	cfg  *Config
}

type Config struct {
	MinPasswordLength int
	MaxPasswordLength int
	MaxEmailLength    int
	MaxNameLength     int

	SessionTTLMinutes     int
	AccessTokenTTLMinutes int
	JWTSecret             string
}

type Repository interface {
	CreateUser(ctx context.Context, req CreateUserReq) error
	GetUser(ctx context.Context, id uuid.UUID) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetAllUsers(ctx context.Context) ([]User, error)
	UpdateUser(ctx context.Context, req UpdateUserReq) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	CreateSession(ctx context.Context, req Session) error
	GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]Session, error)
	GetSessionByID(ctx context.Context, id uuid.UUID) (Session, error)
	GetSessionByRefreshToken(ctx context.Context, refreshToken string) (Session, error)
	DeleteSession(ctx context.Context, id uuid.UUID) error
	DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error
	UpdateRefreshToken(ctx context.Context, req updateTokenReq) error
	AddUserRole(ctx context.Context, role UserRole) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]UserRole, error)
	DeleteUserRole(ctx context.Context, role UserRole) error
	GetAccessScopeByUserAndRole(ctx context.Context, userID uuid.UUID, role role) ([]uuid.UUID, []uuid.UUID, error)
}

func NewService(repo Repository, cfg Config) *UserService {
	return &UserService{
		repo: repo,
		cfg:  &cfg,
	}
}

func (s *UserService) CreateUser(ctx context.Context, req CreateUserReq) error {
	_, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err == nil {
		err = &apperror.Error{
			Message:  "user with this email already exists",
			Code:     apperror.BadRequest,
			LogLevel: apperror.LogLevelWarn,
		}
		logger.Error(ctx, err).Str("email", req.Email).Msg("UserService.CreateUser: user with this email already exists")
		return fmt.Errorf("UserService.CreateUser: %w", err)
	}
	if !errors.Is(err, ErrUserNotFound) {
		logger.Error(ctx, err).Str("email", req.Email).Msg("UserService.CreateUser.GetUserByEmail")
		return fmt.Errorf("UserService.CreateUser: %w", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		logger.Error(ctx, err).Msg("UserService.CreateUser: failed to generate UUID")
		return fmt.Errorf("UserService.CreateUser.uuid.NewV7(): %w", err)
	}
	hash, err := hashPassword(req.Password)
	if err != nil {
		logger.Error(ctx, err).Str("email", req.Email).Msg("UserService.CreateUser: failed to hash password")
		return fmt.Errorf("UserService.CreateUser.hashPassword: %w", err)
	}

	req.ID = id
	req.Email = strings.ToLower(req.Email)
	req.PasswordHash = hash

	err = s.repo.CreateUser(ctx, req)
	if err != nil {
		logger.Error(ctx, err).Interface("request", req).Msg("UserService.CreateUser")
		return fmt.Errorf("UserService.CreateUser.repo.CreateUser: %w", err)
	}

	return nil
}

func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (User, error) {
	err := s.CheckSelfOrAdmin(ctx, id)
	if err != nil {
		logger.Error(ctx, err).Str("user_id", id.String()).Msg("UserService.GetUser.IsAdmin")
		return User{}, fmt.Errorf("UserService.GetUser: %w", err)
	}

	user, err := s.repo.GetUser(ctx, id)
	if err != nil {
		logger.Error(ctx, err).Str("id", id.String()).Msg("UserService.GetUser")
		return User{}, fmt.Errorf("UserService.GetUser.repo.GetUser: %w", err)
	}
	return user, nil
}

func (s *UserService) GetAllUsers(ctx context.Context) ([]User, error) {
	err := s.CheckIsAdmin(ctx)
	if err != nil {
		logger.Error(ctx, err).Msg("UserService.GetAllUsers.CheckIsAdmin")
		return nil, fmt.Errorf("UserService.GetAllUsers: %w", err)
	}
	users, err := s.repo.GetAllUsers(ctx)
	if err != nil {
		logger.Error(ctx, err).Msg("UserService.GetAllUsers")
		return nil, fmt.Errorf("UserService.GetAllUsers: %w", err)
	}
	return users, nil
}

func (s *UserService) UpdateUser(ctx context.Context, req UpdateUserReq) error {
	err := s.CheckSelfOrAdmin(ctx, req.ID)
	if err != nil {
		logger.Error(ctx, err).Str("user_id", req.ID.String()).Msg("UserService.UpdateUser.IsAdmin")
		return fmt.Errorf("UserService.UpdateUser: %w", err)
	}

	req.Email = strings.ToLower(req.Email)

	err = s.repo.UpdateUser(ctx, req)
	if err != nil {
		logger.Error(ctx, err).Interface("request", req).Msg("UserService.UpdateUser")
		return fmt.Errorf("UserService.UpdateUser.repo.UpdateUser: %w", err)
	}
	return nil
}

func (s *UserService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	err := s.CheckIsAdmin(ctx)
	if err != nil {
		logger.Error(ctx, err).Str("user_id", id.String()).Msg("UserService.DeleteUser.CheckIsAdmin")
		return fmt.Errorf("UserService.DeleteUser: %w", err)
	}

	err = s.repo.DeleteUser(ctx, id)
	if err != nil {
		logger.Error(ctx, err).Str("id", id.String()).Msg("UserService.DeleteUser")
		return fmt.Errorf("UserService.DeleteUser.repo.DeleteUser: %w", err)
	}
	return nil
}

func (s *UserService) GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	err := s.CheckSelfOrAdmin(ctx, userID)
	if err != nil {
		logger.Error(ctx, err).Str("user_id", userID.String()).Msg("UserService.GetSessionsByUserID.IsAdmin")
		return nil, fmt.Errorf("UserService.GetSessionsByUserID: %w", err)
	}

	sessions, err := s.repo.GetSessionsByUserID(ctx, userID)
	if err != nil {
		logger.Error(ctx, err).Str("user_id", userID.String()).
			Msg("UserService.GetSessionsByUserID.repo.GetSessionsByUserID")
		return nil, fmt.Errorf("UserService.GetSessionsByUserID: %w", err)
	}
	return sessions, nil
}

func (s *UserService) DeleteSession(ctx context.Context, id uuid.UUID) error {
	session, err := s.repo.GetSessionByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			err = &apperror.Error{
				Message:  "session not found",
				Code:     apperror.NotFound,
				LogLevel: apperror.LogLevelWarn,
			}
		}
		logger.Error(ctx, err).Str("id", id.String()).Msg("UserService.DeleteSession.GetSessionByID")
		return fmt.Errorf("UserService.DeleteSession.repo.GetSessionByID: %w", err)
	}
	err = s.CheckSelfOrAdmin(ctx, session.UserID)
	if err != nil {
		logger.Error(ctx, err).Str("session_id", id.String()).Msg("UserService.DeleteSession.CheckIsAdmin")
		return fmt.Errorf("UserService.DeleteSession: %w", err)
	}

	err = s.repo.DeleteSession(ctx, id)
	if err != nil {
		logger.Error(ctx, err).Str("id", id.String()).Msg("UserService.DeleteSession")
		return fmt.Errorf("UserService.DeleteSession.repo.DeleteSession: %w", err)
	}
	return nil
}

func (s *UserService) DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error {
	err := s.CheckSelfOrAdmin(ctx, userID)
	if err != nil {
		logger.Error(ctx, err).Str("user_id", userID.String()).Msg("UserService.DeleteSessionsByUserID.CheckIsAdmin")
		return fmt.Errorf("UserService.DeleteSessionsByUserID: %w", err)
	}

	err = s.repo.DeleteSessionsByUserID(ctx, userID)
	if err != nil {
		logger.Error(ctx, err).Str("user_id", userID.String()).
			Msg("UserService.DeleteSessionsByUserID.repo.DeleteSessionsByUserID")
		return fmt.Errorf("UserService.DeleteSessionsByUserID: %w", err)
	}
	return nil
}

func (s *UserService) AddUserRole(ctx context.Context, role UserRole) error {
	err := s.CheckIsAdmin(ctx)
	if err != nil {
		logger.Error(ctx, err).Str("role", role.Role.String()).Msg("UserService.AddUserRole.CheckIsAdmin")
		return fmt.Errorf("UserService.AddUserRole: %w", err)
	}

	err = s.repo.AddUserRole(ctx, role)
	if err != nil {
		logger.Error(ctx, err).Interface("role", role).Msg("UserService.AddUserRole")
		return fmt.Errorf("UserService.AddUserRole.repo.AddUserRole: %w", err)
	}
	return nil
}

func (s *UserService) DeleteUserRole(ctx context.Context, role UserRole) error {
	err := s.CheckIsAdmin(ctx)
	if err != nil {
		logger.Error(ctx, err).Str("role", role.Role.String()).Msg("UserService.DeleteUserRole.CheckIsAdmin")
		return fmt.Errorf("UserService.DeleteUserRole: %w", err)
	}

	err = s.repo.DeleteUserRole(ctx, role)
	if err != nil {
		logger.Error(ctx, err).Interface("role", role).Msg("UserService.DeleteUserRole")
		return fmt.Errorf("UserService.DeleteUserRole.repo.DeleteUserRole: %w", err)
	}
	return nil
}

func (s *UserService) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]UserRole, error) {
	err := s.CheckSelfOrAdmin(ctx, userID)
	if err != nil {
		logger.Error(ctx, err).Str("user_id", userID.String()).Msg("UserService.GetUserRoles.CheckIsAdmin")
		return nil, fmt.Errorf("UserService.GetUserRoles: %w", err)
	}

	roles, err := s.repo.GetUserRoles(ctx, userID)
	if err != nil {
		logger.Error(ctx, err).Str("user_id", userID.String()).Msg("UserService.GetUserRoles")
		return nil, fmt.Errorf("UserService.GetUserRoles.repo.GetUserRoles: %w", err)
	}
	return roles, nil
}

func (s *UserService) RefreshTokens(ctx context.Context, refreshToken string) (string, string, error) {
	session, err := s.repo.GetSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			err = &apperror.Error{
				Message:  "refresh token not found",
				Code:     apperror.Unauthorized,
				LogLevel: apperror.LogLevelWarn,
			}
		}
		logger.Error(ctx, err).Str("refresh_token", refreshToken).Msg("UserService.RefreshTokens.GetSessionByRefreshToken")
		return "", "", fmt.Errorf("UserService.RefreshTokens.repo.GetSessionByRefreshToken: %w", err)
	}

	now := time.Now().UTC()
	newRefreshToken, err := generateRefreshToken()
	if err != nil {
		logger.Error(ctx, err).Str("session_id", session.ID.String()).Msg("UserService.RefreshTokens.generateRefreshToken")
		return "", "", fmt.Errorf("UserService.RefreshTokens.generateRefreshToken: %w", err)
	}

	err = s.repo.UpdateRefreshToken(ctx, updateTokenReq{
		ID:           session.ID,
		RefreshToken: newRefreshToken,
		ExpiresAt:    now.Add(time.Duration(s.cfg.SessionTTLMinutes) * time.Minute),
	})
	if err != nil {
		logger.Error(ctx, err).Str("session_id", session.ID.String()).Msg("UserService.RefreshTokens.UpdateRefreshToken")
		return "", "", fmt.Errorf("UserService.RefreshTokens.repo.UpdateRefreshToken: %w", err)
	}

	accessToken, err := s.generateAccessToken(session.UserID, session.ID, now)
	if err != nil {
		logger.Error(ctx, err).Str("session_id", session.ID.String()).Msg("UserService.RefreshTokens.generateAccessToken")
		return "", "", fmt.Errorf("UserService.RefreshTokens.generateAccessToken: %w", err)
	}

	return accessToken, newRefreshToken, nil

}

func (s *UserService) Login(ctx context.Context, email, password, userAgent string) (string, string, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			err = &apperror.Error{
				Message:  "user not found",
				Code:     apperror.BadRequest,
				LogLevel: apperror.LogLevelWarn,
			}
		}
		logger.Error(ctx, err).Str("email", email).Msg("UserService.Login.GetUserByEmail")
		return "", "", fmt.Errorf("UserService.Login.GetUserByEmail: %w", err)
	}

	if !checkPassword(password, user.PasswordHash) {
		err = &apperror.Error{
			Message:  "invalid password",
			Code:     apperror.Unauthorized,
			LogLevel: apperror.LogLevelWarn,
		}
		logger.Error(ctx, err).Str("email", email).Msg("UserService.Login: invalid password")
		return "", "", fmt.Errorf("UserService.Login.checkPassword: %w", err)
	}

	sessionID, err := uuid.NewV7()
	if err != nil {
		logger.Error(ctx, err).Str("email", email).Msg("UserService.Login: failed to generate session UUID")
		return "", "", fmt.Errorf("UserService.Login.uuid.NewV7(): %w", err)
	}
	refreshToken, err := generateRefreshToken()
	if err != nil {
		logger.Error(ctx, err).Str("email", email).Msg("UserService.Login: failed to generate refresh token")
		return "", "", fmt.Errorf("UserService.Login.generateRefreshToken: %w", err)
	}
	now := time.Now().UTC()
	session := Session{
		ID:           sessionID,
		UserID:       user.ID,
		UserAgent:    userAgent,
		RefreshToken: refreshToken,
		CreatedAt:    now,
		ExpiresAt:    now.Add(time.Duration(s.cfg.SessionTTLMinutes) * time.Minute),
	}

	err = s.repo.CreateSession(ctx, session)
	if err != nil {
		logger.Error(ctx, err).Str("email", email).Interface("request", session).
			Msg("UserService.Login.CreateSession")
		return "", "", fmt.Errorf("UserService.Login.repo.CreateSession: %w", err)
	}

	accessToken, err := s.generateAccessToken(user.ID, sessionID, now)
	if err != nil {
		logger.Error(ctx, err).Str("email", email).Msg("UserService.Login.generateAccessToken")
		return "", "", fmt.Errorf("UserService.Login.generateAccessToken: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *UserService) CheckSelfOrAdmin(ctx context.Context, targetUserID uuid.UUID) error {
	currentUserID, ok := contextutil.GetFromContext[uuid.UUID](ctx, contextutil.ContextKeyUserID)
	if !ok {
		return fmt.Errorf("UserService.CheckIsAdmin: failed to get user ID from context")
	}
	if currentUserID != targetUserID {
		return s.CheckIsAdmin(ctx)
	}
	return nil
}

func (s *UserService) CheckIsAdmin(ctx context.Context) error {
	currentUserID, ok := contextutil.GetFromContext[uuid.UUID](ctx, contextutil.ContextKeyUserID)
	if !ok {
		return fmt.Errorf("UserService.CheckIsAdmin: failed to get user ID from context")
	}
	isAdmin, err := s.checkAdminRights(ctx, currentUserID)
	if err != nil {
		return fmt.Errorf("UserService.CheckIsAdmin: %w", err)
	}

	if !isAdmin {
		err = &apperror.Error{
			Message:  "permission denied",
			Code:     apperror.Forbidden,
			LogLevel: apperror.LogLevelWarn,
		}
		return fmt.Errorf("UserService.CheckIsAdmin: %w", err)
	}

	return nil
}

func (s *UserService) GetAccessScopeByUserAndRole(ctx context.Context, role role) (departmentIDs []uuid.UUID, articleIDs []uuid.UUID, isAdmin bool, err error) {
	currentUserID, ok := contextutil.GetFromContext[uuid.UUID](ctx, contextutil.ContextKeyUserID)
	if !ok {
		return nil, nil, false, fmt.Errorf("UserService.GetAccessScopeByUserAndRole: failed to get user ID from context")
	}

	isAdmin, err = s.checkAdminRights(ctx, currentUserID)
	if err != nil {
		logger.Error(ctx, err).Str("user_id", currentUserID.String()).Msg("UserService.GetAccessScopeByUserAndRole.checkAdminRights")
		return nil, nil, false, fmt.Errorf("UserService.GetAccessScopeByUserAndRole: %w", err)
	}
	if isAdmin {
		return nil, nil, true, nil
	}

	departmentIDs, articleIDs, err = s.repo.GetAccessScopeByUserAndRole(ctx, currentUserID, role)
	if err != nil {
		logger.Error(ctx, err).Str("user_id", currentUserID.String()).Str("role", role.String()).
			Msg("UserService.GetAccessScopeByUserAndRole.GetAccessScopeByUserAndRole")
		return nil, nil, false, fmt.Errorf("UserService.GetAccessScopeByUserAndRole: %w", err)
	}

	return departmentIDs, articleIDs, false, nil
}

func (s *UserService) checkAdminRights(ctx context.Context, userID uuid.UUID) (bool, error) {
	roles, err := s.repo.GetUserRoles(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("checkAdminRights: %w", err)
	}

	for _, role := range roles {
		if role.Role == RoleAdmin {
			return true, nil
		}
	}

	return false, nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateRefreshToken() (string, error) {
	bytes := make([]byte, 32) // 32 bytes = 256 bits of entropy
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes), nil
}

func (s *UserService) generateAccessToken(userID uuid.UUID, sessionID uuid.UUID, now time.Time) (string, error) {
	claims := auth.AccessTokenClaims{
		SID: sessionID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.cfg.AccessTokenTTLMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return accessToken.SignedString([]byte(s.cfg.JWTSecret))
}

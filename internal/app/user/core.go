package user

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/secure"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Repository interface {
	CreateUser(ctx context.Context, req CreateUserReq, id uuid.UUID, passwordHash string) error
	GetUser(ctx context.Context, id uuid.UUID) (User, string, error)
	GetUserByEmail(ctx context.Context, email string) (User, string, error)
	GetAllUsers(ctx context.Context) ([]User, error)
	UpdateUser(ctx context.Context, req UpdateUserReq) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	ChangePassword(ctx context.Context, id uuid.UUID, newPasswordHash string) error
}

type IDGenerator interface {
	New() (uuid.UUID, error)
}

type Config struct {
	MaxEmailLength    int `mapstructure:"max_email_length" json:"max_email_length"`
	MaxNameLength     int `mapstructure:"max_name_length" json:"max_name_length"`
	MinPasswordLength int `mapstructure:"min_password_length" json:"min_password_length"`
	MaxPasswordLength int `mapstructure:"max_password_length" json:"max_password_length"`

	PasswordHashCost int `mapstructure:"password_hash_cost" json:"password_hash_cost"`
}

type core struct {
	repo        Repository
	idGenerator IDGenerator
	cfg         Config
}

func NewCore(repo Repository, idGenerator IDGenerator, cfg Config) *core {
	if cfg.MaxEmailLength <= 0 {
		panic("ValidationConfig.MaxEmailLength must be > 0")
	}
	if cfg.MaxNameLength <= 0 {
		panic("ValidationConfig.MaxNameLength must be > 0")
	}
	if cfg.MinPasswordLength <= 0 {
		panic("ValidationConfig.MinPasswordLength must be > 0")
	}
	if cfg.MaxPasswordLength <= 0 || cfg.MaxPasswordLength > 72 {
		panic("ValidationConfig.MaxPasswordLength must be > 0 and <= 72 (bcrypt limit)")
	}
	if cfg.MinPasswordLength > cfg.MaxPasswordLength {
		panic("ValidationConfig.MinPasswordLength must be <= MaxPasswordLength")
	}
	if cfg.PasswordHashCost < bcrypt.MinCost || cfg.PasswordHashCost > bcrypt.MaxCost {
		panic("invalid bcrypt cost")
	}

	return &core{repo: repo, cfg: cfg, idGenerator: idGenerator}
}

func (c *core) CreateUser(ctx context.Context, req CreateUserReq) error {
	req.Name = normalizeName(req.Name)
	if err := c.validateName(req.Name); err != nil {
		return fmt.Errorf("user.core.CreateUser: %w", err)
	}
	req.Email = normalizeEmail(req.Email)
	if err := c.validateEmail(req.Email, true); err != nil {
		return fmt.Errorf("user.core.CreateUser: %w", err)
	}
	if err := c.validatePassword(req.Password); err != nil {
		return fmt.Errorf("user.core.CreateUser: %w", err)
	}
	passwordHash, err := secure.HashPassword(req.Password, c.cfg.PasswordHashCost)
	if err != nil {
		return fmt.Errorf("user.core.CreateUser: %w", err)
	}

	id, err := c.idGenerator.New()
	if err != nil {
		return fmt.Errorf("user.core.CreateUser: %w", err)
	}
	if err = c.repo.CreateUser(ctx, req, id, string(passwordHash)); err != nil {
		return fmt.Errorf("user.core.CreateUser: %w", err)
	}

	return nil
}

func (c *core) GetUser(ctx context.Context, id uuid.UUID) (User, string, error) {
	if id == uuid.Nil {
		return User{}, "", fmt.Errorf("user.core.GetUser: %w", apperr.ErrNilUUID(FieldUserID))
	}

	user, passwordHash, err := c.repo.GetUser(ctx, id)
	if err != nil {
		return User{}, "", fmt.Errorf("user.core.GetUser: %w", err)
	}

	return user, passwordHash, nil
}

func (c *core) GetAllUsers(ctx context.Context) ([]User, error) {
	users, err := c.repo.GetAllUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("user.core.GetAllUsers: %w", err)
	}

	return users, nil
}

func (c *core) UpdateUser(ctx context.Context, req UpdateUserReq) error {
	if req.UserID == uuid.Nil {
		return fmt.Errorf("user.core.UpdateUser: %w", apperr.ErrNilUUID(FieldUserID))
	}
	req.Name = normalizeName(req.Name)
	if err := c.validateName(req.Name); err != nil {
		return fmt.Errorf("user.core.UpdateUser: %w", err)
	}
	req.Email = normalizeEmail(req.Email)
	if err := c.validateEmail(req.Email, true); err != nil {
		return fmt.Errorf("user.core.UpdateUser: %w", err)
	}

	if err := c.repo.UpdateUser(ctx, req); err != nil {
		return fmt.Errorf("user.core.UpdateUser: %w", err)
	}

	return nil
}

func (c *core) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if err := c.repo.DeleteUser(ctx, id); err != nil {
		return fmt.Errorf("user.core.DeleteUser: %w", err)
	}

	return nil
}

func (c *core) ChangePassword(ctx context.Context, id uuid.UUID, newPassword []byte) error {
	if id == uuid.Nil {
		return fmt.Errorf("user.core.ChangePassword: %w", apperr.ErrNilUUID(FieldUserID))
	}
	if err := c.validatePassword(newPassword); err != nil {
		return fmt.Errorf("user.core.ChangePassword: %w", err)
	}
	newPasswordHash, err := secure.HashPassword(newPassword, c.cfg.PasswordHashCost)
	if err != nil {
		return fmt.Errorf("user.core.ChangePassword: %w", err)
	}
	if err = c.repo.ChangePassword(ctx, id, string(newPasswordHash)); err != nil {
		return fmt.Errorf("user.core.ChangePassword: %w", err)
	}

	return nil
}

func (c *core) GetUserByEmail(ctx context.Context, email string) (User, string, error) {
	email = normalizeEmail(email)
	if err := c.validateEmail(email, false); err != nil {
		return User{}, "", fmt.Errorf("user.core.GetAuthByEmail: %w", err)
	}

	user, passwordHash, err := c.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return User{}, "", fmt.Errorf("user.core.GetAuthByEmail: %w", err)
	}
	return user, passwordHash, nil
}

func (c *core) validatePassword(password []byte) error {
	n := utf8.RuneCount(password)
	if n < c.cfg.MinPasswordLength {
		return ErrPasswordTooShort(c.cfg.MinPasswordLength)
	}
	if n > c.cfg.MaxPasswordLength {
		return ErrPasswordTooLong(c.cfg.MaxPasswordLength)
	}
	return nil
}

func (c *core) validateEmail(address string, validateLength bool) error {
	_, err := mail.ParseAddress(address)
	if err != nil {
		return fmt.Errorf("validateEmail: %w", ErrInvalidEmail())
	}
	if validateLength && len(address) > c.cfg.MaxEmailLength {
		return fmt.Errorf("validateEmail: %w", ErrEmailTooLong(c.cfg.MaxEmailLength))
	}
	return nil
}

func (c *core) validateName(name string) error {
	if name == "" {
		return fmt.Errorf("validateName: %w", ErrNameEmpty())
	}
	if len(name) > c.cfg.MaxNameLength {
		return fmt.Errorf("validateName: %w", ErrNameTooLong(c.cfg.MaxNameLength))
	}
	return nil
}

func normalizeName(name string) string {
	return strings.TrimSpace(name)
}

func normalizeEmail(address string) string {
	return strings.TrimSpace(strings.ToLower(address))
}

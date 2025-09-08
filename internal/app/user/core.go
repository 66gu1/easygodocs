package user

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
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

type PasswordHasher interface {
	HashPassword(password []byte, cost int) ([]byte, error)
}

type Validator interface {
	ValidatePassword(password []byte) error
	ValidateEmail(address string, validateLength bool) error
	ValidateName(name string) error
	NormalizeName(name string) string
	NormalizeEmail(address string) string
}

type Config struct {
	PasswordHashCost int `mapstructure:"password_hash_cost" json:"password_hash_cost"`
}

type core struct {
	repo           Repository
	idGenerator    IDGenerator
	passwordHasher PasswordHasher
	validator      Validator
	cfg            Config
}

func NewCore(repo Repository, idGenerator IDGenerator, passwordHasher PasswordHasher, validator Validator, cfg Config) (*core, error) {
	if cfg.PasswordHashCost < bcrypt.MinCost || cfg.PasswordHashCost > bcrypt.MaxCost {
		return nil, fmt.Errorf("user.NewCore: %w", fmt.Errorf("Config.PasswordHashCost must be between %d and %d", bcrypt.MinCost, bcrypt.MaxCost))
	}
	if idGenerator == nil || passwordHasher == nil || repo == nil || validator == nil {
		return nil, fmt.Errorf("user.NewCore: %w", fmt.Errorf("nil dependency"))
	}

	return &core{repo: repo, cfg: cfg, idGenerator: idGenerator, passwordHasher: passwordHasher, validator: validator}, nil
}

func (c *core) CreateUser(ctx context.Context, req CreateUserReq) error {
	req.Name = c.validator.NormalizeName(req.Name)
	if err := c.validator.ValidateName(req.Name); err != nil {
		return fmt.Errorf("user.core.CreateUser: %w", err)
	}
	req.Email = c.validator.NormalizeEmail(req.Email)
	if err := c.validator.ValidateEmail(req.Email, true); err != nil {
		return fmt.Errorf("user.core.CreateUser: %w", err)
	}
	if err := c.validator.ValidatePassword(req.Password); err != nil {
		return fmt.Errorf("user.core.CreateUser: %w", err)
	}
	passwordHash, err := c.passwordHasher.HashPassword(req.Password, c.cfg.PasswordHashCost)
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
	req.Name = c.validator.NormalizeName(req.Name)
	if err := c.validator.ValidateName(req.Name); err != nil {
		return fmt.Errorf("user.core.UpdateUser: %w", err)
	}
	req.Email = c.validator.NormalizeEmail(req.Email)
	if err := c.validator.ValidateEmail(req.Email, true); err != nil {
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
	if err := c.validator.ValidatePassword(newPassword); err != nil {
		return fmt.Errorf("user.core.ChangePassword: %w", err)
	}
	newPasswordHash, err := c.passwordHasher.HashPassword(newPassword, c.cfg.PasswordHashCost)
	if err != nil {
		return fmt.Errorf("user.core.ChangePassword: %w", err)
	}
	if err = c.repo.ChangePassword(ctx, id, string(newPasswordHash)); err != nil {
		return fmt.Errorf("user.core.ChangePassword: %w", err)
	}

	return nil
}

func (c *core) GetUserByEmail(ctx context.Context, email string) (User, string, error) {
	email = c.validator.NormalizeEmail(email)
	if err := c.validator.ValidateEmail(email, false); err != nil {
		return User{}, "", fmt.Errorf("user.core.GetAuthByEmail: %w", err)
	}

	user, passwordHash, err := c.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return User{}, "", fmt.Errorf("user.core.GetAuthByEmail: %w", err)
	}
	return user, passwordHash, nil
}

type ValidationConfig struct {
	MaxEmailLength    int `mapstructure:"max_email_length" json:"max_email_length"`
	MaxNameLength     int `mapstructure:"max_name_length" json:"max_name_length"`
	MinPasswordLength int `mapstructure:"min_password_length" json:"min_password_length"`
	MaxPasswordLength int `mapstructure:"max_password_length" json:"max_password_length"`
}

type validator struct {
	cfg ValidationConfig
}

func NewValidator(cfg ValidationConfig) (Validator, error) {
	if cfg.MaxEmailLength <= 0 {
		return nil, fmt.Errorf("NewValidator: %w", fmt.Errorf("ValidationConfig.MaxEmailLength must be > 0"))
	}
	if cfg.MaxNameLength <= 0 {
		return nil, fmt.Errorf("NewValidator: %w", fmt.Errorf("ValidationConfig.MaxNameLength must be > 0"))
	}
	if cfg.MinPasswordLength <= 0 {
		return nil, fmt.Errorf("NewValidator: %w", fmt.Errorf("ValidationConfig.MinPasswordLength must be > 0"))
	}
	if cfg.MaxPasswordLength < cfg.MinPasswordLength {
		return nil, fmt.Errorf("NewValidator: %w", fmt.Errorf("ValidationConfig.MaxPasswordLength must be >= MinPasswordLength"))
	}
	if cfg.MaxPasswordLength > 72 {
		return nil, fmt.Errorf("NewValidator: %w", fmt.Errorf("ValidationConfig.MaxPasswordLength must be > 0 and <= 72"))
	}

	return &validator{cfg: cfg}, nil
}

func (v *validator) ValidatePassword(password []byte) error {
	n := utf8.RuneCount(password)
	if n < v.cfg.MinPasswordLength {
		return fmt.Errorf("ValidatePassword: %w", ErrPasswordTooShort(v.cfg.MinPasswordLength))
	}
	if n > v.cfg.MaxPasswordLength {
		return fmt.Errorf("ValidatePassword: %w", ErrPasswordTooLong(v.cfg.MaxPasswordLength))
	}
	return nil
}

func (v *validator) ValidateEmail(address string, validateLength bool) error {
	_, err := mail.ParseAddress(address)
	if err != nil {
		return fmt.Errorf("ValidateEmail: %w", ErrInvalidEmail())
	}
	if validateLength && len(address) > v.cfg.MaxEmailLength {
		return fmt.Errorf("validateEmail: %w", ErrEmailTooLong(v.cfg.MaxEmailLength))
	}
	return nil
}

func (v *validator) ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("ValidateName: %w", ErrNameEmpty())
	}
	if len(name) > v.cfg.MaxNameLength {
		return fmt.Errorf("ValidateName: %w", ErrNameTooLong(v.cfg.MaxNameLength))
	}
	return nil
}

func (v *validator) NormalizeName(name string) string {
	return strings.TrimSpace(name)
}

func (v *validator) NormalizeEmail(address string) string {
	return strings.TrimSpace(strings.ToLower(address))
}

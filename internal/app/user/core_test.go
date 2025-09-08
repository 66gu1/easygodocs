package user_test

//go:generate minimock -o ./mocks -s _mock.go

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/66gu1/easygodocs/internal/app/user"
	"github.com/66gu1/easygodocs/internal/app/user/mocks"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func cfg() user.Config {
	return user.Config{
		PasswordHashCost: bcrypt.MinCost,
	}
}

type mock struct {
	validator      *mocks.ValidatorMock
	passwordHasher *mocks.PasswordHasherMock
	idGen          *mocks.IDGeneratorMock
	repo           *mocks.RepositoryMock
}

func TestNewCore(t *testing.T) {
	t.Parallel()

	var (
		repo           = mocks.NewRepositoryMock(t)
		idGen          = mocks.NewIDGeneratorMock(t)
		passwordHasher = mocks.NewPasswordHasherMock(t)
		validator      = mocks.NewValidatorMock(t)
	)

	tests := []struct {
		name    string
		repo    user.Repository
		idGen   user.IDGenerator
		hasher  user.PasswordHasher
		v       user.Validator
		cfg     user.Config
		wantErr bool
	}{
		{
			name:    "success",
			repo:    repo,
			idGen:   idGen,
			hasher:  passwordHasher,
			v:       validator,
			cfg:     cfg(),
			wantErr: false,
		},
		{
			name:    "error/nil_repo",
			repo:    nil,
			idGen:   idGen,
			hasher:  passwordHasher,
			v:       validator,
			cfg:     cfg(),
			wantErr: true,
		},
		{
			name:    "error/nil_idGen",
			repo:    repo,
			idGen:   nil,
			hasher:  passwordHasher,
			v:       validator,
			cfg:     cfg(),
			wantErr: true,
		},
		{
			name:    "error/nil_hasher",
			repo:    repo,
			idGen:   idGen,
			hasher:  nil,
			v:       validator,
			cfg:     cfg(),
			wantErr: true,
		},
		{
			name:    "error/nil_validator",
			repo:    repo,
			idGen:   idGen,
			hasher:  passwordHasher,
			v:       nil,
			cfg:     cfg(),
			wantErr: true,
		},
		{
			name:   "error_invalid_hash_cost/below_min",
			repo:   repo,
			idGen:  idGen,
			hasher: passwordHasher,
			v:      validator,
			cfg: user.Config{
				PasswordHashCost: bcrypt.MinCost - 1,
			},
			wantErr: true,
		},
		{
			name:   "error_invalid_hash_cost/above_max",
			repo:   repo,
			idGen:  idGen,
			hasher: passwordHasher,
			v:      validator,
			cfg: user.Config{
				PasswordHashCost: bcrypt.MaxCost + 1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := user.NewCore(tt.repo, tt.idGen, tt.hasher, tt.v, tt.cfg)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCore_CreateUser(t *testing.T) {
	t.Parallel()

	var (
		ctx             = context.Background()
		id              = uuid.New()
		hash            = []byte("hashed-pa$$123")
		req             = user.CreateUserReq{Email: " AAA@mail.com ", Name: " User ", Password: []byte("pa$1")}
		expErr          = errors.New(`expected error`)
		normalizedName  = "User"
		normalizedEmail = "aaa@mail.com"
		expReq          = user.CreateUserReq{Email: normalizedEmail, Name: normalizedName, Password: req.Password}
	)

	tests := []struct {
		name  string
		setup func(mocks mock)
		err   error
	}{
		{
			name: "success/normalized",
			setup: func(mocks mock) {
				mocks.validator.NormalizeNameMock.Expect(req.Name).Return(normalizedName)
				mocks.validator.ValidateNameMock.Expect(normalizedName).Return(nil)
				mocks.validator.NormalizeEmailMock.Expect(req.Email).Return(normalizedEmail)
				mocks.validator.ValidateEmailMock.Expect(normalizedEmail, true).Return(nil)
				mocks.validator.ValidatePasswordMock.Expect(req.Password).Return(nil)
				mocks.passwordHasher.HashPasswordMock.Expect(req.Password, cfg().PasswordHashCost).Return(hash, nil)
				mocks.idGen.NewMock.Return(id, nil)
				mocks.repo.CreateUserMock.Expect(ctx, expReq, id, string(hash)).Return(nil)
			},
		},
		{
			name: "error/validation/name",
			setup: func(mocks mock) {
				mocks.validator.NormalizeNameMock.Expect(req.Name).Return(normalizedName)
				mocks.validator.ValidateNameMock.Expect(normalizedName).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "error/validation/email",
			setup: func(mocks mock) {
				mocks.validator.NormalizeNameMock.Expect(req.Name).Return(normalizedName)
				mocks.validator.ValidateNameMock.Expect(normalizedName).Return(nil)
				mocks.validator.NormalizeEmailMock.Expect(req.Email).Return(normalizedEmail)
				mocks.validator.ValidateEmailMock.Expect(normalizedEmail, true).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "error/validation/password",
			setup: func(mocks mock) {
				mocks.validator.NormalizeNameMock.Expect(req.Name).Return(normalizedName)
				mocks.validator.ValidateNameMock.Expect(normalizedName).Return(nil)
				mocks.validator.NormalizeEmailMock.Expect(req.Email).Return(normalizedEmail)
				mocks.validator.ValidateEmailMock.Expect(normalizedEmail, true).Return(nil)
				mocks.validator.ValidatePasswordMock.Expect(req.Password).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "error/HashPassword",
			setup: func(mocks mock) {
				mocks.validator.NormalizeNameMock.Expect(req.Name).Return(normalizedName)
				mocks.validator.ValidateNameMock.Expect(normalizedName).Return(nil)
				mocks.validator.NormalizeEmailMock.Expect(req.Email).Return(normalizedEmail)
				mocks.validator.ValidateEmailMock.Expect(normalizedEmail, true).Return(nil)
				mocks.validator.ValidatePasswordMock.Expect(req.Password).Return(nil)
				mocks.passwordHasher.HashPasswordMock.Expect(req.Password, cfg().PasswordHashCost).Return(nil, expErr)
			},
			err: expErr,
		},
		{
			name: "error/idGen",
			setup: func(mocks mock) {
				mocks.validator.NormalizeNameMock.Expect(req.Name).Return(normalizedName)
				mocks.validator.ValidateNameMock.Expect(normalizedName).Return(nil)
				mocks.validator.NormalizeEmailMock.Expect(req.Email).Return(normalizedEmail)
				mocks.validator.ValidateEmailMock.Expect(normalizedEmail, true).Return(nil)
				mocks.validator.ValidatePasswordMock.Expect(req.Password).Return(nil)
				mocks.passwordHasher.HashPasswordMock.Expect(req.Password, cfg().PasswordHashCost).Return(hash, nil)
				mocks.idGen.NewMock.Return(uuid.Nil, expErr)
			},
			err: expErr,
		},
		{
			name: "error/repo",
			setup: func(mocks mock) {
				mocks.validator.NormalizeNameMock.Expect(req.Name).Return(normalizedName)
				mocks.validator.ValidateNameMock.Expect(normalizedName).Return(nil)
				mocks.validator.NormalizeEmailMock.Expect(req.Email).Return(normalizedEmail)
				mocks.validator.ValidateEmailMock.Expect(normalizedEmail, true).Return(nil)
				mocks.validator.ValidatePasswordMock.Expect(req.Password).Return(nil)
				mocks.passwordHasher.HashPasswordMock.Expect(req.Password, cfg().PasswordHashCost).Return(hash, nil)
				mocks.idGen.NewMock.Return(id, nil)
				mocks.repo.CreateUserMock.Expect(ctx, expReq, id, string(hash)).Return(expErr)
			},
			err: expErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := mock{
				validator:      mocks.NewValidatorMock(t),
				passwordHasher: mocks.NewPasswordHasherMock(t),
				idGen:          mocks.NewIDGeneratorMock(t),
				repo:           mocks.NewRepositoryMock(t),
			}

			if tt.setup != nil {
				tt.setup(m)
			}

			core, err := user.NewCore(m.repo, m.idGen, m.passwordHasher, m.validator, cfg())
			require.NoError(t, err)
			err = core.CreateUser(ctx, req)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCore_GetUser(t *testing.T) {
	t.Parallel()

	var (
		ctx  = context.Background()
		id   = uuid.New()
		hash = "hashed-pa$$123"
		want = user.User{
			ID:             id,
			Email:          "email",
			Name:           "name",
			SessionVersion: 0,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			DeletedAt:      nil,
		}
		expErr = errors.New(`expected error`)
	)

	tests := []struct {
		name  string
		setup func(mocks mock)
		in    uuid.UUID
		err   error
	}{
		{
			name: "success",
			in:   id,
			setup: func(mocks mock) {
				mocks.repo.GetUserMock.Expect(ctx, id).Return(want, hash, nil)
			},
		},
		{
			name: "error/validation/id",
			in:   uuid.Nil,
			err:  apperr.ErrNilUUID(""),
		},
		{
			name: "error/repo",
			err:  expErr,
			in:   id,
			setup: func(mocks mock) {
				mocks.repo.GetUserMock.Expect(ctx, id).Return(want, hash, expErr)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := mock{
				repo: mocks.NewRepositoryMock(t),
			}

			if tt.setup != nil {
				tt.setup(m)
			}

			core, err := user.NewCore(m.repo, m.idGen, m.passwordHasher, m.validator, cfg())
			require.NoError(t, err)
			usr, h, err := core.GetUser(ctx, tt.in)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, want, usr)
			require.Equal(t, hash, h)
		})
	}
}

func TestCore_GetAllUsers(t *testing.T) {
	t.Parallel()

	var (
		ctx    = context.Background()
		want   = []user.User{{ID: uuid.New(), Email: "email", Name: "name", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
		expErr = errors.New(`expected error`)
	)
	tests := []struct {
		name  string
		setup func(mocks mock)
		err   error
	}{
		{
			name: "success",
			setup: func(mocks mock) {
				mocks.repo.GetAllUsersMock.Expect(ctx).Return(want, nil)
			},
		},
		{
			name: "error/repo",
			err:  expErr,
			setup: func(mocks mock) {
				mocks.repo.GetAllUsersMock.Expect(ctx).Return(nil, expErr)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := mock{
				repo: mocks.NewRepositoryMock(t),
			}

			if tt.setup != nil {
				tt.setup(m)
			}

			core, err := user.NewCore(m.repo, m.idGen, m.passwordHasher, m.validator, cfg())
			require.NoError(t, err)
			users, err := core.GetAllUsers(ctx)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, want, users)
		})
	}
}

func TestCore_UpdateUser(t *testing.T) {
	t.Parallel()

	var (
		ctx             = context.Background()
		req             = user.UpdateUserReq{UserID: uuid.New(), Email: "mail", Name: "name"}
		expErr          = errors.New(`expected error`)
		normalizedName  = "n_name"
		normalizedEmail = "n_mail"
		expReq          = user.UpdateUserReq{UserID: req.UserID, Email: normalizedEmail, Name: normalizedName}
	)
	tests := []struct {
		name  string
		in    user.UpdateUserReq
		setup func(mocks mock)
		err   error
	}{
		{
			name: "success/normalized",
			in:   req,
			setup: func(mocks mock) {
				mocks.validator.NormalizeNameMock.Expect(req.Name).Return(normalizedName)
				mocks.validator.ValidateNameMock.Expect(normalizedName).Return(nil)
				mocks.validator.NormalizeEmailMock.Expect(req.Email).Return(normalizedEmail)
				mocks.validator.ValidateEmailMock.Expect(normalizedEmail, true).Return(nil)
				mocks.repo.UpdateUserMock.Expect(ctx, expReq).Return(nil)
			},
		},
		{
			name: "error/validation/id",
			in:   user.UpdateUserReq{UserID: uuid.Nil},
			err:  apperr.ErrNilUUID(""),
		},
		{
			name: "error/validation/name",
			in:   req,
			setup: func(mocks mock) {
				mocks.validator.NormalizeNameMock.Expect(req.Name).Return(normalizedName)
				mocks.validator.ValidateNameMock.Expect(normalizedName).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "error/validation/email",
			in:   req,
			setup: func(mocks mock) {
				mocks.validator.NormalizeNameMock.Expect(req.Name).Return(normalizedName)
				mocks.validator.ValidateNameMock.Expect(normalizedName).Return(nil)
				mocks.validator.NormalizeEmailMock.Expect(req.Email).Return(normalizedEmail)
				mocks.validator.ValidateEmailMock.Expect(normalizedEmail, true).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "error/repo",
			in:   req,
			err:  expErr,
			setup: func(mocks mock) {
				mocks.validator.NormalizeNameMock.Expect(req.Name).Return(normalizedName)
				mocks.validator.ValidateNameMock.Expect(normalizedName).Return(nil)
				mocks.validator.NormalizeEmailMock.Expect(req.Email).Return(normalizedEmail)
				mocks.validator.ValidateEmailMock.Expect(normalizedEmail, true).Return(nil)
				mocks.repo.UpdateUserMock.Expect(ctx, expReq).Return(expErr)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := mock{
				validator: mocks.NewValidatorMock(t),
				repo:      mocks.NewRepositoryMock(t),
			}

			if tt.setup != nil {
				tt.setup(m)
			}

			core, err := user.NewCore(m.repo, m.idGen, m.passwordHasher, m.validator, cfg())
			require.NoError(t, err)
			err = core.UpdateUser(ctx, tt.in)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCore_DeleteUser(t *testing.T) {
	t.Parallel()

	var (
		ctx    = context.Background()
		id     = uuid.New()
		expErr = errors.New(`expected error`)
	)
	tests := []struct {
		name  string
		setup func(mocks mock)
		in    uuid.UUID
		err   error
	}{
		{
			name: "success",
			in:   id,
			setup: func(mocks mock) {
				mocks.repo.DeleteUserMock.Expect(ctx, id).Return(nil)
			},
		},
		{
			name: "error/repo",
			in:   id,
			err:  expErr,
			setup: func(mocks mock) {
				mocks.repo.DeleteUserMock.Expect(ctx, id).Return(expErr)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := mock{
				repo: mocks.NewRepositoryMock(t),
			}

			if tt.setup != nil {
				tt.setup(m)
			}

			core, err := user.NewCore(m.repo, m.idGen, m.passwordHasher, m.validator, cfg())
			require.NoError(t, err)
			err = core.DeleteUser(ctx, tt.in)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCore_ChangePassword(t *testing.T) {
	t.Parallel()

	var (
		ctx      = context.Background()
		id       = uuid.New()
		password = []byte("pa$1")
		hash     = []byte("hashed-pa$$123")
		expErr   = errors.New(`expected error`)
	)
	tests := []struct {
		name  string
		setup func(mocks mock)
		in    uuid.UUID
		err   error
	}{
		{
			name: "success",
			in:   id,
			setup: func(mocks mock) {
				mocks.validator.ValidatePasswordMock.Expect(password).Return(nil)
				mocks.passwordHasher.HashPasswordMock.Expect(password, cfg().PasswordHashCost).Return(hash, nil)
				mocks.repo.ChangePasswordMock.Expect(ctx, id, string(hash)).Return(nil)
			},
		},
		{
			name: "error/validation/id",
			in:   uuid.Nil,
			err:  apperr.ErrNilUUID(""),
		},
		{
			name: "error/validation/password",
			in:   id,
			setup: func(mocks mock) {
				mocks.validator.ValidatePasswordMock.Expect(password).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "error/hash_password",
			in:   id,
			setup: func(mocks mock) {
				mocks.validator.ValidatePasswordMock.Expect(password).Return(nil)
				mocks.passwordHasher.HashPasswordMock.Expect(password, cfg().PasswordHashCost).Return(nil, expErr)
			},
			err: expErr,
		},
		{
			name: "error/repo",
			in:   id,
			err:  expErr,
			setup: func(mocks mock) {
				mocks.validator.ValidatePasswordMock.Expect(password).Return(nil)
				mocks.passwordHasher.HashPasswordMock.Expect(password, cfg().PasswordHashCost).Return(hash, nil)
				mocks.repo.ChangePasswordMock.Expect(ctx, id, string(hash)).Return(expErr)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := mock{
				repo:           mocks.NewRepositoryMock(t),
				passwordHasher: mocks.NewPasswordHasherMock(t),
				validator:      mocks.NewValidatorMock(t),
			}

			if tt.setup != nil {
				tt.setup(m)
			}

			core, err := user.NewCore(m.repo, m.idGen, m.passwordHasher, m.validator, cfg())
			require.NoError(t, err)
			err = core.ChangePassword(ctx, tt.in, password)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCore_GetUserByEmail(t *testing.T) {
	t.Parallel()

	var (
		ctx             = context.Background()
		email           = "email"
		normalizedEmail = "n_email"
		id              = uuid.New()
		hash            = "hashed-pa$$123"
		want            = user.User{
			ID:             id,
			Email:          "email",
			Name:           "name",
			SessionVersion: 0,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			DeletedAt:      nil,
		}
		expErr = errors.New(`expected error`)
	)
	tests := []struct {
		name  string
		setup func(mocks mock)
		in    string
		err   error
	}{
		{
			name: "success/normalized",
			in:   email,
			setup: func(mocks mock) {
				mocks.validator.NormalizeEmailMock.Expect(email).Return(normalizedEmail)
				mocks.validator.ValidateEmailMock.Expect(normalizedEmail, false).Return(nil)
				mocks.repo.GetUserByEmailMock.Expect(ctx, normalizedEmail).Return(want, hash, nil)
			},
		},
		{
			name: "error/validation/email",
			in:   email,
			setup: func(mocks mock) {
				mocks.validator.NormalizeEmailMock.Expect(email).Return(normalizedEmail)
				mocks.validator.ValidateEmailMock.Expect(normalizedEmail, false).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "error/repo",
			in:   email,
			err:  expErr,
			setup: func(mocks mock) {
				mocks.validator.NormalizeEmailMock.Expect(email).Return(normalizedEmail)
				mocks.validator.ValidateEmailMock.Expect(normalizedEmail, false).Return(nil)
				mocks.repo.GetUserByEmailMock.Expect(ctx, normalizedEmail).Return(want, hash, expErr)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := mock{
				repo:      mocks.NewRepositoryMock(t),
				validator: mocks.NewValidatorMock(t),
			}

			if tt.setup != nil {
				tt.setup(m)
			}

			core, err := user.NewCore(m.repo, m.idGen, m.passwordHasher, m.validator, cfg())
			require.NoError(t, err)
			usr, h, err := core.GetUserByEmail(ctx, tt.in)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, want, usr)
			require.Equal(t, hash, h)
		})
	}
}

func vCFG() user.ValidationConfig {
	return user.ValidationConfig{
		MaxEmailLength:    15,
		MaxNameLength:     5,
		MinPasswordLength: 3,
		MaxPasswordLength: 5,
	}
}

func TestNewValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     user.ValidationConfig
		wantErr bool
	}{
		{
			name: "success",
			cfg:  vCFG(),
		},
		{
			name: "error/max_email_length",
			cfg: user.ValidationConfig{
				MaxEmailLength:    0,
				MaxNameLength:     5,
				MinPasswordLength: 3,
				MaxPasswordLength: 5,
			},
			wantErr: true,
		},
		{
			name: "error/max_name_length",
			cfg: user.ValidationConfig{
				MaxEmailLength:    15,
				MaxNameLength:     0,
				MinPasswordLength: 3,
				MaxPasswordLength: 5,
			},
			wantErr: true,
		},
		{
			name: "error/min_password_length",
			cfg: user.ValidationConfig{
				MaxEmailLength:    15,
				MaxNameLength:     5,
				MinPasswordLength: 0,
				MaxPasswordLength: 5,
			},
			wantErr: true,
		},
		{
			name: "error/max_password_length < min",
			cfg: user.ValidationConfig{
				MaxEmailLength:    15,
				MaxNameLength:     5,
				MinPasswordLength: 6,
				MaxPasswordLength: 5,
			},
			wantErr: true,
		},
		{
			name: "error/max_password_length > 72",
			cfg: user.ValidationConfig{
				MaxEmailLength:    15,
				MaxNameLength:     5,
				MinPasswordLength: 3,
				MaxPasswordLength: 73,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := user.NewValidator(tt.cfg)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidator_ValidateName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		err  error
	}{
		{
			name: "success",
			in:   "name",
		},
		{
			name: "error/empty",
			in:   "",
			err:  user.ErrNameEmpty(),
		},
		{
			name: "error/too long",
			in:   "long_name",
			err:  user.ErrNameTooLong(vCFG().MaxNameLength),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v, err := user.NewValidator(vCFG())
			require.NoError(t, err)

			err = v.ValidateName(tt.in)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidator_ValidateEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		in             string
		validateLength bool
		err            error
	}{
		{
			name:           "success",
			in:             "email@email.com",
			validateLength: true,
		},
		{
			name:           "success/no length validation",
			in:             "long_email@email.com",
			validateLength: false,
		},
		{
			name:           "error/invalid",
			in:             "invalid_email",
			validateLength: true,
			err:            user.ErrInvalidEmail(),
		},
		{
			name:           "error/too long",
			in:             "long_email@email.com",
			validateLength: true,
			err:            user.ErrEmailTooLong(vCFG().MaxEmailLength),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v, err := user.NewValidator(vCFG())
			require.NoError(t, err)

			err = v.ValidateEmail(tt.in, tt.validateLength)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidator_ValidatePassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []byte
		err  error
	}{
		{
			name: "success",
			in:   []byte("pa$1"),
		},
		{
			name: "error/too short",
			in:   []byte("p$"),
			err:  user.ErrPasswordTooShort(vCFG().MinPasswordLength),
		},
		{
			name: "error/too long",
			in:   []byte("long_pa$1"),
			err:  user.ErrPasswordTooLong(vCFG().MaxPasswordLength),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v, err := user.NewValidator(vCFG())
			require.NoError(t, err)

			err = v.ValidatePassword(tt.in)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidator_NormalizeEmail(t *testing.T) {
	t.Parallel()

	var (
		in   = "  MAIL@mail.com  "
		want = "mail@mail.com"
	)
	v, err := user.NewValidator(vCFG())
	require.NoError(t, err)
	got := v.NormalizeEmail(in)
	require.Equal(t, want, got)
}

func TestValidator_NormalizeName(t *testing.T) {
	t.Parallel()

	var (
		in   = "  Name  "
		want = "Name"
	)
	v, err := user.NewValidator(vCFG())
	require.NoError(t, err)
	got := v.NormalizeName(in)
	require.Equal(t, want, got)
}

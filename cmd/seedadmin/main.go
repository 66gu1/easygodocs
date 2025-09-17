package main

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"time"

	"github.com/66gu1/easygodocs/config"
	"github.com/66gu1/easygodocs/internal/app/auth"
	authrepo "github.com/66gu1/easygodocs/internal/app/auth/repo/gorm"
	"github.com/66gu1/easygodocs/internal/app/user"
	userrepo "github.com/66gu1/easygodocs/internal/app/user/repo/gorm"
	"github.com/66gu1/easygodocs/internal/infrastructure/secure"
	"github.com/66gu1/easygodocs/internal/infrastructure/system"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	err := godotenv.Overload(".env")
	if err != nil {
		log.Debug().Err(err).Msg("failed to load .env.local file, using environment variables")
	}
	dsn := os.Getenv("DATABASE_DSN")
	email := os.Getenv("ADMIN_EMAIL")
	pass := os.Getenv("ADMIN_PASSWORD")

	if dsn == "" || email == "" || pass == "" {
		panic("DATABASE_DSN, ADMIN_EMAIL and ADMIN_PASSWORD environment variables are required")
	}

	_ = config.LoadConfig()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(err)
	}

	authRepo, err := authrepo.NewRepository(db)
	if err != nil {
		panic(err)
	}
	// we don't need jwt secret here or config, because we just assign role
	authCore, err := auth.NewCore(authRepo, secure.NewTokenCodec(ephemeralKey()), &system.UUIDv7Generator{}, &system.RNDGenerator{}, &system.TimeGenerator{}, secure.NewPasswordHasher(), auth.Config{SessionTTLMinutes: 1, AccessTokenTTLMinutes: 1})
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	id := createUser(ctx, db, email, pass)
	err = authCore.AddUserRole(ctx, auth.UserRole{
		UserID: id,
		Role:   auth.RoleAdmin,
	})
	if err != nil {
		if errors.Is(err, auth.ErrDuplicateUserRole()) {
			log.Warn().Msgf("User with email already has admin role, skip adding role")
		} else {
			panic(err)
		}
	} else {
		log.Info().Msgf("Admin role added to user with ID: %s", id.String())
	}
}

func createUser(ctx context.Context, db *gorm.DB, email, pass string) uuid.UUID {
	userRepo, err := userrepo.NewRepository(db)
	if err != nil {
		panic(err)
	}
	cfg, vCFG := config.GetUserConfigs()
	validator, err := user.NewValidator(vCFG)
	if err != nil {
		panic(err)
	}
	core, err := user.NewCore(userRepo, &system.UUIDv7Generator{}, secure.NewPasswordHasher(), validator, cfg)
	if err != nil {
		panic(err)
	}

	id, err := core.CreateUser(ctx, user.CreateUserReq{
		Email:    email,
		Password: []byte(pass),
		Name:     "Admin",
	})
	if err != nil {
		if errors.Is(err, user.ErrUserWithEmailAlreadyExists()) {
			log.Warn().Msgf("User with email already exists, skip creating admin user")
			usr, _, err := core.GetUserByEmail(ctx, email)
			if err != nil {
				panic(err)
			}
			id = usr.ID
			log.Info().Msgf("Admin user ID: %s", id.String())
		} else {
			panic(err)
		}
	} else {
		log.Info().Msgf("Admin user created with ID: %s", id.String())
	}
	return id
}

func ephemeralKey() []byte {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}

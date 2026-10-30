// @title        EasyGoDocs API
// @version      1.0
// @description  Demo wiki backend in Go
// @BasePath     /api/v1
// @schemes       http

// @securityDefinitions.apikey BearerAuth
// @type          apiKey
// @in header
// @name Authorization
// (format: "Bearer <token>")
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/66gu1/easygodocs/config"
	"github.com/66gu1/easygodocs/docs"
	"github.com/66gu1/easygodocs/internal/infrastructure/secure"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	cfg := config.LoadConfig()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(cfg.LogLevel.ZeroLog())
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load doesn't override variables that are already set: the environment wins over the file,
	// so a setting can be overridden at launch (REDIS_URL=... make run) and a stray .env can't
	// silently replace the values a container is given.
	err := godotenv.Load(".env")
	if err != nil {
		log.Debug().Err(err).Msg("failed to load .env file, using environment variables")
	}

	dsn, ok := os.LookupEnv("DATABASE_DSN")
	if !ok {
		panic("DATABASE_DSN environment variable is required")
	}
	db := mustOpenDB(dsn, cfg.DBPool)

	redisURL, ok := os.LookupEnv("REDIS_URL")
	if !ok {
		panic("REDIS_URL environment variable is required")
	}
	redisClient := mustOpenRedis(redisURL)

	docs.SwaggerInfo.BasePath = "/api/v1"
	userCfg, userValidationCfg := config.GetUserConfigs()
	entityCfg, entityValidationCfg := config.GetEntityConfigs()
	handler, err := newHandler(db, redisClient, appConfig{
		App:              cfg,
		User:             userCfg,
		UserValidation:   userValidationCfg,
		Auth:             config.GetAuthConfigs(),
		Entity:           entityCfg,
		EntityValidation: entityValidationCfg,
		JWTSecret:        mustJWTSecret(),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to set up the application")
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Info().Msg(fmt.Sprintf("starting server on :%s", cfg.Port))
	if err = srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Msg("server error")
	}
}

// mustJWTSecret reads JWT_SECRET and refuses to start with a missing or weak secret.
func mustJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < secure.MinJWTSecretLength {
		log.Fatal().Msgf("JWT_SECRET must be at least %d bytes long", secure.MinJWTSecretLength)
	}
	return []byte(secret)
}

// mustOpenRedis connects to Redis and fails fast if it isn't reachable at startup,
// so a misconfigured address is noticed immediately rather than hidden by the rate limiter failing open.
func mustOpenRedis(url string) *redis.Client {
	opts, err := redis.ParseURL(url)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse REDIS_URL")
	}
	// Without it go-redis ignores context deadlines for network I/O, and the limiter timeout wouldn't apply.
	opts.ContextTimeoutEnabled = true

	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = client.Ping(ctx).Err()
	cancel()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}
	return client
}

func mustOpenDB(dsn string, pool config.DBPoolConfig) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open database")
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get *sql.DB from gorm")
	}
	sqlDB.SetMaxOpenConns(pool.MaxOpenConns)
	sqlDB.SetMaxIdleConns(pool.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(pool.ConnMaxLifetimeMinutes) * time.Minute)

	return db
}

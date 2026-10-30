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
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/66gu1/easygodocs/config"
	"github.com/66gu1/easygodocs/docs"
	"github.com/66gu1/easygodocs/internal/app/auth"
	authrepo "github.com/66gu1/easygodocs/internal/app/auth/repo/gorm"
	authhttp "github.com/66gu1/easygodocs/internal/app/auth/transport/http"
	authusecase "github.com/66gu1/easygodocs/internal/app/auth/usecase"
	"github.com/66gu1/easygodocs/internal/app/entity"
	entityrepo "github.com/66gu1/easygodocs/internal/app/entity/repo/gorm"
	entityhttp "github.com/66gu1/easygodocs/internal/app/entity/transport/http"
	entityusecase "github.com/66gu1/easygodocs/internal/app/entity/usecase"
	"github.com/66gu1/easygodocs/internal/app/user"
	userrepo "github.com/66gu1/easygodocs/internal/app/user/repo/gorm"
	userhttp "github.com/66gu1/easygodocs/internal/app/user/transport/http"
	userusecase "github.com/66gu1/easygodocs/internal/app/user/usecase"
	dbinfra "github.com/66gu1/easygodocs/internal/infrastructure/db"
	"github.com/66gu1/easygodocs/internal/infrastructure/httpx"
	"github.com/66gu1/easygodocs/internal/infrastructure/ratelimit"
	"github.com/66gu1/easygodocs/internal/infrastructure/secure"
	"github.com/66gu1/easygodocs/internal/infrastructure/system"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	httpSwagger "github.com/swaggo/http-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	cfg := config.LoadConfig()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(cfg.LogLevel.ZeroLog())
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	err := godotenv.Overload(".env")
	if err != nil {
		log.Debug().Err(err).Msg("failed to load .env.local file, using environment variables")
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
	publicLimiter := mustPublicLimiter(redisClient, cfg.RateLimit)
	loginAttempts := mustLoginAttempts(redisClient, cfg.RateLimit)

	jwtCodec := secure.NewTokenCodec(mustJWTSecret(), auth.TokenIssuer)

	idGen := &system.UUIDv7Generator{}
	timeGen := &system.TimeGenerator{}
	rndGen := &system.RNDGenerator{}
	passwordHasher := secure.NewPasswordHasher()

	userCfg, userValidationCfg := config.GetUserConfigs()
	userRepo, err := userrepo.NewRepository(db)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create user repository")
	}
	userValidator, err := user.NewValidator(userValidationCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create user validator")
	}
	userCore, err := user.NewCore(userRepo, idGen, passwordHasher, userValidator, userCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create user core")
	}

	authCfg := config.GetAuthConfigs()
	authRepo, err := authrepo.NewRepository(db)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create auth repository")
	}
	authCore, err := auth.NewCore(authRepo, jwtCodec, idGen, rndGen, timeGen, secure.NewRefreshTokenHasher(), authCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create auth core")
	}

	entityCfg, entityValidationCfg := config.GetEntityConfigs()
	entityRepo, err := entityrepo.NewRepository(db)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create entity repository")
	}
	entityValidator, err := entity.NewValidator(entityValidationCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create entity validator")
	}
	txManager, err := dbinfra.NewTxManager(db)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create tx manager")
	}
	entityCore, err := entity.NewCore(entityRepo, txManager, entity.Generators{
		ID:   idGen,
		Time: timeGen,
	}, entityValidator, entityCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create entity core")
	}

	userService := userusecase.NewService(userCore, authCore, passwordHasher)
	userHandler := userhttp.NewHandler(userService)

	dummyPasswordHash := mustDummyPasswordHash(passwordHasher, userCfg.PasswordHashCost)
	authService := authusecase.NewService(authCore, userCore, passwordHasher, loginAttempts, dummyPasswordHash)
	authHandler := authhttp.NewHandler(authService)

	entityService := entityusecase.NewService(entityCore, entityusecase.NewPermissionChecker(entityCore, authCore))
	entityHandler := entityhttp.NewHandler(entityService)

	docs.SwaggerInfo.BasePath = "/api/v1"
	// --- set up chi router
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(httpx.ClientIPMiddleware(cfg.TrustedProxies))
	r.Use(middleware.Recoverer)
	r.Use(httpx.Logger)
	r.Use(httpx.MaxBodyBytes(cfg.MaxBodySize))

	r.Route("/api/v1", func(r chi.Router) {
		// with auth
		r.Group(func(r chi.Router) {
			r.Use(authhttp.AuthMiddleware(jwtCodec, authService))
			// --- user routes
			r.Route("/users", func(r chi.Router) {
				r.Get("/", userHandler.GetAllUsers) // GET    /users

				r.Route(fmt.Sprintf("/{%s}", userhttp.URLParamUserID), func(r chi.Router) {
					r.Get("/", userHandler.GetUser)                 // GET    /users/{user_id}
					r.Put("/", userHandler.UpdateUser)              // PUT    /users/{user_id}
					r.Delete("/", userHandler.DeleteUser)           // DELETE /users/{user_id}
					r.Post("/password", userHandler.ChangePassword) // POST   /users/{user_id}/password
				})
			})

			// --- session routes
			r.Route("/sessions", func(r chi.Router) {
				r.Get("/", authHandler.GetSessionsByUserID)       // GET    /sessions?user_id={user_id}
				r.Delete("/", authHandler.DeleteSessionsByUserID) // DELETE /sessions?user_id={user_id}

				r.Route(fmt.Sprintf("/{%s}", authhttp.URLParamSessionID), func(r chi.Router) {
					r.Delete("/", authHandler.DeleteSession) // DELETE /sessions/{session_id}?user_id={user_id}
				})
			})

			// --- roles routes
			r.Route("/roles", func(r chi.Router) {
				r.Get("/", authHandler.ListUserRoles)     // GET /roles
				r.Post("/", authHandler.AddUserRole)      // POST /roles
				r.Delete("/", authHandler.DeleteUserRole) // DELETE /roles
			})

			// --- entity routes
			r.Route("/entities", func(r chi.Router) {
				r.Post("/", entityHandler.Create) // POST /entities
				r.Get("/", entityHandler.GetTree) // GET /entities

				r.Route(fmt.Sprintf("/{%s}", entityhttp.URLParamEntityID), func(r chi.Router) {
					r.Get("/", entityHandler.Get)       // GET    /entities/{entity_id}
					r.Put("/", entityHandler.Update)    // PUT    /entities/{entity_id}
					r.Delete("/", entityHandler.Delete) // DELETE /entities/{entity_id}

					r.Route("/versions", func(r chi.Router) {
						r.Get("/", entityHandler.GetVersionsList) // GET /entities/{entity_id}/versions

						r.Route(fmt.Sprintf("/{%s}", entityhttp.URLParamVersion), func(r chi.Router) {
							r.Get("/", entityHandler.GetVersion) // GET /entities/{entity_id}/versions/{version}
						})
					})

					r.Route("/draft", func(r chi.Router) {
						r.Get("/", entityHandler.GetDraft)             // GET    /entities/{entity_id}/draft
						r.Put("/", entityHandler.SaveDraft)            // PUT    /entities/{entity_id}/draft
						r.Delete("/", entityHandler.DeleteDraft)       // DELETE /entities/{entity_id}/draft
						r.Post("/publish", entityHandler.PublishDraft) // POST   /entities/{entity_id}/draft/publish
					})
				})
			})
		})

		// without auth
		r.Group(func(r chi.Router) {
			r.Use(httpx.RateLimit(publicLimiter, httpx.KeyByClientIP))
			r.Post("/login", authHandler.Login)           // POST /login
			r.Post("/refresh", authHandler.RefreshTokens) // POST /refresh
			r.Post("/register", userHandler.CreateUser)   // POST /register
		})

		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("/api/v1/swagger/doc.json"),
		))
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
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

// mustDummyPasswordHash creates a hash with the same cost as real password hashes.
// It is used to equalize login timing for unknown emails.
func mustDummyPasswordHash(hasher *secure.PasswordHasher, cost int) []byte {
	password := make([]byte, 32)
	if _, err := rand.Read(password); err != nil {
		log.Fatal().Err(err).Msg("failed to generate dummy password")
	}
	hash, err := hasher.HashPassword(password, cost)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create dummy password hash")
	}
	return hash
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

// mustPublicLimiter limits unauthenticated endpoints per client IP.
func mustPublicLimiter(client *redis.Client, cfg config.RateLimitConfig) *ratelimit.Limiter {
	limiter, err := ratelimit.NewLimiter(client, ratelimit.Config{
		Prefix:  "ratelimit:public:",
		Limit:   cfg.PublicRequestsPerMinute,
		Window:  time.Minute,
		Timeout: time.Duration(cfg.TimeoutMilliseconds) * time.Millisecond,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create rate limiter")
	}
	return limiter
}

// mustLoginAttempts limits password guesses per account.
func mustLoginAttempts(client *redis.Client, cfg config.RateLimitConfig) *ratelimit.LoginAttempts {
	attempts, err := ratelimit.NewLoginAttempts(client, ratelimit.LoginAttemptsConfig{
		PerEmailAndIP: cfg.LoginAttemptsPerEmailAndIP,
		PerEmail:      cfg.LoginAttemptsPerEmail,
		Window:        time.Duration(cfg.LoginWindowMinutes) * time.Minute,
		Timeout:       time.Duration(cfg.TimeoutMilliseconds) * time.Millisecond,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create login attempts limiter")
	}
	return attempts
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

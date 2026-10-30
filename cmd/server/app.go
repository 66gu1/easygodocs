package main

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"github.com/66gu1/easygodocs/config"
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
	"github.com/66gu1/easygodocs/internal/infrastructure/revocation"
	"github.com/66gu1/easygodocs/internal/infrastructure/secure"
	"github.com/66gu1/easygodocs/internal/infrastructure/system"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
	"gorm.io/gorm"
)

type appConfig struct {
	App              config.Config
	User             user.Config
	UserValidation   user.ValidationConfig
	Auth             auth.Config
	Entity           entity.Config
	EntityValidation entity.ValidationConfig
	JWTSecret        []byte
}

// newHandler wires the application and returns its HTTP handler.
// The integration tests use it too, so they run the same wiring and routes as the server.
func newHandler(db *gorm.DB, redisClient *redis.Client, cfg appConfig) (http.Handler, error) {
	redisTimeout := time.Duration(cfg.App.RateLimit.TimeoutMilliseconds) * time.Millisecond
	publicLimiter, err := ratelimit.NewLimiter(redisClient, ratelimit.Config{
		Prefix:  "ratelimit:public:",
		Limit:   cfg.App.RateLimit.PublicRequestsPerMinute,
		Window:  time.Minute,
		Timeout: redisTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("newHandler: %w", err)
	}
	loginAttempts, err := ratelimit.NewLoginAttempts(redisClient, ratelimit.LoginAttemptsConfig{
		PerEmailAndIP: cfg.App.RateLimit.LoginAttemptsPerEmailAndIP,
		PerEmail:      cfg.App.RateLimit.LoginAttemptsPerEmail,
		Window:        time.Duration(cfg.App.RateLimit.LoginWindowMinutes) * time.Minute,
		Timeout:       redisTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("newHandler: %w", err)
	}
	// Entries live as long as a session can, see revocation.Store.
	revocations, err := revocation.NewStore(redisClient, time.Duration(cfg.Auth.SessionTTLMinutes)*time.Minute, redisTimeout)
	if err != nil {
		return nil, fmt.Errorf("newHandler: %w", err)
	}

	jwtCodec := secure.NewTokenCodec(cfg.JWTSecret, auth.TokenIssuer)
	idGen := &system.UUIDv7Generator{}
	timeGen := &system.TimeGenerator{}
	rndGen := &system.RNDGenerator{}
	passwordHasher := secure.NewPasswordHasher()

	userRepo, err := userrepo.NewRepository(db)
	if err != nil {
		return nil, fmt.Errorf("newHandler: %w", err)
	}
	userValidator, err := user.NewValidator(cfg.UserValidation)
	if err != nil {
		return nil, fmt.Errorf("newHandler: %w", err)
	}
	userCore, err := user.NewCore(userRepo, idGen, passwordHasher, userValidator, cfg.User)
	if err != nil {
		return nil, fmt.Errorf("newHandler: %w", err)
	}

	authRepo, err := authrepo.NewRepository(db)
	if err != nil {
		return nil, fmt.Errorf("newHandler: %w", err)
	}
	authCore, err := auth.NewCore(authRepo, jwtCodec, idGen, rndGen, timeGen, secure.NewRefreshTokenHasher(), cfg.Auth)
	if err != nil {
		return nil, fmt.Errorf("newHandler: %w", err)
	}

	entityRepo, err := entityrepo.NewRepository(db)
	if err != nil {
		return nil, fmt.Errorf("newHandler: %w", err)
	}
	entityValidator, err := entity.NewValidator(cfg.EntityValidation)
	if err != nil {
		return nil, fmt.Errorf("newHandler: %w", err)
	}
	txManager, err := dbinfra.NewTxManager(db)
	if err != nil {
		return nil, fmt.Errorf("newHandler: %w", err)
	}
	entityCore, err := entity.NewCore(entityRepo, txManager, entity.Generators{
		ID:   idGen,
		Time: timeGen,
	}, entityValidator, cfg.Entity)
	if err != nil {
		return nil, fmt.Errorf("newHandler: %w", err)
	}

	dummyPasswordHash, err := dummyPasswordHash(passwordHasher, cfg.User.PasswordHashCost)
	if err != nil {
		return nil, fmt.Errorf("newHandler: %w", err)
	}
	authService := authusecase.NewService(authusecase.Deps{
		Core:           authCore,
		UserCore:       userCore,
		PasswordHasher: passwordHasher,
		LoginAttempts:  loginAttempts,
		Revocations:    revocations,
		TimeGenerator:  timeGen,
	}, dummyPasswordHash)
	authHandler := authhttp.NewHandler(authService)

	userService := userusecase.NewService(userCore, authCore, authService, passwordHasher)
	userHandler := userhttp.NewHandler(userService)

	entityService := entityusecase.NewService(entityCore, entityusecase.NewPermissionChecker(entityCore, authCore))
	entityHandler := entityhttp.NewHandler(entityService)

	// --- set up chi router
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(httpx.ClientIPMiddleware(cfg.App.TrustedProxies))
	r.Use(middleware.Recoverer)
	r.Use(httpx.Logger)
	r.Use(httpx.MaxBodyBytes(cfg.App.MaxBodySize))

	r.Route("/api/v1", func(r chi.Router) {
		// with auth
		r.Group(func(r chi.Router) {
			r.Use(authhttp.AuthMiddleware(jwtCodec, revocations, authService))
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

	return r, nil
}

// dummyPasswordHash creates a hash with the same cost as real password hashes.
// It is used to equalize login timing for unknown emails.
func dummyPasswordHash(hasher *secure.PasswordHasher, cost int) ([]byte, error) {
	password := make([]byte, 32)
	if _, err := rand.Read(password); err != nil {
		return nil, fmt.Errorf("dummyPasswordHash: %w", err)
	}
	hash, err := hasher.HashPassword(password, cost)
	if err != nil {
		return nil, fmt.Errorf("dummyPasswordHash: %w", err)
	}
	return hash, nil
}

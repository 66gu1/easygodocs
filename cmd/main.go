package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

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
	"github.com/66gu1/easygodocs/internal/infrastructure/httpx"
	"github.com/66gu1/easygodocs/internal/infrastructure/secure"
	"github.com/66gu1/easygodocs/internal/infrastructure/system"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := loadConfig()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(cfg.LogLevel.zeroLog())
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	err := godotenv.Overload(".env")
	if err != nil {
		log.Debug().Err(err).Msg("failed to load .env.local file, using environment variables")
	}

	password := os.Getenv("DB_PASSWORD")
	dsn := fmt.Sprintf("%s password=%s", cfg.DatabaseDSN, password)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		panic(err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	jwtCodec := secure.NewTokenCodec([]byte(jwtSecret))

	idGen := &system.UUIDv7Generator{}
	timeGen := &system.TimeGenerator{}
	rndGen := &system.RNDGenerator{}
	passwordHasher := secure.NewPasswordHasher()

	userCfg, userValidationCfg := getUserConfigs()
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

	authCfg := getAuthConfigs()
	authRepo, err := authrepo.NewRepository(db)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create auth repository")
	}
	authCore, err := auth.NewCore(authRepo, jwtCodec, idGen, rndGen, timeGen, passwordHasher, authCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create auth core")
	}

	entityCfg, entityValidationCfg := getEntityConfigs()
	entityRepo, err := entityrepo.NewRepository(db, entityCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create entity repository")
	}
	entityValidator, err := entity.NewValidator(entityValidationCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create entity validator")
	}
	entityCore, err := entity.NewCore(entityRepo, entity.Generators{
		ID:   idGen,
		Time: timeGen,
	}, entityValidator)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create entity core")
	}

	userService := userusecase.NewService(userCore, authCore, passwordHasher)
	userHandler := userhttp.NewHandler(userService)

	authService := authusecase.NewService(authCore, userCore, passwordHasher)
	authHandler := authhttp.NewHandler(authService)

	entityService := entityusecase.NewService(entityCore, entityusecase.NewPermissionChecker(entityCore, authCore))
	entityHandler := entityhttp.NewHandler(entityService)

	// --- set up chi router
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Use(httpx.MaxBodyBytes(cfg.MaxBodySize))

	// with auth
	r.Group(func(r chi.Router) {
		r.Use(authhttp.AuthMiddleware(jwtCodec))
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
			})
		})
	})

	// without auth
	r.Group(func(r chi.Router) {
		r.Post("/login", authHandler.Login)           // POST /login
		r.Post("/refresh", authHandler.RefreshTokens) // POST /refresh
		r.Post("/register", userHandler.CreateUser)   // POST /register
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

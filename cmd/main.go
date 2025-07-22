package main

import (
	"errors"
	"fmt"
	"github.com/66gu1/easygodocs/internal/app/article"
	"github.com/66gu1/easygodocs/internal/app/department"
	"github.com/66gu1/easygodocs/internal/infrastructure/auth"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"net/http"
	"os"
	"time"
)

func main() {
	cfg := getConfig()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(cfg.LogLevel.zeroLog())
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	err := godotenv.Load() // looks for .env in current directory
	if err != nil {
		log.Debug().Err(err).Msg("failed to load .env file, using environment variables")
	}

	password := os.Getenv("DB_PASSWORD")
	dsn := fmt.Sprintf("%s password=%s", cfg.DatabaseDSN, password)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NowFunc: func() time.Time {
			return time.Now().UTC()
		}})
	if err != nil {
		panic(err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal().Msg("JWT_SECRET environment variable not set")
	}
	authService := auth.New(jwtSecret)

	// --- set up chi router
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)

	departmentService := department.NewService(department.NewRepository(db), nil)
	articleService := article.NewService(article.NewRepository(db), departmentService)
	departmentService.SetArticleService(articleService)

	departmentHandler := department.NewHandler(departmentService)
	articleHandler := article.NewHandler(articleService)

	r.Group(func(p chi.Router) {
		p.Use(authService.AuthMiddleware)
		// --- department routes
		p.Route("/departments", func(r chi.Router) {
			p.Get("/", departmentHandler.GetDepartmentTree) // GET    /departments
			p.Post("/", departmentHandler.Create)           // POST   /departments

			p.Route("/{id}", func(r chi.Router) {
				p.Put("/", departmentHandler.Update)    // PUT    /departments/{id}
				p.Delete("/", departmentHandler.Delete) // DELETE /departments/{id}
			})
		})

		// --- article routes
		p.Route("/articles", func(r chi.Router) {
			p.Post("/", articleHandler.Create) // POST   /articles
			p.Post("/draft", articleHandler.CreateDraft)

			p.Route("/{id}", func(r chi.Router) {
				p.Get("/", articleHandler.Get)                          // GET    /articles/{id}
				p.Put("/", articleHandler.Update)                       // PUT    /articles/{id}
				p.Delete("/", articleHandler.Delete)                    // DELETE /articles/{id}
				p.Get("/versions", articleHandler.GetVersionsList)      // GET    /articles/{id}/versions
				p.Get("/versions/{version}", articleHandler.GetVersion) // GET    /articles/{id}/versions/{version}
			})
		})
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Info().Msg("starting server on :8080")
	if err = srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Msg("server error")
	}

}

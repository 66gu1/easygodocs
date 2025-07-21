package main

import (
	"errors"
	"fmt"
	"github.com/66gu1/easygodocs/internal/app/article"
	"github.com/66gu1/easygodocs/internal/app/department"
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
	// --- department routes
	r.Route("/departments", func(r chi.Router) {
		r.Get("/", departmentHandler.GetDepartmentTree) // GET    /departments
		r.Post("/", departmentHandler.Create)           // POST   /departments

		r.Route("/{id}", func(r chi.Router) {
			r.Put("/", departmentHandler.Update)    // PUT    /departments/{id}
			r.Delete("/", departmentHandler.Delete) // DELETE /departments/{id}
		})
	})

	artcleHandler := article.NewHandler(articleService)
	// --- article routes
	r.Route("/articles", func(r chi.Router) {
		r.Post("/", artcleHandler.Create) // POST   /articles
		r.Post("/draft", artcleHandler.CreateDraft)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", artcleHandler.Get)                          // GET    /articles/{id}
			r.Put("/", artcleHandler.Update)                       // PUT    /articles/{id}
			r.Delete("/", artcleHandler.Delete)                    // DELETE /articles/{id}
			r.Get("/versions", artcleHandler.GetVersionsList)      // GET    /articles/{id}/versions
			r.Get("/versions/{version}", artcleHandler.GetVersion) // GET    /articles/{id}/versions/{version}
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

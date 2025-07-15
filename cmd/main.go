package main

import (
	"github.com/66gu1/easygodocs/internal/app/department"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"net/http"
	"os"
	"time"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	dsn := "host=localhost user=postgres password=4502 dbname=easy_go_docs port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NowFunc: func() time.Time {
			return time.Now().UTC()
		}})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	departmentService := department.NewService(department.NewRepository(db))

	// --- create handler
	h := department.NewHandler(departmentService)

	// --- set up chi router
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)    // logs start/end of each request
	r.Use(middleware.Recoverer) // recovers from panics
	r.Use(middleware.Logger)

	// --- department routes
	r.Route("/departments", func(r chi.Router) {
		r.Get("/", h.GetDepartmentTree) // GET    /departments
		r.Post("/", h.Create)           // POST   /departments

		r.Route("/{id}", func(r chi.Router) {
			r.Put("/", h.Update)    // PUT    /departments/{id}
			r.Delete("/", h.Delete) // DELETE /departments/{id}
		})
	})

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Info().Msg("starting server on :8080")
	if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("server error")
	}

}

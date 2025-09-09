//go:build testutil

package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func init() {
	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}
}

const (
	defaultUser     = "postgres"
	defaultPassword = "postgres"
	defaultDB       = "postgres"
)

type TestDB struct {
	Container tc.Container
	Host      string
	Port      string
	User      string
	Password  string
}

func StartPostgres() (td *TestDB, cleanup func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: tc.ContainerRequest{
			Image:        "postgres:16-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     defaultUser,
				"POSTGRES_PASSWORD": defaultPassword,
				"POSTGRES_DB":       defaultDB,
			},
			WaitingFor: wait.ForSQL(nat.Port("5432/tcp"), "pgx",
				func(host string, port nat.Port) string {
					// DSN для ping-а контейнера
					return fmt.Sprintf(
						"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable timezone=UTC",
						host, port.Port(), defaultUser, defaultPassword, defaultDB,
					)
				},
			).WithStartupTimeout(60 * time.Second).
				WithPollInterval(200 * time.Millisecond).
				WithQuery("SELECT 1"),
		},
		Started: true,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to start postgres container: %v", err))
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(context.Background()) //nolint:errcheck
		panic(fmt.Sprintf("get container host: %v", err))
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		_ = container.Terminate(context.Background()) //nolint:errcheck
		panic(fmt.Sprintf("get mapped port: %v", err))
	}

	td = &TestDB{
		Container: container,
		Host:      host,
		Port:      port.Port(),
		User:      defaultUser,
		Password:  defaultPassword,
	}

	cleanup = func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = container.Terminate(ctx) //nolint:errcheck
	}

	return td, cleanup
}

func (td *TestDB) CreateIsolatedDB(t *testing.T) (*gorm.DB, *sql.DB, func()) {
	t.Helper()

	admin, err := sql.Open("pgx", td.adminDSN(defaultDB))
	if err != nil {
		t.Fatalf("sql open (admin): %v", err)
	}
	defer admin.Close()

	dbName := fmt.Sprintf("test_%s", uuid.New().String())

	if _, err = admin.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)); err != nil {
		t.Fatalf("fail to create database %s: %v", dbName, err)
	}

	sqlDSN := td.sqlDSN(dbName)
	sqlDB, err := sql.Open("pgx", sqlDSN)
	if err != nil {
		t.Fatalf("sql open (test db): %v", err)
	}

	runGooseUp(t, sqlDB)

	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxLifetime(2 * time.Minute)

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		t.Fatalf("gorm open: %v", err)
	}

	once := &sync.Once{}
	cleanup := func() {
		once.Do(func() {
			_, _ = admin.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, dbName)) //nolint:errcheck
			_ = sqlDB.Close()
		})
	}
	return gdb, sqlDB, cleanup
}

// --- DSN helpers ---

func (td *TestDB) adminDSN(dbname string) string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable timezone=UTC",
		td.Host, td.Port, td.User, td.Password, dbname)
}

func (td *TestDB) sqlDSN(dbname string) string {
	return td.adminDSN(dbname)
}

// --- goose ---

func runGooseUp(t *testing.T, sdb *sql.DB) {
	t.Helper()

	migrationsDir := findMigrationsDir()
	if _, err := os.Stat(migrationsDir); err != nil {
		t.Fatalf("migrations dir not found: %s (%v). Set MIGRATIONS_DIR env var.", migrationsDir, err)
	}

	if err := goose.Up(sdb, migrationsDir); err != nil {
		t.Fatalf("goose.Up: %v", err)
	}
}

func findMigrationsDir() string {
	if v := os.Getenv("MIGRATIONS_DIR"); v != "" {
		return v
	}
	d, err := os.Getwd()
	if err != nil {
		return "migrations" // fallback
	}
	for {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return filepath.Join(d, "migrations")
		}
		p := filepath.Dir(d)
		if p == d {
			return "migrations" // fallback
		}
		d = p
	}
}

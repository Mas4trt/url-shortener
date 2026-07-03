package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
	"url-shortener/internal/storage/postgres"

	"github.com/golang-migrate/migrate/v4"
	mpostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestPostgresRepository_Integration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	pgContainer, err := tcpostgres.RunContainer(
		ctx,
		testcontainers.WithImage("postgres:18.4-bookworm"),
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("user"),
		tcpostgres.WithPassword("pass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(10*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	t.Cleanup(func() {
		_ = pgContainer.Terminate(ctx)
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	driver, err := mpostgres.WithInstance(db, &mpostgres.Config{})
	if err != nil {
		t.Fatal(err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://../../../migrations", "postgres", driver)
	if err != nil {
		t.Fatal(err)
	}

	// 2. Накатываем миграции
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatal(err)
	}

	repo, err := postgres.New(ctx, connStr)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	t.Run("Save and Get URL", func(t *testing.T) {
		require.NoError(t, repo.Save(ctx, "https://google.com", "google"))

		res, err := repo.Get(ctx, "google")
		require.NoError(t, err)
		require.Equal(t, "https://google.com", res)

		require.NoError(t, repo.Delete(ctx, "google"))
	})
}

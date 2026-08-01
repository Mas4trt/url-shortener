package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"
	"url-shortener/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	insertURLQuery = `
        INSERT INTO urlshortener.url (url, alias)
        VALUES ($1, $2);
    `

	selectURLQuery = `
        SELECT url
        FROM urlshortener.url
        WHERE alias = $1;
    `

	deleteURLQuery = `
        DELETE FROM urlshortener.url
        WHERE alias = $1;
    `
)

const (
	pgUniqueViolation  = "23505"
	urlAliasConstraint = "url_alias_key"

	// defaultQueryTimeout bounds a single query regardless of what
	// deadline (if any) the caller's context already carries — nested
	// context.WithTimeout takes the earlier of the two deadlines, so this
	// only ever tightens the bound, never loosens it.
	defaultQueryTimeout = 5 * time.Second
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

// New is a lightweight constructor for tests and one-off tooling — it
// skips the pool tuning InitPostgres applies for production traffic.
// Prefer InitPostgres for anything long-running.
func New(ctx context.Context, connString string) (*PostgresRepo, error) {
	const op = "storage.postgres.New"

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	storage := &PostgresRepo{
		pool: pool,
	}

	return storage, nil
}

// Close закрывает соединение с БД
func (s *PostgresRepo) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *PostgresRepo) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Save сохраняет URL и его алиас в БД
func (s *PostgresRepo) Save(ctx context.Context, rawURL string, alias string) error {
	const op = "storage.postgres.Save"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	tag, err := s.pool.Exec(ctx, insertURLQuery, rawURL, alias)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgUniqueViolation &&
				pgErr.ConstraintName == urlAliasConstraint {
				return fmt.Errorf("%s: %w", op, domain.ErrURLExist)
			}
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%s: unexpected rows affected: %d", op, tag.RowsAffected())
	}

	return nil
}

func (s *PostgresRepo) Get(ctx context.Context, alias string) (string, error) {
	const op = "storage.postgres.Get"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	var urlToGet string

	err := s.pool.QueryRow(ctx, selectURLQuery, alias).Scan(&urlToGet)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("%s: %w", op, domain.ErrURLNotFound)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return urlToGet, nil
}

func (s *PostgresRepo) Delete(ctx context.Context, alias string) error {
	const op = "storage.postgres.Delete"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	tag, err := s.pool.Exec(ctx, deleteURLQuery, alias)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%s: %w", op, domain.ErrURLNotFound)
	}

	return nil
}

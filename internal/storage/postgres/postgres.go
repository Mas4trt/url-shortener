package postgres

import (
	"context"
	"errors"
	"fmt"
	"url-shortener/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	pool *pgxpool.Pool
}

// New инициализирует базу данных и создает необходимые таблицы
func New(ctx context.Context, connString string) (*Storage, error) {
	const op = "storage.postgres.New"

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("%s: ping failed: %w", op, err)
	}

	return &Storage{pool: pool}, nil
}

// Close закрывает соединение с БД
func (s *Storage) Close() {
	s.pool.Close()
}

// SaveURL сохраняет URL и его алиас в БД
func (s *Storage) SaveURL(ctx context.Context, urlToSave string, alias string) (int64, error) {
	const op = "storage.postgres.SaveURL"

	var id int64
	const insertURLQuery = `
		INSERT INTO urlshortener.url (url, alias) 
		VALUES ($1, $2) 
		RETURNING id
	`

	err := s.pool.QueryRow(ctx, insertURLQuery, urlToSave, alias).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch {
			case pgErr.Code == "23505" && pgErr.ConstraintName == "url_alias_key":
				return 0, fmt.Errorf("%s: %w", op, domain.ErrURLExist)
			}
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetURL(ctx context.Context, alias string) (string, error) {
	const op = "storage.postgres.GetURL"

	var urlToGet string
	const selectURLQuery = `
		SELECT url FROM urlshortener.url 
		WHERE alias = $1
	`

	err := s.pool.QueryRow(ctx, selectURLQuery, alias).Scan(&urlToGet)
	if err != nil {
		// Проверяем, вернулась ли ошибка из-за того, что запись не найдена
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("%s: %w", op, domain.ErrURLNotFound)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return urlToGet, nil
}

func (s *Storage) DeleteURL(ctx context.Context, alias string) error {
	const op = "storage.postgres.DeleteURL"

	const deleteURLQuery = `
		DELETE FROM url 
		WHERE alias = $1
	`

	tag, err := s.pool.Exec(ctx, deleteURLQuery, alias)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// Если ни одна строка не затронута, значит алиас не найден
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%s: %w", op, domain.ErrURLNotFound)
	}

	return nil
}

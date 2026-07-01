package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"url-shortener/internal/storage"

	"modernc.org/sqlite"
)

const (
	// errConstraintUnique — код ошибки уникальности SQLite
	errConstraintUnique = 2067
)

type Storage struct {
	db *sql.DB
}

// New инициализирует базу данных и создает необходимые таблицы
func New(ctx context.Context, storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s : %w", op, err)
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("%s :ping failed: %w", op, err)
	}

	_, err = db.ExecContext(ctx, `
	CREATE TABLE IF NOT EXISTS url(
		id INTEGER PRIMARY KEY,
		alias TEXT NOT NULL UNIQUE,
		url TEXT NOT NULL);
	CREATE INDEX IF NOT EXISTS idx_alias ON url(alias);
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to create tables %w", op, err)
	}

	return &Storage{db: db}, nil
}

// Close закрывает соединение с БД
func (s *Storage) Close() error {
	return s.db.Close()
}

// SaveURL сохраняет URL и его алиас в БД
func (s *Storage) SaveURL(ctx context.Context, urlToSave string, alias string) (int64, error) {
	const op = "storage.sqlite.SaveURL"

	res, err := s.db.ExecContext(ctx, "INSERT INTO url(url, alias) VALUES(?, ?)", urlToSave, alias)
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code() == errConstraintUnique {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrURLExist)
		}
		return 0, fmt.Errorf("%s : %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get last insert id: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetURL(ctx context.Context, alias string) (string, error) {
	const op = "storage.sqlite.GetURL"

	var urlToGet string

	// QueryRowContext выполняет запрос, ожидая ровно одну строку
	// Сразу же вызываем Scan, чтобы записать результат в urlToGet
	err := s.db.QueryRowContext(ctx, `SELECT url FROM url WHERE alias = ?`, alias).Scan(&urlToGet)
	if err != nil {
		// Проверяем, вернулась ли ошибка из-за того, что запись не найдена
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("%s: %w", op, storage.ErrURLNotFound)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return urlToGet, nil
}

func (s *Storage) DeleteURL(ctx context.Context, alias string) error {
	const op = "storage.sqlite.DeleteURL"

	res, err := s.db.ExecContext(ctx, `DELETE FROM url WHERE alias = ?`, alias)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// Получаем количество удаленных строк
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: failed to get rows affected: %w", op, err)
	}

	// Если ни одна строка не затронута, значит алиас не найден
	if rowsAffected == 0 {
		return fmt.Errorf("%s : %w", op, storage.ErrURLNotFound)
	}

	return nil
}

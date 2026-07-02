package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"url-shortener/internal/domian"
	"url-shortener/internal/lib/random"
)

// TODO: вынести в конфиг
const (
	aliasLength = 6
	maxRetries  = 5
)

// URLSaver определяет интерфейс для сохранения URL
type URLSaver interface {
	SaveURL(ctx context.Context, urlToSave string, alias string) (int64, error)
}

type URLService struct {
	log      *slog.Logger
	urlSaver URLSaver
}

func New(log *slog.Logger, urlSaver URLSaver) *URLService {
	return &URLService{
		log:      log,
		urlSaver: urlSaver,
	}
}

func (s *URLService) Save(ctx context.Context, rawURL string, customAlias string) (string, error) {
	const op = "service.url.Save"

	// Если пользователь передал свой алиас — пробуем сохранить его
	if customAlias != "" {
		_, err := s.urlSaver.SaveURL(ctx, rawURL, customAlias)
		if err != nil {
			if errors.Is(err, domian.ErrURLExist) {
				return "", domian.ErrURLExist
			}
			return "", fmt.Errorf("%s : %w", op, err)
		}
		return customAlias, nil
	}

	// Если алиас пустой — генерируем случайный с повторными попытками
	for i := 0; i < maxRetries; i++ {
		alias, err := random.NewRandomString(aliasLength)
		if err != nil {
			return "", fmt.Errorf("%s: failed to generate alias: %w", op, err)
		}

		_, err = s.urlSaver.SaveURL(ctx, rawURL, alias)
		if err == nil {
			return alias, nil
		}

		if errors.Is(err, domian.ErrURLExist) {
			s.log.Warn("alias collision occurred, retrying", slog.String("alias", alias))
			continue // Пробуем сгенерировать снова
		}

		// Какая-то другая ошибка БД
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return "", fmt.Errorf("%s: failed to generate unique alias after %d retries", op, maxRetries)
}

package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"url-shortener/internal/domain"
	"url-shortener/pkg/logger/sl"
)

// URLRepository определяет интерфейс для сохранения URL
type URLRepository interface {
	Save(ctx context.Context, urlToSave string, alias string) error
	Get(ctx context.Context, alias string) (string, error)
	Delete(ctx context.Context, alias string) error
}

type AliasGenerator interface {
	Generate() (string, error)
}

type Config struct {
	MaxRetries int
}

type Service struct {
	log        *slog.Logger
	repository URLRepository
	generator  AliasGenerator
	cfg        Config
}

func New(log *slog.Logger, Repository URLRepository, generator AliasGenerator, cfg Config) (*Service, error) {
	if cfg.MaxRetries <= 0 {
		return nil, fmt.Errorf("invalid config: max retries must be positive")
	}
	return &Service{
		log:        log,
		repository: Repository,
		generator:  generator,
		cfg:        cfg,
	}, nil
}

func (s *Service) Save(ctx context.Context, url string, alias string) (string, error) {
	const op = "service.URLService.Save"

	if url == "" {
		return "", fmt.Errorf("%s: %w", op, domain.ErrInvalidURL)
	}

	if alias != "" {
		return s.saveAlias(ctx, url, alias)
	}

	return s.saveGeneratedAlias(ctx, url)
}

func (s *Service) Get(ctx context.Context, alias string) (string, error) {
	const op = "service.URLService.Get"

	if alias == "" {
		return "", fmt.Errorf("%s: %w", op, domain.ErrEmptyAlias)
	}

	url, err := s.repository.Get(ctx, alias)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return url, nil
}

func (s *Service) Delete(ctx context.Context, alias string) error {
	const op = "service.URLService.Delete"

	if alias == "" {
		return fmt.Errorf("%s: %w", op, domain.ErrEmptyAlias)
	}

	if err := s.repository.Delete(ctx, alias); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Service) saveAlias(ctx context.Context, url string, alias string) (string, error) {
	const op = "service.URLService.saveAlias"

	if err := s.repository.Save(ctx, url, alias); err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return alias, nil
}

func (s *Service) saveGeneratedAlias(ctx context.Context, url string) (string, error) {
	const op = "service.URLService.saveGeneratedAlias"

	log := sl.LoggerWithCtx(ctx, s.log)

	for attempt := 1; attempt <= s.cfg.MaxRetries; attempt++ {

		alias, err := s.generator.Generate()
		if err != nil {
			return "", fmt.Errorf("%s: failed to generate alias: %w", op, err)
		}

		err = s.repository.Save(ctx, url, alias)
		if err == nil {
			return alias, nil
		}

		if errors.Is(err, domain.ErrURLExist) {
			log.Warn(
				"alias collision",
				slog.Int("max_attempts", s.cfg.MaxRetries),
				slog.Int("attempt", attempt),
				slog.String("alias", alias),
			)
			continue
		}

		return "", fmt.Errorf("%s: %w", op, err)
	}

	return "", fmt.Errorf("%s: failed to generate unique alias after %d retries", op, s.cfg.MaxRetries)
}

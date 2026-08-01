package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"url-shortener/internal/domain"
	sl "url-shortener/pkg/logger/sl"
)

// URLRepository is everything the service needs from persistence. The
// concrete implementation (e.g. the Redis-fronted Postgres cache) lives
// in internal/storage and is wired in at startup — the service only ever
// sees this interface, so it can be tested with a mock and swapped
// without touching business logic.
type URLRepository interface {
	Save(ctx context.Context, urlToSave string, alias string) error
	Get(ctx context.Context, alias string) (string, error)
	Delete(ctx context.Context, alias string) error
}

// AliasGenerator produces a candidate alias. Implementations are expected
// to be safe for concurrent use.
type AliasGenerator interface {
	Generate() (string, error)
}

// Config holds tunables for the service. Validated once in New so every
// other method can assume it's sane.
type Config struct {
	// MaxRetries bounds how many times Save will re-roll a generated
	// alias after a collision before giving up. Must be positive.
	MaxRetries int
}

func (c Config) validate() error {
	if c.MaxRetries <= 0 {
		return fmt.Errorf("%w: max_retries must be positive, got %d", ErrInvalidConfig, c.MaxRetries)
	}
	return nil
}

type Service struct {
	log        *slog.Logger
	repository URLRepository
	generator  AliasGenerator
	cfg        Config
}

// New constructs a Service. It returns ErrInvalidConfig (via errors.Is)
// rather than panicking, since config here can come from environment
// variables at process startup and a hard crash with a clear error beats
// an implicit nil-pointer panic three calls deep.
func New(log *slog.Logger, repository URLRepository, generator AliasGenerator, cfg Config) (*Service, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &Service{
		log:        log.With(slog.String("layer", "service")),
		repository: repository,
		generator:  generator,
		cfg:        cfg,
	}, nil
}

// Save persists url under alias if alias is non-empty, otherwise it
// generates one, retrying on collision up to cfg.MaxRetries times. It
// returns the alias that was actually stored.
func (s *Service) Save(ctx context.Context, url string, alias string) (string, error) {
	const op = "service.URLService.Save"

	url = strings.TrimSpace(url)
	alias = strings.TrimSpace(alias)

	if url == "" {
		return "", fmt.Errorf("%s: %w", op, domain.ErrInvalidURL)
	}

	if alias != "" {
		return s.saveAlias(ctx, url, alias)
	}

	return s.saveGeneratedAlias(ctx, url)
}

// Get resolves alias to its target URL.
func (s *Service) Get(ctx context.Context, alias string) (string, error) {
	const op = "service.URLService.Get"

	alias = strings.TrimSpace(alias)
	if alias == "" {
		return "", fmt.Errorf("%s: %w", op, domain.ErrEmptyAlias)
	}

	url, err := s.repository.Get(ctx, alias)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return url, nil
}

// Delete removes alias and its target URL.
func (s *Service) Delete(ctx context.Context, alias string) error {
	const op = "service.URLService.Delete"

	alias = strings.TrimSpace(alias)
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

// saveGeneratedAlias rolls a new alias and attempts to save it, retrying
// on domain.ErrURLExist (an alias collision — expected and harmless at
// low probability) up to cfg.MaxRetries times. It bails out early if ctx
// is canceled so a client disconnect doesn't burn retries against a slow
// or degraded repository.
func (s *Service) saveGeneratedAlias(ctx context.Context, url string) (string, error) {
	const op = "service.URLService.saveGeneratedAlias"

	log := sl.LoggerWithCtx(ctx, s.log).With(slog.String("op", op))

	for attempt := 1; attempt <= s.cfg.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("%s: %w", op, err)
		}

		alias, err := s.generator.Generate()
		if err != nil {
			return "", fmt.Errorf("%s: failed to generate alias: %w", op, err)
		}

		err = s.repository.Save(ctx, url, alias)
		if err == nil {
			return alias, nil
		}

		if errors.Is(err, domain.ErrURLExist) {
			log.WarnContext(ctx, "alias collision",
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

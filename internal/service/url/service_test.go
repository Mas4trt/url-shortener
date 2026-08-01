package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/domain"
	amocks "url-shortener/internal/service/mocks"
	service "url-shortener/internal/service/url"
	"url-shortener/internal/storage/mocks"
)

func defaultTestConfig() service.Config {
	return service.Config{
		MaxRetries: 2,
	}
}

func setupTest(t *testing.T, cfg service.Config) (*mocks.URLRepository, *amocks.AliasGenerator, *service.Service) {
	t.Helper()
	repo := mocks.NewURLRepository(t)
	gen := amocks.NewAliasGenerator(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	srv, err := service.New(logger, repo, gen, cfg)
	require.NoError(t, err)

	return repo, gen, srv
}

func TestNewService(t *testing.T) {
	repo := mocks.NewURLRepository(t)
	gen := amocks.NewAliasGenerator(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name        string
		cfg         service.Config
		expectError bool
	}{
		{
			name:        "Valid config",
			cfg:         service.Config{MaxRetries: 3},
			expectError: false,
		},
		{
			name:        "Invalid max retries",
			cfg:         service.Config{MaxRetries: -1},
			expectError: true,
		},
		{
			name:        "Zero max retries",
			cfg:         service.Config{MaxRetries: 0},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, err := service.New(logger, repo, gen, tt.cfg)
			if tt.expectError {
				require.Error(t, err)
				require.ErrorIs(t, err, service.ErrInvalidConfig, "config errors should be identifiable via errors.Is")
				require.Nil(t, srv)
			} else {
				require.NoError(t, err)
				require.NotNil(t, srv)
			}
		})
	}
}

func TestService_Save(t *testing.T) {
	ctx := context.Background()
	testURL := "https://google.com"

	tests := []struct {
		name          string
		url           string
		alias         string
		mockBehavior  func(repo *mocks.URLRepository, gen *amocks.AliasGenerator, cfg service.Config)
		expectedAlias string
		expectedErr   error
	}{
		{
			name:  "Empty URL",
			url:   "",
			alias: "",
			mockBehavior: func(repo *mocks.URLRepository, gen *amocks.AliasGenerator, cfg service.Config) {
				// Mocks must not be called.
			},
			expectedErr: domain.ErrInvalidURL,
		},
		{
			name:  "Whitespace-only URL is treated as empty",
			url:   "   ",
			alias: "",
			mockBehavior: func(repo *mocks.URLRepository, gen *amocks.AliasGenerator, cfg service.Config) {
			},
			expectedErr: domain.ErrInvalidURL,
		},
		{
			name:  "Success with custom alias",
			url:   testURL,
			alias: "goog",
			mockBehavior: func(repo *mocks.URLRepository, gen *amocks.AliasGenerator, cfg service.Config) {
				repo.On("Save", mock.Anything, testURL, "goog").Return(nil).Once()
			},
			expectedAlias: "goog",
			expectedErr:   nil,
		},
		{
			name:  "Error on save custom alias",
			url:   testURL,
			alias: "goog",
			mockBehavior: func(repo *mocks.URLRepository, gen *amocks.AliasGenerator, cfg service.Config) {
				repo.On("Save", mock.Anything, testURL, "goog").Return(errors.New("db error")).Once()
			},
			expectedErr: errors.New("db error"),
		},
		{
			name:  "Success with generated alias (first attempt)",
			url:   testURL,
			alias: "",
			mockBehavior: func(repo *mocks.URLRepository, gen *amocks.AliasGenerator, cfg service.Config) {
				gen.On("Generate").Return("rnd01", nil).Once()
				repo.On("Save", mock.Anything, testURL, "rnd01").Return(nil).Once()
			},
			expectedAlias: "rnd01",
			expectedErr:   nil,
		},
		{
			name:  "Generator failure",
			url:   testURL,
			alias: "",
			mockBehavior: func(repo *mocks.URLRepository, gen *amocks.AliasGenerator, cfg service.Config) {
				gen.On("Generate").Return("", errors.New("gen error")).Once()
			},
			expectedErr: errors.New("failed to generate alias"),
		},
		{
			name:  "Success with generated alias after 1 collision (retry)",
			url:   testURL,
			alias: "",
			mockBehavior: func(repo *mocks.URLRepository, gen *amocks.AliasGenerator, cfg service.Config) {
				gen.On("Generate").Return("colli", nil).Once()
				repo.On("Save", mock.Anything, testURL, "colli").Return(domain.ErrURLExist).Once()
				gen.On("Generate").Return("rnd02", nil).Once()
				repo.On("Save", mock.Anything, testURL, "rnd02").Return(nil).Once()
			},
			expectedAlias: "rnd02",
			expectedErr:   nil,
		},
		{
			name:  "Fail due to max retries exceeded",
			url:   testURL,
			alias: "",
			mockBehavior: func(repo *mocks.URLRepository, gen *amocks.AliasGenerator, cfg service.Config) {
				gen.On("Generate").Return("colli", nil).Times(cfg.MaxRetries)
				repo.On("Save", mock.Anything, testURL, mock.Anything).Return(domain.ErrURLExist).Times(cfg.MaxRetries)
			},
			expectedErr: errors.New("failed to generate unique alias"),
		},
		{
			name:  "Fail due to unknown DB error on generate",
			url:   testURL,
			alias: "",
			mockBehavior: func(repo *mocks.URLRepository, gen *amocks.AliasGenerator, cfg service.Config) {
				gen.On("Generate").Return("rnd03", nil).Once()
				repo.On("Save", mock.Anything, testURL, "rnd03").Return(errors.New("db connection lost")).Once()
			},
			expectedErr: errors.New("db connection lost"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultTestConfig()
			repo, gen, srv := setupTest(t, cfg)
			tt.mockBehavior(repo, gen, cfg)

			alias, err := srv.Save(ctx, tt.url, tt.alias)

			if tt.expectedErr != nil {
				require.Error(t, err)
				if errors.Is(tt.expectedErr, domain.ErrInvalidURL) {
					require.ErrorIs(t, err, domain.ErrInvalidURL)
				} else {
					require.ErrorContains(t, err, tt.expectedErr.Error())
				}
				require.Empty(t, alias)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedAlias, alias)
			}
		})
	}
}

// TestService_Save_ContextCanceled ensures the retry loop bails out
// immediately once the caller's context is canceled, instead of burning
// through every remaining retry attempt against a repository that may
// itself be slow to fail.
func TestService_Save_ContextCanceled(t *testing.T) {
	repo, gen, srv := setupTest(t, service.Config{MaxRetries: 5})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	alias, err := srv.Save(ctx, "https://example.com", "")

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, alias)

	gen.AssertNotCalled(t, "Generate")
	repo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything, mock.Anything)
}

func TestService_Get(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		alias        string
		mockBehavior func(repo *mocks.URLRepository)
		expectedURL  string
		expectedErr  error
	}{
		{
			name:  "Empty alias",
			alias: "",
			mockBehavior: func(repo *mocks.URLRepository) {
			},
			expectedErr: domain.ErrEmptyAlias,
		},
		{
			name:  "Success",
			alias: "goog",
			mockBehavior: func(repo *mocks.URLRepository) {
				repo.On("Get", mock.Anything, "goog").Return("https://google.com", nil).Once()
			},
			expectedURL: "https://google.com",
			expectedErr: nil,
		},
		{
			name:  "Not found",
			alias: "unknown",
			mockBehavior: func(repo *mocks.URLRepository) {
				repo.On("Get", mock.Anything, "unknown").Return("", domain.ErrURLNotFound).Once()
			},
			expectedErr: domain.ErrURLNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, srv := setupTest(t, defaultTestConfig())
			tt.mockBehavior(repo)

			url, err := srv.Get(ctx, tt.alias)

			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				require.Empty(t, url)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedURL, url)
			}
		})
	}
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		alias        string
		mockBehavior func(repo *mocks.URLRepository)
		expectedErr  error
	}{
		{
			name:  "Empty alias",
			alias: "",
			mockBehavior: func(repo *mocks.URLRepository) {
			},
			expectedErr: domain.ErrEmptyAlias,
		},
		{
			name:  "Success",
			alias: "goog",
			mockBehavior: func(repo *mocks.URLRepository) {
				repo.On("Delete", mock.Anything, "goog").Return(nil).Once()
			},
			expectedErr: nil,
		},
		{
			name:  "Repository error",
			alias: "error",
			mockBehavior: func(repo *mocks.URLRepository) {
				repo.On("Delete", mock.Anything, "error").Return(errors.New("db error")).Once()
			},
			expectedErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, srv := setupTest(t, defaultTestConfig())
			tt.mockBehavior(repo)

			err := srv.Delete(ctx, tt.alias)

			if tt.expectedErr != nil {
				require.Error(t, err)
				if errors.Is(tt.expectedErr, domain.ErrEmptyAlias) {
					require.ErrorIs(t, err, domain.ErrEmptyAlias)
				} else {
					require.ErrorContains(t, err, tt.expectedErr.Error())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"url-shortener/pkg/logger/sl"
)

const shutdownTimeout = 10 * time.Second

type App struct {
	log    *slog.Logger
	server *http.Server
}

func New(
	log *slog.Logger,
	server *http.Server,
) *App {
	return &App{
		log:    log,
		server: server,
	}
}

func (a *App) Run(cleanup func()) error {
	if cleanup != nil {
		defer cleanup()
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	serverErr := make(chan error, 1)

	go func() {
		a.log.Info(
			"starting HTTP server",
			slog.String("address", a.server.Addr),
		)

		if err := a.server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case sig := <-signals:
		a.log.Info(
			"shutdown signal received",
			slog.String("signal", sig.String()),
		)

	case err := <-serverErr:
		return fmt.Errorf("http server: %w", err)
	}

	a.log.Info("shutting down HTTP server")

	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		shutdownTimeout,
	)
	defer shutdownCancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		a.log.Error(
			"failed to shutdown server",
			sl.Err(err),
		)
	}

	a.log.Info("server stopped")
	return nil
}

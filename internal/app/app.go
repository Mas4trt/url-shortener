package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"url-shortener/pkg/logger/sl"
)

type App struct {
	log    *slog.Logger
	server *http.Server
}

func New(log *slog.Logger, server *http.Server) *App {
	return &App{
		log:    log,
		server: server,
	}
}

func (a *App) Run() error {
	const op = "app.Run"

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		a.log.Info("starting HTTP server", slog.String("address", a.server.Addr))
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.log.Error("failed to start server", sl.Err(err))
		}
	}()

	<-done
	a.log.Info("stopping server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		a.log.Error("failed to shutdown server gracefully", sl.Err(err))
	}

	a.log.Info("server stopped completely")
	return nil
}

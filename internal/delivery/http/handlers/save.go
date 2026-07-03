package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"url-shortener/internal/domain"
	sl "url-shortener/internal/lib/logger/sl"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type SaveRequest struct {
	URL   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty"`
}

type SaveResponse struct {
	Response
	Alias string `json:"alias,omitempty"`
}

// Service — интерфейс, который хэндлер ожидает от бизнес-логики
type Service interface {
	Save(ctx context.Context, url string, alias string) (string, error)
	Get(ctx context.Context, alias string) (string, error)
	Delete(ctx context.Context, alias string) error
}

type Handler struct {
	log       *slog.Logger
	service   Service
	validator *validator.Validate
}

func New(log *slog.Logger, service Service, validator *validator.Validate) *Handler {
	return &Handler{
		log: log.With(
			slog.String("layer", "handler"),
		),
		service:   service,
		validator: validator,
	}
}

// Save выполняет обработку POST /url
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.Handler.Save"

	log := h.logger(r, op)

	var req SaveRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		log.Error("failed to decode request body", sl.Err(err))
		respond(w, r, http.StatusBadRequest, Error("failed to decode request"))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		var validationErr validator.ValidationErrors
		if errors.As(err, &validationErr) {
			log.Error("invalid request", sl.Err(err))
			respond(w, r, http.StatusBadRequest, ValidationError(validationErr))
			return
		}
		log.Error("validator failure", sl.Err(err))
		respond(w, r, http.StatusInternalServerError, Error("validation error"))
		return
	}

	alias, err := h.service.Save(r.Context(), req.URL, req.Alias)
	switch {
	case errors.Is(err, domain.ErrURLExist):
		log.Debug("url already exists", slog.String("alias", req.Alias))
		respond(w, r, http.StatusConflict, Error("url already exists"))
		return
	case err != nil:
		log.Error("failed add url", sl.Err(err))
		respond(w, r, http.StatusInternalServerError, Error("failed add url"))
		return
	}

	log.Info("url added", slog.String("alias", alias))

	respond(w, r, http.StatusOK, SaveResponse{
		Response: Ok(),
		Alias:    alias,
	})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.Handler.Get"

	log := h.logger(r, op)

	alias := chi.URLParam(r, "alias")
	if alias == "" {
		log.Debug("alias is empty")
		respond(w, r, http.StatusBadRequest, Error("invalid request"))
		return
	}

	resURL, err := h.service.Get(r.Context(), alias)
	switch {
	case errors.Is(err, domain.ErrURLNotFound):
		log.Debug("url not found", slog.String("alias", alias))
		respond(w, r, http.StatusNotFound, Error("not found"))
		return
	case err != nil:
		log.Error("failed to get url", sl.Err(err))
		respond(w, r, http.StatusInternalServerError, Error("internal error"))
		return
	}

	log.Info("got alias", slog.String("alias", alias))

	http.Redirect(w, r, resURL, http.StatusFound)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.Handler.Delete"

	log := h.logger(r, op)

	alias := chi.URLParam(r, "alias")
	if alias == "" {
		log.Debug("alias is empty")
		respond(w, r, http.StatusBadRequest, Error("invalid request"))
		return
	}

	err := h.service.Delete(r.Context(), alias)
	switch {
	case errors.Is(err, domain.ErrURLNotFound):
		respond(w, r, http.StatusNotFound, Error("not found"))
		return
	case err != nil:
		log.Error("failed to delete url", sl.Err(err))
		respond(w, r, http.StatusInternalServerError, Error("internal error"))
		return
	}

	log.Info("url successfully deleted", slog.String("alias", alias))

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) logger(r *http.Request, op string) *slog.Logger {
	return h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)
}

func respond(w http.ResponseWriter, r *http.Request, code int, v any) {
	render.Status(r, code)
	render.JSON(w, r, v)
}

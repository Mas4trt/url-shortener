package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"url-shortener/internal/domain"
	"url-shortener/internal/transport/http/response"
	sl "url-shortener/pkg/logger/sl"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

// maxSaveRequestBytes caps the JSON body accepted by POST /url. Without a
// cap, render.DecodeJSON will happily read an unbounded body into memory
// off an unauthenticated-until-parsed request — an easy memory-exhaustion
// vector. 1 MiB is generous for a {url, alias} payload.
const maxSaveRequestBytes = 1 << 20 // 1 MiB

type SaveRequest struct {
	URL   string `json:"url" validate:"required,url,url_scheme"`
	Alias string `json:"alias" validate:"omitempty,min=4,max=20,alias"`
}

type SaveResponse struct {
	response.Response
	Alias string `json:"alias,omitempty"`
}

// URLService is what the handler expects from the business-logic layer —
// satisfied by *service.Service. Kept here (not in the service package)
// so the service package doesn't need to know it's used over HTTP.
type URLService interface {
	Save(ctx context.Context, url string, alias string) (string, error)
	Get(ctx context.Context, alias string) (string, error)
	Delete(ctx context.Context, alias string) error
}

type Handler struct {
	log       *slog.Logger
	service   URLService
	validator *validator.Validate
}

func New(log *slog.Logger, service URLService, validator *validator.Validate) *Handler {
	return &Handler{
		log: log.With(
			slog.String("layer", "handler"),
		),
		service:   service,
		validator: validator,
	}
}

// Save handles POST /url.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.Handler.Save"

	log := h.logger(r, op)

	r.Body = http.MaxBytesReader(w, r.Body, maxSaveRequestBytes)

	var req SaveRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		log.Debug("failed to decode request body", sl.Err(err))
		response.Respond(w, r, http.StatusBadRequest, response.Error("failed to decode request"))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		var validationErr validator.ValidationErrors
		if errors.As(err, &validationErr) {
			log.Debug("invalid request", sl.Err(err))
			response.Respond(w, r, http.StatusBadRequest, response.ValidationError(validationErr))
			return
		}
		log.Error("validator failure", sl.Err(err))
		response.Respond(w, r, http.StatusInternalServerError, response.Error("validation error"))
		return
	}

	alias, err := h.service.Save(r.Context(), req.URL, req.Alias)
	if err != nil {
		response.RespondError(w, r, log, err,
			http.StatusInternalServerError, "failed add url",
			response.Case(domain.ErrURLExist, http.StatusConflict, "url already exists"),
			response.Case(domain.ErrInvalidURL, http.StatusBadRequest, "url is required"),
		)
		return
	}

	log.Info("url added", slog.String("alias", alias))

	response.Respond(w, r, http.StatusOK, SaveResponse{
		Response: response.Ok(),
		Alias:    alias,
	})
}

// Get handles GET /{alias} — resolves and 302-redirects to the target URL.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.Handler.Get"

	log := h.logger(r, op)

	alias := chi.URLParam(r, "alias")
	if alias == "" {
		log.Debug("alias is empty")
		response.Respond(w, r, http.StatusBadRequest, response.Error("invalid request"))
		return
	}

	resURL, err := h.service.Get(r.Context(), alias)
	if err != nil {
		response.RespondError(w, r, log, err,
			http.StatusInternalServerError, "internal error",
			response.Case(domain.ErrURLNotFound, http.StatusNotFound, "not found"),
		)
		return
	}

	log.Info("got alias", slog.String("alias", alias))

	http.Redirect(w, r, resURL, http.StatusFound)
}

// Delete handles DELETE /{alias}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.Handler.Delete"

	log := h.logger(r, op)

	alias := chi.URLParam(r, "alias")
	if alias == "" {
		log.Debug("alias is empty")
		response.Respond(w, r, http.StatusBadRequest, response.Error("invalid request"))
		return
	}

	if err := h.service.Delete(r.Context(), alias); err != nil {
		response.RespondError(w, r, log, err,
			http.StatusInternalServerError, "internal error",
			response.Case(domain.ErrURLNotFound, http.StatusNotFound, "not found"),
		)
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

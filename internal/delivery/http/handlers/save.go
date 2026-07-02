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

type Request struct {
	URL   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty"`
}

type SaveResponse struct {
	Response
	URL   string `json:"url,omitempty"`
	Alias string `json:"alias,omitempty"`
}

// URLService — интерфейс, который хэндлер ожидает от бизнес-логики
type URLService interface {
	Save(ctx context.Context, rawURL string, customAlias string) (string, error)
	Get(ctx context.Context, customAlias string) (string, error)
}

type URLHandler struct {
	log       *slog.Logger
	service   URLService
	validator *validator.Validate
}

func NewURLHandler(log *slog.Logger, service URLService, validator *validator.Validate) *URLHandler {
	return &URLHandler{
		log:       log,
		service:   service,
		validator: validator,
	}
}

// Save выполняет обработку POST /url
func (h *URLHandler) Save(w http.ResponseWriter, r *http.Request) {
	const op = "Handlers.URLHandler.Save"

	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	var req Request
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		log.Error("failed to decode request body", sl.Err(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, Error("failed to decode request"))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		validatorErr := err.(validator.ValidationErrors)
		log.Error("invalid request", sl.Err(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ValidationError(validatorErr))
		return
	}

	alias, err := h.service.Save(r.Context(), req.URL, req.Alias)
	if errors.Is(err, domain.ErrURLExist) {
		log.Info("url already exists", slog.String("url", req.URL))
		render.Status(r, http.StatusConflict)
		render.JSON(w, r, Error("url already exists"))
		return
	}
	if err != nil {
		log.Error("failed add url", sl.Err(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, Error("failed add url"))
		return
	}

	log.Info("url added", slog.String("alias", alias))

	// Если все удачно возвращаем Status:200 и Response
	render.Status(r, http.StatusOK)
	render.JSON(w, r, SaveResponse{
		Response: Ok(),
		Alias:    alias,
	})
}

func (h *URLHandler) Get(w http.ResponseWriter, r *http.Request) {
	const op = "Handlers.URLHandler.Get"

	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	alias := chi.URLParam(r, "alias")
	if alias == "" {
		log.Info("alias is empty")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, Error("invalid request"))
		return
	}

	resURL, err := h.service.Get(r.Context(), alias)
	if errors.Is(err, domain.ErrURLNotFound) {
		log.Info("url not found", slog.String("alias", alias))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, Error("not found"))
		return
	}
	if err != nil {
		log.Error("failed to get url", sl.Err(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, Error("internal error"))
		return
	}

	log.Info("got url", slog.String("url", resURL))

	http.Redirect(w, r, resURL, http.StatusFound)
}

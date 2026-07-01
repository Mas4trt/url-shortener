package save

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "url-shortener/internal/lib/api/response"
	sl "url-shortener/internal/lib/logger/slog"
	"url-shortener/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Request struct {
	URL   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty"`
}

type Response struct {
	resp.Response
	Alias string `json:"alias,omitempty"`
}

// URLService — интерфейс, который хэндлер ожидает от бизнес-логики
type URLService interface {
	Save(ctx context.Context, rawURL string, customAlias string) (string, error)
}

func New(log *slog.Logger, urlService URLService, v *validator.Validate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.save.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("failed to decode request"))
			return
		}

		// log.Info("request body decoded", slog.Any("request", req))

		if err := v.Struct(req); err != nil {
			validatorErr := err.(validator.ValidationErrors)
			log.Error("invalid request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationError(validatorErr))
			return
		}

		alias, err := urlService.Save(r.Context(), req.URL, req.Alias)
		if errors.Is(err, storage.ErrURLExist) {
			log.Info("url already exists", slog.String("url", req.URL))
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, resp.Error("url already exists"))
			return
		}
		if err != nil {
			log.Error("failed add url", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("failed add url"))
			return
		}

		log.Info("url added", slog.String("alias", alias))

		// Если все удачно возвращаем Status:200 и Response
		render.Status(r, http.StatusOK)
		render.JSON(w, r, Response{
			Response: resp.Ok(),
			Alias:    alias,
		})
	}
}

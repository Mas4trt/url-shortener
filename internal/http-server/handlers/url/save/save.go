package save

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "url-shortener/internal/lib/api/response"
	sl "url-shortener/internal/lib/logger/slog"
	"url-shortener/internal/lib/random"
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

const aliasLength = 6

type URLSaver interface {
	SaveURL(ctx context.Context, urlToSave string, alias string) (int64, error)
}

func New(log *slog.Logger, urlSaver URLSaver, v *validator.Validate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.save.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("failed to decode request"))

			return
		}

		log.Info("request body decoded", slog.Any("request", req))

		if err := v.Struct(req); err != nil {
			validatorErr := err.(validator.ValidationErrors)

			log.Error("invalid request", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationError(validatorErr))

			return
		}

		alias := req.Alias
		if alias != "" {
			id, err := urlSaver.SaveURL(r.Context(), req.URL, alias)
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

			log.Info("url added", slog.Int64("id", id))

		} else {
			const maxRetries = 5
			var id int64
			var err error
			for i := 0; i < maxRetries; i++ {
				alias, err = random.NewRandomString(aliasLength)
				if err != nil {
					log.Error("failed to generate random alias", sl.Err(err))
					render.Status(r, http.StatusInternalServerError)
					render.JSON(w, r, resp.Error("internal error"))
					return
				}

				id, err = urlSaver.SaveURL(r.Context(), req.URL, alias)
				if err == nil {
					log.Info("url added", slog.Int64("id", id))
					break // Успешно сохранили, выходим из цикла
				}

				if errors.Is(err, storage.ErrURLExist) {
					continue // Пробуем сгенерировать снова
				}

				// Какая-то другая ошибка БД
				log.Error("failed to save url", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.Error("failed to save url"))
				return
			}

			// Если за 5 попыток так и не смогли сгенерировать уникальный alias
			if err != nil {
				log.Error("failed to generate unique alias after retries")
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.Error("please try again later"))
				return
			}

		}
		// Если все удачно возвращаем Status:200 и Response
		render.Status(r, http.StatusOK)
		render.JSON(w, r, Response{
			Response: resp.Ok(),
			Alias:    alias,
		})
	}
}

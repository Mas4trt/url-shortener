package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"

	"url-shortener/internal/ssoclient"
	"url-shortener/internal/transport/http/response"
	sl "url-shortener/pkg/logger/sl"

	"github.com/go-chi/render"
)

const maxAuthRequestBytes = 1 << 20 // 1 MiB

const minPasswordLength = 8

// AuthService is what the auth handler expects from sso — satisfied by
// *ssoclient.Client.
type AuthService interface {
	Register(ctx context.Context, email, password string) (uint64, error)
	Login(ctx context.Context, email, password string) (access, refresh string, err error)
	RefreshTokens(ctx context.Context, refreshToken string) (access, refresh string, err error)
	Logout(ctx context.Context, refreshToken string) error
}

type AuthHandler struct {
	log *slog.Logger
	sso AuthService
}

func NewAuth(log *slog.Logger, sso AuthService) *AuthHandler {
	return &AuthHandler{
		log: log.With(slog.String("layer", "auth_handler")),
		sso: sso,
	}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenResponse struct {
	response.Response
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthRequestBytes)

	var req registerRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		response.Respond(w, r, http.StatusBadRequest, response.Error("failed to decode request"))
		return
	}

	if err := validateEmail(req.Email); err != nil {
		response.Respond(w, r, http.StatusBadRequest, response.Error(err.Error()))
		return
	}
	if len(req.Password) < minPasswordLength {
		response.Respond(w, r, http.StatusBadRequest,
			response.Error(fmt.Sprintf("password must be at least %d characters", minPasswordLength)))
		return
	}

	_, err := h.sso.Register(r.Context(), req.Email, req.Password)
	switch {
	case errors.Is(err, ssoclient.ErrUserExists):
		response.Respond(w, r, http.StatusConflict, response.Error("user already exists"))
		return
	case err != nil:
		h.log.Error("register failed", sl.Err(err))
		response.Respond(w, r, http.StatusInternalServerError, response.Error("failed to register"))
		return
	}

	response.Respond(w, r, http.StatusCreated, response.Ok())
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthRequestBytes)

	var req loginRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		response.Respond(w, r, http.StatusBadRequest, response.Error("failed to decode request"))
		return
	}

	if err := validateEmail(req.Email); err != nil {
		response.Respond(w, r, http.StatusBadRequest, response.Error(err.Error()))
		return
	}
	if req.Password == "" {
		response.Respond(w, r, http.StatusBadRequest, response.Error("password is required"))
		return
	}

	access, refresh, err := h.sso.Login(r.Context(), req.Email, req.Password)
	switch {
	case errors.Is(err, ssoclient.ErrInvalidCredentials):
		response.Respond(w, r, http.StatusUnauthorized, response.Error("invalid email or password"))
		return
	case err != nil:
		h.log.Error("login failed", sl.Err(err))
		response.Respond(w, r, http.StatusInternalServerError, response.Error("failed to login"))
		return
	}

	response.Respond(w, r, http.StatusOK, tokenResponse{
		Response:     response.Ok(),
		AccessToken:  access,
		RefreshToken: refresh,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthRequestBytes)

	var req refreshRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil || req.RefreshToken == "" {
		response.Respond(w, r, http.StatusBadRequest, response.Error("refresh_token is required"))
		return
	}

	access, refresh, err := h.sso.RefreshTokens(r.Context(), req.RefreshToken)
	switch {
	case errors.Is(err, ssoclient.ErrRefreshInvalid):
		response.Respond(w, r, http.StatusUnauthorized, response.Error("refresh token invalid or expired"))
		return
	case err != nil:
		h.log.Error("refresh failed", sl.Err(err))
		response.Respond(w, r, http.StatusInternalServerError, response.Error("failed to refresh"))
		return
	}

	response.Respond(w, r, http.StatusOK, tokenResponse{
		Response:     response.Ok(),
		AccessToken:  access,
		RefreshToken: refresh,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthRequestBytes)

	var req refreshRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil || req.RefreshToken == "" {
		response.Respond(w, r, http.StatusBadRequest, response.Error("refresh_token is required"))
		return
	}

	if err := h.sso.Logout(r.Context(), req.RefreshToken); err != nil {
		h.log.Error("logout failed", sl.Err(err))
		response.Respond(w, r, http.StatusInternalServerError, response.Error("failed to logout"))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func validateEmail(email string) error {
	if email == "" {
		return errors.New("email is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return errors.New("email must be a valid address")
	}
	return nil
}

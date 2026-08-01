package response

import (
	"errors"
	"log/slog"
	"net/http"
)

// ErrCase maps a sentinel error to the HTTP status and message the
// client should see for it.
type ErrCase struct {
	Target error
	Status int
	Msg    string
}

// Case is a small constructor to keep handler call sites terse.
func Case(target error, status int, msg string) ErrCase {
	return ErrCase{Target: target, Status: status, Msg: msg}
}

// RespondError inspects err against cases in order (using errors.Is, so
// wrapped errors match) and writes the corresponding status/body. If no
// case matches, it logs the error server-side (never leaking internals
// to the client) and responds with fallbackStatus/fallbackMsg.
//
// This replaces a hand-rolled `switch { case errors.Is(err, X): ... }` in
// every handler with a declarative list, so adding a new domain error
// case is a one-line change instead of a copy-pasted switch arm.
func RespondError(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error, fallbackStatus int, fallbackMsg string, cases ...ErrCase) {
	for _, c := range cases {
		if errors.Is(err, c.Target) {
			if c.Status >= http.StatusInternalServerError {
				log.Error(c.Msg, slog.String("error", err.Error()))
			} else {
				log.Debug(c.Msg, slog.String("error", err.Error()))
			}
			Respond(w, r, c.Status, Error(c.Msg))
			return
		}
	}

	log.Error(fallbackMsg, slog.String("error", err.Error()))
	Respond(w, r, fallbackStatus, Error(fallbackMsg))
}

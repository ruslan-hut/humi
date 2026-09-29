package account

import (
	"context"
	"log/slog"
	"net/http"

	"humi/entity"
	"humi/internal/http-server/middleware/userauth"
	"humi/internal/lib/request"
	"humi/internal/lib/response"
	"humi/internal/lib/sl"
)

// Handler manages the signed-in user's own password and devices.
type Handler interface {
	ChangePassword(ctx context.Context, ses *entity.Session, current, next string) error
	Sessions(ctx context.Context, cur *entity.Session) ([]entity.Session, error)
	RevokeSession(ctx context.Context, userID, id int64) error
}

// Password changes the password and signs out every other device.
func Password(log *slog.Logger, h Handler) http.HandlerFunc {
	log = log.With(sl.Module("handlers.account.password"))

	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Current string `json:"current"`
			New     string `json:"new"`
		}
		if !request.Decode(w, r, &in) {
			return
		}
		if err := h.ChangePassword(r.Context(), userauth.Session(r.Context()), in.Current, in.New); err != nil {
			response.Err(w, r, log, "change password", err)
			return
		}
		response.Send(w, r, nil)
	}
}

// Sessions lists the user's signed-in devices.
func Sessions(log *slog.Logger, h Handler) http.HandlerFunc {
	log = log.With(sl.Module("handlers.account.sessions"))

	return func(w http.ResponseWriter, r *http.Request) {
		list, err := h.Sessions(r.Context(), userauth.Session(r.Context()))
		if err != nil {
			response.Err(w, r, log, "list sessions", err)
			return
		}
		response.Send(w, r, list)
	}
}

// Revoke signs out one of the user's devices.
func Revoke(log *slog.Logger, h Handler) http.HandlerFunc {
	log = log.With(sl.Module("handlers.account.revoke"))

	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := request.ID(w, r, "id")
		if !ok {
			return
		}
		if err := h.RevokeSession(r.Context(), userauth.User(r.Context()).ID, id); err != nil {
			response.Err(w, r, log, "revoke session", err)
			return
		}
		response.Send(w, r, nil)
	}
}

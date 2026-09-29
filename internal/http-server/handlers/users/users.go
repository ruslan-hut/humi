package users

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

// Handler manages who can sign in. Admin only.
type Handler interface {
	Users(ctx context.Context) ([]entity.UserInfo, error)
	SetRole(ctx context.Context, actor *entity.User, id int64, role string) (*entity.User, error)
	DeleteUser(ctx context.Context, actor *entity.User, id int64) error
	ResetLink(ctx context.Context, actor *entity.User, id int64) (*entity.Invite, error)
}

// List returns every user with session activity.
func List(log *slog.Logger, h Handler) http.HandlerFunc {
	log = log.With(sl.Module("handlers.users.list"))

	return func(w http.ResponseWriter, r *http.Request) {
		list, err := h.Users(r.Context())
		if err != nil {
			response.Err(w, r, log, "list users", err)
			return
		}
		response.Send(w, r, list)
	}
}

// Update changes the role of a user.
func Update(log *slog.Logger, h Handler) http.HandlerFunc {
	log = log.With(sl.Module("handlers.users.update"))

	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := request.ID(w, r, "id")
		if !ok {
			return
		}
		var in struct {
			Role string `json:"role"`
		}
		if !request.Decode(w, r, &in) {
			return
		}
		u, err := h.SetRole(r.Context(), userauth.User(r.Context()), id, in.Role)
		if err != nil {
			response.Err(w, r, log, "set role", err)
			return
		}
		response.Send(w, r, u)
	}
}

// Delete removes a user.
func Delete(log *slog.Logger, h Handler) http.HandlerFunc {
	log = log.With(sl.Module("handlers.users.delete"))

	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := request.ID(w, r, "id")
		if !ok {
			return
		}
		if err := h.DeleteUser(r.Context(), userauth.User(r.Context()), id); err != nil {
			response.Err(w, r, log, "delete user", err)
			return
		}
		response.Send(w, r, nil)
	}
}

// Reset issues a password reset link for a user.
func Reset(log *slog.Logger, h Handler) http.HandlerFunc {
	log = log.With(sl.Module("handlers.users.reset"))

	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := request.ID(w, r, "id")
		if !ok {
			return
		}
		inv, err := h.ResetLink(r.Context(), userauth.User(r.Context()), id)
		if err != nil {
			response.Err(w, r, log, "reset link", err)
			return
		}
		response.Send(w, r, inv)
	}
}

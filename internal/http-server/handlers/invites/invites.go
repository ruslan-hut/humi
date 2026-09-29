package invites

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

// Handler manages pending join and reset links. Admin only.
type Handler interface {
	Invites(ctx context.Context) ([]entity.Invite, error)
	CreateInvite(ctx context.Context, actor *entity.User, role string) (*entity.Invite, error)
	DeleteInvite(ctx context.Context, id int64) error
}

// List returns the links that are still pending.
func List(log *slog.Logger, h Handler) http.HandlerFunc {
	log = log.With(sl.Module("handlers.invites.list"))

	return func(w http.ResponseWriter, r *http.Request) {
		list, err := h.Invites(r.Context())
		if err != nil {
			response.Err(w, r, log, "list invites", err)
			return
		}
		response.Send(w, r, list)
	}
}

// Create issues a join link; its token is in this response only.
func Create(log *slog.Logger, h Handler) http.HandlerFunc {
	log = log.With(sl.Module("handlers.invites.create"))

	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Role string `json:"role"`
		}
		if !request.Decode(w, r, &in) {
			return
		}
		inv, err := h.CreateInvite(r.Context(), userauth.User(r.Context()), in.Role)
		if err != nil {
			response.Err(w, r, log, "create invite", err)
			return
		}
		response.Send(w, r, inv)
	}
}

// Delete revokes a pending link.
func Delete(log *slog.Logger, h Handler) http.HandlerFunc {
	log = log.With(sl.Module("handlers.invites.delete"))

	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := request.ID(w, r, "id")
		if !ok {
			return
		}
		if err := h.DeleteInvite(r.Context(), id); err != nil {
			response.Err(w, r, log, "delete invite", err)
			return
		}
		response.Send(w, r, nil)
	}
}

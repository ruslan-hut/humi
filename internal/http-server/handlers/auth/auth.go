package auth

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"humi/entity"
	"humi/internal/http-server/middleware/userauth"
	"humi/internal/lib/request"
	"humi/internal/lib/response"
	"humi/internal/lib/sl"
)

// Handler signs people in and out.
type Handler interface {
	Login(ctx context.Context, username, password, userAgent string) (*entity.User, string, error)
	Logout(ctx context.Context, ses *entity.Session) error
	Invite(ctx context.Context, token string) (*entity.Invite, error)
	AcceptInvite(ctx context.Context, token, username, password, userAgent string) (*entity.User, string, error)
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login checks a password and sets the session cookie.
func Login(log *slog.Logger, h Handler, cookies userauth.Cookies) http.HandlerFunc {
	log = log.With(sl.Module("handlers.auth.login"))

	return func(w http.ResponseWriter, r *http.Request) {
		var in credentials
		if !request.Decode(w, r, &in) {
			return
		}
		u, token, err := h.Login(r.Context(), in.Username, in.Password, r.UserAgent())
		if err != nil {
			response.Err(w, r, log, "login", err)
			return
		}
		cookies.Set(w, token)
		response.Send(w, r, u)
	}
}

// Logout revokes the current session and clears the cookie.
func Logout(log *slog.Logger, h Handler, cookies userauth.Cookies) http.HandlerFunc {
	log = log.With(sl.Module("handlers.auth.logout"))

	return func(w http.ResponseWriter, r *http.Request) {
		if err := h.Logout(r.Context(), userauth.Session(r.Context())); err != nil {
			response.Err(w, r, log, "logout", err)
			return
		}
		cookies.Clear(w)
		response.Send(w, r, nil)
	}
}

// Me returns the signed-in user.
func Me() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.Send(w, r, userauth.User(r.Context()))
	}
}

// Invite describes a pending link so the page can show the right form.
func Invite(log *slog.Logger, h Handler) http.HandlerFunc {
	log = log.With(sl.Module("handlers.auth.invite"))

	return func(w http.ResponseWriter, r *http.Request) {
		inv, err := h.Invite(r.Context(), chi.URLParam(r, "token"))
		if err != nil {
			response.Err(w, r, log, "invite", err)
			return
		}
		response.Send(w, r, inv)
	}
}

// Accept redeems a join or reset link and signs the user in.
func Accept(log *slog.Logger, h Handler, cookies userauth.Cookies) http.HandlerFunc {
	log = log.With(sl.Module("handlers.auth.accept"))

	return func(w http.ResponseWriter, r *http.Request) {
		var in credentials
		if !request.Decode(w, r, &in) {
			return
		}
		u, token, err := h.AcceptInvite(r.Context(), chi.URLParam(r, "token"), in.Username, in.Password, r.UserAgent())
		if err != nil {
			response.Err(w, r, log, "accept invite", err)
			return
		}
		cookies.Set(w, token)
		response.Send(w, r, u)
	}
}

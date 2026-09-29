package userauth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"humi/entity"
	"humi/internal/lib/response"
	"humi/internal/lib/sl"
)

// CookieName carries the session token. It is HttpOnly, so the page never
// sees it; the dashboard asks /auth/me who is signed in.
const CookieName = "humi_session"

type ctxKey int

const (
	userKey ctxKey = iota
	sessionKey
)

// Authenticator resolves a session token to its user. refreshed reports that
// the session expiry slid forward.
type Authenticator interface {
	Resume(ctx context.Context, token string) (ses *entity.Session, u *entity.User, refreshed bool, err error)
}

// Cookies writes the session cookie. Secure is off only for local http.
type Cookies struct {
	Secure bool
	MaxAge int
}

// Set stores the session token on the client.
func (c Cookies) Set(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: token, Path: "/", MaxAge: c.MaxAge,
		HttpOnly: true, Secure: c.Secure, SameSite: http.SameSiteStrictMode,
	})
}

// Clear removes the session cookie.
func (c Cookies) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: c.Secure, SameSite: http.SameSiteStrictMode,
	})
}

// New rejects requests without a live session and puts the user and session
// in the context.
func New(log *slog.Logger, auth Authenticator, cookies Cookies) func(next http.Handler) http.Handler {
	log = log.With(sl.Module("middleware.userauth"))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(CookieName)
			if err != nil || c.Value == "" {
				response.Fail(w, r, http.StatusUnauthorized, "sign in required")
				return
			}

			ses, u, refreshed, err := auth.Resume(r.Context(), c.Value)
			if errors.Is(err, entity.ErrUnauthorized) {
				cookies.Clear(w)
			}
			if err != nil {
				response.Err(w, r, log, "resume session", err)
				return
			}
			if refreshed {
				cookies.Set(w, c.Value)
			}

			ctx := context.WithValue(r.Context(), userKey, u)
			ctx = context.WithValue(ctx, sessionKey, ses)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Admin rejects signed-in users who are not admins. It runs after New.
func Admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u := User(r.Context()); u == nil || u.Role != entity.RoleAdmin {
			response.Fail(w, r, http.StatusForbidden, "admin only")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// User returns the signed-in user of this request.
func User(ctx context.Context) *entity.User {
	u, _ := ctx.Value(userKey).(*entity.User)
	return u
}

// Session returns the session this request came with.
func Session(ctx context.Context) *entity.Session {
	s, _ := ctx.Value(sessionKey).(*entity.Session)
	return s
}

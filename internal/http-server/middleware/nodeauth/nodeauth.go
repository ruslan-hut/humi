package nodeauth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"humi/entity"
	"humi/internal/lib/response"
	"humi/internal/lib/sl"
)

type ctxKey int

const nodeKey ctxKey = iota

// Authenticator resolves an ingest token to the node that owns it.
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*entity.Node, error)
}

// New rejects ingest requests that do not carry a known node bearer token.
func New(log *slog.Logger, auth Authenticator) func(next http.Handler) http.Handler {
	log = log.With(sl.Module("middleware.nodeauth"))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if token == "" || token == r.Header.Get("Authorization") {
				response.Fail(w, r, http.StatusUnauthorized, "bearer token required")
				return
			}

			node, err := auth.Authenticate(r.Context(), token)
			if err != nil {
				log.Warn("rejected ingest", slog.String("remote", r.RemoteAddr))
				response.Fail(w, r, http.StatusUnauthorized, "unknown token")
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), nodeKey, node)))
		})
	}
}

// FromContext returns the node authenticated for this request.
func FromContext(ctx context.Context) *entity.Node {
	node, _ := ctx.Value(nodeKey).(*entity.Node)
	return node
}

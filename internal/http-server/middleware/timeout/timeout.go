package timeout

import (
	"context"
	"net/http"
	"time"
)

// Timeout caps the lifetime of every request context.
func Timeout(seconds int) func(next http.Handler) http.Handler {
	d := time.Duration(seconds) * time.Second
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

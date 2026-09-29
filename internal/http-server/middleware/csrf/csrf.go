package csrf

import (
	"net/http"

	"humi/internal/lib/response"
)

// Header and Value mark a request as sent by the dashboard. A cross-site form
// cannot set a custom header, and a cross-site fetch that does needs a CORS
// preflight this service never grants.
const (
	Header = "X-Requested-With"
	Value  = "humi"
)

// Require rejects state-changing requests that lack the header. The session
// cookie is SameSite=Strict already; this is the second lock on the door.
func Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if r.Header.Get(Header) != Value {
				response.Fail(w, r, http.StatusForbidden, "missing X-Requested-With")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

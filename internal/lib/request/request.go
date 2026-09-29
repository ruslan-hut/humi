package request

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"humi/internal/lib/response"
)

// Decode reads a JSON body into v, answering 400 when it cannot.
func Decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		response.Fail(w, r, http.StatusBadRequest, "invalid payload")
		return false
	}
	return true
}

// ID reads a numeric URL parameter, answering 400 when it is not one.
func ID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil || id <= 0 {
		response.Fail(w, r, http.StatusBadRequest, "invalid "+name)
		return 0, false
	}
	return id, true
}

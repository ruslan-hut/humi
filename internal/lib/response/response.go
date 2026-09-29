package response

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/render"

	"humi/entity"
	"humi/internal/lib/clock"
	"humi/internal/lib/sl"
)

// Response is the envelope every endpoint returns.
type Response struct {
	Data          interface{} `json:"data,omitempty"`
	Success       bool        `json:"success"`
	StatusMessage string      `json:"status_message"`
	Timestamp     string      `json:"timestamp"`
}

// Ok wraps a payload in a success envelope.
func Ok(data interface{}) Response {
	return Response{Data: data, Success: true, StatusMessage: "Success", Timestamp: clock.Now()}
}

// Error wraps a message in a failure envelope.
func Error(message string) Response {
	return Response{Success: false, StatusMessage: message, Timestamp: clock.Now()}
}

// Fail writes a failure envelope with the given HTTP status.
func Fail(w http.ResponseWriter, r *http.Request, status int, message string) {
	render.Status(r, status)
	render.JSON(w, r, Error(message))
}

// Send writes a success envelope with status 200.
func Send(w http.ResponseWriter, r *http.Request, data interface{}) {
	render.JSON(w, r, Ok(data))
}

// Err writes the failure matching a domain error. Client errors carry their
// own message; anything else is logged and reported as a bare 500.
func Err(w http.ResponseWriter, r *http.Request, log *slog.Logger, op string, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, entity.ErrInvalid):
		status = http.StatusBadRequest
	case errors.Is(err, entity.ErrUnauthorized):
		status = http.StatusUnauthorized
	case errors.Is(err, entity.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, entity.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, entity.ErrConflict):
		status = http.StatusConflict
	}
	if status == http.StatusInternalServerError {
		log.Error(op, sl.Err(err))
		Fail(w, r, status, "internal error")
		return
	}
	Fail(w, r, status, err.Error())
}

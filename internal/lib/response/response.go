package response

import (
	"net/http"

	"github.com/go-chi/render"

	"humi/internal/lib/clock"
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

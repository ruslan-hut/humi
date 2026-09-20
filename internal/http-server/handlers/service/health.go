package service

import (
	"context"
	"log/slog"
	"net/http"

	"humi/internal/lib/response"
	"humi/internal/lib/sl"
)

// Handler reports service state.
type Handler interface {
	Stat(ctx context.Context) (map[string]any, error)
}

// Health answers readiness probes and reports storage counters.
func Health(log *slog.Logger, h Handler) http.HandlerFunc {
	log = log.With(sl.Module("handlers.service.health"))

	return func(w http.ResponseWriter, r *http.Request) {
		stat, err := h.Stat(r.Context())
		if err != nil {
			log.Error("health", sl.Err(err))
			response.Fail(w, r, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		response.Send(w, r, stat)
	}
}

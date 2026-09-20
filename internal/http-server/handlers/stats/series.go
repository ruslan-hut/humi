package stats

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"humi/entity"
	"humi/internal/lib/response"
	"humi/internal/lib/sl"

	"github.com/go-chi/chi/v5"
)

// Handler serves aggregated history.
type Handler interface {
	Series(ctx context.Context, slug string, from, to int64, bucketS int) (*entity.Series, error)
}

// Series returns a bucketed history for one node.
// Query: ?from=<unix>&to=<unix>&bucket=<seconds>, defaulting to the last 24 hours.
func Series(log *slog.Logger, h Handler) http.HandlerFunc {
	log = log.With(sl.Module("handlers.stats.series"))

	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		now := time.Now().UTC().Unix()

		from := queryInt(r.URL.Query().Get("from"), now-24*3600)
		to := queryInt(r.URL.Query().Get("to"), now)
		bucket := int(queryInt(r.URL.Query().Get("bucket"), 300))

		series, err := h.Series(r.Context(), slug, from, to, bucket)
		if err != nil {
			log.Error("series", slog.String("node", slug), sl.Err(err))
			response.Fail(w, r, http.StatusBadRequest, err.Error())
			return
		}

		response.Send(w, r, series)
	}
}

func queryInt(raw string, fallback int64) int64 {
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return v
}

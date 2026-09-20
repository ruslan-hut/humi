package readings

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"humi/entity"
	"humi/internal/http-server/middleware/nodeauth"
	"humi/internal/lib/response"
	"humi/internal/lib/sl"
)

// Handler accepts measurements from nodes.
type Handler interface {
	Ingest(ctx context.Context, node *entity.Node, in []entity.ReadingInput) (int, error)
}

// payload accepts either a single reading or a batch, so a node that buffered
// while the server was down can flush everything in one request.
type payload struct {
	single entity.ReadingInput
	batch  []entity.ReadingInput
}

func (p *payload) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '[' {
		return json.Unmarshal(b, &p.batch)
	}
	if err := json.Unmarshal(b, &p.single); err != nil {
		return err
	}
	p.batch = []entity.ReadingInput{p.single}
	return nil
}

// Ingest stores readings posted by an authenticated node.
func Ingest(log *slog.Logger, h Handler) http.HandlerFunc {
	log = log.With(sl.Module("handlers.readings.ingest"))

	return func(w http.ResponseWriter, r *http.Request) {
		node := nodeauth.FromContext(r.Context())
		if node == nil {
			response.Fail(w, r, http.StatusUnauthorized, "no node in context")
			return
		}

		var p payload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			response.Fail(w, r, http.StatusBadRequest, "invalid payload")
			return
		}

		n, err := h.Ingest(r.Context(), node, p.batch)
		if err != nil {
			log.Error("ingest", slog.String("node", node.Slug), sl.Err(err))
			response.Fail(w, r, http.StatusBadRequest, err.Error())
			return
		}

		response.Send(w, r, map[string]any{"stored": n, "interval_s": node.IntervalS})
	}
}

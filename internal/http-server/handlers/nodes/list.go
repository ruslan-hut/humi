package nodes

import (
	"context"
	"log/slog"
	"net/http"

	"humi/entity"
	"humi/internal/lib/response"
	"humi/internal/lib/sl"
)

// Handler serves node state to the dashboard.
type Handler interface {
	Nodes(ctx context.Context) ([]entity.NodeState, error)
}

// List returns every node with its latest reading and online flag.
func List(log *slog.Logger, h Handler) http.HandlerFunc {
	log = log.With(sl.Module("handlers.nodes.list"))

	return func(w http.ResponseWriter, r *http.Request) {
		states, err := h.Nodes(r.Context())
		if err != nil {
			log.Error("list nodes", sl.Err(err))
			response.Fail(w, r, http.StatusInternalServerError, "cannot read nodes")
			return
		}
		response.Send(w, r, states)
	}
}

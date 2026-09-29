package nodes

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"humi/entity"
	"humi/internal/lib/request"
	"humi/internal/lib/response"
	"humi/internal/lib/sl"
)

// Settings changes nodes and their rules. Everything but Rules is admin only.
type Settings interface {
	CreateNode(ctx context.Context, in entity.NodeInput) (*entity.Node, string, error)
	UpdateNode(ctx context.Context, slug string, p entity.NodePatch) (*entity.Node, error)
	DeleteNode(ctx context.Context, slug string) error
	RotateToken(ctx context.Context, slug string) (string, error)
	Rules(ctx context.Context, slug string) ([]entity.Rule, error)
	ReplaceRules(ctx context.Context, slug string, rules []entity.Rule) ([]entity.Rule, error)
}

// Create registers a node. The ingest token is in this response only.
func Create(log *slog.Logger, h Settings) http.HandlerFunc {
	log = log.With(sl.Module("handlers.nodes.create"))

	return func(w http.ResponseWriter, r *http.Request) {
		var in entity.NodeInput
		if !request.Decode(w, r, &in) {
			return
		}
		node, token, err := h.CreateNode(r.Context(), in)
		if err != nil {
			response.Err(w, r, log, "create node", err)
			return
		}
		response.Send(w, r, map[string]any{"node": node, "token": token})
	}
}

// Update changes the settings of a node.
func Update(log *slog.Logger, h Settings) http.HandlerFunc {
	log = log.With(sl.Module("handlers.nodes.update"))

	return func(w http.ResponseWriter, r *http.Request) {
		var p entity.NodePatch
		if !request.Decode(w, r, &p) {
			return
		}
		node, err := h.UpdateNode(r.Context(), chi.URLParam(r, "slug"), p)
		if err != nil {
			response.Err(w, r, log, "update node", err)
			return
		}
		response.Send(w, r, node)
	}
}

// Delete removes a node with all of its readings.
func Delete(log *slog.Logger, h Settings) http.HandlerFunc {
	log = log.With(sl.Module("handlers.nodes.delete"))

	return func(w http.ResponseWriter, r *http.Request) {
		if err := h.DeleteNode(r.Context(), chi.URLParam(r, "slug")); err != nil {
			response.Err(w, r, log, "delete node", err)
			return
		}
		response.Send(w, r, nil)
	}
}

// Token issues a new ingest token; the old one stops working at once.
func Token(log *slog.Logger, h Settings) http.HandlerFunc {
	log = log.With(sl.Module("handlers.nodes.token"))

	return func(w http.ResponseWriter, r *http.Request) {
		token, err := h.RotateToken(r.Context(), chi.URLParam(r, "slug"))
		if err != nil {
			response.Err(w, r, log, "rotate token", err)
			return
		}
		response.Send(w, r, map[string]string{"token": token})
	}
}

// Rules returns the alert rules of a node.
func Rules(log *slog.Logger, h Settings) http.HandlerFunc {
	log = log.With(sl.Module("handlers.nodes.rules"))

	return func(w http.ResponseWriter, r *http.Request) {
		rules, err := h.Rules(r.Context(), chi.URLParam(r, "slug"))
		if err != nil {
			response.Err(w, r, log, "rules", err)
			return
		}
		response.Send(w, r, rules)
	}
}

// ReplaceRules swaps the whole rule set of a node.
func ReplaceRules(log *slog.Logger, h Settings) http.HandlerFunc {
	log = log.With(sl.Module("handlers.nodes.replace_rules"))

	return func(w http.ResponseWriter, r *http.Request) {
		var in []entity.Rule
		if !request.Decode(w, r, &in) {
			return
		}
		rules, err := h.ReplaceRules(r.Context(), chi.URLParam(r, "slug"), in)
		if err != nil {
			response.Err(w, r, log, "replace rules", err)
			return
		}
		response.Send(w, r, rules)
	}
}

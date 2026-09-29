package core

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"humi/entity"
)

// CreateNode registers a node with the default rules and returns its ingest
// token, which is shown once.
func (c *Core) CreateNode(ctx context.Context, in entity.NodeInput) (*entity.Node, string, error) {
	in.Slug = strings.TrimSpace(in.Slug)
	in.Name = strings.TrimSpace(in.Name)
	in.Location = strings.TrimSpace(in.Location)
	if in.IntervalS == 0 {
		in.IntervalS = 900
	}
	for _, err := range []error{
		validateSlug(in.Slug), validateName(in.Name), validateLocation(in.Location), validateInterval(in.IntervalS),
	} {
		if err != nil {
			return nil, "", err
		}
	}

	t, err := c.db.CreateNode(ctx, in.Slug, in.Name, in.Location, in.IntervalS)
	if errors.Is(err, entity.ErrConflict) {
		return nil, "", entity.Errorf(entity.ErrConflict, "slug taken")
	}
	if err != nil {
		return nil, "", err
	}
	node, err := c.db.NodeBySlug(ctx, in.Slug)
	if err != nil {
		return nil, "", err
	}

	c.log.Info("node created", slog.String("node", node.Slug))
	return node, t, nil
}

// UpdateNode applies the fields set in the patch. A new interval reaches the
// node in the reply to its next reading.
func (c *Core) UpdateNode(ctx context.Context, slug string, p entity.NodePatch) (*entity.Node, error) {
	node, err := c.node(ctx, slug)
	if err != nil {
		return nil, err
	}
	if p.Name != nil {
		node.Name = strings.TrimSpace(*p.Name)
		if err = validateName(node.Name); err != nil {
			return nil, err
		}
	}
	if p.Location != nil {
		node.Location = strings.TrimSpace(*p.Location)
		if err = validateLocation(node.Location); err != nil {
			return nil, err
		}
	}
	if p.IntervalS != nil {
		if err = validateInterval(*p.IntervalS); err != nil {
			return nil, err
		}
		node.IntervalS = *p.IntervalS
	}
	if p.Enabled != nil {
		node.Enabled = *p.Enabled
	}

	if err = c.db.UpdateNode(ctx, node); err != nil {
		return nil, err
	}
	return node, nil
}

// DeleteNode removes a node and everything it recorded.
func (c *Core) DeleteNode(ctx context.Context, slug string) error {
	node, err := c.node(ctx, slug)
	if err != nil {
		return err
	}
	if err = c.db.DeleteNode(ctx, node.ID); err != nil {
		return err
	}
	c.log.Info("node deleted", slog.String("node", slug))
	return nil
}

// RotateToken issues a new ingest token for a node; the old one stops working.
func (c *Core) RotateToken(ctx context.Context, slug string) (string, error) {
	node, err := c.node(ctx, slug)
	if err != nil {
		return "", err
	}
	t, err := c.db.RotateNodeToken(ctx, node.ID)
	if err != nil {
		return "", err
	}
	c.log.Info("node token rotated", slog.String("node", slug))
	return t, nil
}

// Rules returns the alert rules of one node.
func (c *Core) Rules(ctx context.Context, slug string) ([]entity.Rule, error) {
	node, err := c.node(ctx, slug)
	if err != nil {
		return nil, err
	}
	rules, err := c.db.Rules(ctx, node.ID)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		rules[i].NodeSlug = slug
	}
	return rules, nil
}

// ReplaceRules swaps the whole rule set of a node.
func (c *Core) ReplaceRules(ctx context.Context, slug string, in []entity.Rule) ([]entity.Rule, error) {
	node, err := c.node(ctx, slug)
	if err != nil {
		return nil, err
	}
	rules, err := normalizeRules(in)
	if err != nil {
		return nil, err
	}
	if err = c.db.ReplaceRules(ctx, node.ID, rules); err != nil {
		return nil, err
	}
	return c.Rules(ctx, slug)
}

func (c *Core) node(ctx context.Context, slug string) (*entity.Node, error) {
	node, err := c.db.NodeBySlug(ctx, slug)
	if errors.Is(err, entity.ErrNotFound) {
		return nil, entity.Errorf(entity.ErrNotFound, "node %q not found", slug)
	}
	return node, err
}

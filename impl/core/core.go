package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"humi/entity"
	"humi/internal/config"
	"humi/internal/lib/clock"
	"humi/internal/lib/sl"
)

// maxPoints caps how many buckets a series may return, so a phone never has to
// draw more than it can show.
const maxPoints = 480

// Database is the storage contract the core depends on.
type Database interface {
	NodeByToken(ctx context.Context, token string) (*entity.Node, error)
	NodeBySlug(ctx context.Context, slug string) (*entity.Node, error)
	Nodes(ctx context.Context) ([]entity.NodeState, error)
	SaveReadings(ctx context.Context, nodeID int64, readings []entity.Reading) error
	Series(ctx context.Context, nodeID int64, from, to int64, bucketS int) ([]entity.Bucket, error)
	Stat(ctx context.Context) (map[string]any, error)

	CreateNode(ctx context.Context, slug, name, location string, intervalS int) (string, error)
	UpdateNode(ctx context.Context, n *entity.Node) error
	DeleteNode(ctx context.Context, id int64) error
	RotateNodeToken(ctx context.Context, id int64) (string, error)
	Rules(ctx context.Context, nodeID int64) ([]entity.Rule, error)
	ReplaceRules(ctx context.Context, nodeID int64, rules []entity.Rule) error

	CreateUser(ctx context.Context, username, passHash, role string) (*entity.User, error)
	UserByUsername(ctx context.Context, username string) (*entity.User, string, error)
	UserByID(ctx context.Context, id int64) (*entity.User, string, error)
	Users(ctx context.Context, now int64) ([]entity.UserInfo, error)
	SetUserRole(ctx context.Context, id int64, role string) error
	DeleteUser(ctx context.Context, id int64) error
	SetPassword(ctx context.Context, id int64, passHash string) error

	CreateSession(ctx context.Context, userID int64, userAgent string, now, expiresAt int64) (*entity.Session, string, error)
	SessionByToken(ctx context.Context, token string, now int64) (*entity.Session, *entity.User, error)
	TouchSession(ctx context.Context, id, now, expiresAt int64) error
	Sessions(ctx context.Context, userID, now int64) ([]entity.Session, error)
	DeleteSession(ctx context.Context, userID, id int64) error
	DeleteOtherSessions(ctx context.Context, userID, keepID int64) error

	CreateInvite(ctx context.Context, kind, role string, userID, createdBy, now, expiresAt int64) (*entity.Invite, error)
	InviteByToken(ctx context.Context, token string, now int64) (*entity.Invite, error)
	Invites(ctx context.Context, now int64) ([]entity.Invite, error)
	DeleteInvite(ctx context.Context, id int64) error
	AcceptJoin(ctx context.Context, token, username, passHash string, now int64) (*entity.User, error)
	AcceptReset(ctx context.Context, token, passHash string, now int64) (*entity.User, error)
}

// Core wires the business logic on top of the database.
type Core struct {
	db   Database
	conf *config.Config
	log  *slog.Logger
}

// New returns a core bound to the given database.
func New(db Database, conf *config.Config, log *slog.Logger) *Core {
	return &Core{db: db, conf: conf, log: log.With(sl.Module("core"))}
}

// Authenticate resolves an ingest bearer token to the node that owns it.
func (c *Core) Authenticate(ctx context.Context, token string) (*entity.Node, error) {
	return c.db.NodeByToken(ctx, token)
}

// Ingest validates and stores a batch of readings for one node.
func (c *Core) Ingest(ctx context.Context, node *entity.Node, in []entity.ReadingInput) (int, error) {
	if len(in) == 0 {
		return 0, errors.New("empty batch")
	}
	if len(in) > c.conf.Ingest.MaxBatch {
		return 0, fmt.Errorf("batch of %d exceeds limit %d", len(in), c.conf.Ingest.MaxBatch)
	}

	now := clock.Unix()
	readings := make([]entity.Reading, 0, len(in))
	for _, r := range in {
		if r.RH < 0 || r.RH > 100 {
			return 0, fmt.Errorf("rh %.1f out of range", r.RH)
		}
		if r.Temp != nil && (*r.Temp < -50 || *r.Temp > 100) {
			return 0, fmt.Errorf("temp %.1f out of range", *r.Temp)
		}
		age := r.AgeS
		if age < 0 || age > c.conf.Ingest.MaxAgeS {
			age = 0
		}
		readings = append(readings, entity.Reading{
			ReceivedAt: now - int64(age),
			RH:         r.RH,
			Temp:       r.Temp,
			VBat:       r.VBat,
			RSSI:       r.RSSI,
		})
	}

	if err := c.db.SaveReadings(ctx, node.ID, readings); err != nil {
		return 0, err
	}

	c.log.Debug("readings stored", slog.String("node", node.Slug), slog.Int("count", len(readings)))
	return len(readings), nil
}

// Nodes returns dashboard state for every node, with staleness resolved.
func (c *Core) Nodes(ctx context.Context) ([]entity.NodeState, error) {
	states, err := c.db.Nodes(ctx)
	if err != nil {
		return nil, err
	}

	now := clock.Unix()
	for i := range states {
		deadline := int64(states[i].IntervalS * c.conf.Ingest.OfflineRatio)
		states[i].Online = states[i].LastSeen > 0 && now-states[i].LastSeen <= deadline
	}

	return states, nil
}

// Series returns a bucketed history for one node. The bucket is widened when the
// requested range would produce more points than maxPoints.
func (c *Core) Series(ctx context.Context, slug string, from, to int64, bucketS int) (*entity.Series, error) {
	node, err := c.db.NodeBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if to <= from {
		return nil, errors.New("empty range")
	}
	if bucketS <= 0 {
		bucketS = 300
	}
	if span := int((to - from) / int64(bucketS)); span > maxPoints {
		bucketS = int((to-from)/maxPoints) + 1
	}

	points, err := c.db.Series(ctx, node.ID, from, to, bucketS)
	if err != nil {
		return nil, err
	}

	return &entity.Series{Slug: slug, From: from, To: to, BucketS: bucketS, Points: points}, nil
}

// Stat reports storage counters for the health endpoint.
func (c *Core) Stat(ctx context.Context) (map[string]any, error) {
	return c.db.Stat(ctx)
}

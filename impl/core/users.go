package core

import (
	"context"
	"errors"
	"log/slog"

	"humi/entity"
	"humi/internal/lib/clock"
)

// Users lists everyone who can sign in.
func (c *Core) Users(ctx context.Context) ([]entity.UserInfo, error) {
	return c.db.Users(ctx, clock.Unix())
}

// UserByName looks a user up by name, for userctl.
func (c *Core) UserByName(ctx context.Context, username string) (*entity.User, error) {
	u, _, err := c.db.UserByUsername(ctx, username)
	if errors.Is(err, entity.ErrNotFound) {
		return nil, entity.Errorf(entity.ErrNotFound, "user %q not found", username)
	}
	return u, err
}

// SetRole changes the role of a user. The last admin cannot be demoted.
func (c *Core) SetRole(ctx context.Context, actor *entity.User, id int64, role string) (*entity.User, error) {
	if err := validateRole(role); err != nil {
		return nil, err
	}
	if err := c.userErr(c.db.SetUserRole(ctx, id, role)); err != nil {
		return nil, err
	}
	u, _, err := c.db.UserByID(ctx, id)
	if err != nil {
		return nil, c.userErr(err)
	}
	c.log.Info("role changed", slog.String("by", actor.Username), slog.String("user", u.Username), slog.String("role", role))
	return u, nil
}

// DeleteUser removes a user and signs them out. Nobody can delete themselves,
// and the last admin stays.
func (c *Core) DeleteUser(ctx context.Context, actor *entity.User, id int64) error {
	if actor.ID == id {
		return entity.Errorf(entity.ErrInvalid, "you cannot delete yourself")
	}
	if err := c.userErr(c.db.DeleteUser(ctx, id)); err != nil {
		return err
	}
	c.log.Info("user deleted", slog.String("by", actor.Username), slog.Int64("user", id))
	return nil
}

// ResetLink issues a single-use link that sets a new password for a user.
// actor is nil when the link comes from userctl.
func (c *Core) ResetLink(ctx context.Context, actor *entity.User, id int64) (*entity.Invite, error) {
	if _, _, err := c.db.UserByID(ctx, id); err != nil {
		return nil, c.userErr(err)
	}
	now := clock.Unix()
	return c.db.CreateInvite(ctx, entity.InviteReset, "", id, actorID(actor), now, now+resetTTL)
}

// Invites lists the links that are still pending.
func (c *Core) Invites(ctx context.Context) ([]entity.Invite, error) {
	return c.db.Invites(ctx, clock.Unix())
}

// CreateInvite issues a join link for a new user with the given role.
// actor is nil when the link comes from userctl.
func (c *Core) CreateInvite(ctx context.Context, actor *entity.User, role string) (*entity.Invite, error) {
	if err := validateRole(role); err != nil {
		return nil, err
	}
	now := clock.Unix()
	return c.db.CreateInvite(ctx, entity.InviteJoin, role, 0, actorID(actor), now, now+joinTTL)
}

// DeleteInvite revokes a pending link.
func (c *Core) DeleteInvite(ctx context.Context, id int64) error {
	err := c.db.DeleteInvite(ctx, id)
	if errors.Is(err, entity.ErrNotFound) {
		return entity.Errorf(entity.ErrNotFound, "invite not found")
	}
	return err
}

// userErr names the user in a lookup failure; the last-admin refusal from the
// database is client-facing already.
func (c *Core) userErr(err error) error {
	if errors.Is(err, entity.ErrNotFound) {
		return entity.Errorf(entity.ErrNotFound, "user not found")
	}
	return err
}

func actorID(u *entity.User) int64 {
	if u == nil {
		return 0
	}
	return u.ID
}

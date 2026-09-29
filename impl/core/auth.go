package core

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"golang.org/x/crypto/bcrypt"

	"humi/entity"
	"humi/internal/lib/clock"
)

const (
	// sessionRefresh throttles the sliding expiry to one write per hour.
	sessionRefresh = 3600
	joinTTL        = 7 * 24 * 3600
	resetTTL       = 24 * 3600
	maxUserAgent   = 200
)

var errBadCredentials = entity.Errorf(entity.ErrUnauthorized, "wrong username or password")

// dummyHash is compared against when the username is unknown, so a failed
// login takes as long whether or not the user exists.
var dummyHash = sync.OnceValue(func() []byte {
	h, _ := bcrypt.GenerateFromPassword([]byte("humi-dummy-password"), bcrypt.DefaultCost)
	return h
})

// Login checks a password and opens a session. It returns the plaintext
// session token for the cookie.
func (c *Core) Login(ctx context.Context, username, password, userAgent string) (*entity.User, string, error) {
	u, hash, err := c.db.UserByUsername(ctx, username)
	if errors.Is(err, entity.ErrNotFound) {
		_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(password))
		return nil, "", errBadCredentials
	}
	if err != nil {
		return nil, "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		c.log.Warn("failed login", slog.String("user", u.Username))
		return nil, "", errBadCredentials
	}
	return c.startSession(ctx, u, userAgent)
}

// Resume resolves a session token. refreshed reports that the expiry slid
// forward and the cookie should be sent again.
func (c *Core) Resume(ctx context.Context, token string) (*entity.Session, *entity.User, bool, error) {
	now := clock.Unix()
	ses, u, err := c.db.SessionByToken(ctx, token, now)
	if errors.Is(err, entity.ErrNotFound) {
		return nil, nil, false, entity.Errorf(entity.ErrUnauthorized, "sign in required")
	}
	if err != nil {
		return nil, nil, false, err
	}
	if now-ses.LastUsed < sessionRefresh {
		return ses, u, false, nil
	}
	if err = c.db.TouchSession(ctx, ses.ID, now, now+entity.SessionTTL); err != nil {
		return nil, nil, false, err
	}
	ses.LastUsed, ses.ExpiresAt = now, now+entity.SessionTTL
	return ses, u, true, nil
}

// Logout revokes the session the request came with.
func (c *Core) Logout(ctx context.Context, ses *entity.Session) error {
	err := c.db.DeleteSession(ctx, ses.UserID, ses.ID)
	if errors.Is(err, entity.ErrNotFound) {
		return nil
	}
	return err
}

// Invite describes a pending link to the page that redeems it, without
// revealing anything but what the form needs.
func (c *Core) Invite(ctx context.Context, token string) (*entity.Invite, error) {
	inv, err := c.db.InviteByToken(ctx, token, clock.Unix())
	if errors.Is(err, entity.ErrNotFound) {
		return nil, errInviteGone
	}
	if err != nil {
		return nil, err
	}
	return &entity.Invite{Kind: inv.Kind, Role: inv.Role, Username: inv.Username, ExpiresAt: inv.ExpiresAt}, nil
}

var errInviteGone = entity.Errorf(entity.ErrNotFound, "invite not found or expired")

// AcceptInvite redeems a link: a join link creates the user, a reset link sets
// a new password and signs the user out everywhere else. Either way the
// caller ends up signed in.
func (c *Core) AcceptInvite(ctx context.Context, token, username, password, userAgent string) (*entity.User, string, error) {
	now := clock.Unix()
	inv, err := c.db.InviteByToken(ctx, token, now)
	if errors.Is(err, entity.ErrNotFound) {
		return nil, "", errInviteGone
	}
	if err != nil {
		return nil, "", err
	}
	if inv.Kind == entity.InviteJoin {
		if err = validateUsername(username); err != nil {
			return nil, "", err
		}
	}
	if err = validatePassword(password); err != nil {
		return nil, "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	var u *entity.User
	if inv.Kind == entity.InviteJoin {
		u, err = c.db.AcceptJoin(ctx, token, username, string(hash), now)
	} else {
		u, err = c.db.AcceptReset(ctx, token, string(hash), now)
	}
	switch {
	case errors.Is(err, entity.ErrConflict):
		return nil, "", entity.Errorf(entity.ErrConflict, "username taken")
	case errors.Is(err, entity.ErrNotFound):
		return nil, "", errInviteGone
	case err != nil:
		return nil, "", err
	}

	c.log.Info("invite accepted", slog.String("kind", inv.Kind), slog.String("user", u.Username))
	return c.startSession(ctx, u, userAgent)
}

// ChangePassword checks the current password, stores the new one and signs
// out every other session of the user.
func (c *Core) ChangePassword(ctx context.Context, ses *entity.Session, current, next string) error {
	_, hash, err := c.db.UserByID(ctx, ses.UserID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(current)) != nil {
		return entity.Errorf(entity.ErrInvalid, "wrong current password")
	}
	if err = validatePassword(next); err != nil {
		return err
	}
	h, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err = c.db.SetPassword(ctx, ses.UserID, string(h)); err != nil {
		return err
	}
	return c.db.DeleteOtherSessions(ctx, ses.UserID, ses.ID)
}

// Sessions lists the signed-in devices of a user, marking the current one.
func (c *Core) Sessions(ctx context.Context, cur *entity.Session) ([]entity.Session, error) {
	list, err := c.db.Sessions(ctx, cur.UserID, clock.Unix())
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].Current = list[i].ID == cur.ID
	}
	return list, nil
}

// RevokeSession signs out one device of the user.
func (c *Core) RevokeSession(ctx context.Context, userID, id int64) error {
	err := c.db.DeleteSession(ctx, userID, id)
	if errors.Is(err, entity.ErrNotFound) {
		return entity.Errorf(entity.ErrNotFound, "session not found")
	}
	return err
}

func (c *Core) startSession(ctx context.Context, u *entity.User, userAgent string) (*entity.User, string, error) {
	if len(userAgent) > maxUserAgent {
		userAgent = userAgent[:maxUserAgent]
	}
	now := clock.Unix()
	_, t, err := c.db.CreateSession(ctx, u.ID, userAgent, now, now+entity.SessionTTL)
	if err != nil {
		return nil, "", err
	}
	return u, t, nil
}

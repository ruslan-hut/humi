package database

import (
	"context"
	"database/sql"
	"errors"

	"humi/entity"
	"humi/internal/lib/token"
)

// CreateUser stores a new user. The username is unique ignoring case.
func (s *SQLite) CreateUser(ctx context.Context, username, passHash, role string) (*entity.User, error) {
	return createUser(ctx, s.db, username, passHash, role)
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func createUser(ctx context.Context, db execer, username, passHash, role string) (*entity.User, error) {
	res, err := db.ExecContext(ctx,
		`INSERT INTO users (username, pass_hash, role, created_at) VALUES (?, ?, ?, unixepoch())`,
		username, passHash, role)
	if err != nil {
		return nil, conflict(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	u, _, err := userBy(ctx, db, `id = ?`, id)
	return u, err
}

// UserByUsername returns a user and its password hash.
func (s *SQLite) UserByUsername(ctx context.Context, username string) (*entity.User, string, error) {
	return userBy(ctx, s.db, `username = ?`, username)
}

// UserByID returns a user and its password hash.
func (s *SQLite) UserByID(ctx context.Context, id int64) (*entity.User, string, error) {
	return userBy(ctx, s.db, `id = ?`, id)
}

func userBy(ctx context.Context, db execer, where string, arg any) (*entity.User, string, error) {
	var u entity.User
	var hash string
	err := db.QueryRowContext(ctx,
		`SELECT id, username, role, created_at, pass_hash FROM users WHERE `+where, arg).
		Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	return &u, hash, nil
}

// Users lists every user with its session activity.
func (s *SQLite) Users(ctx context.Context, now int64) ([]entity.UserInfo, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT u.id, u.username, u.role, u.created_at,
		        COALESCE(MAX(s.last_used), 0),
		        COUNT(s.id) FILTER (WHERE s.expires_at > ?)
		 FROM users u LEFT JOIN sessions s ON s.user_id = u.id
		 GROUP BY u.id ORDER BY u.username`, now)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	users := make([]entity.UserInfo, 0, 4)
	for rows.Next() {
		var u entity.UserInfo
		if err = rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt, &u.LastActive, &u.Sessions); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// SetUserRole changes a role, refusing to demote the last admin.
func (s *SQLite) SetUserRole(ctx context.Context, id int64, role string) error {
	return s.guardAdmins(ctx, id, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE users SET role = ? WHERE id = ?`, role, id)
		return affected(res, err)
	}, role != entity.RoleAdmin)
}

// DeleteUser removes a user with its sessions, refusing to delete the last admin.
func (s *SQLite) DeleteUser(ctx context.Context, id int64) error {
	return s.guardAdmins(ctx, id, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
		return affected(res, err)
	}, true)
}

// guardAdmins runs change in a transaction and rolls it back when it removed
// the last admin. The count happens inside the transaction, so two admins
// demoting each other at once cannot both succeed.
func (s *SQLite) guardAdmins(ctx context.Context, id int64, change func(tx *sql.Tx) error, removes bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err = change(tx); err != nil {
		return err
	}
	if removes {
		var admins int
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role = 'admin'`).Scan(&admins); err != nil {
			return err
		}
		if admins == 0 {
			return ErrLastAdmin
		}
	}
	return tx.Commit()
}

// SetPassword replaces a user's password hash.
func (s *SQLite) SetPassword(ctx context.Context, id int64, passHash string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE users SET pass_hash = ? WHERE id = ?`, passHash, id)
	return affected(res, err)
}

// CreateSession signs a user in and returns the plaintext session token.
// Expired sessions of every user are swept on the way.
func (s *SQLite) CreateSession(ctx context.Context, userID int64, userAgent string, now, expiresAt int64) (*entity.Session, string, error) {
	t, err := token.New()
	if err != nil {
		return nil, "", err
	}
	if _, err = s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, now); err != nil {
		return nil, "", err
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (user_id, token_hash, created_at, last_used, expires_at, user_agent)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		userID, token.Hash(t), now, now, expiresAt, userAgent)
	if err != nil {
		return nil, "", err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, "", err
	}
	return &entity.Session{
		ID: id, UserID: userID, CreatedAt: now, LastUsed: now, ExpiresAt: expiresAt, UserAgent: userAgent,
	}, t, nil
}

// SessionByToken resolves a session token that has not expired to its user.
func (s *SQLite) SessionByToken(ctx context.Context, t string, now int64) (*entity.Session, *entity.User, error) {
	var ses entity.Session
	var u entity.User
	err := s.db.QueryRowContext(ctx,
		`SELECT s.id, s.user_id, s.created_at, s.last_used, s.expires_at, s.user_agent,
		        u.id, u.username, u.role, u.created_at
		 FROM sessions s JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = ? AND s.expires_at > ?`, token.Hash(t), now).
		Scan(&ses.ID, &ses.UserID, &ses.CreatedAt, &ses.LastUsed, &ses.ExpiresAt, &ses.UserAgent,
			&u.ID, &u.Username, &u.Role, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	return &ses, &u, nil
}

// TouchSession records use of a session and slides its expiry forward.
func (s *SQLite) TouchSession(ctx context.Context, id, now, expiresAt int64) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET last_used = ?, expires_at = ? WHERE id = ?`, now, expiresAt, id)
	return affected(res, err)
}

// Sessions lists the live sessions of one user, most recently used first.
func (s *SQLite) Sessions(ctx context.Context, userID, now int64) ([]entity.Session, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, created_at, last_used, expires_at, user_agent FROM sessions
		 WHERE user_id = ? AND expires_at > ? ORDER BY last_used DESC, id DESC`, userID, now)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	sessions := make([]entity.Session, 0, 4)
	for rows.Next() {
		var ses entity.Session
		if err = rows.Scan(&ses.ID, &ses.UserID, &ses.CreatedAt, &ses.LastUsed, &ses.ExpiresAt, &ses.UserAgent); err != nil {
			return nil, err
		}
		sessions = append(sessions, ses)
	}
	return sessions, rows.Err()
}

// DeleteSession revokes one session of a user.
func (s *SQLite) DeleteSession(ctx context.Context, userID, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ? AND user_id = ?`, id, userID)
	return affected(res, err)
}

// DeleteOtherSessions revokes every session of a user except keepID; 0 revokes all.
func (s *SQLite) DeleteOtherSessions(ctx context.Context, userID, keepID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ? AND id != ?`, userID, keepID)
	return err
}

// CreateInvite stores a single-use link and returns it with its plaintext token.
// role applies to join links, userID to reset links, createdBy is 0 for userctl.
func (s *SQLite) CreateInvite(ctx context.Context, kind, role string, userID, createdBy, now, expiresAt int64) (*entity.Invite, error) {
	t, err := token.New()
	if err != nil {
		return nil, err
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO invites (token_hash, kind, role, user_id, created_by, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		token.Hash(t), kind, nullString(role), nullID(userID), nullID(createdBy), now, expiresAt)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	inv, err := inviteBy(ctx, s.db, `i.id = ?`, id, now)
	if err != nil {
		return nil, err
	}
	inv.Token = t
	return inv, nil
}

// InviteByToken returns an invite that is unused and has not expired.
func (s *SQLite) InviteByToken(ctx context.Context, t string, now int64) (*entity.Invite, error) {
	return inviteBy(ctx, s.db, `i.token_hash = ?`, token.Hash(t), now)
}

const inviteColumns = `SELECT i.id, i.kind, COALESCE(i.role, ''), COALESCE(i.user_id, 0),
	        COALESCE(t.username, ''), COALESCE(c.username, ''), i.created_at, i.expires_at
	 FROM invites i
	 LEFT JOIN users t ON t.id = i.user_id
	 LEFT JOIN users c ON c.id = i.created_by`

func inviteBy(ctx context.Context, db execer, where string, arg any, now int64) (*entity.Invite, error) {
	var inv entity.Invite
	err := db.QueryRowContext(ctx,
		inviteColumns+` WHERE `+where+` AND i.used_at IS NULL AND i.expires_at > ?`, arg, now).
		Scan(&inv.ID, &inv.Kind, &inv.Role, &inv.UserID, &inv.Username, &inv.CreatedBy, &inv.CreatedAt, &inv.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// Invites lists pending invites, newest first. Used and expired ones are swept.
func (s *SQLite) Invites(ctx context.Context, now int64) ([]entity.Invite, error) {
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM invites WHERE used_at IS NOT NULL OR expires_at <= ?`, now); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, inviteColumns+` ORDER BY i.created_at DESC, i.id DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	invites := make([]entity.Invite, 0, 4)
	for rows.Next() {
		var inv entity.Invite
		if err = rows.Scan(&inv.ID, &inv.Kind, &inv.Role, &inv.UserID, &inv.Username, &inv.CreatedBy,
			&inv.CreatedAt, &inv.ExpiresAt); err != nil {
			return nil, err
		}
		invites = append(invites, inv)
	}
	return invites, rows.Err()
}

// DeleteInvite revokes a pending invite.
func (s *SQLite) DeleteInvite(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM invites WHERE id = ?`, id)
	return affected(res, err)
}

// AcceptJoin redeems a join link: creates the user and burns the link in one
// transaction, so a link cannot be used twice.
func (s *SQLite) AcceptJoin(ctx context.Context, t, username, passHash string, now int64) (*entity.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	inv, err := inviteBy(ctx, tx, `i.token_hash = ? AND i.kind = 'join'`, token.Hash(t), now)
	if err != nil {
		return nil, err
	}
	u, err := createUser(ctx, tx, username, passHash, inv.Role)
	if err != nil {
		return nil, err
	}
	if err = useInvite(ctx, tx, inv.ID, now); err != nil {
		return nil, err
	}
	return u, tx.Commit()
}

// AcceptReset redeems a reset link: sets the password, signs the user out
// everywhere and burns the link.
func (s *SQLite) AcceptReset(ctx context.Context, t, passHash string, now int64) (*entity.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	inv, err := inviteBy(ctx, tx, `i.token_hash = ? AND i.kind = 'reset'`, token.Hash(t), now)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE users SET pass_hash = ? WHERE id = ?`, passHash, inv.UserID); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, inv.UserID); err != nil {
		return nil, err
	}
	if err = useInvite(ctx, tx, inv.ID, now); err != nil {
		return nil, err
	}
	u, _, err := userBy(ctx, tx, `id = ?`, inv.UserID)
	if err != nil {
		return nil, err
	}
	return u, tx.Commit()
}

func useInvite(ctx context.Context, tx *sql.Tx, id, now int64) error {
	res, err := tx.ExecContext(ctx, `UPDATE invites SET used_at = ? WHERE id = ? AND used_at IS NULL`, now, id)
	return affected(res, err)
}

func nullString(v string) sql.NullString {
	return sql.NullString{String: v, Valid: v != ""}
}

func nullID(v int64) sql.NullInt64 {
	return sql.NullInt64{Int64: v, Valid: v != 0}
}

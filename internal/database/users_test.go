package database

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"humi/entity"
)

func mustUser(t *testing.T, db *SQLite, name, role string) *entity.User {
	t.Helper()
	u, err := db.CreateUser(context.Background(), name, "hash", role)
	if err != nil {
		t.Fatalf("create user %s: %v", name, err)
	}
	return u
}

func TestCreateUserUniqueIgnoringCase(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	db := newTestDB(t)
	mustUser(t, db, "Ruslan", entity.RoleAdmin)

	if _, err := db.CreateUser(context.Background(), "ruslan", "hash", entity.RoleViewer); !errors.Is(err, ErrConflict) {
		t.Fatalf("err = %v, want ErrConflict", err)
	}
	u, _, err := db.UserByUsername(context.Background(), "RUSLAN")
	if err != nil || u.Username != "Ruslan" {
		t.Fatalf("lookup ignoring case = %+v, %v", u, err)
	}
}

func TestLastAdminIsKept(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	db := newTestDB(t)
	admin := mustUser(t, db, "admin", entity.RoleAdmin)
	viewer := mustUser(t, db, "viewer", entity.RoleViewer)

	if err := db.SetUserRole(ctx, admin.ID, entity.RoleViewer); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("demote last admin: err = %v, want ErrLastAdmin", err)
	}
	if err := db.DeleteUser(ctx, admin.ID); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("delete last admin: err = %v, want ErrLastAdmin", err)
	}
	if u, _, _ := db.UserByID(ctx, admin.ID); u == nil || u.Role != entity.RoleAdmin {
		t.Fatalf("refused change was not rolled back: %+v", u)
	}

	if err := db.SetUserRole(ctx, viewer.ID, entity.RoleAdmin); err != nil {
		t.Fatalf("promote: %v", err)
	}
	if err := db.SetUserRole(ctx, admin.ID, entity.RoleViewer); err != nil {
		t.Fatalf("demote with another admin left: %v", err)
	}
	if err := db.DeleteUser(ctx, admin.ID); err != nil {
		t.Fatalf("delete viewer: %v", err)
	}
	if err := db.DeleteUser(ctx, admin.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete twice: err = %v, want ErrNotFound", err)
	}
}

func TestSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	db := newTestDB(t)
	u := mustUser(t, db, "ruslan", entity.RoleAdmin)
	other := mustUser(t, db, "guest", entity.RoleViewer)

	phone, phoneToken, err := db.CreateSession(ctx, u.ID, "iPhone", 1_000, 5_000)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	_, laptopToken, err := db.CreateSession(ctx, u.ID, "Mac", 1_100, 5_000)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	ses, got, err := db.SessionByToken(ctx, phoneToken, 2_000)
	if err != nil || got.ID != u.ID || ses.ID != phone.ID {
		t.Fatalf("resolve = %+v %+v, %v", ses, got, err)
	}
	if _, _, err = db.SessionByToken(ctx, phoneToken, 5_000); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired session: err = %v, want ErrNotFound", err)
	}

	if err = db.TouchSession(ctx, phone.ID, 4_000, 9_000); err != nil {
		t.Fatalf("touch: %v", err)
	}
	if _, _, err = db.SessionByToken(ctx, phoneToken, 6_000); err != nil {
		t.Fatalf("touched session expired: %v", err)
	}

	list, err := db.Sessions(ctx, u.ID, 2_000)
	if err != nil || len(list) != 2 || list[0].ID != phone.ID {
		t.Fatalf("sessions = %+v, %v; want phone first", list, err)
	}

	if err = db.DeleteSession(ctx, other.ID, phone.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revoking someone else's session: err = %v, want ErrNotFound", err)
	}
	if err = db.DeleteOtherSessions(ctx, u.ID, phone.ID); err != nil {
		t.Fatalf("delete others: %v", err)
	}
	if _, _, err = db.SessionByToken(ctx, laptopToken, 2_000); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other session survived: %v", err)
	}
	if _, _, err = db.SessionByToken(ctx, phoneToken, 2_000); err != nil {
		t.Fatalf("kept session gone: %v", err)
	}
}

func TestJoinInvite(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	db := newTestDB(t)
	admin := mustUser(t, db, "admin", entity.RoleAdmin)

	inv, err := db.CreateInvite(ctx, entity.InviteJoin, entity.RoleViewer, 0, admin.ID, 1_000, 2_000)
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}
	if inv.Token == "" || inv.CreatedBy != "admin" || inv.Role != entity.RoleViewer {
		t.Fatalf("invite = %+v", inv)
	}

	if _, err = db.InviteByToken(ctx, inv.Token, 2_000); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired invite: err = %v, want ErrNotFound", err)
	}

	// A taken username rolls back and leaves the link usable.
	if _, err = db.AcceptJoin(ctx, inv.Token, "ADMIN", "hash", 1_500); !errors.Is(err, ErrConflict) {
		t.Fatalf("taken username: err = %v, want ErrConflict", err)
	}
	u, err := db.AcceptJoin(ctx, inv.Token, "guest", "hash", 1_500)
	if err != nil || u.Role != entity.RoleViewer {
		t.Fatalf("accept = %+v, %v", u, err)
	}
	if _, err = db.AcceptJoin(ctx, inv.Token, "guest2", "hash", 1_500); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second use: err = %v, want ErrNotFound", err)
	}

	pending, err := db.Invites(ctx, 1_500)
	if err != nil || len(pending) != 0 {
		t.Fatalf("pending = %+v, %v; want none", pending, err)
	}
}

func TestResetInvite(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	db := newTestDB(t)
	u := mustUser(t, db, "ruslan", entity.RoleAdmin)
	_, sesToken, err := db.CreateSession(ctx, u.ID, "", 1_000, 9_000)
	if err != nil {
		t.Fatalf("session: %v", err)
	}

	inv, err := db.CreateInvite(ctx, entity.InviteReset, "", u.ID, 0, 1_000, 2_000)
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}
	if got, e := db.InviteByToken(ctx, inv.Token, 1_500); e != nil || got.Username != "ruslan" {
		t.Fatalf("lookup = %+v, %v", got, e)
	}
	// A reset link cannot be redeemed as a join link.
	if _, err = db.AcceptJoin(ctx, inv.Token, "mallory", "hash", 1_500); !errors.Is(err, ErrNotFound) {
		t.Fatalf("reset as join: err = %v, want ErrNotFound", err)
	}

	if _, err = db.AcceptReset(ctx, inv.Token, "new-hash", 1_500); err != nil {
		t.Fatalf("accept reset: %v", err)
	}
	if _, hash, _ := db.UserByID(ctx, u.ID); hash != "new-hash" {
		t.Fatalf("hash = %q, want new-hash", hash)
	}
	if _, _, err = db.SessionByToken(ctx, sesToken, 1_600); !errors.Is(err, ErrNotFound) {
		t.Fatalf("session survived a reset: %v", err)
	}
}

func TestNodeRules(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	db := newTestDB(t)

	if _, err := db.CreateNode(ctx, "bath", "Bath", "", 900); err != nil {
		t.Fatalf("create node: %v", err)
	}
	if _, err := db.CreateNode(ctx, "bath", "Bath again", "", 900); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate slug: err = %v, want ErrConflict", err)
	}
	node, _ := db.NodeBySlug(ctx, "bath")

	rules, err := db.Rules(ctx, node.ID)
	if err != nil || len(rules) != len(defaultRules) {
		t.Fatalf("seeded %d rules (%v), want %d", len(rules), err, len(defaultRules))
	}
	states, _ := db.Nodes(ctx)
	if low, high := states[0].RHLow, states[0].RHHigh; low == nil || *low != 30 || high == nil || *high != 65 {
		t.Fatalf("bands = %v / %v, want 30 / 65", low, high)
	}

	err = db.ReplaceRules(ctx, node.ID, []entity.Rule{
		{Metric: entity.MetricRH, Op: entity.OpGT, Threshold: 70, ForMin: 30, Enabled: false},
		{Metric: entity.MetricRH, Op: entity.OpLT, Threshold: 35, ForMin: 30, Enabled: true},
	})
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	states, _ = db.Nodes(ctx)
	if low, high := states[0].RHLow, states[0].RHHigh; low == nil || *low != 35 || high != nil {
		t.Fatalf("bands = %v / %v, want 35 / none (disabled)", low, high)
	}

	dup := []entity.Rule{
		{Metric: entity.MetricRH, Op: entity.OpGT, Threshold: 70},
		{Metric: entity.MetricRH, Op: entity.OpGT, Threshold: 75},
	}
	if err = db.ReplaceRules(ctx, node.ID, dup); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate rule: err = %v, want ErrConflict", err)
	}
	if rules, _ = db.Rules(ctx, node.ID); len(rules) != 2 {
		t.Fatalf("failed replace was not rolled back: %d rules", len(rules))
	}
}

func TestNodeTokenAndDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	db := newTestDB(t)

	old, err := db.CreateNode(ctx, "hall", "Hall", "", 900)
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	node, _ := db.NodeBySlug(ctx, "hall")

	fresh, err := db.RotateNodeToken(ctx, node.ID)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if _, err = db.NodeByToken(ctx, old); !errors.Is(err, ErrNotFound) {
		t.Fatalf("old token still works: %v", err)
	}
	if _, err = db.NodeByToken(ctx, fresh); err != nil {
		t.Fatalf("new token rejected: %v", err)
	}

	node.Enabled = false
	if err = db.UpdateNode(ctx, node); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if _, err = db.NodeByToken(ctx, fresh); !errors.Is(err, ErrNotFound) {
		t.Fatalf("disabled node accepted: %v", err)
	}

	if err = db.SaveReadings(ctx, node.ID, []entity.Reading{{ReceivedAt: 1, RH: 50}}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err = db.DeleteNode(ctx, node.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	stat, _ := db.Stat(ctx)
	if stat["readings"] != int64(0) {
		t.Fatalf("readings left after delete: %v", stat["readings"])
	}
}

func TestMigrationSeedsRulesForExistingNodes(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "old.db")
	db, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err = db.CreateNode(ctx, "cellar", "Cellar", "", 900); err != nil {
		t.Fatalf("create node: %v", err)
	}

	// Roll the file back to how 0001 left it: a node without rules.
	for _, q := range []string{
		`DELETE FROM rules`,
		`DROP TABLE invites`, `DROP TABLE sessions`, `DROP TABLE users`,
		`DROP INDEX idx_rules_node_metric`,
		`DELETE FROM schema_migrations WHERE name = '0002_users.sql'`,
	} {
		if _, err = db.db.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	_ = db.Close()

	db, err = NewSQLite(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = db.Close() }()

	node, _ := db.NodeBySlug(ctx, "cellar")
	rules, err := db.Rules(ctx, node.ID)
	if err != nil || len(rules) != len(defaultRules) {
		t.Fatalf("migration seeded %d rules (%v), want %d", len(rules), err, len(defaultRules))
	}
	for i, r := range rules {
		want := findRule(defaultRules, r.Metric, r.Op)
		if want == nil || want.Threshold != r.Threshold || want.ForMin != r.ForMin || want.Enabled != r.Enabled {
			t.Errorf("rule %d = %+v, differs from the Go defaults", i, r)
		}
	}
}

func findRule(rules []entity.Rule, metric, op string) *entity.Rule {
	for i := range rules {
		if rules[i].Metric == metric && rules[i].Op == op {
			return &rules[i]
		}
	}
	return nil
}

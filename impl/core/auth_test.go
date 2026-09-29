package core

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"humi/entity"
	"humi/internal/database"
)

// realCore runs on a SQLite file: the auth paths are mostly storage, and a
// stub would only test itself.
func realCore(t *testing.T) *Core {
	t.Helper()
	db, err := database.NewSQLite(filepath.Join(t.TempDir(), "core.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return testCore(db)
}

// join creates a user through an invite link, the only way users are made.
func join(t *testing.T, c *Core, role, username, password string) (*entity.User, string) {
	t.Helper()
	ctx := context.Background()
	inv, err := c.CreateInvite(ctx, nil, role)
	if err != nil {
		t.Fatalf("invite: %v", err)
	}
	u, token, err := c.AcceptInvite(ctx, inv.Token, username, password, "test")
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	return u, token
}

func TestLogin(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	c := realCore(t)
	join(t, c, entity.RoleAdmin, "ruslan", "correct horse")

	tests := []struct {
		name, user, pass string
		ok               bool
	}{
		{"right password", "ruslan", "correct horse", true},
		{"username ignores case", "Ruslan", "correct horse", true},
		{"wrong password", "ruslan", "wrong horse", false},
		{"unknown user", "nobody", "correct horse", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, token, err := c.Login(ctx, tt.user, tt.pass, "ua")
			if tt.ok {
				if err != nil || u.Username != "ruslan" || token == "" {
					t.Fatalf("login = %+v %q, %v", u, token, err)
				}
				return
			}
			// Unknown user and wrong password must be indistinguishable.
			if err == nil || err.Error() != "wrong username or password" || !errors.Is(err, entity.ErrUnauthorized) {
				t.Fatalf("err = %v, want the generic credentials error", err)
			}
		})
	}
}

func TestResumeAndLogout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	c := realCore(t)
	_, token := join(t, c, entity.RoleAdmin, "ruslan", "correct horse")

	ses, u, refreshed, err := c.Resume(ctx, token)
	if err != nil || u.Username != "ruslan" || refreshed {
		t.Fatalf("resume = %+v refreshed %v, %v", u, refreshed, err)
	}
	if err = c.Logout(ctx, ses); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, _, _, err = c.Resume(ctx, token); !errors.Is(err, entity.ErrUnauthorized) {
		t.Fatalf("resume after logout: err = %v, want ErrUnauthorized", err)
	}
}

func TestChangePasswordSignsOutOtherDevices(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	c := realCore(t)
	_, phone := join(t, c, entity.RoleAdmin, "ruslan", "correct horse")
	_, laptop, err := c.Login(ctx, "ruslan", "correct horse", "laptop")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	ses, _, _, _ := c.Resume(ctx, phone)

	if err = c.ChangePassword(ctx, ses, "wrong", "battery staple"); !errors.Is(err, entity.ErrInvalid) {
		t.Fatalf("wrong current: err = %v, want ErrInvalid", err)
	}
	if err = c.ChangePassword(ctx, ses, "correct horse", "short"); !errors.Is(err, entity.ErrInvalid) {
		t.Fatalf("short new: err = %v, want ErrInvalid", err)
	}
	if err = c.ChangePassword(ctx, ses, "correct horse", "battery staple"); err != nil {
		t.Fatalf("change: %v", err)
	}

	if _, _, _, err = c.Resume(ctx, laptop); err == nil {
		t.Fatal("the other device is still signed in")
	}
	if _, _, _, err = c.Resume(ctx, phone); err != nil {
		t.Fatalf("the current device was signed out: %v", err)
	}
	if _, _, err = c.Login(ctx, "ruslan", "battery staple", ""); err != nil {
		t.Fatalf("login with new password: %v", err)
	}
}

func TestAcceptInvite(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	c := realCore(t)
	admin, _ := join(t, c, entity.RoleAdmin, "ruslan", "correct horse")

	inv, _ := c.CreateInvite(ctx, admin, entity.RoleViewer)
	preview, err := c.Invite(ctx, inv.Token)
	if err != nil || preview.Kind != entity.InviteJoin || preview.Role != entity.RoleViewer || preview.ID != 0 {
		t.Fatalf("preview = %+v, %v", preview, err)
	}

	if _, _, err = c.AcceptInvite(ctx, inv.Token, "x", "correct horse", ""); !errors.Is(err, entity.ErrInvalid) {
		t.Fatalf("bad username: err = %v, want ErrInvalid", err)
	}
	if _, _, err = c.AcceptInvite(ctx, inv.Token, "RUSLAN", "correct horse", ""); err == nil || err.Error() != "username taken" {
		t.Fatalf("taken username: err = %v, want username taken", err)
	}
	if _, _, err = c.AcceptInvite(ctx, inv.Token, "guest", "guest password", ""); err != nil {
		t.Fatalf("accept: %v", err)
	}
	if _, err = c.Invite(ctx, inv.Token); err == nil || err.Error() != "invite not found or expired" {
		t.Fatalf("used invite: err = %v", err)
	}

	// A reset link needs no username and keeps the account.
	reset, err := c.ResetLink(ctx, admin, admin.ID)
	if err != nil {
		t.Fatalf("reset link: %v", err)
	}
	u, _, err := c.AcceptInvite(ctx, reset.Token, "", "new password", "")
	if err != nil || u.ID != admin.ID {
		t.Fatalf("reset = %+v, %v", u, err)
	}
	if _, _, err = c.Login(ctx, "ruslan", "new password", ""); err != nil {
		t.Fatalf("login after reset: %v", err)
	}
}

func TestUserAdministration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	c := realCore(t)
	admin, _ := join(t, c, entity.RoleAdmin, "ruslan", "correct horse")
	guest, _ := join(t, c, entity.RoleViewer, "guest", "guest password")

	if err := c.DeleteUser(ctx, admin, admin.ID); !errors.Is(err, entity.ErrInvalid) {
		t.Fatalf("delete self: err = %v, want ErrInvalid", err)
	}
	if _, err := c.SetRole(ctx, admin, admin.ID, entity.RoleViewer); !errors.Is(err, entity.ErrConflict) {
		t.Fatalf("demote last admin: err = %v, want ErrConflict", err)
	}
	if _, err := c.SetRole(ctx, admin, 999, entity.RoleViewer); !errors.Is(err, entity.ErrNotFound) {
		t.Fatalf("unknown user: err = %v, want ErrNotFound", err)
	}

	list, err := c.Users(ctx)
	if err != nil || len(list) != 2 || list[1].Sessions != 1 || list[1].LastActive == 0 {
		t.Fatalf("users = %+v, %v", list, err)
	}
	if err = c.DeleteUser(ctx, admin, guest.ID); err != nil {
		t.Fatalf("delete guest: %v", err)
	}
}

func TestNodeSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	c := realCore(t)

	if _, _, err := c.CreateNode(ctx, entity.NodeInput{Slug: "Bath", Name: "Bath"}); !errors.Is(err, entity.ErrInvalid) {
		t.Fatalf("bad slug: err = %v, want ErrInvalid", err)
	}
	node, token, err := c.CreateNode(ctx, entity.NodeInput{Slug: "bath", Name: " Bath ", IntervalS: 0})
	if err != nil || token == "" || node.Name != "Bath" || node.IntervalS != 900 {
		t.Fatalf("create = %+v %q, %v", node, token, err)
	}
	if _, _, err = c.CreateNode(ctx, entity.NodeInput{Slug: "bath", Name: "Bath"}); err == nil || err.Error() != "slug taken" {
		t.Fatalf("duplicate: err = %v, want slug taken", err)
	}

	interval, off := 300, false
	node, err = c.UpdateNode(ctx, "bath", entity.NodePatch{IntervalS: &interval, Enabled: &off})
	if err != nil || node.IntervalS != 300 || node.Enabled || node.Name != "Bath" {
		t.Fatalf("patch = %+v, %v", node, err)
	}
	if _, err = c.UpdateNode(ctx, "nope", entity.NodePatch{}); !errors.Is(err, entity.ErrNotFound) {
		t.Fatalf("unknown node: err = %v, want ErrNotFound", err)
	}

	rules, err := c.ReplaceRules(ctx, "bath", []entity.Rule{{Metric: "rh", Op: "gt", Threshold: 70, ForMin: 30, Enabled: true}})
	if err != nil || len(rules) != 1 || rules[0].NodeSlug != "bath" || rules[0].ID == 0 {
		t.Fatalf("rules = %+v, %v", rules, err)
	}
	if err = c.DeleteNode(ctx, "bath"); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

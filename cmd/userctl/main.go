// Command userctl bootstraps and rescues dashboard access: it prints single-use
// links that create a user or reset a password. The first admin comes from here.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"humi/entity"
	"humi/impl/core"
	"humi/internal/config"
	"humi/internal/database"
)

func main() {
	configPath := flag.String("conf", "config.yml", "path to config file")
	invite := flag.String("invite", "", "print a join link for a new user with this role: admin | viewer")
	reset := flag.String("reset", "", "print a password reset link for this username")
	list := flag.Bool("list", false, "list users")
	flag.Parse()

	modes := 0
	for _, set := range []bool{*invite != "", *reset != "", *list} {
		if set {
			modes++
		}
	}
	if modes != 1 {
		fmt.Fprintln(os.Stderr, "usage: userctl -invite admin | -reset <username> | -list")
		os.Exit(2)
	}

	conf := config.MustLoad(*configPath)

	db, err := database.NewSQLite(conf.Database.Path)
	if err != nil {
		fail("open database", err)
	}
	defer func() { _ = db.Close() }()

	c := core.New(db, conf, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := context.Background()

	switch {
	case *list:
		users, e := c.Users(ctx)
		if e != nil {
			fail("list users", e)
		}
		for _, u := range users {
			fmt.Printf("%-4d %-24s %-7s sessions %d\n", u.ID, u.Username, u.Role, u.Sessions)
		}

	case *invite != "":
		inv, e := c.CreateInvite(ctx, nil, *invite)
		if e != nil {
			fail("create invite", e)
		}
		printLink(conf, inv, fmt.Sprintf("join link for a new %s", inv.Role))

	default:
		u, e := c.UserByName(ctx, *reset)
		if e != nil {
			fail("reset", e)
		}
		inv, e := c.ResetLink(ctx, nil, u.ID)
		if e != nil {
			fail("reset", e)
		}
		printLink(conf, inv, fmt.Sprintf("password reset link for %s", u.Username))
	}
}

func printLink(conf *config.Config, inv *entity.Invite, what string) {
	base := strings.TrimRight(conf.Web.BaseURL, "/")
	if base == "" {
		base = "http://localhost:4200"
	}
	fmt.Printf("%s, single use, valid until %s:\n\n%s/join/%s\n",
		what, time.Unix(inv.ExpiresAt, 0).Format("2 Jan 15:04"), base, inv.Token)
}

func fail(what string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", what, err)
	os.Exit(1)
}

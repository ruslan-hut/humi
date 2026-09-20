// Command nodectl registers measuring nodes and prints their ingest token.
// The token is shown once; only its hash is stored.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"humi/internal/config"
	"humi/internal/database"
)

func main() {
	configPath := flag.String("conf", "config.yml", "path to config file")
	slug := flag.String("slug", "", "node slug, e.g. bedroom")
	name := flag.String("name", "", "display name")
	location := flag.String("location", "", "optional free-form location")
	interval := flag.Int("interval", 900, "expected reporting interval in seconds")
	flag.Parse()

	if *slug == "" || *name == "" {
		fmt.Fprintln(os.Stderr, "usage: nodectl -slug bedroom -name Bedroom [-interval 900]")
		os.Exit(2)
	}

	conf := config.MustLoad(*configPath)

	db, err := database.NewSQLite(conf.Database.Path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open database:", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	token, err := db.CreateNode(context.Background(), *slug, *name, *location, *interval)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create node:", err)
		os.Exit(1)
	}

	fmt.Printf("node %q created\ntoken: %s\n\nflash this into the node, it is not recoverable\n", *slug, token)
}

package main

import (
	"flag"
	"log/slog"

	"humi/impl/core"
	"humi/internal/config"
	"humi/internal/database"
	"humi/internal/http-server/api"
	"humi/internal/lib/logger"
	"humi/internal/lib/sl"
)

func main() {
	configPath := flag.String("conf", "config.yml", "path to config file")
	logPath := flag.String("log", "/var/log/humi", "path to log file directory")
	flag.Parse()

	conf := config.MustLoad(*configPath)
	lg := logger.SetupLogger(conf.Env, *logPath)

	lg.Info("starting humi", slog.String("config", *configPath), slog.String("env", conf.Env))

	db, err := database.NewSQLite(conf.Database.Path)
	if err != nil {
		lg.Error("sqlite open", sl.Err(err))
		return
	}
	defer func() {
		if e := db.Close(); e != nil {
			lg.Error("sqlite close", sl.Err(e))
		}
	}()
	lg.Info("database ready", slog.String("path", conf.Database.Path))

	handler := core.New(db, conf, lg)

	// *** blocking start with http server ***
	if err = api.New(conf, lg, handler); err != nil {
		lg.Error("server start", sl.Err(err))
		return
	}
	lg.Error("service stopped")
}

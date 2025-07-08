package main

import (
	"context"
	"fileserver/internal/app"
	"fileserver/internal/config"
	"fileserver/internal/http/server"
	"log/slog"
	"os"
)

const (
	envDev   = "dev"
	envProd  = "prod"
	envLocal = "local"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("starting application", "env", cfg.Env)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app, err := app.NewApp(ctx, log, cfg.DB, cfg.Cache, cfg.FileStorage, cfg.AdminToken)
	if err != nil {
		log.Error("failed to init app", slog.String("error", err.Error()))
		os.Exit(1)
	}

	err = server.StartServer(ctx, &cfg.HTTPServer, log, app.DocumentService, app.AuthService, app.UserService)
	if err != nil {
		log.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	return log
}

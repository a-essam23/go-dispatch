package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/a-essam23/go-dispatch/internal/engine"
	"github.com/a-essam23/go-dispatch/internal/server"
	"github.com/a-essam23/go-dispatch/pkg/config"
	"github.com/a-essam23/go-dispatch/pkg/logging"
)

func main() {
	logger := logging.New(logging.LevelDebug)
	slog.SetDefault(logger)
	engine.RegisterCoreActions()
	logger.Info("Action engine initialized.")

	cfg, err := config.Load(logger, "config", engine.GetActionFunc)
	if err != nil {
		logger.Error("Failed to load configuration", slog.Any("error", err))
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background())
	defer stop()

	app := server.NewApp(logger, ctx, cfg)
	if err := app.Run(); err != nil {
		logger.Error("Application run failed", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("Application shut down successfully.")
}

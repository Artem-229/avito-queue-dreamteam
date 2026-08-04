package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"avito-queue/internal/app"
	"avito-queue/internal/config"
)

func main() {
	conf, err := config.ReadConfig()
	if err != nil {
		slog.Error("could not read config", "error", err.Error())
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	application, err := app.New(ctx, conf)
	if err != nil {
		slog.Error("could not initialize app", "error", err.Error())
		os.Exit(1)
	}

	if err := application.Run(ctx); err != nil {
		slog.Error("app stopped with error", "error", err.Error())
		os.Exit(1)
	}
}

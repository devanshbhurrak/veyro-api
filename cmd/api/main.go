// Command api runs the Veyro HTTP API.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"backend/internal/app"
	"backend/internal/platform/config"
	"backend/internal/platform/logger"
	"backend/internal/server"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "api: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	application, err := app.New(cfg)
	if err != nil {
		return err
	}
	defer application.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := server.New(fmt.Sprintf(":%d", cfg.App.Port), application.Handler, application.Log,
		cfg.HTTP.ReadTimeout, cfg.HTTP.WriteTimeout, cfg.HTTP.IdleTimeout, cfg.HTTP.ShutdownTimeout)
	application.Log.Info("starting api", "environment", cfg.App.Environment, "port", cfg.App.Port,
		"log_level", cfg.App.LogLevel)
	if err := srv.Run(ctx); err != nil {
		logger.FromContext(ctx).Error("http server stopped with error", "error", err.Error())
		return err
	}
	application.Log.Info("api stopped")
	return nil
}

// Command worker runs asynchronous background processing (outbox email
// delivery). It is a separate process built from the same codebase.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"backend/internal/app"
	"backend/internal/platform/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "worker: %v\n", err)
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

	application.Log.Info("starting worker", "environment", cfg.App.Environment,
		"outbox_interval", cfg.Outbox.WorkerInterval.String())
	application.Outbox.Run(ctx, cfg.Outbox.WorkerInterval)
	application.Log.Info("worker stopped")
	return nil
}

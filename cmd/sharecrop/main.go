package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/e6qu/sharecrop/internal/app"
	"github.com/e6qu/sharecrop/internal/db"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/web"
)

func main() {
	os.Exit(run(context.Background(), os.Args, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	cfg := app.LoadConfig()
	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{}))

	if len(args) > 1 {
		switch args[1] {
		case "migrate":
			return runMigrate(ctx, args[2:], cfg, stdout, logger)
		case "serve":
			return runServe(ctx, cfg, logger)
		default:
			_, _ = fmt.Fprintf(stderr, "unknown command: %s\n", args[1])
			return 2
		}
	}

	return runServe(ctx, cfg, logger)
}

func runMigrate(ctx context.Context, args []string, cfg app.Config, stdout io.Writer, logger *slog.Logger) int {
	if len(args) != 1 || args[0] != "up" {
		_, _ = fmt.Fprintln(stdout, "usage: sharecrop migrate up")
		return 2
	}

	pool, err := db.Open(ctx, cfg.DatabaseURL())
	if err != nil {
		logger.Error("open database", "error", err)
		return 1
	}
	defer pool.Close()

	if err := db.MigrateUp(ctx, pool, cfg.MigrationsDir()); err != nil {
		logger.Error("run migrations", "error", err)
		return 1
	}

	_, _ = fmt.Fprintln(stdout, "migrations applied")
	return 0
}

func runServe(ctx context.Context, cfg app.Config, logger *slog.Logger) int {
	staticFiles, err := web.StaticFiles()
	if err != nil {
		logger.Error("load static files", "error", err)
		return 1
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddress(),
		Handler:           httpserver.New(staticFiles),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errs := make(chan error, 1)
	go func() {
		logger.Info("starting sharecrop", "address", cfg.HTTPAddress())
		errs <- server.ListenAndServe()
	}()

	stopCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-stopCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown server", "error", err)
			return 1
		}
		return 0
	case err := <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			return 0
		}
		logger.Error("serve", "error", err)
		return 1
	}
}

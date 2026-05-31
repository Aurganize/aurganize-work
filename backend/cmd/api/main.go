package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/config"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/logger"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage"
)

func main() {
	// All initialisation runs through a single function that returns an exit
	// code. This lets defers fire before os.Exit, which is otherwise impossible.
	os.Exit(run())
}

func run() int {
	// === Load Config ===
	cfg, err := config.Load()
	if err != nil {
		code, _ := os.Stderr.WriteString("config: " + err.Error() + "\n")
		return code
	}

	// === Build Logger ===
	logger := logger.New(cfg.LogLevel, cfg.LogFormat, os.Stdout)
	slog.SetDefault(logger) // ensure third-party libs using slog see our config

	logger.Info("starting server",
		slog.String("env", cfg.AppEnv),
		slog.String("port", cfg.AppPort))

	// === App pool (aurganize_app — RLS-respecting) ===
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	appPool, err := storage.NewPool(ctx, cfg.DatabaseUrl)
	cancel()
	if err != nil {
		logger.Error("failed to connect to app user database.",
			slog.Any("err", err))
		return 1
	}

	defer appPool.Close()
	logger.Info("app database connected", slog.Int("max_conns", 25))

	// === Auth pool (aurganize_auth — BYPASSRLS, read-only) ===
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	authPool, err := storage.NewPool(ctx, cfg.DatabaseAuthUrl)
	cancel()
	if err != nil {
		logger.Error("failed to connect to auth user database.",
			slog.Any("err", err))
		return 1
	}

	defer appPool.Close()
	logger.Info("auth database connected", slog.Int("max_conns", 4))

	// === Build the HTTP router (gin Router) ===
	router := buildRouter(cfg, logger, appPool, authPool)

	// === Start the HTTP server ===
	// We build the server configuration
	srv := http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	// Run the server in a goroutine so main() can wait on a shutdown signal
	serverErrors := make(chan error, 1) // server error channel
	go func() {
		logger.Info("listening", slog.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// === Wait for shutdown signal or server error ===
	shutdown := make(chan os.Signal, 1) // shutdown signal channel
	// We register to get - system terminate signal to be send to shutdown signal
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logger.Error("server crashed", slog.Any("err", err))
	case sig := <-shutdown:
		logger.Info("shutdown signal recieved", slog.String("signal", sig.String()))
	}

	// === Graceful Shutdown ===
	// Give in-flight requests up to 30s to finish before forcing the
	// server to close.
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("forced shutdown", slog.Any("err", err))
		_ = srv.Close()
		return 1
	}

	logger.Info("server stopped cleanly")

	return 0
}

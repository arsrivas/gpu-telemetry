package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"gpu-telemetry/config"
	_ "gpu-telemetry/docs" // swag generated
	"gpu-telemetry/pkg/api"
	"gpu-telemetry/pkg/storage"
	"gpu-telemetry/pkg/util/logger"

	"go.uber.org/zap"
)

// @title GPU Telemetry API
// @version 1.0
// @description REST API for querying GPU telemetry
// @BasePath /
// @schemes http
func main() {
	cfg, err := config.LoadAPIConfig()
	if err != nil {
		log.Fatal(err)
	}

	logg, err := logger.NewLogger(cfg.LogLevel)
	if err != nil {
		log.Fatal(err)
	}
	defer logg.Sync()

	store, err := storage.NewPostgres(cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("failed to connect to postgres URL %s with error %v", cfg.PostgresDSN, err)
	}
	defer store.Close()

	h := api.NewHandler(store, logg)

	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: api.Router(h),
	}

	// Context cancelled on SIGINT / SIGTERM
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	// Start HTTP server
	go func() {
		logg.Info("API server starting",
			zap.String("port", cfg.ServerPort),
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logg.Fatal("server failed",
				zap.Error(err),
			)
		}
	}()

	// Block until signal received
	<-ctx.Done()
	logg.Info("shutdown signal received")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logg.Error("graceful shutdown failed",
			zap.Error(err),
		)
	} else {
		logg.Info("server shut down gracefully")
	}
}

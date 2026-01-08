package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"gpu-telemetry/config"
	"gpu-telemetry/pkg/mq"
	mqhttp "gpu-telemetry/pkg/transport/mqhttp"
	"gpu-telemetry/pkg/util/logger"

	"go.uber.org/zap"
)

func main() {
	cfg := config.LoadMQConfig()

	logg, err := logger.NewLogger(cfg.LogLevel)
	if err != nil {
		log.Fatal(err)
	}
	defer logg.Sync()

	queue := mq.NewQueue[mq.Envelope](cfg.MQPartitions)
	server := mqhttp.NewServer(queue, logg)

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: server.Router(),
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
		logg.Info("MQ server started", zap.String("addr", ":8080"))

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logg.Fatal("MQ server failed to start",
				zap.Error(err),
			)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	logg.Info("shutdown signal received")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logg.Error("MQ server graceful shutdown failed",
			zap.Error(err),
		)
	} else {
		logg.Info("MQ server shut down gracefully")
	}
}

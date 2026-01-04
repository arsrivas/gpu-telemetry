package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"gpu-telemetry/config"
	"gpu-telemetry/pkg/client"
	"gpu-telemetry/pkg/collector"
	"gpu-telemetry/pkg/storage"
	"gpu-telemetry/pkg/util/logger"

	"go.uber.org/zap"
)

func main() {
	cfg, err := config.LoadCollectorConfig()
	if err != nil {
		panic(err)
	}
	logg, err := logger.NewLogger("info")
	if err != nil {
		panic(err)
	}
	defer func() { _ = logg.Sync() }()

	store, err := storage.NewPostgres(cfg.PostgresDSN)
	if err != nil {
		logg.Fatal("failed to connect to postgres", zap.Error(err))
	}
	defer store.Close()

	mq := client.NewMQClient(cfg.MQURL)
	c := collector.New(mq, store, logg)

	// ---- context & graceful shutdown ----
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logg.Info("telemetry collector service started")

	c.Run(ctx)
}

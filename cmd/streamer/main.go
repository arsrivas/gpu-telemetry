package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"gpu-telemetry/config"
	"gpu-telemetry/pkg/client"
	"gpu-telemetry/pkg/streamer"
	"gpu-telemetry/pkg/util/logger"
)

func main() {
	cfg, err := config.LoadStreamerConfig()
	if err != nil {
		log.Fatal(err)
	}
	logg, err := logger.NewLogger(cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	interval := 1 * time.Second
	if v := os.Getenv("INTERVAL_MS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			interval = time.Duration(i) * time.Millisecond
		}
	}

	mq := client.NewMQClient(cfg.MQURL)
	s := streamer.New(mq, cfg.MetricsCsvFilePath, interval, logg)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logg.Info("telemetry data streamer service started")
	s.Run(ctx)
}

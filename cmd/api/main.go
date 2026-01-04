package main

import (
	"log"
	"net/http"
	"os"

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

	logg.Info("API server starting",
		zap.String("port", cfg.ServerPort),
	)

	if err := http.ListenAndServe(":"+cfg.ServerPort, api.Router(h)); err != nil {
		logg.Error("API server stopped",
			zap.Error(err),
		)
		os.Exit(1)
	}

}

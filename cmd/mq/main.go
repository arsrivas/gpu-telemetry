package main

import (
	"log"
	"net/http"

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

	queue := mq.NewQueue[mq.Envelope](cfg.MQPartitions)
	server := mqhttp.NewServer(queue, logg)

	logg.Info("mq server started", zap.String("addr", ":8080"))

	if err := http.ListenAndServe(":8080", server.Router()); err != nil {
		logg.Fatal("mq server failed to start",
			zap.Error(err),
		)
	}
}

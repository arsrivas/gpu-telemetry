package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"gpu-telemetry/pkg/mq"
	mqhttp "gpu-telemetry/pkg/transport/mqhttp"
	"gpu-telemetry/pkg/util/logger"

	"go.uber.org/zap"
)

func main() {
	partitions := 4
	if v := os.Getenv("MQ_PARTITIONS"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			partitions = p
		}
	}
	logg, err := logger.NewLogger("info")
	if err != nil {
		log.Fatal(err)
	}

	queue := mq.NewQueue[mq.Envelope](partitions)
	server := mqhttp.NewServer(queue, logg)

	logg.Info("mq server started", zap.String("addr", ":8080"))

	if err := http.ListenAndServe(":8080", server.Router()); err != nil {
		logg.Fatal("mq server failed to start",
			zap.Error(err),
		)
	}
}

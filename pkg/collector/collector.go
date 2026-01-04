package collector

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"gpu-telemetry/model"
	"gpu-telemetry/pkg/client"
	"gpu-telemetry/pkg/storage"

	"go.uber.org/zap"
)

type Collector struct {
	MQ    *client.MQClient
	Store storage.Store
	log   *zap.Logger
}

func New(mq *client.MQClient, store storage.Store, log *zap.Logger) *Collector {
	return &Collector{
		MQ:    mq,
		Store: store,
		log:   log,
	}
}

func (c *Collector) Run(ctx context.Context) {

	// health server
	go c.startHealthServer()

	for {
		select {
		case <-ctx.Done():
			c.log.Info("collector shutting down")
			return
		default:
		}

		msgs, err := c.MQ.Poll(10)
		if err != nil {
			c.log.Warn("poll failed", zap.Error(err))
			time.Sleep(2 * time.Second)
			continue
		}

		if len(msgs) == 0 {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		for _, msg := range msgs {
			var t model.Telemetry
			if err := json.Unmarshal(msg.Payload, &t); err != nil {
				c.log.Warn("unable to deserialize telemetry data")
				continue
			}
			if err := c.Store.Insert(t); err != nil {
				c.log.Warn("failed to persist telemetry",
					zap.String("id", t.ID),
					zap.String("gpu", t.GPUId),
					zap.Error(err),
				)
				continue
			}

			c.log.Debug("telemetry stored",
				zap.String("id", t.ID),
				zap.String("gpu", t.GPUId),
			)

			if err := c.MQ.Ack(t.ID); err != nil {
				c.log.Warn("ack failed",
					zap.String("id", t.ID),
					zap.Error(err),
				)
			}
		}
	}
}

func (c *Collector) startHealthServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", c.Health)

	srv := &http.Server{
		Addr:    ":8082",
		Handler: mux,
	}

	c.log.Info("health server started", zap.String("addr", ":8082"))
	_ = srv.ListenAndServe()
}

func (c *Collector) Health(w http.ResponseWriter, _ *http.Request) {
	if err := c.Store.Ping(); err != nil {
		http.Error(w, "db not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

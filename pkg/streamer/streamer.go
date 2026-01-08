package streamer

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"gpu-telemetry/pkg/client"
	"gpu-telemetry/pkg/mq"
)

type Streamer struct {
	source   DataSource
	mq       *client.MQClient
	interval time.Duration
	log      *zap.Logger
}

func New(source DataSource, mq *client.MQClient, interval time.Duration, log *zap.Logger) *Streamer {
	return &Streamer{
		source:   source,
		mq:       mq,
		interval: interval,
		log:      log,
	}
}

func (s *Streamer) Run(ctx context.Context) {
	s.log.Info("streamer started")

	ch, err := s.source.DataStream(ctx)
	if err != nil {
		s.log.Fatal("failed to start data source", zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			s.log.Info("streamer shutting down")
			return

		case t, ok := <-ch:
			if !ok {
				s.log.Info("data source closed, restarting")
				time.Sleep(time.Second)
				return
			}

			payload, _ := json.Marshal(t)
			env := mq.Envelope{
				ID:      t.ID,
				Key:     t.GPUId,
				Payload: payload,
				Type:    "telemetry",
			}

			if err := s.mq.Enqueue(env); err != nil {
				s.log.Warn("enqueue failed",
					zap.String("gpu", t.GPUId),
					zap.Error(err),
				)
			}

			time.Sleep(s.interval)
		}
	}
}

package streamer

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"gpu-telemetry/model"
	"gpu-telemetry/pkg/client"
	"gpu-telemetry/pkg/mq"
)

// Streamer reads telemetry data from a CSV file and continuously
// streams it to the message queue.
type Streamer struct {
	MQ       *client.MQClient
	Interval time.Duration
	FilePath string
	log      *zap.Logger
}

// New constructs a new Streamer instance.
func New(mq *client.MQClient, file string, interval time.Duration, log *zap.Logger) *Streamer {
	return &Streamer{
		MQ:       mq,
		FilePath: file,
		Interval: interval,
		log:      log,
	}
}

// Run starts the main streaming loop.
func (s *Streamer) Run(ctx context.Context) {
	normalize := func(s string) string {
		s = strings.TrimSpace(s)
		s = strings.TrimPrefix(s, "\ufeff")
		return strings.ToLower(s)
	}

	s.log.Info("streamer started",
		zap.String("file", s.FilePath),
		zap.Duration("interval", s.Interval),
	)

	for {
		select {
		case <-ctx.Done():
			s.log.Info("streamer shutting down")
			return
		default:
		}

		file, err := os.Open(s.FilePath)
		if err != nil {
			s.log.Error("failed to open csv", zap.Error(err))
			time.Sleep(time.Second)
			continue
		}

		reader := csv.NewReader(file)
		reader.FieldsPerRecord = -1
		reader.TrimLeadingSpace = true
		reader.LazyQuotes = true

		header, err := reader.Read()
		if err != nil {
			s.log.Error("failed to read csv header", zap.Error(err))
			file.Close()
			continue
		}

		col := make(map[string]int)
		for i, h := range header {
			col[normalize(h)] = i
		}

		get := func(row []string, key string) string {
			i, ok := col[normalize(key)]
			if !ok || i >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[i])
		}

		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				s.log.Warn("csv read error", zap.Error(err))
				continue
			}

			valueStr := get(row, "value")
			if valueStr == "" {
				s.log.Warn("missing value field")
				continue
			}

			value, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				s.log.Warn("invalid value", zap.String("value", valueStr))
				continue
			}

			gpuUUID := get(row, "uuid")
			if gpuUUID == "" {
				s.log.Warn("missing gpu uuid")
				continue
			}

			msg := model.Telemetry{
				ID:        uuid.NewString(),
				GPUId:     gpuUUID,
				Timestamp: time.Now().UTC(),
				Metric:    get(row, "metric_name"),
				Value:     value,
				Labels:    get(row, "labels_raw"),
			}
			payload, _ := json.Marshal(msg)
			env := mq.Envelope{
				ID:      msg.ID,
				Key:     msg.GPUId,
				Payload: payload,
				Type:    "telemetry",
			}
			if err := s.MQ.Enqueue(env); err != nil {
				s.log.Warn("enqueue failed",
					zap.String("gpu", msg.GPUId),
					zap.Error(err),
				)
			}

			time.Sleep(s.Interval)
		}

		file.Close()
		s.log.Info("csv replay completed, restarting")
	}
}

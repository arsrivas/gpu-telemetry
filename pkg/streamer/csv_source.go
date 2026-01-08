package streamer

import (
	"context"
	"encoding/csv"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"gpu-telemetry/model"
)

type CSVSource struct {
	filePath string
	log      *zap.Logger
}

func NewCSVSource(path string, log *zap.Logger) *CSVSource {
	return &CSVSource{
		filePath: path,
		log:      log,
	}
}

func (c *CSVSource) DataStream(ctx context.Context) (<-chan model.Telemetry, error) {
	out := make(chan model.Telemetry)

	go func() {
		defer close(out)

		normalize := func(s string) string {
			s = strings.TrimSpace(s)
			s = strings.TrimPrefix(s, "\ufeff")
			return strings.ToLower(s)
		}

		for {
			file, err := os.Open(c.filePath)
			if err != nil {
				c.log.Error("failed to open csv", zap.Error(err))
				time.Sleep(time.Second)
				continue
			}

			reader := csv.NewReader(file)
			reader.FieldsPerRecord = -1
			reader.TrimLeadingSpace = true
			reader.LazyQuotes = true

			header, err := reader.Read()
			if err != nil {
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
				select {
				case <-ctx.Done():
					file.Close()
					return
				default:
				}

				row, err := reader.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					continue
				}

				value, err := strconv.ParseFloat(get(row, "value"), 64)
				if err != nil {
					continue
				}

				out <- model.Telemetry{
					ID:        uuid.NewString(),
					GPUId:     get(row, "uuid"),
					Timestamp: time.Now().UTC(),
					Metric:    get(row, "metric_name"),
					Value:     value,
					Labels:    get(row, "labels_raw"),
				}
			}

			file.Close()
		}
	}()

	return out, nil
}

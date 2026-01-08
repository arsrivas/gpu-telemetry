package streamer

import (
	"context"

	"gpu-telemetry/model"
)

// DataSource produces telemetry records
type DataSource interface {
	DataStream(ctx context.Context) (<-chan model.Telemetry, error)
}

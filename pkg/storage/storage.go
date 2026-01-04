package storage

import (
	"gpu-telemetry/model"
	"time"
)

// Store defines the persistence contract used by Collector and API
type Store interface {
	Insert(entry model.Telemetry) error

	// GPUs returns all GPU IDs for which telemetry exists
	GPUs() ([]string, error)

	// Telemetry returns telemetry for a GPU ordered by time
	// startTs / endTs are optional (unix seconds)
	Telemetry(gpuID string, startTs, endTs *time.Time) ([]model.Telemetry, error)
	GPUExists(gpuID string) (bool, error)
	Ping() error
	Close() error
}

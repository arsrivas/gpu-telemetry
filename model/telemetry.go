package model

import "time"

type Telemetry struct {
	ID        string    `json:"id"`
	GPUId     string    `json:"gpu_id"`
	Timestamp time.Time `json:"timestamp"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Labels    string    `json:"labels"`
}

type TelemetryResponse struct {
	ID        string    `json:"id"`
	GPUId     string    `json:"gpu_id"`
	Timestamp time.Time `json:"timestamp"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Labels    string    `json:"labels"`
}

type ErrorResponse struct {
	Message string `json:"message" example:"invalid start_time"`
}

func (t Telemetry) IDKey() string {
	return t.ID
}

func (t Telemetry) PartitionKey() string {
	return t.GPUId
}

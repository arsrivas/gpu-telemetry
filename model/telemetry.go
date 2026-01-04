package model

import "time"

// Telemetry represents a single telemetry datapoint emitted by a GPU.
type Telemetry struct {
	// ID is a globally unique identifier for the telemetry record.
	ID string `json:"id"`
	// GPUId identifies the GPU that emitted this telemetry datapoint.
	GPUId string `json:"gpu_id"`
	// Timestamp represents the ingestion time of the telemetry datapoint.
	Timestamp time.Time `json:"timestamp"`
	// Metric represents the name of the telemetry metric
	Metric string `json:"metric"`
	// Value is the numeric value of the telemetry metric.
	Value float64 `json:"value"`
	// Labels contains optional metadata associated with the telemetry datapoint.
	Labels string `json:"labels"`
}

// TelemetryResponse represents the API-facing view of telemetry data.
// It intentionally mirrors Telemetry today but allows future divergence
type TelemetryResponse struct {
	ID        string    `json:"id"`
	GPUId     string    `json:"gpu_id"`
	Timestamp time.Time `json:"timestamp"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Labels    string    `json:"labels"`
}

// ErrorResponse represents a standardized error payload returned by the API.
type ErrorResponse struct {
	Message string `json:"message" example:"invalid start_time"`
}

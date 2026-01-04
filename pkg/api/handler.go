package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"gpu-telemetry/model"
	"gpu-telemetry/pkg/storage"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Handler struct {
	store storage.Store
	log   *zap.Logger
}

// NewHandler constructs a new API handler with its dependencies injected.
func NewHandler(store storage.Store, log *zap.Logger) *Handler {
	return &Handler{
		store: store,
		log:   log,
	}
}

// ListGPUs godoc
// @Summary      List all GPUs
// @Description  Returns all GPU IDs for which telemetry is available
// @Tags         GPUs
// @Produce      json
// @Success      200 {array} string
// @Failure      500 {object} model.ErrorResponse
// @Router       /api/v1/gpus [get]
//
// ListGPUs returns a list of GPU identifiers for which telemetry
// has been persisted.
func (h *Handler) ListGPUs(w http.ResponseWriter, _ *http.Request) {
	h.log.Info("List GPUs request received")
	gpus, err := h.store.GPUs()
	if err != nil {
		h.log.Error("failed to list GPUs", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to list GPUs")
		return
	}
	h.log.Info("List GPUs request served successfully", zap.Int("gpu_count", len(gpus)))
	writeJSON(w, http.StatusOK, gpus)
}

// GetTelemetry godoc
// @Summary      Query telemetry by GPU
// @Description  Returns telemetry entries ordered by time
// @Tags         Telemetry
// @Produce      json
// @Param        id path string true "GPU ID"
// @Param        start_time query int64 false "Start time (Unix timestamp in seconds)"
// @Param        end_time query int64 false "End time (Unix timestamp in seconds)"
// @Success      200 {array} model.TelemetryResponse
// @Failure      400 {object} model.ErrorResponse
// @Failure      404 {object} model.ErrorResponse
// @Failure      500 {object} model.ErrorResponse
// @Router       /api/v1/gpus/{id}/telemetry [get]
//
// GetTelemetry returns telemetry datapoints for a specific GPU,
// optionally filtered by a time window.
func (h *Handler) GetTelemetry(w http.ResponseWriter, r *http.Request) {
	gpuID := chi.URLParam(r, "id")
	h.log.Info("get telemetry request received", zap.String("gpu_id", gpuID))

	var start, end *int64

	if v := r.URL.Query().Get("start_time"); v != "" {
		ts, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			h.log.Warn("invalid start_time", zap.String("gpu_id", gpuID), zap.String("value", v))
			writeError(w, http.StatusBadRequest, "invalid start_time")
			return
		}
		start = &ts
	}

	if v := r.URL.Query().Get("end_time"); v != "" {
		ts, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			h.log.Warn("invalid end_time", zap.String("gpu_id", gpuID), zap.String("value", v))
			writeError(w, http.StatusBadRequest, "invalid end_time")
			return
		}
		end = &ts
	}

	if start != nil && end != nil && *start > *end {
		writeError(w, http.StatusBadRequest, "start_time must be <= end_time")
		return
	}

	exists, err := h.store.GPUExists(gpuID)
	if err != nil {
		h.log.Error("failed to check GPU existence", zap.String("gpu_id", gpuID), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal server error while checking for GPU")
		return
	}
	if !exists {
		h.log.Info("GPU not found", zap.String("gpu_id", gpuID))
		writeError(w, http.StatusNotFound, "GPU not found")
		return
	}

	data, err := h.store.Telemetry(gpuID, start, end)
	if err != nil {
		h.log.Error("failed to fetch telemetry", zap.String("gpu_id", gpuID), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to fetch telemetry data")
		return
	}

	resp := make([]model.TelemetryResponse, 0, len(data))
	for _, t := range data {
		resp = append(resp, model.TelemetryResponse{
			ID:        t.ID,
			GPUId:     t.GPUId,
			Timestamp: t.Timestamp,
			Metric:    t.Metric,
			Value:     t.Value,
			Labels:    t.Labels,
		})
	}
	h.log.Info("telemetry query succeeded",
		zap.String("gpu_id", gpuID),
		zap.Int("count", len(data)),
	)
	writeJSON(w, http.StatusOK, resp)
}

// Health implements a readiness probe for the API.
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	if err := h.store.Ping(); err != nil {
		writeError(w, http.StatusServiceUnavailable, "DB not ready")
		return
	}
	w.WriteHeader(http.StatusOK)
}

// writeJSON serializes the given value as JSON and writes it
// with the provided HTTP status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a standardized JSON error response.
// It is intentionally minimal to keep error handling consistent.
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(model.ErrorResponse{
		Message: msg,
	})
}

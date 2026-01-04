package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"gpu-telemetry/model"
	"gpu-telemetry/pkg/api"
	"gpu-telemetry/pkg/storage"
	mocks "gpu-telemetry/pkg/storage/mock"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHandler_ListGPUs(t *testing.T) {
	tests := []struct {
		name           string
		store          storage.Store
		log            *zap.Logger
		wantStatusCode int
		wantGPUCount   int
	}{
		{
			name: "success with multiple gpus",
			store: &mocks.StoreMock{
				GPUsFn: func() ([]string, error) {
					return []string{"GPU-1", "GPU-2"}, nil
				},
			},
			log:            zap.NewNop(),
			wantStatusCode: http.StatusOK,
			wantGPUCount:   2,
		},
		{
			name: "success with no gpus",
			store: &mocks.StoreMock{
				GPUsFn: func() ([]string, error) {
					return []string{}, nil
				},
			},
			log:            zap.NewNop(),
			wantStatusCode: http.StatusOK,
			wantGPUCount:   0,
		},
		{
			name: "store returns error",
			store: &mocks.StoreMock{
				GPUsFn: func() ([]string, error) {
					return nil, errors.New("db failure")
				},
			},
			log:            zap.NewNop(),
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := api.NewHandler(tt.store, tt.log)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus", nil)
			rr := httptest.NewRecorder()

			h.ListGPUs(rr, req)

			require.Equal(t, tt.wantStatusCode, rr.Code)

			if tt.wantStatusCode == http.StatusOK {
				var resp []string
				require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
				require.Len(t, resp, tt.wantGPUCount)
			}
		})
	}
}

func TestHandler_Health(t *testing.T) {
	tests := []struct {
		name       string
		store      *mocks.StoreMock
		wantStatus int
	}{
		{
			name: "db ready",
			store: &mocks.StoreMock{
				PingFn: func() error { return nil },
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "db not ready",
			store: &mocks.StoreMock{
				PingFn: func() error { return errors.New("db down") },
			},
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	log := zap.NewNop()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := api.NewHandler(tt.store, log)

			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			rr := httptest.NewRecorder()

			h.Health(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestHandler_GetTelemetry(t *testing.T) {
	tests := []struct {
		name       string
		store      *mocks.StoreMock
		gpuID      string
		query      string
		wantStatus int
	}{
		{
			name:  "gpu not found",
			gpuID: "GPU-X",
			store: &mocks.StoreMock{
				GPUExistsFn: func(id string) (bool, error) {
					return false, nil
				},
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:  "telemetry success",
			gpuID: "GPU-1",
			store: &mocks.StoreMock{
				GPUExistsFn: func(id string) (bool, error) {
					return true, nil
				},
				TelemetryFn: func(id string, _, _ *int64) ([]model.Telemetry, error) {
					return []model.Telemetry{
						{ID: "1", GPUId: id, Metric: "util", Value: 80},
					}, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "invalid start_time",
			gpuID: "GPU-1",
			query: "?start_time=abc",
			store: &mocks.StoreMock{
				GPUExistsFn: func(id string) (bool, error) {
					return true, nil
				},
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "invalid end_time",
			gpuID: "GPU-1",
			query: "?end_time=not-a-number",
			store: &mocks.StoreMock{
				GPUExistsFn: func(id string) (bool, error) {
					return true, nil
				},
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "gpu exists check fails",
			gpuID: "GPU-1",
			store: &mocks.StoreMock{
				GPUExistsFn: func(id string) (bool, error) {
					return false, errors.New("db error during exists check")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:  "telemetry fetch fails",
			gpuID: "GPU-1",
			store: &mocks.StoreMock{
				GPUExistsFn: func(id string) (bool, error) {
					return true, nil
				},
				TelemetryFn: func(id string, _, _ *int64) ([]model.Telemetry, error) {
					return nil, errors.New("db error during telemetry fetch")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	log := zap.NewNop()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := api.NewHandler(tt.store, log)

			req := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/gpus/"+tt.gpuID+"/telemetry"+tt.query,
				nil,
			)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.gpuID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()
			h.GetTelemetry(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

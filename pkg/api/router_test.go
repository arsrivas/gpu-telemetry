package api

import (
	"gpu-telemetry/pkg/storage/mock"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestRouter(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "health endpoint",
			method: http.MethodGet,
			path:   "/healthz",
		},
		{
			name:   "list gpus endpoint",
			method: http.MethodGet,
			path:   "/api/v1/gpus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				log: zap.NewNop(),
				store: &mock.StoreMock{
					PingFn: func() error {
						return nil
					},
					GPUsFn: func() ([]string, error) {
						return []string{"GPU-1", "GPU-2"}, nil
					},
				},
			}

			router := Router(h)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)
		})
	}
}

package api_test

import (
	"gpu-telemetry/pkg/api"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestRequestLogger(t *testing.T) {
	tests := []struct {
		name string
		log  *zap.Logger
	}{
		{
			name: "Logger middleware",
			log:  zap.NewNop(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test?x=1", nil)
			rr := httptest.NewRecorder()

			mw := api.RequestLogger(tt.log)
			handler := mw(next)

			handler.ServeHTTP(rr, req)
		})
	}
}

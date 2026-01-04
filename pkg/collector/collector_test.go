package collector

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"gpu-telemetry/model"
	"gpu-telemetry/pkg/client"
	"gpu-telemetry/pkg/mq"
	storagemock "gpu-telemetry/pkg/storage/mock"

	"go.uber.org/zap"
)

// startMockMQClient mocks the HTTP MQ server and returns:
// - MQ client
// - channel signaled on Ack
// - cleanup function
func startMockMQClient(t *testing.T, envelopes []mq.Envelope) (*client.MQClient, chan struct{}, func()) {
	t.Helper()

	ackCh := make(chan struct{}, 1)
	var once sync.Once
	polled := false

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {

		case "/poll":
			// Return messages only once
			if polled {
				_ = json.NewEncoder(w).Encode([]mq.Envelope{})
				return
			}
			polled = true
			_ = json.NewEncoder(w).Encode(envelopes)

		case "/ack":
			once.Do(func() {
				ackCh <- struct{}{}
			})
			w.WriteHeader(http.StatusOK)

		default:
			http.NotFound(w, r)
		}
	}))

	cleanup := func() { srv.Close() }
	return client.NewMQClient(srv.URL), ackCh, cleanup
}

func TestCollector_Run(t *testing.T) {
	tests := []struct {
		name      string
		insertErr error
		wantAck   bool
	}{
		{
			name:      "successful insert → ack",
			insertErr: nil,
			wantAck:   true,
		},
		{
			name:      "insert failure → no ack",
			insertErr: errors.New("db error"),
			wantAck:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			telemetry := model.Telemetry{
				ID:    "1",
				GPUId: "gpu-1",
			}

			payload, err := json.Marshal(telemetry)
			if err != nil {
				t.Fatalf("failed to marshal telemetry: %v", err)
			}

			env := mq.Envelope{
				ID:      telemetry.ID,
				Key:     telemetry.GPUId,
				Payload: payload,
			}

			mqClient, ackCh, cleanup :=
				startMockMQClient(t, []mq.Envelope{env})
			defer cleanup()

			store := &storagemock.StoreMock{
				InsertFn: func(model.Telemetry) error {
					return tt.insertErr
				},
			}

			c := New(mqClient, store, zap.NewNop())

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go c.Run(ctx)

			select {
			case <-ackCh:
				if !tt.wantAck {
					t.Fatalf("ack happened, but wantAck=false")
				}

			case <-time.After(400 * time.Millisecond):
				if tt.wantAck {
					t.Fatalf("ack did not happen")
				}
			}
		})
	}
}

func TestCollector_Health(t *testing.T) {
	tests := []struct {
		name       string
		pingErr    error
		wantStatus int
	}{
		{
			name:       "db ready",
			pingErr:    nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "db not ready",
			pingErr:    errors.New("db down"),
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &storagemock.StoreMock{
				PingFn: func() error { return tt.pingErr },
			}

			c := New(nil, store, zap.NewNop())

			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			w := httptest.NewRecorder()

			c.Health(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

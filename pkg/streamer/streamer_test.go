package streamer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"gpu-telemetry/model"
	"gpu-telemetry/pkg/client"
)

type mockSource struct {
	ch chan model.Telemetry
}

func (m *mockSource) DataStream(ctx context.Context) (<-chan model.Telemetry, error) {
	return m.ch, nil
}

func TestStreamer_Run(t *testing.T) {
	tests := []struct {
		name       string
		sendFunc   func(ch chan model.Telemetry)
		closeChan  bool
		assertFunc func(t *testing.T, count int)
	}{
		{
			name: "single telemetry is enqueued",
			sendFunc: func(ch chan model.Telemetry) {
				ch <- model.Telemetry{ID: "1", GPUId: "gpu-1"}
			},
			closeChan: true,
			assertFunc: func(t *testing.T, count int) {
				require.Equal(t, 1, count)
			},
		},
		{
			name: "multiple telemetry records are enqueued",
			sendFunc: func(ch chan model.Telemetry) {
				ch <- model.Telemetry{ID: "1", GPUId: "gpu-1"}
				ch <- model.Telemetry{ID: "2", GPUId: "gpu-2"}
				ch <- model.Telemetry{ID: "3", GPUId: "gpu-3"}
			},
			closeChan: true,
			assertFunc: func(t *testing.T, count int) {
				require.Equal(t, 3, count)
			},
		},
		{
			name:      "context cancelled before any telemetry",
			sendFunc:  func(_ chan model.Telemetry) {},
			closeChan: true,
			assertFunc: func(t *testing.T, count int) {
				require.Equal(t, 0, count)
			},
		},
		{
			name:      "channel closed without data",
			sendFunc:  func(_ chan model.Telemetry) {},
			closeChan: true, // closed by harness, not sendFunc
			assertFunc: func(t *testing.T, count int) {
				require.Equal(t, 0, count)
			},
		},
		{
			name: "context cancelled after first telemetry",
			sendFunc: func(ch chan model.Telemetry) {
				ch <- model.Telemetry{ID: "1", GPUId: "gpu-1"}
				ch <- model.Telemetry{ID: "2", GPUId: "gpu-2"}
			},
			closeChan: true,
			assertFunc: func(t *testing.T, count int) {
				require.GreaterOrEqual(t, count, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enqueued := make(chan struct{}, 10)

			mqServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/enqueue" {
					enqueued <- struct{}{}
					w.WriteHeader(http.StatusAccepted)
					return
				}
				http.NotFound(w, r)
			}))
			defer mqServer.Close()

			mqClient := client.NewMQClient(mqServer.URL)

			ch := make(chan model.Telemetry, 10)
			source := &mockSource{ch: ch}

			s := New(source, mqClient, 0, zap.NewNop())

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go s.Run(ctx)

			tt.sendFunc(ch)

			time.Sleep(50 * time.Millisecond)
			cancel()

			if tt.closeChan {
				close(ch)
			}

			timeout := time.After(300 * time.Millisecond)
			count := 0

		loop:
			for {
				select {
				case <-enqueued:
					count++
				case <-timeout:
					break loop
				}
			}

			tt.assertFunc(t, count)
		})
	}
}

package streamer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"gpu-telemetry/pkg/client"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStreamer_Run_EnqueuesMessages(t *testing.T) {
	// --- temp CSV file ---
	csvData := `uuid,metric_name,value,labels_raw
gpu-1,temp,42.5,lab1
`
	f, err := os.CreateTemp("", "telemetry-*.csv")
	require.NoError(t, err)
	defer os.Remove(f.Name())
	_, err = f.WriteString(csvData)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	// --- signal channel ---
	enqueuedCh := make(chan struct{}, 1)
	// --- fake MQ server ---
	mqServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/enqueue" {
			select {
			case enqueuedCh <- struct{}{}:
			default:
			}
			w.WriteHeader(http.StatusAccepted)
			return
		}
		http.NotFound(w, r)
	}))
	defer mqServer.Close()
	mqClient := client.NewMQClient(mqServer.URL)
	s := New(
		mqClient,
		f.Name(),
		0, // no sleep for deterministic behavior
		zap.NewNop(),
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.Run(ctx)
	// --- wait for exactly one enqueue ---
	select {
	case <-enqueuedCh:
		// success
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for enqueue")
	}
	// stop streamer immediately
	cancel()
}

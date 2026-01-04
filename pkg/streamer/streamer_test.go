package streamer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"gpu-telemetry/pkg/client"

	"go.uber.org/zap"
)

func TestStreamer_Run_EnqueuesMessages(t *testing.T) {
	// --- temp CSV file ---
	csv := `uuid,metric_name,value,labels_raw
gpu-1,temp,42.5,lab1
`

	f, err := os.CreateTemp("", "telemetry-*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	_, _ = f.WriteString(csv)
	f.Close()

	// --- fake MQ server ---
	enqueued := 0
	mqServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/enqueue" {
			enqueued++
			w.WriteHeader(http.StatusAccepted)
			return
		}
		http.NotFound(w, r)
	}))
	defer mqServer.Close()

	mq := client.NewMQClient(mqServer.URL)

	s := New(
		mq,
		f.Name(),
		10*time.Millisecond,
		zap.NewNop(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	go s.Run(ctx)

	time.Sleep(100 * time.Millisecond)
	cancel()

	if enqueued == 0 {
		t.Fatalf("expected at least one enqueue, got %d", enqueued)
	}
}

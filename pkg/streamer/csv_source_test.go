package streamer

import (
	"context"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"

	"gpu-telemetry/model"
)

func TestCSVSource_DataStream(t *testing.T) {
	// --- Arrange ---
	csvContent := `uuid,metric_name,value,labels_raw
gpu-1,temperature,70.5,zone=a
gpu-2,utilization,55.0,zone=b
`

	tmpFile, err := os.CreateTemp("", "telemetry-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(csvContent); err != nil {
		t.Fatalf("failed to write csv: %v", err)
	}
	tmpFile.Close()

	logger := zap.NewNop()
	source := NewCSVSource(tmpFile.Name(), logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Act ---
	ch, err := source.DataStream(ctx)
	if err != nil {
		t.Fatalf("DataStream returned error: %v", err)
	}

	var results []model.Telemetry

	for i := 0; i < 2; i++ {
		select {
		case msg := <-ch:
			results = append(results, msg)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for telemetry")
		}
	}

	// stop the infinite loop
	cancel()

	// --- Assert ---
	if len(results) != 2 {
		t.Fatalf("expected 2 telemetry records, got %d", len(results))
	}

	tests := []struct {
		got  model.Telemetry
		uuid string
		name string
		val  float64
	}{
		{results[0], "gpu-1", "temperature", 70.5},
		{results[1], "gpu-2", "utilization", 55.0},
	}

	for _, tt := range tests {
		if tt.got.GPUId != tt.uuid {
			t.Errorf("GPUId mismatch: got %s, want %s", tt.got.GPUId, tt.uuid)
		}
		if tt.got.Metric != tt.name {
			t.Errorf("Metric mismatch: got %s, want %s", tt.got.Metric, tt.name)
		}
		if tt.got.Value != tt.val {
			t.Errorf("Value mismatch: got %f, want %f", tt.got.Value, tt.val)
		}
		if tt.got.ID == "" {
			t.Error("expected ID to be set")
		}
		if tt.got.Timestamp.IsZero() {
			t.Error("expected Timestamp to be set")
		}
	}
}

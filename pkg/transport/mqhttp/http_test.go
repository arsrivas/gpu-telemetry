package mqhttp

import (
	"bytes"
	"encoding/json"
	"gpu-telemetry/pkg/mq"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

type fakeQueue[T mq.QueueItem] struct {
	items []T
}

func (f *fakeQueue[T]) Enqueue(msg T) {
	f.items = append(f.items, msg)
}

func (f *fakeQueue[T]) Poll(limit int) []T {
	return f.items
}

func (f *fakeQueue[T]) Ack(id string) bool {
	return id == "ok"
}

func (f *fakeQueue[T]) Stats() map[string]int {
	return map[string]int{
		"queued":    len(f.items),
		"in_flight": 0,
	}
}

type testItem struct {
	id  string
	key string
}

func (t testItem) IDKey() string        { return t.id }
func (t testItem) PartitionKey() string { return t.key }

//
// ---- tests ----
//

func TestServer_Enqueue(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		wantStatus int
	}{
		{
			name:       "valid enqueue",
			body:       testItem{id: "1"},
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "invalid json",
			body:       "bad-json",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &fakeQueue[testItem]{}
			s := NewServer[testItem](q, zap.NewNop())

			var buf bytes.Buffer
			_ = json.NewEncoder(&buf).Encode(tt.body)

			req := httptest.NewRequest(http.MethodPost, "/enqueue", &buf)
			w := httptest.NewRecorder()

			s.enqueue(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestServer_Poll(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "poll messages"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &fakeQueue[testItem]{
				items: []testItem{{id: "1"}},
			}
			s := NewServer[testItem](q, zap.NewNop())

			req := httptest.NewRequest(http.MethodGet, "/poll?limit=1", nil)
			w := httptest.NewRecorder()

			s.poll(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d", w.Code)
			}
		})
	}
}

func TestServer_Ack(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{
			name:       "ack success",
			id:         "ok",
			wantStatus: http.StatusOK,
		},
		{
			name:       "ack missing id",
			id:         "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "ack not found",
			id:         "bad",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &fakeQueue[testItem]{}
			s := NewServer[testItem](q, zap.NewNop())

			req := httptest.NewRequest(http.MethodPost, "/ack?id="+tt.id, nil)
			w := httptest.NewRecorder()

			s.ack(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestServer_Stats(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "stats endpoint"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &fakeQueue[testItem]{
				items: []testItem{{id: "1"}},
			}
			s := NewServer[testItem](q, zap.NewNop())

			req := httptest.NewRequest(http.MethodGet, "/stats", nil)
			w := httptest.NewRecorder()

			s.stats(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d", w.Code)
			}
		})
	}
}

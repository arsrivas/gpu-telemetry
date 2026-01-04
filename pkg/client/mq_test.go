package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gpu-telemetry/model"
	"gpu-telemetry/pkg/mq"
)

//
// ──────────────────────────────
// Poll tests
// ──────────────────────────────
//

func TestMQClient_Poll(t *testing.T) {
	tests := []struct {
		name      string
		response  []model.Telemetry
		wantCount int
	}{
		{
			name: "poll returns messages",
			response: []model.Telemetry{
				{ID: "1", GPUId: "gpu-1"},
				{ID: "2", GPUId: "gpu-2"},
			},
			wantCount: 2,
		},
		{
			name:      "poll returns empty list",
			response:  []model.Telemetry{},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// fake MQ server
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/poll" {
					http.NotFound(w, r)
					return
				}
				_ = json.NewEncoder(w).Encode(tt.response)
			}))
			defer srv.Close()

			client := NewMQClient(srv.URL)

			msgs, err := client.Poll(10)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(msgs) != tt.wantCount {
				t.Fatalf("got %d messages, want %d", len(msgs), tt.wantCount)
			}
		})
	}
}

func TestMQClient_Ack(t *testing.T) {
	tests := []struct {
		name       string
		wantCalled bool
	}{
		{
			name:       "ack succeeds",
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/ack" {
					called = true
					w.WriteHeader(http.StatusOK)
					return
				}
				http.NotFound(w, r)
			}))
			defer srv.Close()

			client := NewMQClient(srv.URL)

			err := client.Ack("1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if called != tt.wantCalled {
				t.Fatalf("ack called = %v, want %v", called, tt.wantCalled)
			}
		})
	}
}

func TestMQClient_Enqueue(t *testing.T) {
	tests := []struct {
		name       string
		msg        mq.Envelope
		wantCalled bool
	}{
		{
			name: "enqueue succeeds",
			msg: mq.Envelope{
				ID:      "abcd",
				Key:     "abcd",
				Payload: []byte{1, 2, 3},
			},
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/enqueue" {
					called = true
					w.WriteHeader(http.StatusAccepted)
					return
				}
				http.NotFound(w, r)
			}))
			defer srv.Close()

			client := NewMQClient(srv.URL)

			err := client.Enqueue(tt.msg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if called != tt.wantCalled {
				t.Fatalf("enqueue called = %v, want %v", called, tt.wantCalled)
			}
		})
	}
}

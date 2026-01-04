package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gpu-telemetry/pkg/mq"

	"github.com/stretchr/testify/require"
)

//
// ──────────────────────────────
// Poll tests
// ──────────────────────────────
//

func TestMQClient_Poll(t *testing.T) {
	tests := []struct {
		name      string
		handler   http.HandlerFunc
		wantCount int
		wantErr   bool
	}{
		{
			name: "poll returns messages",
			handler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/poll", r.URL.Path)
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode([]mq.Envelope{
					{ID: "1", Key: "k1"},
					{ID: "2", Key: "k2"},
				})
			},
			wantCount: 2,
		},
		{
			name: "poll returns empty list",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode([]mq.Envelope{})
			},
			wantCount: 0,
		},
		{
			name: "poll returns non-200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name: "poll returns invalid json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("{invalid-json"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			client := NewMQClient(srv.URL)
			envs, err := client.Poll(10)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Len(t, envs, tt.wantCount)
		})
	}
}

//
// ──────────────────────────────
// Enqueue tests
// ──────────────────────────────
//

func TestMQClient_Enqueue(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			name: "enqueue succeeds",
			handler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/enqueue", r.URL.Path)
				require.Equal(t, "application/json", r.Header.Get("Content-Type"))
				w.WriteHeader(http.StatusAccepted)
			},
		},
		{
			name: "enqueue returns non-202",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			client := NewMQClient(srv.URL)

			err := client.Enqueue(mq.Envelope{
				ID:      "id-1",
				Key:     "key-1",
				Payload: []byte{1, 2, 3},
			})

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

//
// ──────────────────────────────
// Ack tests
// ──────────────────────────────
//

func TestMQClient_Ack(t *testing.T) {
	t.Run("ack succeeds", func(t *testing.T) {
		called := false

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/ack", r.URL.Path)
			called = true
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		client := NewMQClient(srv.URL)

		err := client.Ack("123")
		require.NoError(t, err)
		require.True(t, called)
	})

	t.Run("ack http failure", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		srv.Close() // force transport error

		client := NewMQClient(srv.URL)

		err := client.Ack("123")
		require.Error(t, err)
	})
}

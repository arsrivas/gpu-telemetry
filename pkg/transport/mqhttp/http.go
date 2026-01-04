package mqhttp

import (
	"encoding/json"
	"gpu-telemetry/pkg/mq"
	"gpu-telemetry/pkg/transport"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Server exposes HTTP endpoints for interacting with a generic message queue.
type Server[T mq.QueueItem] struct {
	queue transport.Queue[T]
	log   *zap.Logger
}

// NewServer creates a new HTTP server backed by a generic Queue.
func NewServer[T mq.QueueItem](q transport.Queue[T], log *zap.Logger) *Server[T] {
	return &Server[T]{
		queue: q,
		log:   log,
	}
}

// Router builds and returns the HTTP router.
func (s *Server[T]) Router() http.Handler {
	r := chi.NewRouter()

	r.Post("/enqueue", s.enqueue)
	r.Get("/poll", s.poll)
	r.Post("/ack", s.ack)
	r.Get("/stats", s.stats)

	return r
}

// enqueue accepts a message and enqueues it for processing.
func (s *Server[T]) enqueue(w http.ResponseWriter, r *http.Request) {
	var msg T
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		s.log.Warn("failed to decode enqueue request", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.queue.Enqueue(msg)

	s.log.Info("message enqueued",
		zap.String("message_id", msg.IDKey()),
	)

	w.WriteHeader(http.StatusAccepted)
}

// poll retrieves up to `limit` messages from the queue.
func (s *Server[T]) poll(w http.ResponseWriter, r *http.Request) {
	limit := 5
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			if v > 0 {
				limit = v
			}
		}
	}

	msgs := s.queue.Poll(limit)

	s.log.Debug("messages polled",
		zap.Int("limit", limit),
		zap.Int("returned", len(msgs)),
	)

	_ = json.NewEncoder(w).Encode(msgs)
}

// ack acknowledges successful processing of a message.
func (s *Server[T]) ack(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		s.log.Warn("ack called without id")
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	if !s.queue.Ack(id) {
		s.log.Warn("ack failed, message not found", zap.String("id", id))
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	s.log.Info("message acknowledged", zap.String("id", id))
	w.WriteHeader(http.StatusOK)
}

// stats returns queue statistics.
func (s *Server[T]) stats(w http.ResponseWriter, _ *http.Request) {
	stats := s.queue.Stats()

	s.log.Debug("queue stats",
		zap.Int("queued", stats["queued"]),
		zap.Int("in_flight", stats["in_flight"]),
	)

	_ = json.NewEncoder(w).Encode(stats)
}

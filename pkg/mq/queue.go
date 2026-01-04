package mq

import (
	"hash/fnv"
	"sync"
	"time"
)

const InFlightTTL = 30 * time.Second

type inFlight[T QueueItem] struct {
	msg       T
	timestamp time.Time
}

type partition[T QueueItem] struct {
	messages []T
	inFlight map[string]inFlight[T]
	mutex    sync.Mutex
}

// MessageQueue is a generic in-memory queue with partitioning
// and at-least-once delivery semantics.
type MessageQueue[T QueueItem] struct {
	partitions []*partition[T]
}

// NewQueue creates a new partitioned queue.
func NewQueue[T QueueItem](n int) *MessageQueue[T] {
	parts := make([]*partition[T], n)
	for i := 0; i < n; i++ {
		parts[i] = &partition[T]{
			messages: []T{},
			inFlight: make(map[string]inFlight[T]),
		}
	}
	return &MessageQueue[T]{partitions: parts}
}

func (q *MessageQueue[T]) partitionFor(key string) *partition[T] {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	idx := int(h.Sum32()) % len(q.partitions)
	return q.partitions[idx]
}

// Enqueue adds a message to the queue.
func (q *MessageQueue[T]) Enqueue(msg T) {
	p := q.partitionFor(msg.PartitionKey())
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.messages = append(p.messages, msg)
}

// Poll retrieves up to `limit` messages from the queue.
func (q *MessageQueue[T]) Poll(limit int) []T {
	result := []T{}

	for _, p := range q.partitions {
		p.mutex.Lock()

		now := time.Now()
		for id, inflight := range p.inFlight {
			if now.Sub(inflight.timestamp) > InFlightTTL {
				p.messages = append(p.messages, inflight.msg)
				delete(p.inFlight, id)
			}
		}

		for len(p.messages) > 0 && len(result) < limit {
			msg := p.messages[0]
			p.messages = p.messages[1:]

			p.inFlight[msg.IDKey()] = inFlight[T]{
				msg:       msg,
				timestamp: time.Now(),
			}

			result = append(result, msg)
		}

		p.mutex.Unlock()

		if len(result) >= limit {
			break
		}
	}

	return result
}

// Ack removes a message from the in-flight set.
func (q *MessageQueue[T]) Ack(id string) bool {
	for _, p := range q.partitions {
		p.mutex.Lock()
		if _, ok := p.inFlight[id]; ok {
			delete(p.inFlight, id)
			p.mutex.Unlock()
			return true
		}
		p.mutex.Unlock()
	}
	return false
}

// Stats returns queue metrics.
func (q *MessageQueue[T]) Stats() map[string]int {
	queued := 0
	inFlight := 0

	for _, p := range q.partitions {
		p.mutex.Lock()
		queued += len(p.messages)
		inFlight += len(p.inFlight)
		p.mutex.Unlock()
	}

	return map[string]int{
		"queued":    queued,
		"in_flight": inFlight,
	}
}

package transport

import "gpu-telemetry/pkg/mq"

// Queue is the minimal behavior required by the transport layer.
type Queue[T mq.QueueItem] interface {
	Enqueue(T)
	Poll(limit int) []T
	Ack(id string) bool
	Stats() map[string]int
}

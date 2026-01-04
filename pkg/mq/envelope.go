package mq

// Envelope is a transport-level message wrapper.
// The queue understands ONLY this metadata.
// Payload is opaque to the queue.
type Envelope struct {
	ID      string            `json:"id"`
	Key     string            `json:"partition_key"`
	Payload []byte            `json:"payload"`
	Headers map[string]string `json:"headers,omitempty"`
	Type    string            `json:"type,omitempty"`
}

func (e Envelope) IDKey() string {
	return e.ID
}

func (e Envelope) PartitionKey() string {
	return e.Key
}

// QueueItem defines the minimal contract required by the queue.
// Any message that provides a stable ID and partition key can be queued.
type QueueItem interface {
	IDKey() string
	PartitionKey() string
}

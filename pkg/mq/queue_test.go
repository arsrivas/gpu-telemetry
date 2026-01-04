package mq

import (
	"reflect"
	"testing"
)

// ---- test item ----

type testItem struct {
	id  string
	key string
}

func (t testItem) IDKey() string        { return t.id }
func (t testItem) PartitionKey() string { return t.key }

// ---- tests ----

func TestQueue_EnqueueAndPoll(t *testing.T) {
	tests := []struct {
		name  string
		items []testItem
		limit int
		want  []testItem
	}{
		{
			name: "single message",
			items: []testItem{
				{id: "1", key: "gpu-1"},
			},
			limit: 1,
			want: []testItem{
				{id: "1", key: "gpu-1"},
			},
		},
		{
			name: "multiple messages",
			items: []testItem{
				{id: "1", key: "gpu-1"},
				{id: "2", key: "gpu-2"},
			},
			limit: 2,
			want: []testItem{
				{id: "1", key: "gpu-1"},
				{id: "2", key: "gpu-2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue[testItem](4)

			for _, item := range tt.items {
				q.Enqueue(item)
			}

			got := q.Poll(tt.limit)

			if !equalUnordered(got, tt.want) {
				t.Fatalf("Poll() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestQueue_Ack(t *testing.T) {
	tests := []struct {
		name string
		item testItem
	}{
		{
			name: "ack existing message",
			item: testItem{id: "1", key: "gpu-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue[testItem](4)

			q.Enqueue(tt.item)
			got := q.Poll(1)

			if !reflect.DeepEqual(got[0], tt.item) {
				t.Fatalf("unexpected polled message %+v", got[0])
			}

			if ok := q.Ack(tt.item.id); !ok {
				t.Fatalf("expected ack to succeed")
			}
		})
	}
}

func TestQueue_Ack_NotFound(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{
			name: "ack missing message",
			id:   "missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue[testItem](4)

			if ok := q.Ack(tt.id); ok {
				t.Fatalf("expected ack to fail")
			}
		})
	}
}

func TestQueue_Stats(t *testing.T) {
	tests := []struct {
		name           string
		enqueued       []testItem
		expectedQueued int
	}{
		{
			name: "stats reflect queued messages",
			enqueued: []testItem{
				{id: "1", key: "gpu-1"},
				{id: "2", key: "gpu-2"},
			},
			expectedQueued: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue[testItem](4)

			for _, item := range tt.enqueued {
				q.Enqueue(item)
			}

			stats := q.Stats()

			if stats["queued"] != tt.expectedQueued {
				t.Fatalf("queued = %d, want %d", stats["queued"], tt.expectedQueued)
			}

			if stats["in_flight"] != 0 {
				t.Fatalf("in_flight = %d, want 0", stats["in_flight"])
			}
		})
	}
}

func equalUnordered[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	count := make(map[T]int)
	for _, v := range a {
		count[v]++
	}
	for _, v := range b {
		count[v]--
		if count[v] < 0 {
			return false
		}
	}
	return true
}

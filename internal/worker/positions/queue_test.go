package positions

import (
	"testing"

	"github.com/google/uuid"
)

func TestQueueDeduplicatesAndCancels(t *testing.T) {
	queue := NewQueue(1)
	tripID := uuid.New()
	task := Task{TripID: tripID, DriverID: uuid.New()}

	if got := queue.Enqueue(task); got != QueueQueued {
		t.Fatalf("first enqueue = %q, want %q", got, QueueQueued)
	}
	if got := queue.Enqueue(task); got != QueueDuplicate {
		t.Fatalf("duplicate enqueue = %q, want %q", got, QueueDuplicate)
	}

	queue.Cancel(tripID)
	queued, ok := queue.Dequeue()
	if !ok {
		t.Fatal("expected queued task")
	}
	if queue.IsCurrent(queued) {
		t.Fatal("cancelled task is still current")
	}
}

func TestQueueDropsWithoutBlocking(t *testing.T) {
	queue := NewQueue(1)
	if queue.Enqueue(Task{TripID: uuid.New()}) != QueueQueued {
		t.Fatal("first task was not queued")
	}
	if queue.Enqueue(Task{TripID: uuid.New()}) != QueueDropped {
		t.Fatal("second task was not dropped")
	}
}

package positions

import (
	"sync"

	"github.com/google/uuid"
)

type Task struct {
	TripID     uuid.UUID
	DriverID   uuid.UUID
	generation uint64
}

type QueueResult string

const (
	QueueQueued    QueueResult = "queued"
	QueueDuplicate QueueResult = "duplicate"
	QueueDropped   QueueResult = "dropped"
)

type Queue struct {
	mu             sync.Mutex
	tasks          chan Task
	pending        map[uuid.UUID]uint64
	nextGeneration uint64
}

func NewQueue(size int) *Queue {
	return &Queue{
		tasks:   make(chan Task, size),
		pending: make(map[uuid.UUID]uint64),
	}
}

func (q *Queue) Enqueue(task Task) QueueResult {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.pending[task.TripID]; exists {
		return QueueDuplicate
	}

	q.nextGeneration++
	task.generation = q.nextGeneration
	q.pending[task.TripID] = task.generation

	select {
	case q.tasks <- task:
		return QueueQueued
	default:
		delete(q.pending, task.TripID)
		return QueueDropped
	}
}

func (q *Queue) Dequeue() (Task, bool) {
	task, ok := <-q.tasks
	if !ok {
		return Task{}, false
	}

	return task, true
}

func (q *Queue) Done(tripID uuid.UUID) {
	q.DoneTask(Task{TripID: tripID})
}

func (q *Queue) DoneTask(task Task) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if generation, ok := q.pending[task.TripID]; ok &&
		(task.generation == 0 || generation == task.generation) {
		delete(q.pending, task.TripID)
	}
}

func (q *Queue) IsCurrent(task Task) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.pending[task.TripID] == task.generation
}

func (q *Queue) Cancel(tripID uuid.UUID) {
	q.mu.Lock()
	defer q.mu.Unlock()

	delete(q.pending, tripID)
}

func (q *Queue) Close() {
	close(q.tasks)
}

func (q *Queue) Len() int {
	return len(q.tasks)
}

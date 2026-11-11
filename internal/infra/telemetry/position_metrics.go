package telemetry

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/denis-sigma/laboratory-work/internal/worker/positions"
)

type PositionMetrics struct {
	staleChecks     metric.Int64Counter
	queueSize       metric.Int64UpDownCounter
	activeWorkers   metric.Int64UpDownCounter
	requests        metric.Int64Counter
	requestDuration metric.Float64Histogram
	queueMu         sync.Mutex
	lastQueueSize   int
}

func NewPositionMetrics() (*PositionMetrics, error) {
	meter := otel.Meter("tripgo/positions")
	staleChecks, err := meter.Int64Counter("tripgo.stale_checks")
	if err != nil {
		return nil, err
	}
	queueSize, err := meter.Int64UpDownCounter("tripgo.position_requests.queue.size")
	if err != nil {
		return nil, err
	}
	activeWorkers, err := meter.Int64UpDownCounter("tripgo.position_requests.workers.active")
	if err != nil {
		return nil, err
	}
	requests, err := meter.Int64Counter("tripgo.position_requests")
	if err != nil {
		return nil, err
	}
	duration, err := meter.Float64Histogram("tripgo.position_requests.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	return &PositionMetrics{
		staleChecks:     staleChecks,
		queueSize:       queueSize,
		activeWorkers:   activeWorkers,
		requests:        requests,
		requestDuration: duration,
	}, nil
}

func (m *PositionMetrics) StaleChecks(result string) {
	m.staleChecks.Add(context.Background(), 1, metric.WithAttributes(attribute.String("result", result)))
}

func (m *PositionMetrics) QueueSize(size int) {
	m.queueMu.Lock()
	delta := size - m.lastQueueSize
	m.lastQueueSize = size
	m.queueMu.Unlock()
	m.queueSize.Add(context.Background(), int64(delta))
}

func (m *PositionMetrics) ActiveWorkers(delta int) {
	m.activeWorkers.Add(context.Background(), int64(delta))
}

func (m *PositionMetrics) Requests(result string) {
	m.requests.Add(context.Background(), 1, metric.WithAttributes(attribute.String("result", result)))
}

func (m *PositionMetrics) RequestDuration(duration time.Duration) {
	m.requestDuration.Record(context.Background(), duration.Seconds())
}

var _ positions.Metrics = (*PositionMetrics)(nil)

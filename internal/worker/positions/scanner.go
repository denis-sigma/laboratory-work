package positions

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	domain "github.com/denis-sigma/laboratory-work/internal/domain/trip"
)

type TripSource interface {
	FindTripsForPositionRequest(
		ctx context.Context,
		staleAfter time.Duration,
	) ([]domain.Trip, error)
}

type Ticker interface {
	C() <-chan time.Time
	Stop()
}

type Clock interface {
	NewTicker(time.Duration) Ticker
}

type realClock struct{}

func (realClock) NewTicker(interval time.Duration) Ticker {
	return realTicker{time.NewTicker(interval)}
}

type realTicker struct{ *time.Ticker }

func (t realTicker) C() <-chan time.Time { return t.Ticker.C }

type Scanner struct {
	source     TripSource
	queue      *Queue
	interval   time.Duration
	staleAfter time.Duration
	logger     *slog.Logger
	metrics    Metrics
	clock      Clock
}

func NewScanner(
	source TripSource,
	queue *Queue,
	interval time.Duration,
	staleAfter time.Duration,
	logger *slog.Logger,
	metrics ...Metrics,
) *Scanner {
	var scannerMetrics Metrics
	if len(metrics) > 0 {
		scannerMetrics = metrics[0]
	}
	return &Scanner{
		source:     source,
		queue:      queue,
		interval:   interval,
		staleAfter: staleAfter,
		logger:     logger,
		metrics:    scannerMetrics,
		clock:      realClock{},
	}
}

func (s *Scanner) SetClock(clock Clock) { s.clock = clock }

func (s *Scanner) Run(ctx context.Context) {
	ticker := s.clock.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
			s.scan(ctx)
		}
	}
}

func (s *Scanner) scan(ctx context.Context) {
	trips, err := s.source.FindTripsForPositionRequest(
		ctx,
		s.staleAfter,
	)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn(
				"не удалось получить поездки для запроса координат",
				"error", err,
			)
		}
		return
	}

	for _, current := range trips {
		result := s.queue.Enqueue(Task{
			TripID:   current.ID,
			DriverID: current.DriverID,
		})
		if s.metrics != nil {
			s.metrics.StaleChecks(string(result))
			s.metrics.QueueSize(s.queue.Len())
		}

		if s.logger != nil && result == QueueDropped {
			s.logger.Warn(
				"очередь запросов координат заполнена",
				"trip_id", current.ID.String(),
			)
		}
	}
}

func (s *Scanner) Cancel(tripID uuid.UUID) {
	s.queue.Cancel(tripID)
}

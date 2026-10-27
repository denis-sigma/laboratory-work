package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	tripusecase "github.com/denis-sigma/laboratory-work/internal/usecase/trip"
)

type TripMetrics struct {
	created   metric.Int64Counter
	completed metric.Int64Counter
}

func NewTripMetrics() (*TripMetrics, error) {
	meter := otel.Meter("tripgo/trips")

	created, err := meter.Int64Counter(
		"tripgo.trips.created",
		metric.WithDescription("Количество созданных поездок"),
	)
	if err != nil {
		return nil, err
	}

	completed, err := meter.Int64Counter(
		"tripgo.trips.completed",
		metric.WithDescription("Количество завершённых поездок"),
	)
	if err != nil {
		return nil, err
	}

	return &TripMetrics{
		created:   created,
		completed: completed,
	}, nil
}

func (m *TripMetrics) TripsCreated(ctx context.Context) {
	m.created.Add(ctx, 1)
}

func (m *TripMetrics) TripsCompleted(ctx context.Context) {
	m.completed.Add(ctx, 1)
}

var _ tripusecase.Metrics = (*TripMetrics)(nil)

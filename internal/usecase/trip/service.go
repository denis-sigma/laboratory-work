package trip

import (
	"context"
	"time"

	"github.com/google/uuid"

	domain "github.com/denis-sigma/laboratory-work/internal/domain/trip"
)

type Service struct {
	repository TripRepository
	txManager  TxManager
	metrics    Metrics
	logger     Logger
}

func NewService(
	repository TripRepository,
	txManager TxManager,
	metrics Metrics,
	loggers ...Logger,
) *Service {
	var logger Logger
	if len(loggers) > 0 {
		logger = loggers[0]
	}
	return &Service{
		repository: repository,
		txManager:  txManager,
		metrics:    metrics,
		logger:     logger,
	}
}

type CreateTripInput struct {
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint domain.Coordinates
	EndPoint   domain.Coordinates
	Price      int64
}

func (s *Service) Create(
	ctx context.Context,
	input CreateTripInput,
) (domain.Trip, error) {
	now := time.Now().UTC()

	result := domain.Trip{
		ID:         uuid.New(),
		UserID:     input.UserID,
		DriverID:   input.DriverID,
		StartPoint: input.StartPoint,
		EndPoint:   input.EndPoint,
		Price:      input.Price,
		Status:     domain.TripStatusActive,
		StartedAt:  now,
	}

	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		return s.repository.Create(txCtx, result)
	})
	if err != nil {
		return domain.Trip{}, err
	}

	if s.metrics != nil {
		s.metrics.TripsCreated(ctx)
	}
	if s.logger != nil {
		s.logger.InfoContext(ctx, "поездка создана", "trip_id", result.ID.String())
	}

	return result, nil
}

func (s *Service) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.Trip, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *Service) Finish(
	ctx context.Context,
	id uuid.UUID,
) (domain.Trip, error) {
	var result domain.Trip

	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		var err error

		result, err = s.repository.Finish(
			txCtx,
			id,
			time.Now().UTC(),
		)

		return err
	})
	if err != nil {
		return domain.Trip{}, err
	}

	if s.metrics != nil {
		s.metrics.TripsCompleted(ctx)
	}
	if s.logger != nil {
		s.logger.InfoContext(ctx, "поездка завершена", "trip_id", result.ID.String())
	}

	return result, nil
}

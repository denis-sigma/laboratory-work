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
}

func NewService(
	repository TripRepository,
	txManager TxManager,
	metrics Metrics,
) *Service {
	return &Service{
		repository: repository,
		txManager:  txManager,
		metrics:    metrics,
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

	return result, nil
}

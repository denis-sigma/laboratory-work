package trip

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/denis-sigma/laboratory-work/internal/database"
)

type Service struct {
	repository TripRepository
	txManager  database.TxManager
}

func NewService(
	repository TripRepository,
	txManager database.TxManager,
) *Service {
	return &Service{
		repository: repository,
		txManager:  txManager,
	}
}

type CreateTripInput struct {
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint Coordinates
	EndPoint   Coordinates
	Price      int64
}

func (s *Service) Create(
	ctx context.Context,
	input CreateTripInput,
) (Trip, error) {
	now := time.Now().UTC()

	result := Trip{
		ID:         uuid.New(),
		UserID:     input.UserID,
		DriverID:   input.DriverID,
		StartPoint: input.StartPoint,
		EndPoint:   input.EndPoint,
		Price:      input.Price,
		Status:     TripStatusActive,
		StartedAt:  now,
	}

	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		return s.repository.Create(txCtx, result)
	})
	if err != nil {
		return Trip{}, err
	}

	return result, nil
}

func (s *Service) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (Trip, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *Service) Finish(
	ctx context.Context,
	id uuid.UUID,
) (Trip, error) {
	var result Trip

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
		return Trip{}, err
	}

	return result, nil
}

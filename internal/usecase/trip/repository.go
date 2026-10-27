package trip

import (
	"context"
	"time"

	"github.com/google/uuid"

	domain "github.com/denis-sigma/laboratory-work/internal/domain/trip"
)

//go:generate mockgen -source=repository.go -destination=repository_mock_test.go -package=trip
type TripRepository interface {
	Create(ctx context.Context, trip domain.Trip) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.Trip, error)
	Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (domain.Trip, error)
}

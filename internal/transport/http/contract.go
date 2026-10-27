package httptransport

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/denis-sigma/laboratory-work/internal/domain/trip"
	usecase "github.com/denis-sigma/laboratory-work/internal/usecase/trip"
)

//go:generate mockgen -source=contract.go -destination=trip_service_mock_test.go -package=httptransport
type TripService interface {
	Create(ctx context.Context, input usecase.CreateTripInput) (domain.Trip, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Trip, error)
	Finish(ctx context.Context, id uuid.UUID) (domain.Trip, error)
}

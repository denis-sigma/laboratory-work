package trip

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type TripRepository interface {
	Create(ctx context.Context, trip Trip) error
	GetByID(ctx context.Context, id uuid.UUID) (Trip, error)
	Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (Trip, error)
}
package trip

import "context"

type Metrics interface {
	TripsCreated(ctx context.Context)
	TripsCompleted(ctx context.Context)
}

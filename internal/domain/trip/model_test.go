package trip

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTripFinish(t *testing.T) {
	finishTime := time.Date(2026, 10, 27, 12, 0, 0, 0, time.FixedZone("UTC+5", 5*60*60))

	tests := []struct {
		name         string
		status       TripStatus
		wantErr      error
		wantFinished bool
		wantStatus   TripStatus
	}{
		{
			name:         "активную поездку можно завершить",
			status:       TripStatusActive,
			wantStatus:   TripStatusCompleted,
			wantFinished: true,
		},
		{
			name:       "завершённую поездку нельзя завершить повторно",
			status:     TripStatusCompleted,
			wantErr:    ErrTripCompleted,
			wantStatus: TripStatusCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trip := Trip{
				ID:     uuid.New(),
				Status: tt.status,
			}

			err := trip.Finish(finishTime)
			if err != tt.wantErr {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}

			if trip.Status != tt.wantStatus {
				t.Fatalf("status = %s, want %s", trip.Status, tt.wantStatus)
			}

			if (trip.FinishedAt != nil) != tt.wantFinished {
				t.Fatalf("FinishedAt present = %t, want %t", trip.FinishedAt != nil, tt.wantFinished)
			}

			if tt.wantFinished && !trip.FinishedAt.Equal(finishTime.UTC()) {
				t.Fatalf("FinishedAt = %s, want %s", trip.FinishedAt, finishTime.UTC())
			}
		})
	}
}

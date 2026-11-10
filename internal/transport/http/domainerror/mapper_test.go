package domainerror

import (
	"errors"
	"testing"

	"github.com/denis-sigma/laboratory-work/internal/domain/trip"
)

func TestMap(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int32
		code   string
	}{
		{
			name:   "trip not found",
			err:    trip.ErrTripNotFound,
			status: 404,
			code:   "trip_not_found",
		},
		{
			name:   "driver busy",
			err:    trip.ErrDriverBusy,
			status: 409,
			code:   "driver_busy",
		},
		{
			name:   "trip completed",
			err:    trip.ErrTripCompleted,
			status: 409,
			code:   "trip_completed",
		},
		{
			name:   "unknown error",
			err:    errors.New("unknown error"),
			status: 500,
			code:   "internal_error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Map(test.err)

			if got.Status != test.status {
				t.Errorf("status = %d, want %d", got.Status, test.status)
			}

			if got.Code != test.code {
				t.Errorf("code = %q, want %q", got.Code, test.code)
			}
		})
	}
}

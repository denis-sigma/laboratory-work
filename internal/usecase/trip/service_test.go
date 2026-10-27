package trip

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domain "github.com/denis-sigma/laboratory-work/internal/domain/trip"
)

func TestServiceCreate(t *testing.T) {
	ctrl := gomock.NewController(t)

	repository := NewMockTripRepository(ctrl)
	TxManager := NewMockTxManager(ctrl)

	service := NewService(repository, TxManager, nil)

	input := CreateTripInput{
		UserID:   uuid.MustParse("5cb72c04-7650-45c9-a79b-bcdba0631e0c"),
		DriverID: uuid.MustParse("8860b315-ec86-42eb-a17c-7c163d721ff5"),
		StartPoint: domain.Coordinates{
			Latitude:  59.9398,
			Longitude: 30.3146,
		},
		EndPoint: domain.Coordinates{
			Latitude:  59.9290,
			Longitude: 30.3626,
		},
		Price: 1450,
	}

	repository.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(nil)

	TxManager.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			ctx context.Context,
			fn func(context.Context) error,
		) error {
			return fn(ctx)
		})

	result, err := service.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("ожидалось успешное создание, получена ошибка: %v", err)
	}

	if result.UserID != input.UserID {
		t.Errorf("UserID = %s, want %s", result.UserID, input.UserID)
	}

	if result.DriverID != input.DriverID {
		t.Errorf("DriverID = %s, want %s", result.DriverID, input.DriverID)
	}

	if result.Status != domain.TripStatusActive {
		t.Errorf("Status = %s, want %s", result.Status, domain.TripStatusActive)
	}
}

func TestServiceCreateRepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)

	repository := NewMockTripRepository(ctrl)
	txManager := NewMockTxManager(ctrl)
	service := NewService(repository, txManager, nil)

	repositoryError := errors.New("ошибка сохранения поездки")

	repository.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(repositoryError)

	txManager.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			ctx context.Context,
			fn func(context.Context) error,
		) error {
			return fn(ctx)
		})

	_, err := service.Create(context.Background(), CreateTripInput{})

	if !errors.Is(err, repositoryError) {
		t.Fatalf("получена ошибка %v, ожидалась %v", err, repositoryError)
	}
}

func TestServiceFinish(t *testing.T) {
	ctrl := gomock.NewController(t)

	repository := NewMockTripRepository(ctrl)
	txManager := NewMockTxManager(ctrl)
	service := NewService(repository, txManager, nil)

	tripID := uuid.MustParse("5fc5e51f-7456-49ed-a73d-daf5912823f3")
	finishedTrip := domain.Trip{
		ID:     tripID,
		Status: domain.TripStatusCompleted,
	}

	repository.EXPECT().
		Finish(gomock.Any(), tripID, gomock.Any()).
		Return(finishedTrip, nil)

	txManager.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			ctx context.Context,
			fn func(context.Context) error,
		) error {
			return fn(ctx)
		})

	result, err := service.Finish(context.Background(), tripID)
	if err != nil {
		t.Fatalf("ожидалось успешное завершение, получена ошибка: %v", err)
	}

	if result.ID != tripID {
		t.Errorf("ID = %s, want %s", result.ID, tripID)
	}

	if result.Status != domain.TripStatusCompleted {
		t.Errorf("Status = %s, want %s", result.Status, domain.TripStatusCompleted)
	}
}

func TestServiceGetByID(t *testing.T) {
	ctrl := gomock.NewController(t)

	repository := NewMockTripRepository(ctrl)
	txManager := NewMockTxManager(ctrl)
	service := NewService(repository, txManager, nil)

	tripID := uuid.MustParse("5fc5e51f-7456-49ed-a73d-daf5912823f3")
	expectedTrip := domain.Trip{
		ID:     tripID,
		Status: domain.TripStatusActive,
	}

	repository.EXPECT().
		GetByID(gomock.Any(), tripID).
		Return(expectedTrip, nil)

	result, err := service.GetByID(context.Background(), tripID)
	if err != nil {
		t.Fatalf("ожидалось успешное получение, получена ошибка: %v", err)
	}

	if result.ID != expectedTrip.ID {
		t.Errorf("ID = %s, want %s", result.ID, expectedTrip.ID)
	}

	if result.Status != expectedTrip.Status {
		t.Errorf("Status = %s, want %s", result.Status, expectedTrip.Status)
	}
}

//go:build integration

package trip

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/denis-sigma/laboratory-work/internal/domain/trip"
	"github.com/denis-sigma/laboratory-work/internal/infra/postgres/transaction"
)

func newIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("переменная DATABASE_URL не установлена")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("не удалось создать пул: %v", err)
	}

	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("не удалось подключиться к базе: %v", err)
	}

	return pool
}

func TestPostgresTripRepositoryCreate(t *testing.T) {
	pool := newIntegrationPool(t)
	repository := NewPostgresTripRepository(pool, 3*time.Second)

	expected := domain.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: domain.Coordinates{Latitude: 59.9398, Longitude: 30.3146},
		EndPoint:   domain.Coordinates{Latitude: 59.9290, Longitude: 30.3626},
		Price:      1450,
		Status:     domain.TripStatusActive,
		StartedAt:  time.Now().UTC(),
	}

	t.Cleanup(func() {
		_, err := pool.Exec(context.Background(),
			"DELETE FROM trips WHERE id = $1",
			expected.ID,
		)
		if err != nil {
			t.Errorf("не удалось удалить тестовую поездку: %v", err)
		}
	})

	if err := repository.Create(context.Background(), expected); err != nil {
		t.Fatalf("не удалось создать поездку: %v", err)
	}

	actual, err := repository.GetByID(context.Background(), expected.ID)
	if err != nil {
		t.Fatalf("не удалось получить созданную поездку: %v", err)
	}

	if actual.ID != expected.ID {
		t.Errorf("ID = %s, want %s", actual.ID, expected.ID)
	}

	if actual.DriverID != expected.DriverID {
		t.Errorf("DriverID = %s, want %s", actual.DriverID, expected.DriverID)
	}

	if actual.Status != domain.TripStatusActive {
		t.Errorf("Status = %s, want %s", actual.Status, domain.TripStatusActive)
	}
}

func TestPostgresTripRepositoryFinish(t *testing.T) {
	pool := newIntegrationPool(t)
	repository := NewPostgresTripRepository(pool, 3*time.Second)

	tripID := uuid.New()
	expected := domain.Trip{
		ID:         tripID,
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: domain.Coordinates{Latitude: 59.9398, Longitude: 30.3146},
		EndPoint:   domain.Coordinates{Latitude: 59.9290, Longitude: 30.3626},
		Price:      1450,
		Status:     domain.TripStatusActive,
		StartedAt:  time.Now().UTC(),
	}

	t.Cleanup(func() {
		_, err := pool.Exec(
			context.Background(),
			"DELETE FROM trips WHERE id = $1",
			tripID,
		)
		if err != nil {
			t.Errorf("не удалось удалить тестовую поездку: %v", err)
		}
	})

	if err := repository.Create(context.Background(), expected); err != nil {
		t.Fatalf("не удалось создать поездку: %v", err)
	}

	finishedAt := time.Now().UTC()

	result, err := repository.Finish(
		context.Background(),
		tripID,
		finishedAt,
	)
	if err != nil {
		t.Fatalf("не удалось завершить поездку: %v", err)
	}

	if result.Status != domain.TripStatusCompleted {
		t.Errorf("Status = %s, want %s",
			result.Status,
			domain.TripStatusCompleted,
		)
	}

	if result.FinishedAt == nil {
		t.Fatal("FinishedAt не должен быть nil")
	}
}

func TestPostgresTripRepositoryRejectsBusyDriver(t *testing.T) {
	pool := newIntegrationPool(t)
	repository := NewPostgresTripRepository(pool, 3*time.Second)

	driverID := uuid.New()
	firstTripID := uuid.New()
	secondTripID := uuid.New()

	t.Cleanup(func() {
		_, err := pool.Exec(
			context.Background(),
			"DELETE FROM trips WHERE driver_id = $1",
			driverID,
		)
		if err != nil {
			t.Errorf("не удалось удалить тестовые поездки: %v", err)
		}
	})

	makeTrip := func(id uuid.UUID) domain.Trip {
		return domain.Trip{
			ID:         id,
			UserID:     uuid.New(),
			DriverID:   driverID,
			StartPoint: domain.Coordinates{Latitude: 59.9398, Longitude: 30.3146},
			EndPoint:   domain.Coordinates{Latitude: 59.9290, Longitude: 30.3626},
			Price:      1450,
			Status:     domain.TripStatusActive,
			StartedAt:  time.Now().UTC(),
		}
	}

	if err := repository.Create(context.Background(), makeTrip(firstTripID)); err != nil {
		t.Fatalf("первая поездка не создалась: %v", err)
	}

	err := repository.Create(context.Background(), makeTrip(secondTripID))
	if !errors.Is(err, domain.ErrDriverBusy) {
		t.Fatalf("ошибка = %v, ожидалась ErrDriverBusy", err)
	}
}

func TestPostgresTripRepositoryCreateRollsBackOnHistoryError(t *testing.T) {
	pool := newIntegrationPool(t)
	repository := NewPostgresTripRepository(pool, 3*time.Second)
	txManager := transaction.NewTxManager(pool)

	expected := domain.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: domain.Coordinates{Latitude: 59.9398, Longitude: 30.3146},
		EndPoint:   domain.Coordinates{Latitude: 59.9290, Longitude: 30.3626},
		Price:      1450,
		Status:     domain.TripStatusActive,
		StartedAt:  time.Now().UTC(),
	}

	err := txManager.Do(context.Background(), func(ctx context.Context) error {
		if err := repository.Create(ctx, expected); err != nil {
			return err
		}

		executor := transaction.GetExecutor(ctx, pool)

		_, err := executor.Exec(
			ctx,
			`INSERT INTO trip_status_history
				(trip_id, from_status, to_status, reason)
				VALUES ($1, NULL, 'invalid', 'ошибка для проверки rollback')`,
			expected.ID,
		)
		return err
	})

	if err == nil {
		t.Fatal("ожидалась ошибка вставки истории")
	}

	_, err = repository.GetByID(context.Background(), expected.ID)
	if !errors.Is(err, domain.ErrTripNotFound) {
		t.Fatalf(
			"поездка осталась в базе после rollback: %v",
			err,
		)
	}
}

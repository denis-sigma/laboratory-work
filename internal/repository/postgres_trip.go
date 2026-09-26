package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/denis-sigma/laboratory-work/internal/database"
	"github.com/denis-sigma/laboratory-work/internal/trip"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresTripRepository struct {
	pool         *pgxpool.Pool
	builder      squirrel.StatementBuilderType
	queryTimeout time.Duration
}

func NewPostgresTripRepository(
	pool *pgxpool.Pool,
	queryTimeout time.Duration,
) *PostgresTripRepository {
	return &PostgresTripRepository{
		pool:         pool,
		queryTimeout: queryTimeout,
		builder: squirrel.StatementBuilder.
			PlaceholderFormat(squirrel.Dollar),
	}
}

var _ trip.TripRepository = (*PostgresTripRepository)(nil)

func (r *PostgresTripRepository) Create(
	ctx context.Context,
	t trip.Trip,
) error {

	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	executor := database.GetExecutor(queryCtx, r.pool)

	query, args, err := r.builder.
		Insert("trips").
		Columns(
			"id",
			"user_id",
			"driver_id",
			"start_latitude",
			"start_longitude",
			"end_latitude",
			"end_longitude",
			"price",
			"status",
			"started_at",
		).
		Values(
			t.ID,
			t.UserID,
			t.DriverID,
			t.StartPoint.Latitude,
			t.StartPoint.Longitude,
			t.EndPoint.Latitude,
			t.EndPoint.Longitude,
			t.Price,
			t.Status,
			t.StartedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("сборка запроса создания поездки: %w", err)
	}

	_, err = executor.Exec(queryCtx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return trip.ErrDriverBusy
		}

		return fmt.Errorf("создание поездки: %w", err)
	}

	historyQuery, historyArgs, err := r.builder.
		Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(t.ID, nil, t.Status, "поездка создана").
		ToSql()
	if err != nil {
		return fmt.Errorf("сборка запроса истории поездки: %w", err)
	}

	if _, err := executor.Exec(queryCtx, historyQuery, historyArgs...); err != nil {
		return fmt.Errorf("сохранение истории поездки: %w", err)
	}

	return nil
}

func (r *PostgresTripRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (trip.Trip, error) {

	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	executor := database.GetExecutor(queryCtx, r.pool)

	query, args, err := r.builder.
		Select(
			"id",
			"user_id",
			"driver_id",
			"start_latitude",
			"start_longitude",
			"end_latitude",
			"end_longitude",
			"price",
			"status",
			"started_at",
			"finished_at",
		).
		From("trips").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return trip.Trip{}, fmt.Errorf("сборка запроса получения поездки: %w", err)
	}

	var result trip.Trip

	err = executor.QueryRow(queryCtx, query, args...).Scan(
		&result.ID,
		&result.UserID,
		&result.DriverID,
		&result.StartPoint.Latitude,
		&result.StartPoint.Longitude,
		&result.EndPoint.Latitude,
		&result.EndPoint.Longitude,
		&result.Price,
		&result.Status,
		&result.StartedAt,
		&result.FinishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return trip.Trip{}, trip.ErrTripNotFound
		}

		return trip.Trip{}, fmt.Errorf("получение поездки: %w", err)
	}

	return result, nil
}

func (r *PostgresTripRepository) Finish(
	ctx context.Context,
	id uuid.UUID,
	finishedAt time.Time,
) (trip.Trip, error) {

	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	executor := database.GetExecutor(queryCtx, r.pool)

	query, args, err := r.builder.
		Update("trips").
		Set("status", trip.TripStatusCompleted).
		Set("finished_at", finishedAt).
		Set("updated_at", finishedAt).
		Where(squirrel.Eq{
			"id":     id,
			"status": trip.TripStatusActive,
		}).
		ToSql()
	if err != nil {
		return trip.Trip{}, fmt.Errorf("сборка запроса завершения поездки: %w", err)
	}

	result, err := executor.Exec(queryCtx, query, args...)
	if err != nil {
		return trip.Trip{}, fmt.Errorf("завершение поездки: %w", err)
	}

	if result.RowsAffected() == 0 {
		_, err := r.GetByID(queryCtx, id)
		if errors.Is(err, trip.ErrTripNotFound) {
			return trip.Trip{}, trip.ErrTripNotFound
		}

		return trip.Trip{}, trip.ErrTripCompleted
	}

	historyQuery, historyArgs, err := r.builder.
		Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(
			id,
			trip.TripStatusActive,
			trip.TripStatusCompleted,
			"поездка завершена",
		).
		ToSql()
	if err != nil {
		return trip.Trip{}, fmt.Errorf(
			"сборка запроса истории завершения поездки: %w",
			err,
		)
	}

	if _, err := executor.Exec(queryCtx, historyQuery, historyArgs...); err != nil {
		return trip.Trip{}, fmt.Errorf(
			"сохранение истории завершения поездки: %w",
			err,
		)
	}

	return r.GetByID(queryCtx, id)
}

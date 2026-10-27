package trip

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/denis-sigma/laboratory-work/internal/domain/trip"
	"github.com/denis-sigma/laboratory-work/internal/infra/postgres/transaction"
	usecase "github.com/denis-sigma/laboratory-work/internal/usecase/trip"
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

var _ usecase.TripRepository = (*PostgresTripRepository)(nil)

func (r *PostgresTripRepository) Create(
	ctx context.Context,
	t domain.Trip,
) error {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	executor := transaction.GetExecutor(queryCtx, r.pool)

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
			return domain.ErrDriverBusy
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
) (domain.Trip, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	executor := transaction.GetExecutor(queryCtx, r.pool)

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
		return domain.Trip{}, fmt.Errorf("сборка запроса получения поездки: %w", err)
	}

	var result domain.Trip

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
			return domain.Trip{}, domain.ErrTripNotFound
		}

		return domain.Trip{}, fmt.Errorf("получение поездки: %w", err)
	}

	return result, nil
}

func (r *PostgresTripRepository) Finish(
	ctx context.Context,
	id uuid.UUID,
	finishedAt time.Time,
) (domain.Trip, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	executor := transaction.GetExecutor(queryCtx, r.pool)

	current, err := r.GetByID(queryCtx, id)
	if err != nil {
		return domain.Trip{}, err
	}

	if err := current.Finish(finishedAt); err != nil {
		return domain.Trip{}, err
	}

	query, args, err := r.builder.
		Update("trips").
		Set("status", current.Status).
		Set("finished_at", current.FinishedAt).
		Set("updated_at", finishedAt).
		Where(squirrel.Eq{
			"id":     id,
			"status": domain.TripStatusActive,
		}).
		ToSql()
	if err != nil {
		return domain.Trip{}, fmt.Errorf("сборка запроса завершения поездки: %w", err)
	}

	result, err := executor.Exec(queryCtx, query, args...)
	if err != nil {
		return domain.Trip{}, fmt.Errorf("завершение поездки: %w", err)
	}

	if result.RowsAffected() == 0 {
		_, err := r.GetByID(queryCtx, id)
		if errors.Is(err, domain.ErrTripNotFound) {
			return domain.Trip{}, domain.ErrTripNotFound
		}

		return domain.Trip{}, domain.ErrTripCompleted
	}

	historyQuery, historyArgs, err := r.builder.
		Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(
			id,
			domain.TripStatusActive,
			domain.TripStatusCompleted,
			"поездка завершена",
		).
		ToSql()
	if err != nil {
		return domain.Trip{}, fmt.Errorf(
			"сборка запроса истории завершения поездки: %w",
			err,
		)
	}

	if _, err := executor.Exec(queryCtx, historyQuery, historyArgs...); err != nil {
		return domain.Trip{}, fmt.Errorf(
			"сохранение истории завершения поездки: %w",
			err,
		)
	}

	return r.GetByID(queryCtx, id)
}

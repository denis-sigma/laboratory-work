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

const statusColumn = "status"

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
			statusColumn,
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

func (r *PostgresTripRepository) SavePosition(
	ctx context.Context,
	position domain.Position,
) (domain.Position, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	executor := transaction.GetExecutor(queryCtx, r.pool)

	query, args, err := r.builder.
		Insert("trip_positions").
		Columns(
			"trip_id",
			"latitude",
			"longitude",
			"recorded_at",
		).
		Select(
			squirrel.Select(
				"?",
				"?",
				"?",
				"?",
			).
				From("trips").
				Where(squirrel.Eq{
					"id":         position.TripID,
					statusColumn: domain.TripStatusActive,
				}),
		).
		Suffix("RETURNING id").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return domain.Position{}, fmt.Errorf(
			"сборка запроса сохранения координаты: %w",
			err,
		)
	}
	args = append([]any{
		position.TripID,
		position.Coordinates.Latitude,
		position.Coordinates.Longitude,
		position.RecordedAt,
	}, args...)

	err = executor.QueryRow(queryCtx, query, args...).Scan(&position.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			tripResult, tripErr := r.GetByID(queryCtx, position.TripID)
			if errors.Is(tripErr, domain.ErrTripNotFound) {
				return domain.Position{}, domain.ErrTripNotFound
			}
			if tripErr != nil {
				return domain.Position{}, fmt.Errorf(
					"проверка поездки перед сохранением координаты: %w",
					tripErr,
				)
			}
			if tripResult.Status != domain.TripStatusActive {
				return domain.Position{}, domain.ErrTripCompleted
			}
		}

		return domain.Position{}, fmt.Errorf(
			"сохранение координаты: %w",
			err,
		)
	}

	updateQuery, updateArgs, err := r.builder.
		Update("trips").
		Set("last_position_at", squirrel.Expr(
			"GREATEST(COALESCE(last_position_at, ?), ?)",
			position.RecordedAt,
			position.RecordedAt,
		)).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{
			"id":         position.TripID,
			statusColumn: domain.TripStatusActive,
		}).
		ToSql()
	if err != nil {
		return domain.Position{}, fmt.Errorf(
			"сборка запроса обновления времени координаты: %w",
			err,
		)
	}

	if _, err := executor.Exec(queryCtx, updateQuery, updateArgs...); err != nil {
		return domain.Position{}, fmt.Errorf(
			"обновление времени последней координаты: %w",
			err,
		)
	}

	return position, nil
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
			statusColumn,
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
		Set(statusColumn, current.Status).
		Set("finished_at", current.FinishedAt).
		Set("updated_at", finishedAt).
		Where(squirrel.Eq{
			"id":         id,
			statusColumn: domain.TripStatusActive,
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
		if err != nil {
			return domain.Trip{}, fmt.Errorf(
				"проверка поездки после завершения: %w",
				err,
			)
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

func (r *PostgresTripRepository) GetPositions(
	ctx context.Context,
	tripID uuid.UUID,
) ([]domain.Position, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	executor := transaction.GetExecutor(queryCtx, r.pool)

	query, args, err := r.builder.
		Select(
			"id",
			"trip_id",
			"latitude",
			"longitude",
			"recorded_at",
		).
		From("trip_positions").
		Where(squirrel.Eq{"trip_id": tripID}).
		OrderBy("recorded_at ASC", "id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf(
			"сборка запроса получения маршрута: %w",
			err,
		)
	}

	rows, err := executor.Query(queryCtx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("получение маршрута: %w", err)
	}
	defer rows.Close()

	var positions []domain.Position

	for rows.Next() {
		var position domain.Position

		if err := rows.Scan(
			&position.ID,
			&position.TripID,
			&position.Coordinates.Latitude,
			&position.Coordinates.Longitude,
			&position.RecordedAt,
		); err != nil {
			return nil, fmt.Errorf("чтение точки маршрута: %w", err)
		}

		positions = append(positions, position)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("обход точек маршрута: %w", err)
	}

	return positions, nil
}

func (r *PostgresTripRepository) FindTripsForPositionRequest(
	ctx context.Context,
	staleAfter time.Duration,
) ([]domain.Trip, error) {
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
			statusColumn,
			"started_at",
			"finished_at",
		).
		From("trips").
		Where(squirrel.Expr(
			"status = ? AND COALESCE(last_position_at, started_at) < CURRENT_TIMESTAMP - ?::interval",
			domain.TripStatusActive,
			staleAfter.String(),
		)).
		OrderBy("COALESCE(last_position_at, started_at) ASC", "id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf(
			"сборка запроса выборки устаревших поездок: %w",
			err,
		)
	}

	rows, err := executor.Query(queryCtx, query, args...)
	if err != nil {
		return nil, fmt.Errorf(
			"выборка устаревших поездок: %w",
			err,
		)
	}
	defer rows.Close()

	var trips []domain.Trip

	for rows.Next() {
		var current domain.Trip

		if err := rows.Scan(
			&current.ID,
			&current.UserID,
			&current.DriverID,
			&current.StartPoint.Latitude,
			&current.StartPoint.Longitude,
			&current.EndPoint.Latitude,
			&current.EndPoint.Longitude,
			&current.Price,
			&current.Status,
			&current.StartedAt,
			&current.FinishedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"чтение устаревшей поездки: %w",
				err,
			)
		}

		trips = append(trips, current)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"обход устаревших поездок: %w",
			err,
		)
	}

	return trips, nil
}

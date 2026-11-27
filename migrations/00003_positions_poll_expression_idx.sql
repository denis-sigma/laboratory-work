-- +goose Up

DROP INDEX IF EXISTS trips_position_poll_idx;

CREATE INDEX trips_position_poll_idx
    ON trips (status, (COALESCE(last_position_at, started_at)));

-- +goose Down

DROP INDEX trips_position_poll_idx;

CREATE INDEX trips_position_poll_idx
    ON trips (status, last_position_at, started_at);

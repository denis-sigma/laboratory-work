package container

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/denis-sigma/laboratory-work/internal/config"
	api "github.com/denis-sigma/laboratory-work/internal/generated"
	postgres "github.com/denis-sigma/laboratory-work/internal/infra/postgres"
	transaction "github.com/denis-sigma/laboratory-work/internal/infra/postgres/transaction"
	"github.com/denis-sigma/laboratory-work/internal/infra/telemetry"
	repositorytrip "github.com/denis-sigma/laboratory-work/internal/repository/trip"
	httptransport "github.com/denis-sigma/laboratory-work/internal/transport/http"
	middleware "github.com/denis-sigma/laboratory-work/internal/transport/http/middleware"
	usecase "github.com/denis-sigma/laboratory-work/internal/usecase/trip"
)

type Container struct {
	Pool           *pgxpool.Pool
	Handler        http.Handler
	Logger         *slog.Logger
	TracerShutdown func(context.Context) error
	MeterShutdown  func(context.Context) error
}

func New(ctx context.Context, cfg config.Config) (*Container, error) {
	logger, err := telemetry.NewLogger(cfg.LogLevel)
	if err != nil {
		return nil, err
	}

	tracerShutdown, err := telemetry.InitTracer(
		ctx,
		cfg.OTel.ServiceName,
		cfg.OTel.Endpoint,
	)
	if err != nil {
		return nil, err
	}

	meterShutdown, err := telemetry.InitMeter(
		ctx,
		cfg.OTel.ServiceName,
		cfg.OTel.Endpoint,
	)
	if err != nil {
		_ = tracerShutdown(ctx)
		return nil, err
	}

	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}

	txManager := transaction.NewTxManager(pool)

	tripMetrics, err := telemetry.NewTripMetrics()
	if err != nil {
		pool.Close()
		return nil, err
	}

	tripRepository := repositorytrip.NewPostgresTripRepository(
		pool,
		cfg.Database.QueryTimeout,
	)

	tripService := usecase.NewService(
		tripRepository,
		txManager,
		tripMetrics,
	)

	handler := httptransport.NewHandler(
		pool,
		cfg.Database.QueryTimeout,
		tripService,
	)

	httpHandler := api.HandlerWithOptions(
		handler,
		api.ChiServerOptions{
			ErrorHandlerFunc: httptransport.OpenAPIErrorHandler,
		},
	)

	metricsMiddleware, err := middleware.Metrics()
	if err != nil {
		_ = meterShutdown(ctx)
		_ = tracerShutdown(ctx)
		pool.Close()
		return nil, err
	}

	metricsHandler := metricsMiddleware(httpHandler)
	loggedHandler := middleware.Logging(logger)(metricsHandler)

	tracedHandler := otelhttp.NewHandler(
		loggedHandler,
		"trip-service",
		otelhttp.WithSpanNameFormatter(func(
			_ string,
			r *http.Request,
		) string {
			return r.Method + " " + routePattern(r.URL.Path)
		}),
	)

	recoveredHandler := middleware.Recover(logger)(tracedHandler)

	return &Container{
		Pool:           pool,
		Handler:        recoveredHandler,
		Logger:         logger,
		TracerShutdown: tracerShutdown,
		MeterShutdown:  meterShutdown,
	}, nil
}

func routePattern(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) == 4 &&
		parts[0] == "api" &&
		parts[1] == "v1" &&
		parts[2] == "trips" {
		if _, err := uuid.Parse(parts[3]); err == nil {
			return "/api/v1/trips/{tripId}"
		}
	}

	if len(parts) == 5 &&
		parts[0] == "api" &&
		parts[1] == "v1" &&
		parts[2] == "trips" &&
		parts[4] == "finish" {
		if _, err := uuid.Parse(parts[3]); err == nil {
			return "/api/v1/trips/{tripId}/finish"
		}
	}

	return path
}

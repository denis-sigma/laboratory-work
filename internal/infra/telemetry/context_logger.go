package telemetry

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

type ContextLogger struct {
	logger *slog.Logger
}

func NewContextLogger(logger *slog.Logger) *ContextLogger {
	return &ContextLogger{logger: logger}
}

func (l *ContextLogger) InfoContext(ctx context.Context, msg string, args ...any) {
	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.IsValid() {
		args = append(args,
			"trace_id", spanContext.TraceID().String(),
			"span_id", spanContext.SpanID().String(),
		)
	}

	l.logger.InfoContext(ctx, msg, args...)
}

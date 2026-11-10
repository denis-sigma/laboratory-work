package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}

	return w.ResponseWriter.Write(body)
}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			writer := &statusWriter{ResponseWriter: w}
			defer func() {
				if writer.status == 0 {
					writer.status = http.StatusInternalServerError
				}
				logRequest(logger, r, writer.status, time.Since(start))
			}()

			next.ServeHTTP(writer, r)
		})
	}
}

func logRequest(logger *slog.Logger, r *http.Request, status int, elapsed time.Duration) {
	route := r.URL.Path
	if routeContext := chi.RouteContext(r.Context()); routeContext != nil && routeContext.RoutePattern() != "" {
		route = routeContext.RoutePattern()
	}
	spanContext := trace.SpanContextFromContext(r.Context())
	fields := []any{"method", r.Method, "route", route, "status", status, "duration_ms", elapsed.Milliseconds()}
	if spanContext.IsValid() {
		fields = append(fields, "trace_id", spanContext.TraceID().String(), "span_id", spanContext.SpanID().String())
	}
	level := slog.LevelInfo
	if status >= http.StatusInternalServerError {
		level = slog.LevelError
	}
	logger.Log(r.Context(), level, "HTTP запрос", fields...)
}

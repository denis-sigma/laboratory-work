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

			next.ServeHTTP(writer, r)

			route := r.URL.Path
			if routeContext := chi.RouteContext(r.Context()); routeContext != nil &&
				routeContext.RoutePattern() != "" {
				route = routeContext.RoutePattern()
			}

			spanContext := trace.SpanContextFromContext(r.Context())

			logFields := []any{
				"method", r.Method,
				"route", route,
				"status", writer.status,
				"duration_ms", time.Since(start).Milliseconds(),
			}

			if spanContext.IsValid() {
				logFields = append(
					logFields,
					"trace_id", spanContext.TraceID().String(),
					"span_id", spanContext.SpanID().String(),
				)
			}

			level := slog.LevelInfo
			if writer.status >= http.StatusInternalServerError {
				level = slog.LevelError
			}

			logger.Log(r.Context(), level, "HTTP запрос", logFields...)
		})
	}
}

package middleware

import (
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func Metrics() (func(http.Handler) http.Handler, error) {
	meter := otel.Meter("tripgo/http")

	duration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	activeRequests, err := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			route := routePattern(r.URL.Path)

			attributes := metric.WithAttributes(
				attribute.String("http.request.method", r.Method),
				attribute.String("http.route", route),
			)

			activeRequests.Add(r.Context(), 1, attributes)

			writer := &statusWriter{ResponseWriter: w}
			defer func() {
				if writer.status == 0 {
					writer.status = http.StatusInternalServerError
				}
				activeRequests.Add(r.Context(), -1, attributes)
				duration.Record(r.Context(), time.Since(start).Seconds(), metric.WithAttributes(
					attribute.String("http.request.method", r.Method),
					attribute.String("http.route", route),
					attribute.Int("http.response.status_code", writer.status),
				))
			}()
			next.ServeHTTP(writer, r)
		})
	}, nil
}

func routePattern(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) == 4 &&
		parts[0] == "api" &&
		parts[1] == "v1" &&
		parts[2] == "trips" {
		return "/api/v1/trips/{tripId}"
	}

	if len(parts) == 5 &&
		parts[0] == "api" &&
		parts[1] == "v1" &&
		parts[2] == "trips" &&
		parts[4] == "finish" {
		return "/api/v1/trips/{tripId}/finish"
	}

	return path
}

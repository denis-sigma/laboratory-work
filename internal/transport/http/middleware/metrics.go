package middleware

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	httptransport "github.com/denis-sigma/laboratory-work/internal/transport/http"
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
			route := httptransport.RoutePattern(r.URL.Path)

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

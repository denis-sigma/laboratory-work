package httptransport

import "testing"

const unknownRoute = "/unknown"

func TestRoutePattern(t *testing.T) {
	tests := map[string]string{
		"/health":                               "/health",
		"/ready":                                "/ready",
		"/api/v1/trips":                         "/api/v1/trips",
		"/api/v1/trips/not-a-uuid":              "/api/v1/trips/{tripId}",
		"/api/v1/trips/not-a-uuid/finish":       "/api/v1/trips/{tripId}/finish",
		"/unknown/one":                          unknownRoute,
		"/api/v1/trips/not-a-uuid/other":        unknownRoute,
		"/api/v1/trips/not-a-uuid/finish/extra": unknownRoute,
	}

	for path, want := range tests {
		t.Run(path, func(t *testing.T) {
			if got := RoutePattern(path); got != want {
				t.Errorf("RoutePattern(%q) = %q, want %q", path, got, want)
			}
		})
	}
}

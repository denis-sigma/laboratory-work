package httptransport

import "strings"

const (
	apiSegment     = "api"
	versionSegment = "v1"
	tripsSegment   = "trips"
)

// RoutePattern returns a low-cardinality route name for telemetry.
func RoutePattern(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) == 1 && parts[0] == "health" {
		return "/health"
	}
	if len(parts) == 1 && parts[0] == "ready" {
		return "/ready"
	}
	if len(parts) == 3 && parts[0] == apiSegment && parts[1] == versionSegment && parts[2] == tripsSegment {
		return "/api/v1/trips"
	}
	if len(parts) == 4 && parts[0] == apiSegment && parts[1] == versionSegment && parts[2] == tripsSegment {
		return "/api/v1/trips/{tripId}"
	}
	if len(parts) == 5 && parts[0] == apiSegment && parts[1] == versionSegment && parts[2] == tripsSegment && parts[4] == "finish" {
		return "/api/v1/trips/{tripId}/finish"
	}

	return "/unknown"
}

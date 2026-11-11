package positions

import "time"

type Metrics interface {
	StaleChecks(result string)
	QueueSize(size int)
	ActiveWorkers(delta int)
	Requests(result string)
	RequestDuration(duration time.Duration)
}

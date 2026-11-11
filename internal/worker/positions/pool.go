package positions

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

type PushGateway interface {
	RequestPosition(
		ctx context.Context,
		driverID uuid.UUID,
		tripID uuid.UUID,
	) error
}

type Pool struct {
	queue          *Queue
	gateway        PushGateway
	workers        int
	requestTimeout time.Duration
	logger         *slog.Logger
	metrics        Metrics
	group          sync.WaitGroup
}

func NewPool(
	queue *Queue,
	gateway PushGateway,
	workers int,
	requestTimeout time.Duration,
	logger *slog.Logger,
	metrics ...Metrics,
) *Pool {
	var poolMetrics Metrics
	if len(metrics) > 0 {
		poolMetrics = metrics[0]
	}
	return &Pool{
		queue:          queue,
		gateway:        gateway,
		workers:        workers,
		requestTimeout: requestTimeout,
		logger:         logger,
		metrics:        poolMetrics,
	}
}

func (p *Pool) Start() {
	for i := 0; i < p.workers; i++ {
		p.group.Add(1)

		go func() {
			defer p.group.Done()
			p.worker()
		}()
	}
}

func (p *Pool) Stop() {
	p.queue.Close()
	p.group.Wait()
}

func (p *Pool) worker() {
	for {
		task, ok := p.queue.Dequeue()
		if !ok {
			return
		}
		if !p.queue.IsCurrent(task) {
			continue
		}

		requestCtx, cancel := context.WithTimeout(
			context.Background(),
			p.requestTimeout,
		)
		started := time.Now()
		if p.metrics != nil {
			p.metrics.ActiveWorkers(1)
		}

		err := p.gateway.RequestPosition(
			requestCtx,
			task.DriverID,
			task.TripID,
		)

		cancel()
		if p.metrics != nil {
			p.metrics.ActiveWorkers(-1)
			p.metrics.RequestDuration(time.Since(started))
		}
		p.queue.DoneTask(task)
		if p.metrics != nil {
			if err != nil {
				p.metrics.Requests("error")
			} else {
				p.metrics.Requests("success")
			}
			p.metrics.QueueSize(p.queue.Len())
		}

		if err != nil && p.logger != nil {
			p.logger.Warn(
				"не удалось отправить запрос координаты",
				"trip_id", task.TripID.String(),
				"error", err,
			)
		}
	}
}

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
	workCtx        context.Context
	workCancel     context.CancelFunc
	stopOnce       sync.Once
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

func (p *Pool) Start(contexts ...context.Context) {
	ctx := context.Background()
	if len(contexts) > 0 && contexts[0] != nil {
		ctx = contexts[0]
	}
	p.workCtx, p.workCancel = context.WithCancel(ctx)
	for i := 0; i < p.workers; i++ {
		p.group.Add(1)

		go func() {
			defer p.group.Done()
			p.worker()
		}()
	}
}

func (p *Pool) Stop(contexts ...context.Context) {
	ctx := context.Background()
	if len(contexts) > 0 && contexts[0] != nil {
		ctx = contexts[0]
	}
	p.stopOnce.Do(func() { p.queue.Close() })
	done := make(chan struct{})
	go func() {
		p.group.Wait()
		close(done)
	}()
	select {
	case <-done:
		if p.workCancel != nil {
			p.workCancel()
		}
	case <-ctx.Done():
		if p.workCancel != nil {
			p.workCancel()
		}
		<-done
	}
}

func (p *Pool) worker() {
	for {
		task, ok := p.queue.Dequeue(p.workCtx)
		if !ok {
			return
		}
		if p.metrics != nil {
			p.metrics.QueueSize(p.queue.Len())
		}
		if !p.queue.IsCurrent(task) {
			if p.metrics != nil {
				p.metrics.Requests("cancelled")
			}
			continue
		}

		requestCtx, cancel := context.WithTimeout(
			p.workCtx,
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
		result := requestResult(err, p.workCtx)
		if p.metrics != nil {
			p.metrics.ActiveWorkers(-1)
			p.metrics.RequestDuration(time.Since(started), result)
		}
		p.queue.DoneTask(task)
		if p.metrics != nil {
			p.metrics.Requests(result)
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

func requestResult(err error, workCtx context.Context) string {
	if err == nil {
		return "success"
	}
	if workCtx != nil && workCtx.Err() != nil {
		return "cancelled"
	}
	return "error"
}

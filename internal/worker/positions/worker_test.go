package positions

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/goleak"

	trip "github.com/denis-sigma/laboratory-work/internal/domain/trip"
)

func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }

type testGateway struct {
	mu    sync.Mutex
	calls []Task
	err   error
}

type testTicker struct {
	ch chan time.Time
}

func (t *testTicker) C() <-chan time.Time { return t.ch }
func (t *testTicker) Stop()               {}

type testClock struct{ ticker *testTicker }

func (c testClock) NewTicker(time.Duration) Ticker { return c.ticker }

func (g *testGateway) RequestPosition(_ context.Context, driverID, tripID uuid.UUID) error {
	g.mu.Lock()
	g.calls = append(g.calls, Task{DriverID: driverID, TripID: tripID})
	g.mu.Unlock()
	return g.err
}

func TestPoolContinuesAfterGatewayError(t *testing.T) {
	queue := NewQueue(2)
	gateway := &testGateway{err: errors.New("push unavailable")}
	pool := NewPool(queue, gateway, 1, time.Second, nil)
	pool.Start()
	queue.Enqueue(Task{TripID: uuid.New()})
	queue.Enqueue(Task{TripID: uuid.New()})
	pool.Stop()

	if len(gateway.calls) != 2 {
		t.Fatalf("gateway calls = %d, want 2", len(gateway.calls))
	}
}

type testSource struct {
	called chan struct{}
}

func (s testSource) FindTripsForPositionRequest(context.Context, time.Duration) ([]trip.Trip, error) {
	close(s.called)
	return []trip.Trip{{ID: uuid.New(), DriverID: uuid.New()}}, nil
}

func TestScannerUsesInjectedClock(t *testing.T) {
	ticker := &testTicker{ch: make(chan time.Time, 1)}
	source := testSource{called: make(chan struct{})}
	scanner := NewScanner(source, NewQueue(1), time.Hour, time.Minute, nil)
	scanner.SetClock(testClock{ticker: ticker})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { scanner.Run(ctx); close(done) }()
	ticker.ch <- time.Now()

	select {
	case <-source.called:
	case <-time.After(time.Second):
		t.Fatal("scanner did not react to injected tick")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scanner did not stop")
	}
}

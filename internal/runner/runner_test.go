//go:build !wasip1

package runner

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunnerRunsSweepsAndStops(t *testing.T) {
	var runs atomic.Int64
	var failures atomic.Int64
	runner := New(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})), []Sweep{
		{
			Name:     "counting",
			Interval: 5 * time.Millisecond,
			Run: func(_ context.Context) error {
				runs.Add(1)
				return nil
			},
		},
		{
			Name:     "failing",
			Interval: 5 * time.Millisecond,
			Run: func(_ context.Context) error {
				failures.Add(1)
				return errors.New("sweep failed")
			},
		},
	})
	runner.Start(context.Background())

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runs.Load() >= 3 && failures.Load() >= 3 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if runs.Load() < 3 {
		t.Fatalf("counting sweep ran %d times, want >= 3", runs.Load())
	}
	// A failing sweep keeps running: the error is logged, not fatal to the loop.
	if failures.Load() < 3 {
		t.Fatalf("failing sweep ran %d times, want >= 3", failures.Load())
	}

	runner.Stop()
	after := runs.Load()
	time.Sleep(30 * time.Millisecond)
	if runs.Load() != after {
		t.Fatalf("sweep still running after Stop: %d -> %d", after, runs.Load())
	}
}

func TestRunnerStopCancelsInFlightSweep(t *testing.T) {
	started := make(chan struct{})
	unblocked := make(chan struct{})
	runner := New(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})), []Sweep{
		{
			Name:     "blocking",
			Interval: time.Millisecond,
			Run: func(ctx context.Context) error {
				select {
				case started <- struct{}{}:
				default:
				}
				<-ctx.Done()
				close(unblocked)
				return ctx.Err()
			},
		},
	})
	runner.Start(context.Background())
	<-started

	done := make(chan struct{})
	go func() {
		runner.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("Stop did not return; in-flight sweep was not cancelled")
	}
	select {
	case <-unblocked:
	default:
		t.Fatalf("in-flight sweep never observed cancellation")
	}
}

func TestStopWithoutStartIsSafe(t *testing.T) {
	runner := New(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})), []Sweep{})
	runner.Stop()
}

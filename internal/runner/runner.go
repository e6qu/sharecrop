//go:build !wasip1

// Package runner is the host-only lifecycle runner: a set of periodic
// background sweeps (reservation and task expiry, privacy retention,
// rate-limit and MCP-session eviction, webhook dispatch) that run inside
// cmd/sharecrop serve. It is imported ONLY by cmd/sharecrop — never by the
// WASI guest or the browser build — and receives its sweeps as closures, so
// the package itself has no store or database dependencies.
package runner

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Sweep is one named periodic job. Run returns an error to be logged; a
// failing sweep never stops its loop, and the next tick tries again.
type Sweep struct {
	Name     string
	Interval time.Duration
	Run      func(ctx context.Context) error
}

// Runner drives one goroutine per sweep. Start launches the loops; Stop
// cancels them and waits for every in-flight sweep to finish.
type Runner struct {
	logger *slog.Logger
	sweeps []Sweep
	cancel context.CancelFunc
	group  sync.WaitGroup
}

func New(logger *slog.Logger, sweeps []Sweep) *Runner {
	return &Runner{logger: logger, sweeps: sweeps}
}

// Start launches one ticker loop per sweep. The loops stop when Stop is
// called or when the parent context is cancelled.
func (r *Runner) Start(ctx context.Context) {
	runCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	for _, sweep := range r.sweeps {
		r.group.Add(1)
		go r.loop(runCtx, sweep)
	}
}

// Stop cancels every sweep loop and blocks until they have all returned,
// including a sweep run that is currently in flight (the run context is
// cancelled, so a blocked sweep unwinds promptly). Callers must invoke Stop
// before tearing down the resources the sweeps use (guest pool, pgx pool).
func (r *Runner) Stop() {
	if r.cancel == nil {
		return
	}
	r.cancel()
	r.group.Wait()
}

func (r *Runner) loop(ctx context.Context, sweep Sweep) {
	defer r.group.Done()
	ticker := time.NewTicker(sweep.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := sweep.Run(ctx); err != nil {
				// A cancelled context during shutdown is expected unwinding,
				// not a sweep failure worth alarming on.
				if ctx.Err() != nil {
					return
				}
				r.logger.Error("lifecycle sweep failed", "sweep", sweep.Name, "error", err)
			}
		}
	}
}

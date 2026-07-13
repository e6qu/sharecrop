//go:build !wasip1

package rpc

import (
	"context"
	"sync"

	"github.com/tetratelabs/wazero"
)

// Pool serves units of work from a fixed set of long-lived guest instances,
// reusing each across many units instead of instantiating a fresh one per call.
// That moves the guest's ~milliseconds startup cost off the request hot path: a
// pooled instance pays it once, then handles unit after unit over its stdin/
// stdout loop. Each instance is still driven by a single goroutine and touched
// by no other (see session), so pooling adds no shared-instance concurrency -
// concurrency comes from having several independent sessions, one checked out
// per in-flight unit of work.
type Pool struct {
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
	dispatch Dispatcher
	poolCtx  context.Context
	env      map[string]string
	size     int

	idle      chan *session
	startOnce sync.Once
}

// NewPool compiles the guest WASM once and prepares a pool of the given size.
// poolCtx bounds the lifetime of every pooled instance (cancel it, or call
// Close, to tear them all down); the per-call context passed to Call bounds only
// that unit's store queries.
func NewPool(poolCtx context.Context, guestWASM []byte, dispatch Dispatcher, size int) (*Pool, error) {
	return NewPoolWithCache(poolCtx, guestWASM, dispatch, size, "")
}

// NewPoolWithCache is NewPool with an on-disk wazero compilation cache. When
// cacheDir is non-empty and pre-populated (see the wasi-precompile command), the
// guest's machine code is loaded from it instead of being compiled at startup,
// so serve does no build on boot. cacheDir must have been populated by the same
// binary on the same CPU architecture.
func NewPoolWithCache(poolCtx context.Context, guestWASM []byte, dispatch Dispatcher, size int, cacheDir string) (*Pool, error) {
	if size < 1 {
		size = 1
	}
	runtime, compiled, err := compileGuest(poolCtx, guestWASM, cacheDir)
	if err != nil {
		return nil, err
	}
	return &Pool{
		runtime:  runtime,
		compiled: compiled,
		dispatch: dispatch,
		poolCtx:  poolCtx,
		size:     size,
		idle:     make(chan *session, size),
	}, nil
}

// WithGuestEnv sets environment variables handed to every guest instance. It
// must be called before the first Call. It returns the pool for chaining.
func (p *Pool) WithGuestEnv(env map[string]string) *Pool {
	p.env = env
	return p
}

func (p *Pool) start() {
	p.startOnce.Do(func() {
		for i := 0; i < p.size; i++ {
			p.idle <- p.newSession()
		}
	})
}

func (p *Pool) newSession() *session {
	return newSession(p.poolCtx, p.runtime, p.compiled, p.env, p.dispatch)
}

// Call checks out an idle instance, runs one unit of work on it, and checks it
// back in. If the instance died servicing the unit, a fresh one is returned to
// the pool in its place so the pool stays at full size.
func (p *Pool) Call(ctx context.Context, method string, args []byte) ([]byte, error) {
	p.start()

	var s *session
	select {
	case s = <-p.idle:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	result, err := s.call(ctx, method, args)

	if s.alive() {
		p.idle <- s
	} else {
		p.idle <- p.newSession()
	}
	return result, err
}

// Close shuts every pooled instance down and releases the runtime. It waits for
// each in-flight unit of work to finish (an instance is drained only once it
// has been checked back in).
func (p *Pool) Close(ctx context.Context) error {
	p.start()
	for i := 0; i < p.size; i++ {
		_ = (<-p.idle).close()
	}
	return p.runtime.Close(ctx)
}

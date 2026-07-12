//go:build !wasip1

package rpc

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

// Dispatcher services one store call against a real store, returning the
// serialized result or a transport error. A generated host dispatcher for a
// store satisfies this by decoding args, calling the real store method, and
// encoding the result.
type Dispatcher func(ctx context.Context, method string, args []byte) ([]byte, error)

// Caller runs one unit of work (a store method or an HTTP request) through a
// guest and returns its serialized result. Both *Host (a fresh instance per
// call) and *Pool (a reused instance from a pool) satisfy it, so httpbridge and
// the tests can drive either.
type Caller interface {
	Call(ctx context.Context, method string, args []byte) ([]byte, error)
}

// session drives exactly one long-lived guest instance. Two goroutines run per
// session and nothing else ever touches the instance or its pipes: the runner
// goroutine owns the wazero instance (it alone calls InstantiateModule, which
// blocks until the guest's loop exits), and the driver goroutine owns both pipe
// ends (it alone reads the guest's stdout and writes the guest's stdin, so the
// two-write framing can never interleave). Callers reach the driver only through
// the requests channel. This keeps the Phase 1 safety property - one goroutine
// per instance, the pipes as the only cross-goroutine hand-off - even though the
// instance now serves many units of work.
type session struct {
	requests chan sessionRequest
	done     chan struct{}
	runErr   error
}

type sessionRequest struct {
	ctx      context.Context
	method   string
	args     []byte
	resultCh chan sessionResponse
}

type sessionResponse struct {
	result []byte
	err    error
}

func newSession(ctx context.Context, runtime wazero.Runtime, compiled wazero.CompiledModule, env map[string]string, dispatch Dispatcher) *session {
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	s := &session{
		requests: make(chan sessionRequest),
		done:     make(chan struct{}),
	}

	// Driver: the only goroutine that reads stdout and writes stdin.
	go func() {
		for req := range s.requests {
			req.resultCh <- driveUnit(req, stdinWriter, stdoutReader, dispatch)
		}
		// No more work: tell the guest's loop to exit and let it drain its stdin.
		_ = writeHostFrame(stdinWriter, hostFrame{Kind: "shutdown"})
		_ = stdinWriter.Close()
	}()

	// Runner: the only goroutine that touches the wazero instance.
	go func() {
		defer close(s.done)
		config := wazero.NewModuleConfig().
			WithStdin(stdinReader).
			WithStdout(stdoutWriter).
			WithArgs("guest").
			WithName("")
		for key, value := range env {
			config = config.WithEnv(key, value)
		}
		instance, err := runtime.InstantiateModule(ctx, compiled, config)
		if instance != nil {
			_ = instance.Close(ctx)
		}
		// Unblock the driver's next stdout read with EOF.
		_ = stdoutWriter.Close()
		s.runErr = normalizeExit(err)
	}()

	return s
}

// driveUnit sends one unit of work to the guest, services every store call it
// makes back through dispatch, and returns the guest's result. It runs on the
// driver goroutine, so it is the sole reader of stdout and writer of stdin.
func driveUnit(req sessionRequest, stdin io.Writer, stdout io.Reader, dispatch Dispatcher) sessionResponse {
	if err := writeHostFrame(stdin, hostFrame{Kind: "work", Method: req.method, Args: rawOrNull(req.args)}); err != nil {
		return sessionResponse{err: err}
	}
	for {
		frame, err := readGuestFrame(stdout)
		if err != nil {
			return sessionResponse{err: err}
		}
		switch frame.Kind {
		case "call":
			result, dispErr := dispatch(req.ctx, frame.Method, frame.Args)
			reply := hostFrame{Kind: "reply"}
			if dispErr != nil {
				reply.Error = dispErr.Error()
			} else {
				reply.Result = rawOrNull(result)
			}
			if err := writeHostFrame(stdin, reply); err != nil {
				return sessionResponse{err: err}
			}
		case "result":
			if frame.Error != "" {
				return sessionResponse{err: errors.New(frame.Error)}
			}
			return sessionResponse{result: append([]byte(nil), frame.Result...)}
		default:
			return sessionResponse{err: fmt.Errorf("unexpected guest frame %q", frame.Kind)}
		}
	}
}

// call runs one unit of work on this session and blocks for its result. It is
// safe to call from another goroutine, but only one call may be in flight per
// session at a time (the pool guarantees this by checking a session out).
func (s *session) call(ctx context.Context, method string, args []byte) ([]byte, error) {
	resultCh := make(chan sessionResponse, 1)
	select {
	case s.requests <- sessionRequest{ctx: ctx, method: method, args: args, resultCh: resultCh}:
	case <-s.done:
		return nil, errors.New("wasi session has exited")
	}
	resp := <-resultCh
	return resp.result, resp.err
}

// alive reports whether the session's guest is still running.
func (s *session) alive() bool {
	select {
	case <-s.done:
		return false
	default:
		return true
	}
}

// close shuts the session's guest down and waits for it to exit, returning the
// guest's run error if it exited abnormally.
func (s *session) close() error {
	close(s.requests)
	<-s.done
	return s.runErr
}

// Host owns a compiled guest module and a dispatcher for the store it serves.
// Each Call runs one unit of work in a fresh instance, so a Host is the
// instance-per-call shape used by the store dual-run tests and the route tests;
// production uses a Pool instead.
type Host struct {
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
	dispatch Dispatcher
	env      map[string]string
}

// NewHost compiles the guest WASM once and binds it to a dispatcher.
func NewHost(ctx context.Context, guestWASM []byte, dispatch Dispatcher) (*Host, error) {
	runtime, compiled, err := compileGuest(ctx, guestWASM)
	if err != nil {
		return nil, err
	}
	return &Host{runtime: runtime, compiled: compiled, dispatch: dispatch}, nil
}

// WithGuestEnv sets environment variables handed to every guest instance (via
// WASI env), for configuration a guest reads with os.Getenv - e.g. the access
// token secret a request handler needs. It returns the host for chaining.
func (h *Host) WithGuestEnv(env map[string]string) *Host {
	h.env = env
	return h
}

// Close releases the runtime and every module compiled against it.
func (h *Host) Close(ctx context.Context) error {
	return h.runtime.Close(ctx)
}

// Call runs one unit of work in a fresh guest instance and tears it down again.
func (h *Host) Call(ctx context.Context, method string, args []byte) ([]byte, error) {
	s := newSession(ctx, h.runtime, h.compiled, h.env, h.dispatch)
	result, err := s.call(ctx, method, args)
	closeErr := s.close()
	if err != nil {
		return nil, fmt.Errorf("run guest: %w", err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("run guest: %w", closeErr)
	}
	return result, nil
}

func compileGuest(ctx context.Context, guestWASM []byte) (wazero.Runtime, wazero.CompiledModule, error) {
	runtime := wazero.NewRuntime(ctx)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		_ = runtime.Close(ctx)
		return nil, nil, fmt.Errorf("instantiate wasi: %w", err)
	}
	compiled, err := runtime.CompileModule(ctx, guestWASM)
	if err != nil {
		_ = runtime.Close(ctx)
		return nil, nil, fmt.Errorf("compile guest: %w", err)
	}
	return runtime, compiled, nil
}

// normalizeExit treats a wasip1 command's exit(0) - which wazero surfaces as a
// zero-code ExitError - as success.
func normalizeExit(err error) error {
	if err == nil {
		return nil
	}
	var exitErr *sys.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 0 {
		return nil
	}
	return err
}

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

// Host owns a compiled guest module and a dispatcher for the store it serves.
// The runtime and compiled module are built once and reused; each unit of work
// gets a fresh module instance (see Call).
type Host struct {
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
	dispatch Dispatcher
	env      map[string]string
}

// NewHost compiles the guest WASM once and binds it to a dispatcher.
func NewHost(ctx context.Context, guestWASM []byte, dispatch Dispatcher) (*Host, error) {
	runtime := wazero.NewRuntime(ctx)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		_ = runtime.Close(ctx)
		return nil, fmt.Errorf("instantiate wasi: %w", err)
	}
	compiled, err := runtime.CompileModule(ctx, guestWASM)
	if err != nil {
		_ = runtime.Close(ctx)
		return nil, fmt.Errorf("compile guest: %w", err)
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

// Call runs one unit of work: it instantiates a fresh guest instance, hands it
// (method, args) as program arguments, services every store call the guest
// makes back to it through the dispatcher, and returns the result the guest
// reports. Exactly one goroutine (this one) drives the instance; a second pumps
// the guest's stdout. This is the concurrency-safe shape from the Phase 1
// findings - a fresh instance per unit of work, never shared across goroutines.
func (h *Host) Call(ctx context.Context, method string, args []byte) ([]byte, error) {
	guestStdinReader, guestStdinWriter := io.Pipe()
	guestStdoutReader, guestStdoutWriter := io.Pipe()

	var (
		captured  []byte
		gotResult bool
		pumpErr   error
	)
	pumpDone := make(chan struct{})
	go func() {
		defer close(pumpDone)
		for {
			frame, err := readCallFrame(guestStdoutReader)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					pumpErr = err
				}
				return
			}
			switch frame.Kind {
			case "call":
				result, dispErr := h.dispatch(ctx, frame.Method, frame.Args)
				reply := replyFrame{}
				if dispErr != nil {
					reply.Error = dispErr.Error()
				} else {
					reply.Result = rawOrNull(result)
				}
				if err := writeReplyFrame(guestStdinWriter, reply); err != nil {
					pumpErr = err
					return
				}
			case "result":
				captured = append([]byte(nil), frame.Result...)
				gotResult = true
			default:
				pumpErr = fmt.Errorf("unexpected guest frame kind %q", frame.Kind)
				return
			}
		}
	}()

	config := wazero.NewModuleConfig().
		WithStdin(guestStdinReader).
		WithStdout(guestStdoutWriter).
		WithArgs("guest", method, string(args)).
		WithName("")
	for key, value := range h.env {
		config = config.WithEnv(key, value)
	}

	instance, err := h.runtime.InstantiateModule(ctx, h.compiled, config)
	if instance != nil {
		_ = instance.Close(ctx)
	}
	_ = guestStdoutWriter.Close()
	<-pumpDone
	_ = guestStdinWriter.Close()

	if runErr := normalizeExit(err); runErr != nil {
		return nil, fmt.Errorf("run guest: %w", runErr)
	}
	if pumpErr != nil {
		return nil, fmt.Errorf("bridge pump: %w", pumpErr)
	}
	if !gotResult {
		return nil, errors.New("guest exited without reporting a result")
	}
	return captured, nil
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

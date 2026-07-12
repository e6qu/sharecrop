//go:build !wasip1

package wasibridge

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

// CredentialLookup is the slice of a storage adapter the host needs to service
// the spike's one bridged method. In production this is satisfied by
// internal/db.AuthStore against Postgres; the test uses the same real store.
// The host depends on this interface, not on internal/db, so pgx never has to
// compile — the guest build of this package excludes host.go entirely.
type CredentialLookup interface {
	FindCredentialByEmail(ctx context.Context, email auth.EmailAddress) auth.CredentialLookupResult
}

// Host owns a compiled guest module and services the storage calls it makes.
// The wazero runtime and compiled module are built once and reused; each unit
// of work gets a fresh module instance (see LookupCredential).
type Host struct {
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
	store    CredentialLookup
}

// NewHost compiles the guest WASM once and wires it to a storage adapter. The
// caller owns the returned Host and must Close it.
func NewHost(ctx context.Context, guestWASM []byte, store CredentialLookup) (*Host, error) {
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
	return &Host{runtime: runtime, compiled: compiled, store: store}, nil
}

// Close releases the runtime and every module compiled against it.
func (h *Host) Close(ctx context.Context) error {
	return h.runtime.Close(ctx)
}

// LookupCredential runs one unit of work: it instantiates a fresh guest
// instance, hands it email as an argument, services the single storage call the
// guest makes back to the store, and returns the result the guest reports.
//
// The instance is driven by exactly one goroutine (the InstantiateModule call
// here). A second goroutine pumps the guest's stdout: it services store_call
// frames by dispatching to the store and writing the reply to the guest's
// stdin, and captures the final result frame. Because the guest is
// single-threaded and strictly request/response, there is no interleaving to
// resolve. This is the concurrency-safe shape the spike settled on — a fresh
// instance per unit of work rather than a shared long-lived instance.
//
// The returned duration is the wall time of the whole unit of work
// (instantiate + one store round-trip + teardown), which is the number that
// decides whether instance-per-request is affordable.
func (h *Host) LookupCredential(ctx context.Context, email string) (auth.CredentialLookupResult, time.Duration, error) {
	guestStdinReader, guestStdinWriter := io.Pipe()
	guestStdoutReader, guestStdoutWriter := io.Pipe()

	var (
		captured auth.CredentialLookupResult
		pumpErr  error
	)
	pumpDone := make(chan struct{})
	go func() {
		defer close(pumpDone)
		for {
			frame, err := readGuestFrame(guestStdoutReader)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					pumpErr = err
				}
				return
			}
			switch frame.Kind {
			case "store_call":
				reply := h.dispatch(ctx, frame.Call)
				if err := writeHostFrame(guestStdinWriter, hostFrame{Response: &reply}); err != nil {
					pumpErr = err
					return
				}
			case "result":
				if frame.Result != nil {
					captured = credentialFromWire(*frame.Result)
				}
			default:
				pumpErr = fmt.Errorf("unexpected guest frame kind %q", frame.Kind)
				return
			}
		}
	}()

	config := wazero.NewModuleConfig().
		WithStdin(guestStdinReader).
		WithStdout(guestStdoutWriter).
		WithArgs("guest", email).
		// wazero defaults to a deterministic random source; draw from the host's
		// real CSPRNG so the guest's crypto/rand (UUIDs, tokens, salts) is not
		// predictable or collision-prone. See the rpc host for the full rationale.
		WithRandSource(rand.Reader).
		// Anonymous instance name so the same compiled module can be
		// instantiated many times in one runtime without a name collision.
		WithName("")

	start := time.Now()
	instance, err := h.runtime.InstantiateModule(ctx, h.compiled, config)
	elapsed := time.Since(start)
	if instance != nil {
		_ = instance.Close(ctx)
	}

	// The guest has returned; close its stdout writer so the pump sees EOF and
	// finishes, then close its stdin writer to release a dangling reader.
	_ = guestStdoutWriter.Close()
	<-pumpDone
	_ = guestStdinWriter.Close()

	if runErr := normalizeExit(err); runErr != nil {
		return nil, elapsed, fmt.Errorf("run guest: %w", runErr)
	}
	if pumpErr != nil {
		return nil, elapsed, fmt.Errorf("bridge pump: %w", pumpErr)
	}
	if captured == nil {
		return nil, elapsed, errors.New("guest exited without reporting a result")
	}
	return captured, elapsed, nil
}

// dispatch is the entire host side of the bridge: parse the request, call the
// real store method, serialize the result. No business logic lives here — a
// malformed email is turned into the store's own rejection shape and nothing
// is decided beyond argument parsing.
func (h *Host) dispatch(ctx context.Context, call *lookupRequest) credentialWire {
	if call == nil {
		return credentialToWire(rejected(core.ErrorCodeInvalidArgument, "bridge: missing store call"))
	}
	parsed := auth.NewEmailAddress(call.Email)
	accepted, ok := parsed.(auth.EmailAddressAccepted)
	if !ok {
		rejectedEmail := parsed.(auth.EmailAddressRejected)
		return credentialToWire(auth.CredentialLookupRejected{Reason: rejectedEmail.Reason})
	}
	return credentialToWire(h.store.FindCredentialByEmail(ctx, accepted.Value))
}

// normalizeExit treats a wasip1 command's exit(0) — which wazero surfaces as a
// zero-code ExitError — as success.
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

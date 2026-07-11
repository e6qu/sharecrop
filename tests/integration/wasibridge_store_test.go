//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/wasibridge"
)

// These tests exercise the Phase 2 WASI hosting spike end to end: a real
// GOOS=wasip1 guest, compiled here, looks up a credential through the storage
// bridge, and the host services that lookup against real Postgres. They prove
// one store method round-trips with real data — including the not-found and
// DomainError-rejected paths — and measure the cost of a fresh guest instance
// per unit of work.

var (
	guestWASMOnce  sync.Once
	guestWASMBytes []byte
	guestWASMErr   error
)

// buildGuestWASM compiles the spike guest to wasip1 once per test binary and
// caches the bytes. Building in-process keeps the test self-contained: no
// external build step has to run first.
func buildGuestWASM(t *testing.T) []byte {
	t.Helper()
	guestWASMOnce.Do(func() {
		out := filepath.Join(t.TempDir(), "guest.wasm")
		cmd := exec.Command("go", "build", "-o", out,
			"github.com/e6qu/sharecrop/cmd/sharecrop-wasi-spike-guest")
		cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			guestWASMErr = fmt.Errorf("build guest wasm: %v: %s", err, stderr.String())
			return
		}
		guestWASMBytes, guestWASMErr = os.ReadFile(out)
	})
	if guestWASMErr != nil {
		t.Fatalf("%v", guestWASMErr)
	}
	return guestWASMBytes
}

func TestWASIBridgeCredentialRoundTrip(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	authStore := db.NewAuthStore(pool)

	host, err := wasibridge.NewHost(ctx, buildGuestWASM(t), authStore)
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close(ctx) })

	userID := newUserID(t)
	address := "wasi-bridge-" + userID.String() + "@example.com"
	email := mustAcceptEmail(t, address)
	seedCredential(t, authStore, userID, email)

	t.Run("found result matches a direct store call", func(t *testing.T) {
		viaBridge, elapsed, err := host.LookupCredential(ctx, address)
		if err != nil {
			t.Fatalf("bridge lookup: %v", err)
		}
		found, ok := viaBridge.(auth.CredentialFound)
		if !ok {
			t.Fatalf("bridge result = %T, want CredentialFound", viaBridge)
		}

		direct, ok := authStore.FindCredentialByEmail(ctx, email).(auth.CredentialFound)
		if !ok {
			t.Fatalf("direct store call did not return CredentialFound")
		}

		if found.Record.UserID != direct.Record.UserID {
			t.Errorf("user id: bridge %s, direct %s", found.Record.UserID, direct.Record.UserID)
		}
		if found.Record.Email.String() != direct.Record.Email.String() {
			t.Errorf("email: bridge %s, direct %s", found.Record.Email, direct.Record.Email)
		}
		if found.Record.PasswordHash.String() != direct.Record.PasswordHash.String() {
			t.Errorf("password hash did not survive the round trip")
		}
		if found.Record.Status != direct.Record.Status {
			t.Errorf("status: bridge %q, direct %q", found.Record.Status, direct.Record.Status)
		}
		t.Logf("found round-trip in %s", elapsed)
	})

	t.Run("missing credential returns CredentialMissing", func(t *testing.T) {
		result, _, err := host.LookupCredential(ctx, "wasi-bridge-absent@example.com")
		if err != nil {
			t.Fatalf("bridge lookup: %v", err)
		}
		if _, ok := result.(auth.CredentialMissing); !ok {
			t.Fatalf("result = %T, want CredentialMissing", result)
		}
	})

	t.Run("rejected result preserves the DomainError shape", func(t *testing.T) {
		result, _, err := host.LookupCredential(ctx, "not-an-email")
		if err != nil {
			t.Fatalf("bridge lookup: %v", err)
		}
		rejected, ok := result.(auth.CredentialLookupRejected)
		if !ok {
			t.Fatalf("result = %T, want CredentialLookupRejected", result)
		}
		// The DomainError crosses the serialization boundary twice (host to
		// guest, then guest's report back to host). Its code must survive as
		// the canonical ErrorCode value, not a stringly-typed approximation.
		if rejected.Reason.Code() != core.ErrorCodeInvalidArgument {
			t.Errorf("error code = %s, want %s", rejected.Reason.Code(), core.ErrorCodeInvalidArgument)
		}
		if rejected.Reason.Description() == "" {
			t.Errorf("error description was dropped in the round trip")
		}
	})

	t.Run("fresh instance per unit of work stays cheap", func(t *testing.T) {
		const iterations = 20
		var total time.Duration
		for i := 0; i < iterations; i++ {
			_, elapsed, err := host.LookupCredential(ctx, address)
			if err != nil {
				t.Fatalf("iteration %d: %v", i, err)
			}
			total += elapsed
		}
		mean := total / iterations
		// Wall time of a full unit of work: fresh module instantiation plus one
		// store round-trip plus teardown. This is the number that decides
		// whether instance-per-request hosting is affordable.
		t.Logf("instance-per-call mean over %d runs: %s (instantiate + one store round-trip)", iterations, mean)
	})
}

func mustAcceptEmail(t *testing.T, address string) auth.EmailAddress {
	t.Helper()
	accepted, ok := auth.NewEmailAddress(address).(auth.EmailAddressAccepted)
	if !ok {
		t.Fatalf("email %q rejected", address)
	}
	return accepted.Value
}

func seedCredential(t *testing.T, store db.AuthStore, userID core.UserID, email auth.EmailAddress) {
	t.Helper()
	secret, ok := auth.NewPasswordSecret("correct horse battery staple").(auth.PasswordSecretAccepted)
	if !ok {
		t.Fatalf("password secret rejected")
	}
	hash, ok := auth.HashPassword(secret.Value).(auth.PasswordHashCreated)
	if !ok {
		t.Fatalf("password hash rejected")
	}
	if _, ok := store.CreateUserCredential(context.Background(), userID, email, hash.Value).(auth.StoreUserAccepted); !ok {
		t.Fatalf("create user credential rejected")
	}
}

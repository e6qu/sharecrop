//go:build !wasip1

// Command sharecrop-wasi-spike is the native host for the Phase 2 WASI hosting
// spike. It embeds a wazero runtime, owns the Postgres pool, loads the compiled
// guest WASM, and runs a single credential lookup through the storage bridge —
// proving one real store method round-trips from a WASM guest to Postgres and
// back.
//
// Usage:
//
//	export DATABASE_URL=postgres://...
//	GOOS=wasip1 GOARCH=wasm go build -o guest.wasm ./cmd/sharecrop-wasi-spike-guest
//	go run ./cmd/sharecrop-wasi-spike -guest guest.wasm -email someone@example.com
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/wasibridge"
)

func main() {
	guestPath := flag.String("guest", "guest.wasm", "path to the compiled wasip1 guest module")
	email := flag.String("email", "", "email to look up through the bridge")
	flag.Parse()

	if *email == "" {
		fmt.Fprintln(os.Stderr, "-email is required")
		os.Exit(2)
	}

	if err := run(*guestPath, *email); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(guestPath, email string) error {
	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}
	pool, err := db.Open(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()

	guestWASM, err := os.ReadFile(guestPath)
	if err != nil {
		return fmt.Errorf("read guest module: %w", err)
	}

	host, err := wasibridge.NewHost(ctx, guestWASM, db.NewAuthStore(pool))
	if err != nil {
		return fmt.Errorf("build host: %w", err)
	}
	defer host.Close(ctx)

	result, elapsed, err := host.LookupCredential(ctx, email)
	if err != nil {
		return err
	}

	switch typed := result.(type) {
	case auth.CredentialFound:
		fmt.Printf("found: user=%s email=%s status=%s (%.3fms)\n",
			typed.Record.UserID, typed.Record.Email, typed.Record.Status,
			float64(elapsed.Microseconds())/1000)
	case auth.CredentialMissing:
		fmt.Printf("missing: no credential for %q (%.3fms)\n", email, float64(elapsed.Microseconds())/1000)
	case auth.CredentialLookupRejected:
		fmt.Printf("rejected: %s — %s (%.3fms)\n",
			typed.Reason.Code(), typed.Reason.Description(), float64(elapsed.Microseconds())/1000)
	}
	return nil
}

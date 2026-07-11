//go:build !wasip1

// Command sharecrop-wasi-audit-host is the native host for the Phase 3 audit
// store bridge. It embeds a wazero runtime, owns the Postgres pool, loads the
// compiled audit guest, and runs one audit-event lookup through the generated
// bridge - the manual counterpart to the dual-run integration test.
//
// Usage:
//
//	export DATABASE_URL=postgres://...
//	GOOS=wasip1 GOARCH=wasm go build -o audit-guest.wasm ./cmd/sharecrop-wasi-audit-guest
//	go run ./cmd/sharecrop-wasi-audit-host -guest audit-guest.wasm -id <audit-event-id>
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/wasibridge/auditbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

func main() {
	guestPath := flag.String("guest", "audit-guest.wasm", "path to the compiled wasip1 audit guest module")
	rawID := flag.String("id", "", "audit event id to look up through the bridge")
	flag.Parse()

	if *rawID == "" {
		fmt.Fprintln(os.Stderr, "-id is required")
		os.Exit(2)
	}

	if err := run(*guestPath, *rawID); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(guestPath, rawID string) error {
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

	auditStore := db.NewAuditStore(pool)
	host, err := rpc.NewHost(ctx, guestWASM, func(ctx context.Context, method string, args []byte) ([]byte, error) {
		return auditbridge.Dispatch(ctx, auditStore, method, args)
	})
	if err != nil {
		return fmt.Errorf("build host: %w", err)
	}
	defer host.Close(ctx)

	// Drive the full bridge from the host side: the generated GuestStore's calls
	// run a fresh guest, which RPCs back to the dispatcher above.
	store := auditbridge.NewGuestStore(func(method string, args []byte) ([]byte, error) {
		return host.Call(ctx, method, args)
	})

	id, matched := core.ParseAuditEventID(rawID).(core.AuditEventIDCreated)
	if !matched {
		return fmt.Errorf("invalid audit event id %q", rawID)
	}

	switch result := store.Get(ctx, id.Value).(type) {
	case audit.EventFound:
		fmt.Printf("found: id=%s actor=%s action=%s subject=%s/%s\n",
			result.Value.ID, result.Value.ActorUserID, result.Value.Action,
			result.Value.Subject.Kind, result.Value.Subject.ID)
	case audit.GetRejected:
		fmt.Printf("rejected: %s - %s\n", result.Reason.Code(), result.Reason.Description())
	}
	return nil
}

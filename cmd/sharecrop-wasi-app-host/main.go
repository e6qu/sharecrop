//go:build !wasip1

// Command sharecrop-wasi-app-host is the production-shaped host: a real
// net/http.Server that serves authenticated, store-touching routes by running
// the app guest (the real internal/http mux) once per request, with the guest's
// store calls dispatched to real Postgres. It ties the Phase 3 store bridge and
// the Phase 4 HTTP hosting together.
//
// Usage:
//
//	export DATABASE_URL=postgres://...
//	export SHARECROP_ACCESS_TOKEN_SECRET=... (>= 32 bytes)
//	GOOS=wasip1 GOARCH=wasm go build -o app-guest.wasm ./cmd/sharecrop-wasi-app-guest
//	go run ./cmd/sharecrop-wasi-app-host -guest app-guest.wasm -addr :8091
//	curl -s localhost:8091/healthz
//	curl -s -H "Authorization: Bearer <token>" localhost:8091/api/notifications
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/wasibridge/httpbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/storehost"
)

func main() {
	guestPath := flag.String("guest", "app-guest.wasm", "path to the compiled wasip1 app guest module")
	addr := flag.String("addr", ":8091", "address to listen on")
	flag.Parse()

	if err := run(*guestPath, *addr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(guestPath, addr string) error {
	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}
	secret := os.Getenv("SHARECROP_ACCESS_TOKEN_SECRET")
	if secret == "" {
		return fmt.Errorf("SHARECROP_ACCESS_TOKEN_SECRET is not set")
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

	host, err := rpc.NewHost(ctx, guestWASM, storehost.Dispatcher(pool))
	if err != nil {
		return fmt.Errorf("build host: %w", err)
	}
	host.WithGuestEnv(map[string]string{"SHARECROP_ACCESS_TOKEN_SECRET": secret})
	defer host.Close(ctx)

	server := &http.Server{Addr: addr, Handler: httpbridge.Handler(host)}
	log.Printf("listening on %s - each request runs a fresh wasm guest against Postgres", addr)
	return server.ListenAndServe()
}

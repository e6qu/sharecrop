//go:build !wasip1

// Command sharecrop-wasi-http-host is the Phase 4 native host: a real
// net/http.Server that handles each request by running a fresh wasip1 guest
// (the real internal/http mux compiled to WASM). It proves the production
// hosting shape end to end - a request in, a real handler run inside the guest,
// a response out - for a route that needs no database (GET /healthz).
//
// Usage:
//
//	GOOS=wasip1 GOARCH=wasm go build -o http-guest.wasm ./cmd/sharecrop-wasi-http-guest
//	go run ./cmd/sharecrop-wasi-http-host -guest http-guest.wasm -addr :8090
//	curl -s localhost:8090/healthz
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/e6qu/sharecrop/internal/wasibridge/httpbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

func main() {
	guestPath := flag.String("guest", "http-guest.wasm", "path to the compiled wasip1 http guest module")
	addr := flag.String("addr", ":8090", "address to listen on")
	flag.Parse()

	if err := run(*guestPath, *addr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(guestPath, addr string) error {
	ctx := context.Background()

	guestWASM, err := os.ReadFile(guestPath)
	if err != nil {
		return fmt.Errorf("read guest module: %w", err)
	}

	host, err := rpc.NewHost(ctx, guestWASM, noStoreDispatcher)
	if err != nil {
		return fmt.Errorf("build host: %w", err)
	}
	defer host.Close(ctx)

	server := &http.Server{Addr: addr, Handler: &bridgeHandler{host: host}}
	log.Printf("listening on %s - each request runs a fresh wasm guest", addr)
	return server.ListenAndServe()
}

type bridgeHandler struct {
	host *rpc.Host
}

func (h *bridgeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestBytes, err := httpbridge.EncodeRequest(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	responseBytes, err := h.host.Call(r.Context(), "http.handle", requestBytes)
	if err != nil {
		http.Error(w, "bridge error: "+err.Error(), http.StatusBadGateway)
		return
	}
	if err := httpbridge.WriteResponse(w, responseBytes); err != nil {
		log.Printf("write response: %v", err)
	}
}

// noStoreDispatcher rejects every store call. The Phase 4 slice serves /healthz,
// which makes none; a store-touching route would supply a real dispatcher (the
// Phase 3 auditbridge.Dispatch shape).
func noStoreDispatcher(ctx context.Context, method string, args []byte) ([]byte, error) {
	return nil, errors.New("no store is bridged for this route")
}

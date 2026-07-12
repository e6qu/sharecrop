// Command sharecrop-wasi-http-guest is the Phase 4 WASM guest: it runs the real
// internal/http routing table inside a wasip1 module. The host hands it one
// serialized HTTP request; it runs the request through the mux with an httptest
// recorder and reports the serialized response.
//
// Domain services are nil here because this Phase 4 slice handles GET /healthz,
// which touches no service or store. A route that reads a store would bind
// bridge-backed guest stores (the Phase 3 GuestStore) in their place, and those
// calls would RPC back to the host over the same unit-of-work channel.
package main

import (
	"fmt"
	"os"
	"testing/fstest"

	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/wasibridge/httpbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

func main() {
	mux := httpserver.New(fstest.MapFS{}, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err := rpc.Serve(func(_ string, args []byte) ([]byte, error) {
		return httpbridge.Serve(mux, args)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Command sharecrop-wasi-audit-guest is the Phase 3 WASM guest for the audit
// store bridge. Built with GOOS=wasip1 GOARCH=wasm, it receives one store call
// (method, JSON args) from the host, runs it through the generated GuestStore -
// whose calls RPC back to the host, which services them against real Postgres -
// and reports the serialized result.
//
// It never opens a network connection; every store operation goes through the
// host bridge over stdin/stdout.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/e6qu/sharecrop/internal/wasibridge/auditbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

func main() {
	method, args, err := rpc.UnitOfWork()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	store := auditbridge.NewGuestStore(rpc.Invoke)
	result, err := auditbridge.Dispatch(context.Background(), store, method, args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := rpc.ReportResult(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

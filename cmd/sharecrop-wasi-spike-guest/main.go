// Command sharecrop-wasi-spike-guest is the WASM guest for the Phase 2 WASI
// hosting spike. Built with GOOS=wasip1 GOARCH=wasm, it stands in for the
// backend running inside the guest: it takes one email as an argument, looks up
// the credential through the storage bridge (which the host services against
// real Postgres), and reports the result back for the host to verify.
//
// It never opens a network connection — the whole point of the spike is that it
// cannot, and reaches the database only through the host bridge.
package main

import (
	"fmt"
	"os"

	"github.com/e6qu/sharecrop/internal/wasibridge"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "guest: expected an email argument")
		os.Exit(2)
	}
	result := wasibridge.GuestLookupCredential(os.Args[1])
	if err := wasibridge.GuestReportResult(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

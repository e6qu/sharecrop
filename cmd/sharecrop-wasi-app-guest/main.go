// Command sharecrop-wasi-app-guest runs the real internal/http mux inside a
// wasip1 guest with a live domain service wired in - the step that ties the
// Phase 3 store bridge to the Phase 4 HTTP hosting. It serves an authenticated,
// store-touching route (GET /api/notifications): the stateless access-token
// verifier checks the bearer token (no store), and the notification service is
// backed by the generated notification GuestStore, whose reads RPC back to the
// host and hit real Postgres.
//
// Only the notification service is wired; other services are nil, so this guest
// serves /healthz and the notification routes. Its store calls and its final
// HTTP response share the same unit-of-work channel with the host.
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/wasibridge/appmux"
	"github.com/e6qu/sharecrop/internal/wasibridge/httpbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/notificationbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

func main() {
	_, args, err := rpc.UnitOfWork()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	mux, err := buildMux()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	response, err := httpbridge.Serve(mux, args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := rpc.ReportResult(response); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildMux() (http.Handler, error) {
	secret, matched := auth.NewAccessTokenSecret(os.Getenv("SHARECROP_ACCESS_TOKEN_SECRET")).(auth.AccessTokenSecretAccepted)
	if !matched {
		return nil, fmt.Errorf("SHARECROP_ACCESS_TOKEN_SECRET is missing or invalid")
	}
	return appmux.New(secret.Value, notificationbridge.NewGuestStore(rpc.Invoke)), nil
}

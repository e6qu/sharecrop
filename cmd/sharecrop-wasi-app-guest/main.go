// Command sharecrop-wasi-app-guest runs the real internal/http mux inside a
// wasip1 guest with the FULL domain-service graph wired in - the step that ties
// the store bridge to HTTP hosting for every route, not just a slice. The
// stateless access-token verifier checks the bearer token in-guest (no store),
// and all ten domain services are backed by their generated GuestStores, whose
// calls RPC back to the host and hit real Postgres. RuntimeState services with
// no dedicated store (rate limiters, MCP sessions, saved queue views, privacy,
// platform admins, moderation triage) keep their in-memory defaults.
//
// Its store calls and its final HTTP response share the same unit-of-work
// channel with the host.
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/wasibridge/agentbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/appmux"
	"github.com/e6qu/sharecrop/internal/wasibridge/assetsbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/auditbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/authbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/httpbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/ledgerbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/mcpsessionbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/moderationtriagebridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/notificationbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/orgbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/orgcredbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/platformadminbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/privacybridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/ratelimitbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/savedqueueviewbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/submissionbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/taskbridge"
)

func main() {
	// Build the mux and its service graph once per instance, not per request, so
	// a pooled instance amortizes the wiring across every unit of work it serves.
	mux, err := buildMux()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := rpc.Serve(func(_ string, args []byte) ([]byte, error) {
		return httpbridge.Serve(mux, args)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildMux() (http.Handler, error) {
	secret, matched := auth.NewAccessTokenSecret(os.Getenv("SHARECROP_ACCESS_TOKEN_SECRET")).(auth.AccessTokenSecretAccepted)
	if !matched {
		return nil, fmt.Errorf("SHARECROP_ACCESS_TOKEN_SECRET is missing or invalid")
	}
	return appmux.New(secret.Value, appmux.Stores{
		Auth:               authbridge.NewGuestStore(rpc.Invoke),
		Notification:       notificationbridge.NewGuestStore(rpc.Invoke),
		Organization:       orgbridge.NewGuestStore(rpc.Invoke),
		Task:               taskbridge.NewGuestStore(rpc.Invoke),
		Submission:         submissionbridge.NewGuestStore(rpc.Invoke),
		Ledger:             ledgerbridge.NewGuestStore(rpc.Invoke),
		Agent:              agentbridge.NewGuestStore(rpc.Invoke),
		OrgCredential:      orgcredbridge.NewGuestStore(rpc.Invoke),
		Assets:             assetsbridge.NewGuestStore(rpc.Invoke),
		Audit:              auditbridge.NewGuestStore(rpc.Invoke),
		SavedQueueViews:    savedqueueviewbridge.NewGuestStore(rpc.Invoke),
		PlatformAdmins:     platformadminbridge.NewGuestStore(rpc.Invoke),
		ModerationTriage:   moderationtriagebridge.NewGuestStore(rpc.Invoke),
		Privacy:            privacybridge.NewGuestStore(rpc.Invoke),
		IPRateLimiter:      ratelimitbridge.NewGuestRateLimiter(rpc.Invoke, "ip"),
		SubjectRateLimiter: ratelimitbridge.NewGuestRateLimiter(rpc.Invoke, "subject"),
		MCPSessions:        mcpsessionbridge.NewGuestMCPSessionPersistence(rpc.Invoke),
	}), nil
}

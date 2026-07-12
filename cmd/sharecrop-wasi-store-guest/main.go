// Command sharecrop-wasi-store-guest is the WASM guest for the store bridges.
// Built with GOOS=wasip1 GOARCH=wasm, it receives one store call (method, JSON
// args) from the host, routes it by method prefix to the matching generated
// bridge - whose GuestStore RPCs back to the host, which services the call
// against real Postgres - and reports the serialized result.
//
// One guest serves every bridged store; the host picks the store by the method
// name it sends (e.g. "audit.Get", "notification.List").
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/e6qu/sharecrop/internal/wasibridge/agentbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/assetsbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/auditbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/authbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/ledgerbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/moderationtriagebridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/notificationbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/orgbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/orgcredbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/platformadminbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/savedqueueviewbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/submissionbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/taskbridge"
)

func main() {
	if err := rpc.Serve(func(method string, args []byte) ([]byte, error) {
		return dispatch(context.Background(), method, args)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func dispatch(ctx context.Context, method string, args []byte) ([]byte, error) {
	store, _, _ := strings.Cut(method, ".")
	switch store {
	case "assets":
		return assetsbridge.Dispatch(ctx, assetsbridge.NewGuestStore(rpc.Invoke), method, args)
	case "audit":
		return auditbridge.Dispatch(ctx, auditbridge.NewGuestStore(rpc.Invoke), method, args)
	case "notification":
		return notificationbridge.Dispatch(ctx, notificationbridge.NewGuestStore(rpc.Invoke), method, args)
	case "auth":
		return authbridge.Dispatch(ctx, authbridge.NewGuestStore(rpc.Invoke), method, args)
	case "ledger":
		return ledgerbridge.Dispatch(ctx, ledgerbridge.NewGuestStore(rpc.Invoke), method, args)
	case "agent":
		return agentbridge.Dispatch(ctx, agentbridge.NewGuestStore(rpc.Invoke), method, args)
	case "org":
		return orgbridge.Dispatch(ctx, orgbridge.NewGuestStore(rpc.Invoke), method, args)
	case "orgcred":
		return orgcredbridge.Dispatch(ctx, orgcredbridge.NewGuestStore(rpc.Invoke), method, args)
	case "submission":
		return submissionbridge.Dispatch(ctx, submissionbridge.NewGuestStore(rpc.Invoke), method, args)
	case "task":
		return taskbridge.Dispatch(ctx, taskbridge.NewGuestStore(rpc.Invoke), method, args)
	case "savedqueueview":
		return savedqueueviewbridge.Dispatch(ctx, savedqueueviewbridge.NewGuestStore(rpc.Invoke), method, args)
	case "platformadmin":
		return platformadminbridge.Dispatch(ctx, platformadminbridge.NewGuestStore(rpc.Invoke), method, args)
	case "moderationtriage":
		return moderationtriagebridge.Dispatch(ctx, moderationtriagebridge.NewGuestStore(rpc.Invoke), method, args)
	default:
		return nil, fmt.Errorf("no bridge for method %q", method)
	}
}

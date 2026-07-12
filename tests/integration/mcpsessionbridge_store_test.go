//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/wasibridge/mcpsessionbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

// TestMCPSessionBridgeDualRun exercises MCP session persistence - a hand-written
// bridge (its methods return multi-value tuples, so it isn't generated) - through
// the compiled wasip1 guest + host bridge: create a session, count it, append and
// replay events, touch it, and close it, checking the reads match a direct db
// call. A unique session id and subject keep everything private so nothing
// contaminates the shared db-checks database.
func TestMCPSessionBridgeDualRun(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	dbStore := db.NewMCPSessionStore(pool)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-store-guest")
	if err != nil {
		t.Fatalf("compile store guest: %v", err)
	}
	host, err := rpc.NewHost(ctx, guestWASM, func(ctx context.Context, method string, args []byte) ([]byte, error) {
		return mcpsessionbridge.Dispatch(ctx, dbStore, method, args)
	})
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close(ctx) })
	bridgeStore := mcpsessionbridge.NewGuestMCPSessionPersistence(func(method string, args []byte) ([]byte, error) {
		return host.Call(ctx, method, args)
	})

	now := time.Now().UTC()
	cutoff := now.Add(-time.Hour)
	sessionID := "mcp-session-" + newAuditEventID(t).String()
	subject := "mcp-subject-" + newAuditEventID(t).String()

	t.Run("create then count for subject matches a direct call", func(t *testing.T) {
		if err := bridgeStore.CreateMCPSession(ctx, sessionID, subject, now); err != nil {
			t.Fatalf("bridge CreateMCPSession: %v", err)
		}
		viaBridge, err := bridgeStore.ActiveMCPSessionCountForSubject(ctx, subject, cutoff)
		if err != nil {
			t.Fatalf("bridge count: %v", err)
		}
		direct, err := dbStore.ActiveMCPSessionCountForSubject(ctx, subject, cutoff)
		if err != nil {
			t.Fatalf("direct count: %v", err)
		}
		if viaBridge != direct || viaBridge != 1 {
			t.Errorf("session count: bridge %d, direct %d, want 1", viaBridge, direct)
		}
	})

	t.Run("append then list events matches a direct call", func(t *testing.T) {
		payload := []byte(`{"jsonrpc":"2.0","method":"ping"}`)
		eventID, stored, err := bridgeStore.AppendMCPEvent(ctx, sessionID, payload, now)
		if err != nil {
			t.Fatalf("bridge AppendMCPEvent: %v", err)
		}
		if eventID == "" || string(stored) != string(payload) {
			t.Errorf("appended event = (%q, %q)", eventID, stored)
		}

		bridgeIDs, bridgePayloads, err := bridgeStore.ListMCPEvents(ctx, sessionID, "", 10)
		if err != nil {
			t.Fatalf("bridge ListMCPEvents: %v", err)
		}
		directIDs, directPayloads, err := dbStore.ListMCPEvents(ctx, sessionID, "", 10)
		if err != nil {
			t.Fatalf("direct ListMCPEvents: %v", err)
		}
		if len(bridgeIDs) != len(directIDs) || len(bridgeIDs) != 1 {
			t.Fatalf("event counts: bridge %d, direct %d, want 1", len(bridgeIDs), len(directIDs))
		}
		if bridgeIDs[0] != directIDs[0] || bridgeIDs[0] != eventID {
			t.Errorf("event id: bridge %q, direct %q, want %q", bridgeIDs[0], directIDs[0], eventID)
		}
		if string(bridgePayloads[0]) != string(directPayloads[0]) || string(bridgePayloads[0]) != string(payload) {
			t.Errorf("event payload mismatch: bridge %q, direct %q", bridgePayloads[0], directPayloads[0])
		}
	})

	t.Run("touch then close through the bridge", func(t *testing.T) {
		touched, err := bridgeStore.TouchMCPSession(ctx, sessionID, subject, now, cutoff)
		if err != nil {
			t.Fatalf("bridge TouchMCPSession: %v", err)
		}
		if !touched {
			t.Errorf("bridge TouchMCPSession did not touch the active session")
		}
		closed, err := bridgeStore.CloseMCPSession(ctx, sessionID, now)
		if err != nil {
			t.Fatalf("bridge CloseMCPSession: %v", err)
		}
		if !closed {
			t.Errorf("bridge CloseMCPSession did not close the session")
		}
		count, err := bridgeStore.ActiveMCPSessionCountForSubject(ctx, subject, cutoff)
		if err != nil {
			t.Fatalf("bridge count after close: %v", err)
		}
		if count != 0 {
			t.Errorf("subject session count after close = %d, want 0", count)
		}
	})
}

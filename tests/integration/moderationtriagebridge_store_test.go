//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/wasibridge/moderationtriagebridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

// TestModerationTriageBridgeDualRun exercises the moderation-triage service - an
// internal/http RuntimeState service - through the compiled wasip1 guest + host
// bridge: record a report open (from an audit.Event), list it, then update it.
// The seed audit event uses a test-only action so it never collides with the
// scenario-parity run sharing the same db-checks database.
func TestModerationTriageBridgeDualRun(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	dbStore := db.NewModerationTriageStore(pool)
	auditStore := db.NewAuditStore(pool)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-store-guest")
	if err != nil {
		t.Fatalf("compile store guest: %v", err)
	}
	host, err := rpc.NewHost(ctx, guestWASM, func(ctx context.Context, method string, args []byte) ([]byte, error) {
		return moderationtriagebridge.Dispatch(ctx, dbStore, method, args)
	})
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close(ctx) })
	bridgeStore := moderationtriagebridge.NewGuestStore(func(method string, args []byte) ([]byte, error) {
		return host.Call(ctx, method, args)
	})

	actor := createUser(t, pool, "moderation-actor")
	event := recordAuditEvent(t, ctx, auditStore, actor, audit.ActionFromString("wasi-moderation-dualrun"),
		audit.Subject{Kind: "task", ID: "task-" + newAuditEventID(t).String()})

	t.Run("record open then list matches a direct call", func(t *testing.T) {
		saved, matched := bridgeStore.RecordOpen(ctx, event).(httpserver.ModerationTriageSaved)
		if !matched {
			t.Fatalf("bridge RecordOpen did not save")
		}
		if saved.Value.ReportID != event.ID || saved.Value.State != "open" {
			t.Errorf("recorded triage = %+v", saved.Value)
		}

		viaBridge := requireTriageListed(t, bridgeStore.List(ctx, []core.AuditEventID{event.ID}))
		direct := requireTriageListed(t, dbStore.List(ctx, []core.AuditEventID{event.ID}))
		if len(viaBridge) != len(direct) || len(viaBridge) != 1 {
			t.Fatalf("triage counts: bridge %d, direct %d, want 1", len(viaBridge), len(direct))
		}
		if viaBridge[0].ReportID != direct[0].ReportID || viaBridge[0].ReportID != event.ID {
			t.Errorf("listed report = %s, want %s", viaBridge[0].ReportID, event.ID)
		}
		if viaBridge[0].State != direct[0].State {
			t.Errorf("triage state: bridge %q, direct %q", viaBridge[0].State, direct[0].State)
		}
	})

	t.Run("update through the bridge resolves the report", func(t *testing.T) {
		updated, matched := bridgeStore.Update(ctx, actor, event.ID, "resolved", "handled it").(httpserver.ModerationTriageSaved)
		if !matched {
			t.Fatalf("bridge Update did not save")
		}
		if updated.Value.State != "resolved" || updated.Value.ResolutionNote != "handled it" {
			t.Errorf("updated triage = %+v", updated.Value)
		}
	})
}

func requireTriageListed(t *testing.T, result httpserver.ModerationTriageListResult) []httpserver.ModerationTriageRecord {
	t.Helper()
	listed, matched := result.(httpserver.ModerationTriageListed)
	if !matched {
		t.Fatalf("list result = %T, want listed", result)
	}
	return listed.Values
}

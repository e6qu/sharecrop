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
	// The seed event uses the real moderation_report_created action so the
	// pushed-down triage listing (which joins on that action) can see it; the
	// assertions target this event's id, so rows left by other suites sharing
	// the db-checks database do not interfere.
	event := recordAuditEvent(t, ctx, auditStore, actor, audit.ActionModerationReportCreated,
		audit.Subject{Kind: "task", ID: "task-" + newAuditEventID(t).String()})

	t.Run("record open then list matches a direct call", func(t *testing.T) {
		saved, matched := bridgeStore.RecordOpen(ctx, event).(httpserver.ModerationTriageSaved)
		if !matched {
			t.Fatalf("bridge RecordOpen did not save")
		}
		if saved.Value.ReportID != event.ID || saved.Value.State != "open" {
			t.Errorf("recorded triage = %+v", saved.Value)
		}

		filter := httpserver.TriageStateEquals{State: httpserver.ModerationTriageStateOpen}
		viaBridge, bridgeFound := findListedTriage(t, bridgeStore.List(ctx, filter, core.DefaultPage()), event.ID)
		direct, directFound := findListedTriage(t, dbStore.List(ctx, filter, core.DefaultPage()), event.ID)
		if !bridgeFound || !directFound {
			t.Fatalf("open-filtered listing missing the report: bridge %t, direct %t", bridgeFound, directFound)
		}
		if viaBridge.State != direct.State || viaBridge.ReportID != direct.ReportID {
			t.Errorf("triage records differ: bridge %+v, direct %+v", viaBridge, direct)
		}
	})

	t.Run("update through the bridge resolves the report", func(t *testing.T) {
		updated, matched := bridgeStore.Update(ctx, actor, event.ID, httpserver.ModerationTriageStateResolved, "handled it").(httpserver.ModerationTriageSaved)
		if !matched {
			t.Fatalf("bridge Update did not save")
		}
		if updated.Value.State != "resolved" || updated.Value.ResolutionNote != "handled it" {
			t.Errorf("updated triage = %+v", updated.Value)
		}
		if _, stillOpen := findListedTriage(t, bridgeStore.List(ctx, httpserver.TriageStateEquals{State: httpserver.ModerationTriageStateOpen}, core.DefaultPage()), event.ID); stillOpen {
			t.Errorf("resolved report still appears in the open-filtered listing")
		}
	})
}

func findListedTriage(t *testing.T, result httpserver.ModerationTriageListResult, reportID core.AuditEventID) (httpserver.ModerationTriageRecord, bool) {
	t.Helper()
	for _, record := range requireTriageListed(t, result) {
		if record.ReportID == reportID {
			return record, true
		}
	}
	return httpserver.ModerationTriageRecord{}, false
}

func requireTriageListed(t *testing.T, result httpserver.ModerationTriageListResult) []httpserver.ModerationTriageRecord {
	t.Helper()
	listed, matched := result.(httpserver.ModerationTriageListed)
	if !matched {
		t.Fatalf("list result = %T, want listed", result)
	}
	return listed.Values
}

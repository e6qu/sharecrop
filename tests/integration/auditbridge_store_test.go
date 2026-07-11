//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/audit/audittest"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/wasibridge/auditbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

// TestAuditBridgeDualRun is the Phase 3 anti-drift safeguard #3: every audit
// Store method is exercised twice - once against internal/db directly, once
// through the compiled wasip1 guest + host bridge - with the same real Postgres
// behind both. Any behavioral difference is a failure here, not a review
// finding later. The generated bridge (auditbridge.Dispatch / GuestStore) is
// what carries the calls; the same GuestStore type serves as the in-guest
// client and, driven by host.Call, as the host-side bridge driver.
func TestAuditBridgeDualRun(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	dbStore := db.NewAuditStore(pool)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-audit-guest")
	if err != nil {
		t.Fatalf("compile audit guest: %v", err)
	}

	// The host services each store call the guest makes against the real db store.
	host, err := rpc.NewHost(ctx, guestWASM, func(ctx context.Context, method string, args []byte) ([]byte, error) {
		return auditbridge.Dispatch(ctx, dbStore, method, args)
	})
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close(ctx) })

	// bridgeStore is the generated GuestStore driven from the host side: each
	// call runs a fresh guest, which RPCs back to the host dispatcher above.
	bridgeStore := auditbridge.NewGuestStore(func(method string, args []byte) ([]byte, error) {
		return host.Call(ctx, method, args)
	})

	actor := createUser(t, pool, "audit-bridge")

	// A test-only action string that no production query matches. These tests
	// share the db-checks database with the scenario-parity run; using a real
	// action (e.g. moderation_report_created) would leave audit rows that other
	// suites pick up - moderation listing, for one, joins such rows to a triage
	// row this test never creates and 404s on the mismatch.
	action := audit.ActionFromString("wasi-bridge-dualrun")

	t.Run("get: bridge read matches a direct store call", func(t *testing.T) {
		seeded := recordAuditEvent(t, ctx, dbStore, actor, action,
			audit.Subject{Kind: "organization", ID: "org-" + newAuditEventID(t).String()})

		viaBridge := requireEventFound(t, bridgeStore.Get(ctx, seeded.ID))
		direct := requireEventFound(t, dbStore.Get(ctx, seeded.ID))
		assertAuditEventsEqual(t, viaBridge, direct)
	})

	t.Run("get: a missing id rejects the same way both paths", func(t *testing.T) {
		missing := newAuditEventID(t)
		bridgeReason := requireGetRejected(t, bridgeStore.Get(ctx, missing))
		directReason := requireGetRejected(t, dbStore.Get(ctx, missing))
		if bridgeReason.Code() != core.ErrorCodeNotFound || directReason.Code() != core.ErrorCodeNotFound {
			t.Errorf("rejection codes: bridge %s, direct %s, want not_found", bridgeReason.Code(), directReason.Code())
		}
	})

	t.Run("record: writing through the bridge persists to postgres", func(t *testing.T) {
		event := buildAuditEvent(t, actor, action,
			audit.Subject{Kind: "task", ID: "task-" + newAuditEventID(t).String()})

		if _, matched := bridgeStore.Record(ctx, event).(audit.EventRecorded); !matched {
			t.Fatalf("bridge record did not report EventRecorded")
		}
		// The write went guest -> host -> db, so the row must be readable
		// directly from Postgres.
		persisted := requireEventFound(t, dbStore.Get(ctx, event.ID))
		if persisted.ID != event.ID {
			t.Errorf("persisted id = %s, want %s", persisted.ID, event.ID)
		}
		if persisted.Action.String() != event.Action.String() {
			t.Errorf("persisted action = %s, want %s", persisted.Action, event.Action)
		}
		if persisted.Subject != event.Subject {
			t.Errorf("persisted subject = %+v, want %+v", persisted.Subject, event.Subject)
		}
	})

	t.Run("list: bridge listing matches a direct store call", func(t *testing.T) {
		kind := "list-" + newAuditEventID(t).String()
		for i := 0; i < 3; i++ {
			recordAuditEvent(t, ctx, dbStore, actor, action,
				audit.Subject{Kind: kind, ID: newAuditEventID(t).String()})
		}
		page, matched := core.NewPage(50, 0).(core.PageAccepted)
		if !matched {
			t.Fatalf("page rejected")
		}
		filters := audit.ListFilters{
			Action:      audit.AnyAction{},
			SubjectKind: audit.SubjectKindEquals{Value: kind},
			SubjectID:   audit.AnySubjectID{},
		}

		viaBridge := requireEventsListed(t, bridgeStore.List(ctx, filters, page.Value))
		direct := requireEventsListed(t, dbStore.List(ctx, filters, page.Value))
		if len(viaBridge) != 3 || len(direct) != 3 {
			t.Fatalf("list counts: bridge %d, direct %d, want 3", len(viaBridge), len(direct))
		}
		for i := range direct {
			assertAuditEventsEqual(t, viaBridge[i], direct[i])
		}
	})
}

func newAuditEventID(t *testing.T) core.AuditEventID {
	t.Helper()
	created, matched := core.NewAuditEventID().(core.AuditEventIDCreated)
	if !matched {
		t.Fatalf("audit event id rejected")
	}
	return created.Value
}

func buildAuditEvent(t *testing.T, actor core.UserID, action audit.Action, subject audit.Subject) audit.Event {
	t.Helper()
	return audit.Event{
		ID:          newAuditEventID(t),
		ActorUserID: actor,
		Action:      action,
		Subject:     subject,
		Metadata:    audit.EmptyMetadata(),
		CreatedAt:   time.Now().UTC().Round(time.Microsecond),
	}
}

func recordAuditEvent(t *testing.T, ctx context.Context, store db.AuditStore, actor core.UserID, action audit.Action, subject audit.Subject) audit.Event {
	t.Helper()
	event := buildAuditEvent(t, actor, action, subject)
	if _, matched := store.Record(ctx, event).(audit.EventRecorded); !matched {
		t.Fatalf("seed record rejected")
	}
	return event
}

func requireEventFound(t *testing.T, result audit.GetResult) audit.Event {
	t.Helper()
	found, matched := result.(audit.EventFound)
	if !matched {
		t.Fatalf("get result = %T, want EventFound", result)
	}
	return found.Value
}

func requireGetRejected(t *testing.T, result audit.GetResult) core.DomainError {
	t.Helper()
	rejected, matched := result.(audit.GetRejected)
	if !matched {
		t.Fatalf("get result = %T, want GetRejected", result)
	}
	return rejected.Reason
}

func requireEventsListed(t *testing.T, result audit.ListResult) []audit.Event {
	t.Helper()
	listed, matched := result.(audit.EventsListed)
	if !matched {
		t.Fatalf("list result = %T, want EventsListed", result)
	}
	return listed.Values
}

func assertAuditEventsEqual(t *testing.T, got audit.Event, want audit.Event) {
	t.Helper()
	if diff := audittest.EventDiff(got, want); diff != "" {
		t.Errorf("event mismatch: %s", diff)
	}
}

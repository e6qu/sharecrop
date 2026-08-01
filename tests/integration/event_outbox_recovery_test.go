//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/webhook"
)

// The recorded→sweep recovery of a recipient-audience subscription plus its
// inbox notification is covered by TestOutboxCrashWindowIsRecoveredBySweep.
// The two tests here close the remaining crash-window gaps: marketplace
// expansion recovered by the sweep, and a partial dispatch (notifications
// written, store dispatch lost) that must not duplicate on recovery.

// TestOutboxSweepRecoversMarketplaceExpansion crashes before the inline
// dispatch of a public task_opened event that marketplace subscribers are
// watching. The sweep must expand the marketplace deliveries exactly once;
// task_opened is deliberately feed-only, so the sweep must also create no
// inbox rows for it.
func TestOutboxSweepRecoversMarketplaceExpansion(t *testing.T) {
	pool := newPool(t)
	store := db.NewWebhookStore(pool)
	parkActiveSubscriptions(t, pool)
	dispatchAllRecordedEvents(t, pool)

	creator := createUser(t, pool, "market-crash-creator")
	stranger := createUser(t, pool, "market-crash-stranger")
	taskID := insertTask(t, pool, creator, "open", 50)
	insertPublicVisibility(t, pool, taskID)
	subscription := createMarketplaceSubscription(t, store, webhook.OwnerUser{ID: stranger}, webhook.NewMarketplaceAudience())

	// Simulated crash: the mutation's transaction recorded the event (Append
	// leaves it in dispatch_state recorded) and the process died before the
	// inline dispatch ran.
	stored := appendTaskOpenedEvent(t, pool, taskID, creator)
	if row := readEventOutboxRow(t, pool, stored.Event.ID); row.dispatchState != "recorded" {
		t.Fatalf("dispatch state before sweep = %q, want recorded", row.dispatchState)
	}
	if got := countDeliveries(t, pool, subscription.ID); got != 0 {
		t.Fatalf("crash window produced %d deliveries before the sweep", got)
	}

	sweepRecordedOutbox(t, pool)
	if row := readEventOutboxRow(t, pool, stored.Event.ID); row.dispatchState != "dispatched" {
		t.Fatalf("dispatch state after sweep = %q, want dispatched", row.dispatchState)
	}
	if got := countDeliveries(t, pool, subscription.ID); got != 1 {
		t.Fatalf("marketplace deliveries after sweep = %d, want 1", got)
	}
	if got := countNotificationsForEvent(t, pool, stored.Event.ID, creator); got != 0 {
		t.Fatalf("task_opened produced %d inbox rows, want 0 (feed-only kind)", got)
	}

	// A second sweep expands nothing new.
	sweepRecordedOutbox(t, pool)
	if got := countDeliveries(t, pool, subscription.ID); got != 1 {
		t.Fatalf("re-sweep duplicated marketplace deliveries: %d", got)
	}
}

// TestOutboxPartialDispatchIsNotDuplicated crashes half-way through the
// dispatch step: the inbox fan-out leg ran, the store-side leg (webhook
// expansion + marking dispatched) did not. The recovery sweep must finish
// the store-side leg without writing a second inbox row.
func TestOutboxPartialDispatchIsNotDuplicated(t *testing.T) {
	pool := newPool(t)
	ledgerStore := db.NewLedgerStore(pool)
	parkActiveSubscriptions(t, pool)
	dispatchAllRecordedEvents(t, pool)

	grantee := createUser(t, pool, "partial-crash-grantee")
	admin := createUser(t, pool, "partial-crash-admin")
	subscription, _ := createWebhookTestSubscription(t, db.NewWebhookStore(pool), webhook.OwnerUser{ID: grantee}, event.KindCreditGranted)

	command := grantStoreCommand(t, ledger.GrantToUser{ID: grantee}, 40, "partial dispatch grant", "partial-crash-"+grantee.String())
	command.Draft = testEventDraft(t, event.KindCreditGranted, admin)
	granted, matched := ledgerStore.GrantCredits(context.Background(), command).(ledger.CreditsGranted)
	if !matched || len(granted.RecordedEvents) != 1 {
		t.Fatalf("grant rejected or wrong event count")
	}
	draft := granted.RecordedEvents[0]

	// The notification leg completed (as Recorder.Dispatch runs it, keyed to
	// the event id)…
	notifications := notification.NewService(db.NewNotificationStore(pool))
	if _, created := notifications.Notify(context.Background(), grantee, admin, notification.KindCreditGranted,
		event.NotificationSubjectFor(draft.Subject), notification.Metadata{JSON: draft.Metadata.JSON},
		notification.FromEvent{ID: draft.ID}).(notification.NotificationCreated); !created {
		t.Fatalf("notification leg rejected")
	}
	// …but the store-side dispatch never ran: the event is still recorded and
	// no delivery exists.
	if row := readEventOutboxRow(t, pool, draft.ID); row.dispatchState != "recorded" {
		t.Fatalf("dispatch state after partial dispatch = %q, want recorded", row.dispatchState)
	}
	if got := countDeliveries(t, pool, subscription.ID); got != 0 {
		t.Fatalf("partial dispatch produced %d deliveries", got)
	}

	// The sweep completes the dispatch: the delivery appears, the event flips
	// to dispatched, and the already-written inbox row is not duplicated.
	sweepRecordedOutbox(t, pool)
	if row := readEventOutboxRow(t, pool, draft.ID); row.dispatchState != "dispatched" {
		t.Fatalf("dispatch state after sweep = %q, want dispatched", row.dispatchState)
	}
	if got := countDeliveries(t, pool, subscription.ID); got != 1 {
		t.Fatalf("deliveries after sweep = %d, want 1", got)
	}
	if got := countNotificationsForEvent(t, pool, draft.ID, grantee); got != 1 {
		t.Fatalf("notifications after sweep = %d, want exactly the pre-crash row", got)
	}
}

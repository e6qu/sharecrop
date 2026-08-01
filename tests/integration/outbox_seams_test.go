//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/webhook"
	"github.com/jackc/pgx/v5/pgxpool"
)

func maxEventSequence(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	var sequence int64
	if err := pool.QueryRow(context.Background(), "select coalesce(max(seq), 0) from domain_events").Scan(&sequence); err != nil {
		t.Fatalf("read max event seq: %v", err)
	}
	return sequence
}

func listFeedAfter(t *testing.T, pool *pgxpool.Pool, recipient core.UserID, after int64) []event.StoredEvent {
	t.Helper()
	result := db.NewEventStore(pool).ListForRecipient(context.Background(), recipient,
		event.After{Cursor: event.CursorFromSequence(after)}, core.DefaultPage())
	listed, matched := result.(event.ListStoreAccepted)
	if !matched {
		t.Fatalf("feed read rejected: %#v", result)
	}
	return listed.Values
}

// TestCursorFeedNeverSkipsLateCommittingEvent reproduces the outbox ordering
// race: a transaction inserts an event (allocating a lower seq) and stays
// open while a second writer appends a later event. Without commit-ordered
// sequencing the second event commits first, a cursor consumer advances past
// the still-invisible lower seq, and the first event is skipped forever once
// its transaction commits. The domain_event_fence serializes {seq allocation
// -> commit}, so the second writer must block until the first transaction
// commits and the feed can never observe events out of seq order.
func TestCursorFeedNeverSkipsLateCommittingEvent(t *testing.T) {
	pool := newPool(t)
	actor := createUser(t, pool, "fence-actor")
	recipient := createUser(t, pool, "fence-recipient")
	baseline := maxEventSequence(t, pool)

	// Writer 1: an in-flight transaction holding the fence with a committed-
	// later event — the exact statements insertDomainEventInTx runs.
	tx1, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin writer 1: %v", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx1.Rollback(context.Background())
		}
	}()
	if _, err := tx1.Exec(context.Background(), "select id from domain_event_fence where id = 1 for update"); err != nil {
		t.Fatalf("writer 1 fence: %v", err)
	}
	firstID, matched := core.NewDomainEventID().(core.DomainEventIDCreated)
	if !matched {
		t.Fatalf("event id rejected")
	}
	var firstSeq int64
	if err := tx1.QueryRow(context.Background(), `
		insert into domain_events (id, kind, actor_kind, actor_user_id, metadata_json, occurred_at, dispatch_state)
		values ($1, 'task_opened', 'user', $2, '{}'::jsonb, now(), 'recorded')
		returning seq
	`, firstID.Value.String(), actor.String()).Scan(&firstSeq); err != nil {
		t.Fatalf("writer 1 insert: %v", err)
	}
	if _, err := tx1.Exec(context.Background(), `
		insert into domain_event_recipients (event_seq, user_id) values ($1, $2)
	`, firstSeq, recipient.String()); err != nil {
		t.Fatalf("writer 1 recipient: %v", err)
	}

	// Writer 2: the store Append path. It must block on the fence until
	// writer 1 commits.
	appended := make(chan event.AppendStoreResult, 1)
	go func() {
		subject := event.NoSubjectRefs()
		draftCreated := event.NewDraft(event.KindTaskOpened, event.ActorUser{ID: actor}, subject,
			event.EmptyMetadata(), event.NewRecipients(recipient)).(event.DraftCreated)
		appended <- db.NewEventStore(pool).Append(context.Background(), draftCreated.Value.Event(time.Now()), draftCreated.Value.Recipients)
	}()

	// While writer 1 is open, writer 2 must not have committed a later-seq
	// event (this is the pre-fix skip: the feed would have served writer 2's
	// event and the cursor would advance past writer 1's still-invisible
	// seq).
	select {
	case result := <-appended:
		t.Fatalf("append completed while an earlier-seq transaction was still open: %#v", result)
	case <-time.After(500 * time.Millisecond):
	}
	if visible := listFeedAfter(t, pool, recipient, baseline); len(visible) != 0 {
		t.Fatalf("feed served %d events while the fence holder was uncommitted", len(visible))
	}

	if err := tx1.Commit(context.Background()); err != nil {
		t.Fatalf("writer 1 commit: %v", err)
	}
	committed = true
	select {
	case result := <-appended:
		if _, matched := result.(event.AppendStoreAccepted); !matched {
			t.Fatalf("append rejected after fence release: %#v", result)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("append did not complete after the fence holder committed")
	}

	// A cursor walk from the baseline sees both events, in seq order, with
	// writer 1's earlier seq first — no skip.
	visible := listFeedAfter(t, pool, recipient, baseline)
	if len(visible) != 2 {
		t.Fatalf("feed served %d events after both commits, want 2", len(visible))
	}
	if visible[0].Event.ID != firstID.Value {
		t.Fatalf("first served event is %s, want the earlier-seq writer 1 event", visible[0].Event.ID.String())
	}
	if visible[0].Cursor.Sequence() >= visible[1].Cursor.Sequence() {
		t.Fatalf("feed order not by seq: %d then %d", visible[0].Cursor.Sequence(), visible[1].Cursor.Sequence())
	}
	// Resuming from the first event's cursor still serves the second.
	tail := listFeedAfter(t, pool, recipient, visible[0].Cursor.Sequence())
	if len(tail) != 1 || tail[0].Event.ID != visible[1].Event.ID {
		t.Fatalf("cursor resume skipped the later event")
	}
	sweepRecordedOutbox(t, pool)
}

// TestRequestPathReleaseRecordsExpiryEventExactlyOnce covers the swallowed
// reservation_expired fix: a read path that releases a lapsed reservation
// BEFORE the lifecycle sweep runs must record the reservation_expired event
// in its own transaction, the sweep must not record a second one, and the
// dispatch sweep must fan the notification out to the holder exactly once.
func TestRequestPathReleaseRecordsExpiryEventExactlyOnce(t *testing.T) {
	pool := newPool(t)
	store := db.NewTaskStore(pool)
	owner := createUser(t, pool, "req-release-owner")
	worker := createUser(t, pool, "req-release-worker")
	taskID := insertTaskWithRewardKind(t, pool, owner, "open", "none")
	reservationID := insertActiveReservation(t, pool, taskID, worker, true)

	// The read-path housekeeping runs first (a ListReservations call), before
	// the sweep.
	if _, matched := store.ListReservations(context.Background(), taskID, core.DefaultPage()).(task.ListReservationsStoreAccepted); !matched {
		t.Fatalf("list reservations rejected")
	}
	var state string
	if err := pool.QueryRow(context.Background(), "select state from task_reservations where id = $1", reservationID.String()).Scan(&state); err != nil {
		t.Fatalf("read reservation state: %v", err)
	}
	if state != "expired" {
		t.Fatalf("reservation state = %q, want expired", state)
	}
	if got := countEventsByKindForTask(t, pool, taskID, event.KindReservationExpired); got != 1 {
		t.Fatalf("reservation_expired events after read-path release = %d, want 1", got)
	}

	// The sweep runs later: it must not record a second event for the
	// already-released reservation.
	if _, matched := store.ExpireDueReservations(context.Background()).(db.ExpireDueReservationsCompleted); !matched {
		t.Fatalf("reservation sweep rejected")
	}
	if got := countEventsByKindForTask(t, pool, taskID, event.KindReservationExpired); got != 1 {
		t.Fatalf("reservation_expired events after sweep = %d, want 1 (no double emission)", got)
	}

	// Dispatch (inline was skipped on the read path by design) delivers the
	// holder's notification exactly once, and a re-sweep does not duplicate.
	var eventID string
	if err := pool.QueryRow(context.Background(), `
		select id::text from domain_events where task_id = $1 and kind = 'reservation_expired'
	`, taskID.String()).Scan(&eventID); err != nil {
		t.Fatalf("read reservation_expired event id: %v", err)
	}
	parsedEventID, matched := core.ParseDomainEventID(eventID).(core.DomainEventIDCreated)
	if !matched {
		t.Fatalf("event id parse failed")
	}
	sweepRecordedOutbox(t, pool)
	if got := countNotificationsForEvent(t, pool, parsedEventID.Value, worker); got != 1 {
		t.Fatalf("holder notifications = %d, want exactly 1", got)
	}
	sweepRecordedOutbox(t, pool)
	if got := countNotificationsForEvent(t, pool, parsedEventID.Value, worker); got != 1 {
		t.Fatalf("holder notifications after re-sweep = %d, want exactly 1", got)
	}
	if row := readEventOutboxRow(t, pool, parsedEventID.Value); row.dispatchState != "dispatched" {
		t.Fatalf("dispatch state = %q, want dispatched", row.dispatchState)
	}
}

// TestExpirySweepCrashWindowRecoveredExactlyOnce covers the whole-batch loss
// fix: the expiry sweeps record their events inside the transition
// transactions, so a crash before the inline dispatch (simulated by simply
// not dispatching) leaves recorded rows the dispatch sweep completes exactly
// once.
func TestExpirySweepCrashWindowRecoveredExactlyOnce(t *testing.T) {
	pool := newPool(t)
	store := db.NewTaskStore(pool)
	owner := createUser(t, pool, "crash-expiry-owner")
	worker := createUser(t, pool, "crash-expiry-worker")

	reservationTask := insertTaskWithRewardKind(t, pool, owner, "open", "none")
	insertActiveReservation(t, pool, reservationTask, worker, true)
	expiringTask := insertTaskWithRewardKind(t, pool, owner, "open", "none")
	insertActiveReservation(t, pool, expiringTask, worker, false)
	if _, err := pool.Exec(context.Background(), "update tasks set expires_at = now() - interval '1 hour' where id = $1", expiringTask.String()); err != nil {
		t.Fatalf("set task expiry: %v", err)
	}

	// Both sweeps run their store half; the process "crashes" before the
	// runner's inline dispatch.
	if _, matched := store.ExpireDueReservations(context.Background()).(db.ExpireDueReservationsCompleted); !matched {
		t.Fatalf("reservation sweep rejected")
	}
	if _, matched := store.ExpireDueTasks(context.Background()).(db.ExpireDueTasksCompleted); !matched {
		t.Fatalf("task sweep rejected")
	}

	var reservationEventID, taskEventID string
	if err := pool.QueryRow(context.Background(), `
		select id::text from domain_events where task_id = $1 and kind = 'reservation_expired'
	`, reservationTask.String()).Scan(&reservationEventID); err != nil {
		t.Fatalf("reservation_expired event missing (whole-batch loss): %v", err)
	}
	if err := pool.QueryRow(context.Background(), `
		select id::text from domain_events where task_id = $1 and kind = 'task_expired'
	`, expiringTask.String()).Scan(&taskEventID); err != nil {
		t.Fatalf("task_expired event missing (whole-batch loss): %v", err)
	}
	reservationEvent := core.ParseDomainEventID(reservationEventID).(core.DomainEventIDCreated).Value
	taskEvent := core.ParseDomainEventID(taskEventID).(core.DomainEventIDCreated).Value
	if row := readEventOutboxRow(t, pool, reservationEvent); row.dispatchState != "recorded" {
		t.Fatalf("reservation event state before recovery = %q, want recorded", row.dispatchState)
	}
	if row := readEventOutboxRow(t, pool, taskEvent); row.dispatchState != "recorded" {
		t.Fatalf("task event state before recovery = %q, want recorded", row.dispatchState)
	}

	// The dispatch sweep recovers both exactly once.
	sweepRecordedOutbox(t, pool)
	sweepRecordedOutbox(t, pool)
	if got := countNotificationsForEvent(t, pool, reservationEvent, worker); got != 1 {
		t.Fatalf("holder reservation_expired notifications = %d, want 1", got)
	}
	if got := countNotificationsForEvent(t, pool, taskEvent, owner); got != 1 {
		t.Fatalf("owner task_expired notifications = %d, want 1", got)
	}
	if row := readEventOutboxRow(t, pool, taskEvent); row.dispatchState != "dispatched" {
		t.Fatalf("task event state after recovery = %q, want dispatched", row.dispatchState)
	}
}

// TestDispatchSweepSkipsMalformedAndRetiresAtCap covers the poison-pill fix:
// a malformed recorded row must not abort the batch (later rows still
// dispatch), and once its attempt counter reaches the cap it is retired to
// the terminal dispatch_failed state, visible through the store's
// operator read.
func TestDispatchSweepSkipsMalformedAndRetiresAtCap(t *testing.T) {
	pool := newPool(t)
	store := db.NewEventStore(pool)
	dispatchAllRecordedEvents(t, pool)
	actor := createUser(t, pool, "poison-actor")
	recipient := createUser(t, pool, "poison-recipient")

	// A malformed recorded row (unknown kind) older than the healthy rows.
	poisonID, _ := core.NewDomainEventID().(core.DomainEventIDCreated)
	if _, err := pool.Exec(context.Background(), `
		insert into domain_events (id, kind, actor_kind, actor_user_id, metadata_json, occurred_at, dispatch_state)
		values ($1, 'task_exploded', 'user', $2, '{}'::jsonb, now() - interval '1 hour', 'recorded')
	`, poisonID.Value.String(), actor.String()); err != nil {
		t.Fatalf("insert poison row: %v", err)
	}
	// A healthy recorded row behind it.
	healthy := appendWebhookTestEvent(t, pool, event.KindCreditGranted, actor, []core.UserID{recipient}, event.NoOrganization{})

	cutoff := time.Now().UTC().Add(time.Hour)
	listed, matched := store.ListRecordedBefore(context.Background(), cutoff, 500).(db.RecordedEventsListed)
	if !matched {
		t.Fatalf("list recorded rejected")
	}
	if listed.SkippedMalformed < 1 {
		t.Fatalf("skipped = %d, want >= 1 (the poison row)", listed.SkippedMalformed)
	}
	foundHealthy := false
	for _, draft := range listed.Values {
		if draft.ID == poisonID.Value {
			t.Fatalf("poison row was returned as a draft")
		}
		if draft.ID == healthy.Event.ID {
			foundHealthy = true
		}
	}
	if !foundHealthy {
		t.Fatalf("healthy row behind the poison row was not claimed (batch poisoned)")
	}
	var attempts int64
	if err := pool.QueryRow(context.Background(), "select dispatch_attempts from domain_events where id = $1", poisonID.Value.String()).Scan(&attempts); err != nil {
		t.Fatalf("read attempts: %v", err)
	}
	if attempts != 1 {
		t.Fatalf("poison attempts after one pass = %d, want 1", attempts)
	}

	// At the cap the next pass retires the row instead of claiming it again.
	if _, err := pool.Exec(context.Background(), "update domain_events set dispatch_attempts = $2 where id = $1", poisonID.Value.String(), db.MaxEventDispatchAttempts); err != nil {
		t.Fatalf("set attempts to cap: %v", err)
	}
	if _, matched := store.ListRecordedBefore(context.Background(), cutoff, 500).(db.RecordedEventsListed); !matched {
		t.Fatalf("second list rejected")
	}
	var state string
	if err := pool.QueryRow(context.Background(), "select dispatch_state from domain_events where id = $1", poisonID.Value.String()).Scan(&state); err != nil {
		t.Fatalf("read poison state: %v", err)
	}
	if state != "dispatch_failed" {
		t.Fatalf("poison state = %q, want dispatch_failed", state)
	}
	failedListed, failedMatched := store.ListDispatchFailed(context.Background(), 500).(db.DispatchFailedEventsListed)
	if !failedMatched {
		t.Fatalf("list dispatch-failed rejected")
	}
	foundPoison := false
	for _, row := range failedListed.Values {
		if row.EventID == poisonID.Value.String() {
			foundPoison = true
			if row.Kind != "task_exploded" || row.Attempts < int64(db.MaxEventDispatchAttempts) {
				t.Fatalf("failed row = %#v", row)
			}
		}
	}
	if !foundPoison {
		t.Fatalf("dispatch-failed read does not expose the retired row")
	}
	sweepRecordedOutbox(t, pool)
}

// TestClaimHoldPreventsReclaimWithinHold covers the duplicate-POST fix: a
// claimed delivery must stay out of the due window for the full
// caller-supplied hold, and becomes claimable again only after it lapses.
func TestClaimHoldPreventsReclaimWithinHold(t *testing.T) {
	pool := newPool(t)
	store := db.NewWebhookStore(pool)
	parkActiveSubscriptions(t, pool)
	dispatchAllRecordedEvents(t, pool)
	parkPendingDeliveries(t, pool)

	recipient := createUser(t, pool, "claim-hold")
	subscription, _ := createWebhookTestSubscription(t, store, webhook.OwnerUser{ID: recipient}, event.KindTaskOpened)
	dispatchStoredEvent(t, pool, appendWebhookTestEvent(t, pool, event.KindTaskOpened, recipient, []core.UserID{recipient}, event.NoOrganization{}))
	forceDeliveriesDue(t, pool, subscription.ID)

	const hold = 2 * time.Second
	claimOurs := func() int {
		result := store.ClaimDueDeliveries(context.Background(), 10, hold)
		claimed, matched := result.(db.ClaimDueDeliveriesListed)
		if !matched {
			t.Fatalf("claim rejected: %#v", result)
		}
		ours := 0
		for _, value := range claimed.Values {
			if value.Subscription == subscription.ID {
				ours++
			}
		}
		return ours
	}

	if got := claimOurs(); got != 1 {
		t.Fatalf("first claim took %d of our deliveries, want 1", got)
	}
	// Within the hold the row must not be claimable by another dispatcher.
	if got := claimOurs(); got != 0 {
		t.Fatalf("re-claim within the hold took %d deliveries, want 0", got)
	}
	// After the hold lapses the (still pending, unmarked) row is claimable
	// again — the crash-recovery half of at-least-once delivery.
	time.Sleep(hold + 500*time.Millisecond)
	if got := claimOurs(); got != 1 {
		t.Fatalf("claim after the hold lapsed took %d deliveries, want 1", got)
	}
}

// TestCancelResolvesReleasedHoldersInTransaction covers the cancel TOCTOU
// fix: the store, not the service, resolves the holders a cancel releases —
// inside the cancel transaction — and merges them into the recorded event's
// recipients.
func TestCancelResolvesReleasedHoldersInTransaction(t *testing.T) {
	pool := newPool(t)
	store := db.NewTaskStore(pool)
	owner := createUser(t, pool, "cancel-toctou-owner")
	worker := createUser(t, pool, "cancel-toctou-worker")
	taskID := insertTaskWithRewardKind(t, pool, owner, "open", "none")
	insertPublicVisibility(t, pool, taskID)
	insertActiveReservation(t, pool, taskID, worker, false)

	// The service-built draft names only the acting owner — exactly what
	// task.Service.Cancel hands over. The active holder becomes a recipient
	// only if the store resolves them in-transaction.
	subject := event.NoSubjectRefs()
	subject.Task = event.TaskSubject{ID: taskID}
	draft := event.NewDraft(event.KindTaskCancelled, event.ActorUser{ID: owner}, subject,
		event.TaskMetadata(taskID), event.NewRecipients(owner)).(event.DraftCreated).Value

	result := store.ChangeTaskState(context.Background(), taskID, task.StateCancelled, event.Record{Draft: draft})
	if _, matched := result.(task.ChangeTaskStateStoreAccepted); !matched {
		t.Fatalf("cancel rejected: %#v", result)
	}
	row := readEventOutboxRow(t, pool, draft.ID)
	if row.recipients[worker.String()] != 1 {
		t.Fatalf("cancel event recipients = %v, want the released holder resolved in-tx", row.recipients)
	}
	if row.recipients[owner.String()] != 1 {
		t.Fatalf("cancel event recipients = %v, want the owner", row.recipients)
	}
	sweepRecordedOutbox(t, pool)
}

//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/webhook"
	"github.com/e6qu/sharecrop/internal/webhookdispatch"
	"github.com/jackc/pgx/v5/pgxpool"
)

// newDBRecorder builds the same recorder wiring production uses: the event
// store and the inbox both on Postgres.
func newDBRecorder(pool *pgxpool.Pool) event.Recorder {
	return event.NewRecorder(db.NewEventStore(pool), notification.NewService(db.NewNotificationStore(pool)))
}

func newDBLedgerService(pool *pgxpool.Pool) ledger.Service {
	return ledger.NewService(db.NewLedgerStore(pool), newDBRecorder(pool), audit.NewService(db.NewAuditStore(pool)))
}

// sweepRecordedOutbox replays the lifecycle runner's dispatch sweep: list
// every recorded event and dispatch it through the recorder.
func sweepRecordedOutbox(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	recorder := newDBRecorder(pool)
	listed, matched := db.NewEventStore(pool).ListRecordedBefore(context.Background(), time.Now().UTC().Add(time.Hour), 500).(db.RecordedEventsListed)
	if !matched {
		t.Fatalf("list recorded events rejected")
	}
	recorder.Dispatch(context.Background(), listed.Values...)
}

type eventOutboxRow struct {
	kind          string
	actorUserID   string
	dispatchState string
	recipients    map[string]int
}

func readEventOutboxRow(t *testing.T, pool *pgxpool.Pool, id core.DomainEventID) eventOutboxRow {
	t.Helper()
	row := eventOutboxRow{recipients: map[string]int{}}
	var sequence int64
	if err := pool.QueryRow(context.Background(), `
		select seq, kind, coalesce(actor_user_id::text, ''), dispatch_state
		from domain_events where id = $1
	`, id.String()).Scan(&sequence, &row.kind, &row.actorUserID, &row.dispatchState); err != nil {
		t.Fatalf("read event row: %v", err)
	}
	rows, err := pool.Query(context.Background(), `select user_id::text from domain_event_recipients where event_seq = $1`, sequence)
	if err != nil {
		t.Fatalf("read event recipients: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			t.Fatalf("scan recipient: %v", err)
		}
		row.recipients[raw]++
	}
	return row
}

func countNotificationsForEvent(t *testing.T, pool *pgxpool.Pool, id core.DomainEventID, recipient core.UserID) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), `
		select count(*) from notifications where event_id = $1 and recipient_user_id = $2
	`, id.String(), recipient.String()).Scan(&count); err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	return count
}

func countEventsByKindForTask(t *testing.T, pool *pgxpool.Pool, taskID core.TaskID, kind event.Kind) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), `
		select count(*) from domain_events where task_id = $1 and kind = $2
	`, taskID.String(), kind.String()).Scan(&count); err != nil {
		t.Fatalf("count events: %v", err)
	}
	return count
}

// TestOutboxCrashWindowIsRecoveredBySweep simulates the crash window the
// outbox exists for: the mutation transaction commits (recording the event
// row and its recipients) but the process dies before the inline dispatch.
// The dispatch sweep must then produce the notifications and webhook
// deliveries exactly once, and re-running the sweep (or an inline dispatch
// racing it) must not duplicate them.
func TestOutboxCrashWindowIsRecoveredBySweep(t *testing.T) {
	pool := newPool(t)
	store := db.NewLedgerStore(pool)
	parkActiveSubscriptions(t, pool)
	dispatchAllRecordedEvents(t, pool)

	grantee := createUser(t, pool, "outbox-crash-grantee")
	admin := createUser(t, pool, "outbox-crash-admin")

	// The grantee subscribes to credit_granted deliveries.
	subscription, _ := createWebhookTestSubscription(t, db.NewWebhookStore(pool), webhook.OwnerUser{ID: grantee}, event.KindCreditGranted)

	// Simulated crash: call the STORE command directly (as the service does
	// inside its transaction) and never run the inline dispatch.
	command := grantStoreCommand(t, ledger.GrantToUser{ID: grantee}, 55, "crash window grant", "outbox-crash-"+grantee.String())
	command.Draft = testEventDraft(t, event.KindCreditGranted, admin)
	granted, matched := store.GrantCredits(context.Background(), command).(ledger.CreditsGranted)
	if !matched {
		t.Fatalf("grant rejected")
	}
	if _, first := granted.Execution.(ledger.FirstExecution); !first {
		t.Fatalf("fresh grant execution = %T, want FirstExecution", granted.Execution)
	}
	if len(granted.RecordedEvents) != 1 {
		t.Fatalf("recorded events = %d, want 1", len(granted.RecordedEvents))
	}
	eventID := granted.RecordedEvents[0].ID

	row := readEventOutboxRow(t, pool, eventID)
	if row.dispatchState != "recorded" {
		t.Fatalf("dispatch state before sweep = %q, want recorded", row.dispatchState)
	}
	if row.recipients[grantee.String()] != 1 {
		t.Fatalf("event recipients missed the grantee: %v", row.recipients)
	}
	if got := countNotificationsForEvent(t, pool, eventID, grantee); got != 0 {
		t.Fatalf("crash window produced %d notifications before the sweep", got)
	}
	if got := countDeliveries(t, pool, subscription.ID); got != 0 {
		t.Fatalf("crash window produced %d deliveries before the sweep", got)
	}

	// The sweep recovers: notifications + deliveries appear exactly once.
	sweepRecordedOutbox(t, pool)
	if row := readEventOutboxRow(t, pool, eventID); row.dispatchState != "dispatched" {
		t.Fatalf("dispatch state after sweep = %q, want dispatched", row.dispatchState)
	}
	if got := countNotificationsForEvent(t, pool, eventID, grantee); got != 1 {
		t.Fatalf("notifications after sweep = %d, want 1", got)
	}
	if got := countDeliveries(t, pool, subscription.ID); got != 1 {
		t.Fatalf("deliveries after sweep = %d, want 1", got)
	}

	// A second sweep and a late inline dispatch are both no-ops.
	sweepRecordedOutbox(t, pool)
	newDBRecorder(pool).Dispatch(context.Background(), granted.RecordedEvents...)
	if got := countNotificationsForEvent(t, pool, eventID, grantee); got != 1 {
		t.Fatalf("notifications after re-dispatch = %d, want 1", got)
	}
	if got := countDeliveries(t, pool, subscription.ID); got != 1 {
		t.Fatalf("deliveries after re-dispatch = %d, want 1", got)
	}
}

// TestReplayedCommandsRecordNoNewEvents proves the replay-aware store
// surface: fund_task, accept_submission, request_changes, reject_submission,
// refund_task, and grant_credits each detect an idempotent replay, report it,
// and record no second event (the original execution already recorded one).
func TestReplayedCommandsRecordNoNewEvents(t *testing.T) {
	pool := newPool(t)
	service := newDBLedgerService(pool)

	owner := createUser(t, pool, "outbox-replay-owner")
	worker := createUser(t, pool, "outbox-replay-worker")

	// fund_task
	fundTask := insertTask(t, pool, owner, "draft", 20)
	fundKey := idempotencyKey(t, "replay-fund-"+fundTask.String())
	if funded, matched := service.FundTask(context.Background(), owner, fundTask, creditAmount(t, 20), fundKey).(ledger.TaskFunded); !matched {
		t.Fatalf("fund rejected")
	} else if _, first := funded.Execution.(ledger.FirstExecution); !first {
		t.Fatalf("first fund execution = %T", funded.Execution)
	}
	refunded, matched := service.FundTask(context.Background(), owner, fundTask, creditAmount(t, 20), fundKey).(ledger.TaskFunded)
	if !matched {
		t.Fatalf("replayed fund rejected")
	}
	if _, replay := refunded.Execution.(ledger.IdempotentReplay); !replay {
		t.Fatalf("replayed fund execution = %T, want IdempotentReplay", refunded.Execution)
	}
	if len(refunded.RecordedEvents) != 0 {
		t.Fatalf("replayed fund recorded %d events", len(refunded.RecordedEvents))
	}
	if got := countEventsByKindForTask(t, pool, fundTask, event.KindTaskFunded); got != 1 {
		t.Fatalf("task_funded events = %d, want 1", got)
	}

	// accept_submission
	acceptTask := insertTask(t, pool, owner, "draft", 30)
	acceptKey := "replay-accept-" + acceptTask.String()
	if _, matched := service.FundTask(context.Background(), owner, acceptTask, creditAmount(t, 30), idempotencyKey(t, "fund-"+acceptTask.String())).(ledger.TaskFunded); !matched {
		t.Fatalf("fund for accept rejected")
	}
	setTaskState(t, pool, acceptTask, "open")
	acceptSubmission := insertSubmission(t, pool, acceptTask, worker)
	accepted, acceptedMatched := service.AcceptSubmission(context.Background(), owner, acceptTask, acceptSubmission, idempotencyKey(t, acceptKey)).(ledger.SubmissionAccepted)
	if !acceptedMatched {
		t.Fatalf("accept rejected")
	}
	if _, first := accepted.Execution.(ledger.FirstExecution); !first {
		t.Fatalf("first accept execution = %T", accepted.Execution)
	}
	replayedAccept, replayedMatched := service.AcceptSubmission(context.Background(), owner, acceptTask, acceptSubmission, idempotencyKey(t, acceptKey)).(ledger.SubmissionAccepted)
	if !replayedMatched {
		t.Fatalf("replayed accept rejected")
	}
	if _, replay := replayedAccept.Execution.(ledger.IdempotentReplay); !replay {
		t.Fatalf("replayed accept execution = %T", replayedAccept.Execution)
	}
	if len(replayedAccept.RecordedEvents) != 0 {
		t.Fatalf("replayed accept recorded %d events", len(replayedAccept.RecordedEvents))
	}
	if got := countEventsByKindForTask(t, pool, acceptTask, event.KindSubmissionAccepted); got != 1 {
		t.Fatalf("submission_accepted events = %d, want 1", got)
	}
	if got := countEventsByKindForTask(t, pool, acceptTask, event.KindPayoutReceived); got != 1 {
		t.Fatalf("payout_received events = %d, want 1", got)
	}

	// request_changes
	changesTask := insertTask(t, pool, owner, "open", 10)
	changesSubmission := insertSubmission(t, pool, changesTask, worker)
	changesKey := idempotencyKey(t, "replay-changes-"+changesTask.String())
	note := reviewNote(t, "Refresh the numbers, please.")
	if changed, matched := service.RequestChanges(context.Background(), owner, changesTask, changesSubmission, changesKey, note).(ledger.ChangesRequested); !matched {
		t.Fatalf("request changes rejected")
	} else if _, first := changed.Execution.(ledger.FirstExecution); !first {
		t.Fatalf("first request-changes execution = %T", changed.Execution)
	}
	rechanged, rechangedMatched := service.RequestChanges(context.Background(), owner, changesTask, changesSubmission, changesKey, note).(ledger.ChangesRequested)
	if !rechangedMatched {
		t.Fatalf("replayed request changes rejected")
	}
	if _, replay := rechanged.Execution.(ledger.IdempotentReplay); !replay {
		t.Fatalf("replayed request-changes execution = %T", rechanged.Execution)
	}
	if got := countEventsByKindForTask(t, pool, changesTask, event.KindSubmissionChangesRequested); got != 1 {
		t.Fatalf("submission_changes_requested events = %d, want 1", got)
	}

	// reject_submission
	rejectTask := insertTask(t, pool, owner, "open", 10)
	rejectSubmission := insertSubmission(t, pool, rejectTask, worker)
	rejectKey := idempotencyKey(t, "replay-reject-"+rejectTask.String())
	if _, matched := service.RejectSubmission(context.Background(), owner, rejectTask, rejectSubmission, rejectKey, note, ledger.NoCreditReviewSelection{}, ledger.NoTipSelection{}, ledger.NoBanSelection{}).(ledger.SubmissionRejected); !matched {
		t.Fatalf("reject rejected")
	}
	rerejected, rerejectedMatched := service.RejectSubmission(context.Background(), owner, rejectTask, rejectSubmission, rejectKey, note, ledger.NoCreditReviewSelection{}, ledger.NoTipSelection{}, ledger.NoBanSelection{}).(ledger.SubmissionRejected)
	if !rerejectedMatched {
		t.Fatalf("replayed reject rejected")
	}
	if _, replay := rerejected.Execution.(ledger.IdempotentReplay); !replay {
		t.Fatalf("replayed reject execution = %T", rerejected.Execution)
	}
	if got := countEventsByKindForTask(t, pool, rejectTask, event.KindSubmissionRejected); got != 1 {
		t.Fatalf("submission_rejected events = %d, want 1", got)
	}

	// refund_task: a successful refund cancels the task, and a repeated
	// refund of a cancelled task is rejected before the idempotency lookup
	// (lockTaskRefundable's not-yet-awarded gate), so the replay check here
	// is that the retry records no second task_cancelled event.
	refundTask := insertTask(t, pool, owner, "draft", 15)
	refundKeyRaw := "replay-refund-" + refundTask.String()
	if _, matched := service.FundTask(context.Background(), owner, refundTask, creditAmount(t, 15), idempotencyKey(t, "fund-"+refundTask.String())).(ledger.TaskFunded); !matched {
		t.Fatalf("fund for refund rejected")
	}
	if first, matched := service.RefundTask(context.Background(), owner, refundTask, idempotencyKey(t, refundKeyRaw)).(ledger.TaskRefunded); !matched {
		t.Fatalf("refund rejected")
	} else if _, isFirst := first.Execution.(ledger.FirstExecution); !isFirst {
		t.Fatalf("first refund execution = %T", first.Execution)
	}
	if _, rejected := service.RefundTask(context.Background(), owner, refundTask, idempotencyKey(t, refundKeyRaw)).(ledger.RefundRejected); !rejected {
		t.Fatalf("refund of an already-cancelled task must be rejected")
	}
	if got := countEventsByKindForTask(t, pool, refundTask, event.KindTaskCancelled); got != 1 {
		t.Fatalf("task_cancelled events = %d, want 1", got)
	}

	// grant_credits
	grantKey := idempotencyKey(t, "replay-grant-"+worker.String())
	grantNote, grantNoteMatched := ledger.NewGrantNote("replay test grant").(ledger.GrantNoteAccepted)
	if !grantNoteMatched {
		t.Fatalf("grant note rejected")
	}
	if granted, matched := service.GrantCredits(context.Background(), owner, ledger.GrantToUser{ID: worker}, creditAmount(t, 5), grantNote.Value, grantKey).(ledger.CreditsGranted); !matched {
		t.Fatalf("grant rejected")
	} else if _, first := granted.Execution.(ledger.FirstExecution); !first {
		t.Fatalf("first grant execution = %T", granted.Execution)
	}
	regranted, regrantedMatched := service.GrantCredits(context.Background(), owner, ledger.GrantToUser{ID: worker}, creditAmount(t, 5), grantNote.Value, grantKey).(ledger.CreditsGranted)
	if !regrantedMatched {
		t.Fatalf("replayed grant rejected")
	}
	if _, replay := regranted.Execution.(ledger.IdempotentReplay); !replay {
		t.Fatalf("replayed grant execution = %T", regranted.Execution)
	}
	if len(regranted.RecordedEvents) != 0 {
		t.Fatalf("replayed grant recorded %d events", len(regranted.RecordedEvents))
	}
}

// TestAcceptSupersedesCompetingSubmissions proves the multi-submission dead
// end is closed: accepting one submission moves every OTHER still-submitted
// submission to superseded in the same transaction, each with its own event
// addressed to its submitter; already-reviewed submissions are untouched, and
// a replayed accept never re-supersedes.
func TestAcceptSupersedesCompetingSubmissions(t *testing.T) {
	pool := newPool(t)
	service := newDBLedgerService(pool)

	owner := createUser(t, pool, "supersede-owner")
	winner := createUser(t, pool, "supersede-winner")
	loser := createUser(t, pool, "supersede-loser")
	reviewedWorker := createUser(t, pool, "supersede-reviewed")

	taskID := insertTask(t, pool, owner, "draft", 25)
	if _, matched := service.FundTask(context.Background(), owner, taskID, creditAmount(t, 25), idempotencyKey(t, "fund-"+taskID.String())).(ledger.TaskFunded); !matched {
		t.Fatalf("fund rejected")
	}
	setTaskState(t, pool, taskID, "open")
	winningSubmission := insertSubmission(t, pool, taskID, winner)
	competingSubmission := insertSubmission(t, pool, taskID, loser)
	reviewedSubmission := insertSubmission(t, pool, taskID, reviewedWorker)
	if _, err := pool.Exec(context.Background(), "update submissions set state = 'changes_requested' where id = $1", reviewedSubmission.String()); err != nil {
		t.Fatalf("mark reviewed submission: %v", err)
	}

	acceptKey := idempotencyKey(t, "supersede-accept-"+taskID.String())
	accepted, matched := service.AcceptSubmission(context.Background(), owner, taskID, winningSubmission, acceptKey).(ledger.SubmissionAccepted)
	if !matched {
		t.Fatalf("accept rejected")
	}

	states := map[string]string{}
	rows, err := pool.Query(context.Background(), "select id::text, state from submissions where task_id = $1", taskID.String())
	if err != nil {
		t.Fatalf("read submissions: %v", err)
	}
	for rows.Next() {
		var id, state string
		if err := rows.Scan(&id, &state); err != nil {
			t.Fatalf("scan submission: %v", err)
		}
		states[id] = state
	}
	rows.Close()
	if states[winningSubmission.String()] != "accepted" {
		t.Fatalf("winning submission state = %q", states[winningSubmission.String()])
	}
	if states[competingSubmission.String()] != "superseded" {
		t.Fatalf("competing submission state = %q, want superseded", states[competingSubmission.String()])
	}
	if states[reviewedSubmission.String()] != "changes_requested" {
		t.Fatalf("reviewed submission state = %q, must stay changes_requested", states[reviewedSubmission.String()])
	}

	// Exactly one superseded event, addressed to the competing submitter.
	if got := countEventsByKindForTask(t, pool, taskID, event.KindSubmissionSuperseded); got != 1 {
		t.Fatalf("submission_superseded events = %d, want 1", got)
	}
	var supersededDraft event.Draft
	found := false
	for _, draft := range accepted.RecordedEvents {
		if draft.Kind == event.KindSubmissionSuperseded {
			supersededDraft = draft
			found = true
		}
	}
	if !found {
		t.Fatalf("accept result carried no superseded draft: %d drafts", len(accepted.RecordedEvents))
	}
	row := readEventOutboxRow(t, pool, supersededDraft.ID)
	if row.recipients[loser.String()] != 1 || len(row.recipients) != 1 {
		t.Fatalf("superseded recipients = %v, want exactly the competing submitter", row.recipients)
	}

	// Replaying the accept re-supersedes nothing.
	if _, matched := service.AcceptSubmission(context.Background(), owner, taskID, winningSubmission, acceptKey).(ledger.SubmissionAccepted); !matched {
		t.Fatalf("replayed accept rejected")
	}
	if got := countEventsByKindForTask(t, pool, taskID, event.KindSubmissionSuperseded); got != 1 {
		t.Fatalf("submission_superseded events after replay = %d, want 1", got)
	}
}

// TestOrganizationFundCarriesActingUser proves the attribution fix: an
// organization-funded task records the acting member (not the system actor)
// as the event's actor, with the acting user and the task creator as
// recipients.
func TestOrganizationFundCarriesActingUser(t *testing.T) {
	pool := newPool(t)
	service := newDBLedgerService(pool)

	organizationID := createOrganization(t, pool, "org-fund-actor")
	actingUser := createUser(t, pool, "org-fund-acting")
	taskOwner := createUser(t, pool, "org-fund-owner")

	grantNote, _ := ledger.NewGrantNote("org funding budget").(ledger.GrantNoteAccepted)
	if _, matched := service.GrantCredits(context.Background(), actingUser, ledger.GrantToOrganization{ID: organizationID}, creditAmount(t, 200), grantNote.Value, idempotencyKey(t, "org-fund-budget-"+organizationID.String())).(ledger.CreditsGranted); !matched {
		t.Fatalf("organization budget grant rejected")
	}

	taskID := newTaskID(t)
	if _, err := pool.Exec(context.Background(), `
		insert into tasks (id, owner_kind, organization_id, title, description, reward_kind, reward_credit_amount, state, response_schema_json, data_payload_kind, created_by_user_id)
		values ($1, 'organization', $2, 'Org funded task', 'Org funded task description', 'credit', 60, 'draft', '{}'::jsonb, 'none', $3)
	`, taskID.String(), organizationID.String(), taskOwner.String()); err != nil {
		t.Fatalf("insert organization task: %v", err)
	}

	funded, matched := service.FundTaskFromOrganization(context.Background(), actingUser, organizationID, taskID, creditAmount(t, 60), idempotencyKey(t, "org-fund-"+taskID.String())).(ledger.TaskFunded)
	if !matched {
		t.Fatalf("organization fund rejected")
	}
	if len(funded.RecordedEvents) != 1 {
		t.Fatalf("recorded events = %d, want 1", len(funded.RecordedEvents))
	}

	row := readEventOutboxRow(t, pool, funded.RecordedEvents[0].ID)
	if row.kind != "task_funded" {
		t.Fatalf("event kind = %q", row.kind)
	}
	if row.actorUserID != actingUser.String() {
		t.Fatalf("event actor = %q, want the acting user %q", row.actorUserID, actingUser.String())
	}
	if row.recipients[actingUser.String()] != 1 || row.recipients[taskOwner.String()] != 1 {
		t.Fatalf("event recipients = %v, want acting user and task creator", row.recipients)
	}
	// The task creator (not the actor) received the inbox row.
	if got := countNotificationsForEvent(t, pool, funded.RecordedEvents[0].ID, taskOwner); got != 1 {
		t.Fatalf("owner notifications = %d, want 1", got)
	}
}

// TestRefundRecipientsIncludeReleasedHolder proves the second attribution
// fix: refunding (cancelling) a task addresses the task_cancelled event to
// the released active-reservation holder, so the worker learns the task is
// gone.
func TestRefundRecipientsIncludeReleasedHolder(t *testing.T) {
	pool := newPool(t)
	service := newDBLedgerService(pool)

	owner := createUser(t, pool, "refund-holder-owner")
	holder := createUser(t, pool, "refund-holder-worker")
	taskID := insertTask(t, pool, owner, "draft", 30)
	if _, matched := service.FundTask(context.Background(), owner, taskID, creditAmount(t, 30), idempotencyKey(t, "fund-"+taskID.String())).(ledger.TaskFunded); !matched {
		t.Fatalf("fund rejected")
	}
	setTaskState(t, pool, taskID, "open")
	insertActiveReservation(t, pool, taskID, holder, false)

	refunded, matched := service.RefundTask(context.Background(), owner, taskID, idempotencyKey(t, "refund-"+taskID.String())).(ledger.TaskRefunded)
	if !matched {
		t.Fatalf("refund rejected")
	}
	if len(refunded.RecordedEvents) != 1 {
		t.Fatalf("recorded events = %d, want 1", len(refunded.RecordedEvents))
	}
	row := readEventOutboxRow(t, pool, refunded.RecordedEvents[0].ID)
	if row.recipients[holder.String()] != 1 {
		t.Fatalf("task_cancelled recipients = %v, want the released holder", row.recipients)
	}
	if got := countNotificationsForEvent(t, pool, refunded.RecordedEvents[0].ID, holder); got != 1 {
		t.Fatalf("holder notifications = %d, want 1", got)
	}
	var reservationState string
	if err := pool.QueryRow(context.Background(), "select state from task_reservations where task_id = $1 and user_id = $2", taskID.String(), holder.String()).Scan(&reservationState); err != nil {
		t.Fatalf("read reservation: %v", err)
	}
	if reservationState != "cancelled_by_requester" {
		t.Fatalf("reservation state after refund = %q", reservationState)
	}
}

// TestTaskListFundedFilterMatrix proves the funded-state list column and
// filter: a funded credit-reward task, an unfunded one, and a no-credit task
// each surface the right state and are selected by exactly their filter.
func TestTaskListFundedFilterMatrix(t *testing.T) {
	pool := newPool(t)
	taskStore := db.NewTaskStore(pool)
	ledgerStore := db.NewLedgerStore(pool)

	owner := createUser(t, pool, "funded-matrix-owner")
	viewer := createUser(t, pool, "funded-matrix-viewer")
	marker := "funded-matrix-" + owner.String()

	fundedTask := insertTask(t, pool, owner, "draft", 40)
	if _, matched := ledgerStore.FundTask(context.Background(), fundCommand(t, owner, fundedTask, 40, "fund-matrix-"+fundedTask.String())).(ledger.TaskFunded); !matched {
		t.Fatalf("fund rejected")
	}
	unfundedTask := insertTask(t, pool, owner, "draft", 40)
	noRewardTask := insertTaskWithRewardKind(t, pool, owner, "draft", "none")

	expectedStates := map[string]task.FundedState{
		fundedTask.String():   task.FundedStateRewardFunded,
		unfundedTask.String(): task.FundedStateRewardUnfunded,
		noRewardTask.String(): task.FundedStateNoCreditReward,
	}
	for raw := range expectedStates {
		if _, err := pool.Exec(context.Background(), "update tasks set title = $2, state = 'open' where id = $1", raw, "Task "+marker); err != nil {
			t.Fatalf("mark matrix task: %v", err)
		}
		parsed, _ := core.ParseTaskID(raw).(core.TaskIDCreated)
		insertPublicVisibility(t, pool, parsed.Value)
	}

	search, searchMatched := task.NewSearchText(marker).(task.SearchTextAccepted)
	if !searchMatched {
		t.Fatalf("search text rejected")
	}
	listWith := func(funded task.FundedFilter) map[string]task.FundedState {
		filters := task.NoListFilters()
		filters.Search = task.SearchContains{Value: search.Value}
		filters.Funded = funded
		listed, listMatched := taskStore.ListTasks(context.Background(), task.PublicListScope{ViewerID: viewer, IncludeReserved: true}, filters, requirePage(t, 50, 0)).(task.ListTasksStoreAccepted)
		if !listMatched {
			t.Fatalf("list tasks rejected")
		}
		got := map[string]task.FundedState{}
		for _, item := range listed.Values {
			got[item.Task.ID.String()] = item.Funded
		}
		return got
	}

	all := listWith(task.AnyFundedFilter{})
	if len(all) != 3 {
		t.Fatalf("unfiltered matrix listing = %d tasks, want 3", len(all))
	}
	for raw, want := range expectedStates {
		if all[raw] != want {
			t.Fatalf("task %s funded state = %q, want %q", raw, all[raw].String(), want.String())
		}
	}
	for _, expectation := range []struct {
		filter task.FundedState
		want   string
	}{
		{task.FundedStateRewardFunded, fundedTask.String()},
		{task.FundedStateRewardUnfunded, unfundedTask.String()},
		{task.FundedStateNoCreditReward, noRewardTask.String()},
	} {
		got := listWith(task.FundedEquals{Value: expectation.filter})
		if len(got) != 1 {
			t.Fatalf("filter %q matched %d tasks, want 1", expectation.filter.String(), len(got))
		}
		if _, present := got[expectation.want]; !present {
			t.Fatalf("filter %q missed task %s", expectation.filter.String(), expectation.want)
		}
	}
}

// TestTaskListRowsCarryHolderDisplayName proves the active reservation
// holder is named on list rows.
func TestTaskListRowsCarryHolderDisplayName(t *testing.T) {
	pool := newPool(t)
	taskStore := db.NewTaskStore(pool)

	owner := createUser(t, pool, "holder-name-owner")
	holder := createUser(t, pool, "holder-name-worker")
	viewer := createUser(t, pool, "holder-name-viewer")
	taskID := insertTask(t, pool, owner, "open", 10)
	insertPublicVisibility(t, pool, taskID)
	insertActiveReservation(t, pool, taskID, holder, false)

	search, _ := task.NewSearchText(taskID.String()).(task.SearchTextAccepted)
	filters := task.NoListFilters()
	filters.Search = task.SearchContains{Value: search.Value}
	listed, matched := taskStore.ListTasks(context.Background(), task.PublicListScope{ViewerID: viewer, IncludeReserved: true}, filters, requirePage(t, 10, 0)).(task.ListTasksStoreAccepted)
	if !matched || len(listed.Values) != 1 {
		t.Fatalf("list tasks = %#v, want the one reserved task", listed)
	}
	named, holderMatched := listed.Values[0].HolderDisplayName.(task.HolderNamed)
	if !holderMatched {
		t.Fatalf("holder display name absent on a task with an active user reservation")
	}
	if named.DisplayName.String() == "" {
		t.Fatalf("holder display name is empty")
	}
}

// TestNotificationSubmissionSubjectResolvesTaskTitle proves the inbox read
// model resolves the submission subject's task title through the
// submissions→tasks join.
func TestNotificationSubmissionSubjectResolvesTaskTitle(t *testing.T) {
	pool := newPool(t)
	service := notification.NewService(db.NewNotificationStore(pool))

	owner := createUser(t, pool, "subject-title-owner")
	worker := createUser(t, pool, "subject-title-worker")
	taskID := insertTask(t, pool, owner, "open", 10)
	submissionID := insertSubmission(t, pool, taskID, worker)

	created, matched := service.Notify(context.Background(), owner, worker, notification.KindSubmissionCreated,
		notification.Subject{Kind: "submission", ID: submissionID.String()}, notification.EmptyMetadata(), notification.NoSourceEvent{}).(notification.NotificationCreated)
	if !matched {
		t.Fatalf("notify rejected")
	}

	listed, listMatched := service.List(context.Background(), owner, notification.AnyState{}, requirePage(t, 50, 0)).(notification.NotificationsListed)
	if !listMatched {
		t.Fatalf("list rejected")
	}
	for _, value := range listed.Values {
		if value.ID != created.Value.ID {
			continue
		}
		titled, titleMatched := value.SubjectTitle.(notification.TaskSubjectTitle)
		if !titleMatched {
			t.Fatalf("submission-subject notification carries no task title")
		}
		if titled.Title != "Integration task" {
			t.Fatalf("subject title = %q, want the submission's task title", titled.Title)
		}
		return
	}
	t.Fatalf("created notification not found in the inbox listing")
}

// TestClaimedDeliveryCarriesEnrichment proves webhook delivery payloads are
// enriched at claim/build time with the actor's display name and the task's
// title, without touching the stored event.
func TestClaimedDeliveryCarriesEnrichment(t *testing.T) {
	pool := newPool(t)
	store := db.NewWebhookStore(pool)
	parkActiveSubscriptions(t, pool)
	dispatchAllRecordedEvents(t, pool)
	parkPendingDeliveries(t, pool)

	actor := createUser(t, pool, "claim-enrich-actor")
	taskID := insertTask(t, pool, actor, "open", 10)
	subscription, _ := createWebhookTestSubscription(t, store, webhook.OwnerUser{ID: actor}, event.KindTaskOpened)
	dispatchStoredEvent(t, pool, appendTaskOpenedEvent(t, pool, taskID, actor))

	forceDeliveriesDue(t, pool, subscription.ID)
	claimed, matched := store.ClaimDueDeliveries(context.Background(), 10).(db.ClaimDueDeliveriesListed)
	if !matched {
		t.Fatalf("claim rejected")
	}
	var delivery db.ClaimedWebhookDelivery
	found := false
	for _, value := range claimed.Values {
		if value.Subscription == subscription.ID {
			delivery = value
			found = true
		}
	}
	if !found {
		t.Fatalf("subscription's delivery was not claimed")
	}

	named, namedMatched := delivery.Event.ActorName.(event.ActorNamed)
	if !namedMatched || named.DisplayName.String() == "" {
		t.Fatalf("claimed delivery carries no actor display name")
	}
	titled, titledMatched := delivery.Event.TaskTitle.(event.TaskTitled)
	if !titledMatched || titled.Title != "Integration task" {
		t.Fatalf("claimed delivery task title = %#v", delivery.Event.TaskTitle)
	}

	body, err := webhook.EncodeDeliveryBody(delivery.Event, delivery.Subscription.String())
	if err != nil {
		t.Fatalf("encode delivery body: %v", err)
	}
	var decoded struct {
		Event struct {
			ActorDisplayName string `json:"actor_display_name"`
			TaskTitle        string `json:"task_title"`
		} `json:"event"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode delivery body: %v", err)
	}
	if decoded.Event.ActorDisplayName == "" || decoded.Event.TaskTitle != "Integration task" {
		t.Fatalf("delivery body enrichment = %+v", decoded.Event)
	}
}

// TestRunOnceDrainsBacklogBeyondOneBatch proves the dispatcher's drain loop:
// a backlog larger than one claim batch (10) is fully attempted within a
// single RunOnce cycle instead of trickling out one batch per tick.
func TestRunOnceDrainsBacklogBeyondOneBatch(t *testing.T) {
	pool := newPool(t)
	store := db.NewWebhookStore(pool)
	parkActiveSubscriptions(t, pool)
	dispatchAllRecordedEvents(t, pool)
	parkPendingDeliveries(t, pool)
	receiver := newWebhookReceiver(t, http.StatusOK)

	recipient := createUser(t, pool, "drain-recipient")
	created, createdMatched := webhook.NewService(store).Create(context.Background(), webhook.OwnerUser{ID: recipient},
		webhook.NewEndpointURL(receiver.server.URL+"/hooks").(webhook.EndpointURLAccepted).Value,
		webhook.NewKindFilter([]event.Kind{event.KindTaskOpened}).(webhook.KindFilterAccepted).Value,
		webhook.RecipientAudience{},
	).(webhook.SubscriptionCreated)
	if !createdMatched {
		t.Fatalf("create subscription rejected")
	}

	const backlog = 25
	for range backlog {
		dispatchStoredEvent(t, pool, appendWebhookTestEvent(t, pool, event.KindTaskOpened, recipient, []core.UserID{recipient}, event.NoOrganization{}))
	}
	if count := countDeliveries(t, pool, created.Value.ID); count != backlog {
		t.Fatalf("pending deliveries = %d, want %d", count, backlog)
	}

	dispatcher := newTestDispatcher(t, pool, receiver, webhookdispatch.AllowEveryAddress())
	completed := runDispatcherOnce(t, dispatcher)
	if completed.Attempted < backlog {
		t.Fatalf("one RunOnce attempted %d deliveries, want at least the %d-deep backlog", completed.Attempted, backlog)
	}
	if got := len(receiver.recorded()); got != backlog {
		t.Fatalf("receiver saw %d deliveries in one cycle, want %d", got, backlog)
	}
}

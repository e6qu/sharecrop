package db

import (
	"context"
	"errors"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/task"
)

// This file holds the lifecycle-runner sweep queries. They are struct-only
// methods on TaskStore (not part of task.Store), so they are never bridged
// into the WASI guest: the sweeps run host-side only. Both sweeps record
// their event drafts inside the same transaction as the transitions (the
// in-transaction outbox) and return them, so the runner dispatches inline
// and a crash between commit and dispatch is recovered by the event dispatch
// sweep — never by re-emitting.

type ExpireDueTasksResult interface {
	expireDueTasksResult()
}

type ExpireDueTasksCompleted struct {
	// RecordedEvents are the task_expired drafts recorded inside the expire
	// transactions, one per expired task, for the runner's inline dispatch.
	RecordedEvents []event.Draft
}

type ExpireDueTasksRejected struct {
	Reason core.DomainError
}

func (ExpireDueTasksCompleted) expireDueTasksResult() {}

func (ExpireDueTasksRejected) expireDueTasksResult() {}

// ExpireDueTasks moves every open task whose expires_at instant has passed
// (SQL now()) to the expired state. Per task, inside one transaction, it
// re-checks the state under FOR UPDATE with an expected-state predicate
// (mirroring ChangeTaskState's discipline), refunds escrowed credits to the
// funder under the derived idempotency key "expire:<task_id>", returns held
// collectibles, releases non-terminal reservations, and records the
// task_expired event draft (recipients: owner and released holders). A
// concurrent state change or a repeated run leaves the task untouched (the
// sweep is idempotent).
func (store TaskStore) ExpireDueTasks(ctx context.Context) ExpireDueTasksResult {
	// The candidate scan uses the partial index on open tasks with a non-null
	// expires_at (migration 000041).
	rows, err := store.db.Query(ctx, `
		select id::text
		from tasks
		where state = 'open' and expires_at is not null and expires_at <= now()
		order by expires_at asc
	`)
	if err != nil {
		return ExpireDueTasksRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list due tasks failed")}
	}
	rawIDs := make([]string, 0)
	for rows.Next() {
		var rawID string
		if err := rows.Scan(&rawID); err != nil {
			rows.Close()
			return ExpireDueTasksRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan due task failed")}
		}
		rawIDs = append(rawIDs, rawID)
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		return ExpireDueTasksRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read due tasks failed")}
	}

	recorded := make([]event.Draft, 0, len(rawIDs))
	for _, rawID := range rawIDs {
		rowResult := store.expireOneTask(ctx, rawID)
		switch typed := rowResult.(type) {
		case expireOneTaskExpired:
			recorded = append(recorded, typed.draft)
		case expireOneTaskSkipped:
			// A concurrent transition beat the sweep; nothing to report.
		case expireOneTaskRejected:
			return ExpireDueTasksRejected{Reason: typed.reason}
		}
	}
	return ExpireDueTasksCompleted{RecordedEvents: recorded}
}

type expireOneTaskResult interface {
	expireOneTaskResult()
}

type expireOneTaskExpired struct {
	draft event.Draft
}

type expireOneTaskSkipped struct{}

type expireOneTaskRejected struct {
	reason core.DomainError
}

func (expireOneTaskExpired) expireOneTaskResult() {}

func (expireOneTaskSkipped) expireOneTaskResult() {}

func (expireOneTaskRejected) expireOneTaskResult() {}

func (store TaskStore) expireOneTask(ctx context.Context, rawTaskID string) expireOneTaskResult {
	taskIDResult := core.ParseTaskID(rawTaskID)
	taskIDCreated, taskIDMatched := taskIDResult.(core.TaskIDCreated)
	if !taskIDMatched {
		return expireOneTaskRejected{reason: taskIDResult.(core.TaskIDRejected).Reason}
	}
	taskID := taskIDCreated.Value

	tx, err := store.db.Begin(ctx)
	if err != nil {
		return expireOneTaskRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin expire task transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Re-check under lock: the candidate scan ran outside this transaction,
	// so the task may have been cancelled/awarded meanwhile.
	var state string
	var rawOwnerID string
	scanErr := tx.QueryRow(ctx, `
		select state, created_by_user_id::text
		from tasks
		where id = $1 and expires_at is not null and expires_at <= now()
		for update
	`, rawTaskID).Scan(&state, &rawOwnerID)
	if errors.Is(scanErr, ErrNoRows) {
		return expireOneTaskSkipped{}
	}
	if scanErr != nil {
		return expireOneTaskRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "lock expiring task failed")}
	}
	if state != task.StateOpen.String() {
		return expireOneTaskSkipped{}
	}
	ownerIDResult := core.ParseUserID(rawOwnerID)
	ownerIDCreated, ownerIDMatched := ownerIDResult.(core.UserIDCreated)
	if !ownerIDMatched {
		return expireOneTaskRejected{reason: ownerIDResult.(core.UserIDRejected).Reason}
	}

	holders, holdersReason := lockedReservationHolders(ctx, tx, rawTaskID)
	if holdersReason != nil {
		return expireOneTaskRejected{reason: *holdersReason}
	}

	if reason := refundExpiredTaskFunding(ctx, tx, taskID); reason != nil {
		return expireOneTaskRejected{reason: *reason}
	}
	if reason := releaseReservationsOnCancel(ctx, tx, taskID); reason != nil {
		return expireOneTaskRejected{reason: *reason}
	}

	commandTag, err := tx.Exec(ctx, `
		update tasks
		set state = $2, state_recorded_at = now()
		where id = $1 and state = $3
	`, rawTaskID, task.StateExpired.String(), task.StateOpen.String())
	if err != nil {
		return expireOneTaskRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "expire task failed")}
	}
	if commandTag != 1 {
		return expireOneTaskSkipped{}
	}

	// The task_expired event draft is recorded in the same transaction as
	// the transition, so a crash after commit can lose only the dispatch —
	// which the event dispatch sweep replays — never the event itself.
	subject := event.NoSubjectRefs()
	subject.Task = event.TaskSubject{ID: taskID}
	recipients := append([]core.UserID{ownerIDCreated.Value}, holders...)
	draftResult := event.NewDraft(event.KindTaskExpired, event.ActorSystem{}, subject,
		event.TaskMetadata(taskID), event.NewRecipients(recipients...))
	draftCreated, draftMatched := draftResult.(event.DraftCreated)
	if !draftMatched {
		return expireOneTaskRejected{reason: draftResult.(event.DraftRejected).Reason}
	}
	if err := recordEventDraftInTx(ctx, tx, draftCreated.Value); err != nil {
		return expireOneTaskRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "record task expired event failed")}
	}

	if err := tx.Commit(ctx); err != nil {
		return expireOneTaskRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit expire task transaction failed")}
	}
	return expireOneTaskExpired{draft: draftCreated.Value}
}

// lockedReservationHolders lists the distinct users holding a non-terminal
// reservation on the task, read inside the expire/cancel transaction before
// the reservations are released.
func lockedReservationHolders(ctx context.Context, tx Tx, rawTaskID string) ([]core.UserID, *core.DomainError) {
	rows, err := tx.Query(ctx, `
		select distinct requested_by_user_id::text
		from task_reservations
		where task_id = $1 and state in ('active', 'submitted')
	`, rawTaskID)
	if err != nil {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "list reservation holders failed")
		return nil, &reason
	}
	defer rows.Close()
	holders := make([]core.UserID, 0)
	for rows.Next() {
		var rawUserID string
		if err := rows.Scan(&rawUserID); err != nil {
			reason := core.NewDomainError(core.ErrorCodeInvalidState, "scan reservation holder failed")
			return nil, &reason
		}
		userIDResult := core.ParseUserID(rawUserID)
		userIDCreated, matched := userIDResult.(core.UserIDCreated)
		if !matched {
			reason := userIDResult.(core.UserIDRejected).Reason
			return nil, &reason
		}
		holders = append(holders, userIDCreated.Value)
	}
	if err := rows.Err(); err != nil {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "read reservation holders failed")
		return nil, &reason
	}
	return holders, nil
}

// refundExpiredTaskFunding returns an expiring task's allocated credits to
// their funder as a task_refund ledger entry under the derived idempotency
// key "expire:<task_id>", and returns every held collectible. This is a
// system sweep: no requester authorization runs (lockTaskRefundable is the
// user-facing gate), and the task row is already locked by the caller. An
// unfunded task refunds nothing.
func refundExpiredTaskFunding(ctx context.Context, tx Tx, taskID core.TaskID) *core.DomainError {
	if reason := refundTaskCreditFunding(ctx, tx, taskID, stringPointer("expire:"+taskID.String())); reason != nil {
		return reason
	}
	if reason, rejected := refundHeldCollectibleReward(ctx, tx, taskID); rejected {
		return &reason
	}
	return nil
}

type ExpireDueReservationsResult interface {
	expireDueReservationsResult()
}

type ExpireDueReservationsCompleted struct {
	// RecordedEvents are the reservation_expired drafts recorded inside the
	// release transaction, for the runner's inline dispatch.
	RecordedEvents []event.Draft
}

type ExpireDueReservationsRejected struct {
	Reason core.DomainError
}

func (ExpireDueReservationsCompleted) expireDueReservationsResult() {}

func (ExpireDueReservationsRejected) expireDueReservationsResult() {}

// ExpireDueReservations releases every lapsed reservation and records the
// reservation_expired event drafts in the same transaction — the exact
// helper the request-path housekeeping runs (releaseExpiredReservations), so
// whichever runs first records the events exactly once. The recorded drafts
// are returned for inline dispatch.
func (store TaskStore) ExpireDueReservations(ctx context.Context) ExpireDueReservationsResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return ExpireDueReservationsRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin release due reservations failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()
	drafts, releaseErr := releaseExpiredReservationsInTx(ctx, tx)
	if releaseErr != nil {
		return ExpireDueReservationsRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "release due reservations failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return ExpireDueReservationsRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit release due reservations failed")}
	}
	return ExpireDueReservationsCompleted{RecordedEvents: drafts}
}

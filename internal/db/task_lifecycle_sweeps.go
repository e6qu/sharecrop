package db

import (
	"context"
	"errors"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
)

// This file holds the lifecycle-runner sweep queries. They are struct-only
// methods on TaskStore (not part of task.Store), so they are never bridged
// into the WASI guest: the sweeps run host-side only.

// ExpiredTaskRow reports one task the expiry sweep transitioned to expired,
// with the recipients the runner needs for the task_expired event.
type ExpiredTaskRow struct {
	TaskID  core.TaskID
	OwnerID core.UserID
	// ReleasedHolders are the users whose non-terminal reservations on the
	// task were released by the expiration.
	ReleasedHolders []core.UserID
}

type ExpireDueTasksResult interface {
	expireDueTasksResult()
}

type ExpireDueTasksCompleted struct {
	Values []ExpiredTaskRow
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
// collectibles, and releases non-terminal reservations. A concurrent state
// change or a repeated run leaves the task untouched (the sweep is
// idempotent).
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

	expired := make([]ExpiredTaskRow, 0, len(rawIDs))
	for _, rawID := range rawIDs {
		rowResult := store.expireOneTask(ctx, rawID)
		switch typed := rowResult.(type) {
		case expireOneTaskExpired:
			expired = append(expired, typed.value)
		case expireOneTaskSkipped:
			// A concurrent transition beat the sweep; nothing to report.
		case expireOneTaskRejected:
			return ExpireDueTasksRejected{Reason: typed.reason}
		}
	}
	return ExpireDueTasksCompleted{Values: expired}
}

type expireOneTaskResult interface {
	expireOneTaskResult()
}

type expireOneTaskExpired struct {
	value ExpiredTaskRow
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
	if err := tx.Commit(ctx); err != nil {
		return expireOneTaskRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit expire task transaction failed")}
	}
	return expireOneTaskExpired{value: ExpiredTaskRow{TaskID: taskID, OwnerID: ownerIDCreated.Value, ReleasedHolders: holders}}
}

// lockedReservationHolders lists the distinct users holding a non-terminal
// reservation on the task, read inside the expire transaction before the
// reservations are released.
func lockedReservationHolders(ctx context.Context, tx Tx, rawTaskID string) ([]core.UserID, *core.DomainError) {
	rows, err := tx.Query(ctx, `
		select distinct requested_by_user_id::text
		from task_reservations
		where task_id = $1 and state in ('requested', 'active', 'submitted')
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

// ExpiredReservationRow reports one reservation the expiry sweep released,
// with the recipients the runner needs for the reservation_expired event.
type ExpiredReservationRow struct {
	TaskID        core.TaskID
	ReservationID core.TaskReservationID
	// HolderID is the user who requested (and for an active reservation,
	// held) the reservation.
	HolderID core.UserID
	// TaskOwnerID is the task creator, notified alongside the holder.
	TaskOwnerID core.UserID
}

type ExpireDueReservationsResult interface {
	expireDueReservationsResult()
}

type ExpireDueReservationsCompleted struct {
	Values []ExpiredReservationRow
}

type ExpireDueReservationsRejected struct {
	Reason core.DomainError
}

func (ExpireDueReservationsCompleted) expireDueReservationsResult() {}

func (ExpireDueReservationsRejected) expireDueReservationsResult() {}

// ExpireDueReservations releases every requested/active reservation whose
// expires_at instant has passed — the same statement the request-path
// housekeeping runs (expireReservationsSQL) — and returns the released rows
// so the runner can emit reservation_expired events with correct recipients.
func (store TaskStore) ExpireDueReservations(ctx context.Context) ExpireDueReservationsResult {
	rows, err := store.db.Query(ctx, `
		with released as (
			update task_reservations
			set state = 'expired', state_recorded_at = now()
			where state in ('requested', 'active') and expires_at <= now()
			returning id, task_id, requested_by_user_id
		)
		select released.id::text, released.task_id::text, released.requested_by_user_id::text, tasks.created_by_user_id::text
		from released
		join tasks on tasks.id = released.task_id
	`)
	if err != nil {
		return ExpireDueReservationsRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "release due reservations failed")}
	}
	defer rows.Close()

	values := make([]ExpiredReservationRow, 0)
	for rows.Next() {
		var rawReservationID string
		var rawTaskID string
		var rawHolderID string
		var rawOwnerID string
		if err := rows.Scan(&rawReservationID, &rawTaskID, &rawHolderID, &rawOwnerID); err != nil {
			return ExpireDueReservationsRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan released reservation failed")}
		}
		row, reason := parseExpiredReservationRow(rawReservationID, rawTaskID, rawHolderID, rawOwnerID)
		if reason != nil {
			return ExpireDueReservationsRejected{Reason: *reason}
		}
		values = append(values, row)
	}
	if err := rows.Err(); err != nil {
		return ExpireDueReservationsRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read released reservations failed")}
	}
	return ExpireDueReservationsCompleted{Values: values}
}

func parseExpiredReservationRow(rawReservationID string, rawTaskID string, rawHolderID string, rawOwnerID string) (ExpiredReservationRow, *core.DomainError) {
	reservationIDResult := core.ParseTaskReservationID(rawReservationID)
	reservationIDCreated, reservationIDMatched := reservationIDResult.(core.TaskReservationIDCreated)
	if !reservationIDMatched {
		reason := reservationIDResult.(core.TaskReservationIDRejected).Reason
		return ExpiredReservationRow{}, &reason
	}
	taskIDResult := core.ParseTaskID(rawTaskID)
	taskIDCreated, taskIDMatched := taskIDResult.(core.TaskIDCreated)
	if !taskIDMatched {
		reason := taskIDResult.(core.TaskIDRejected).Reason
		return ExpiredReservationRow{}, &reason
	}
	holderIDResult := core.ParseUserID(rawHolderID)
	holderIDCreated, holderIDMatched := holderIDResult.(core.UserIDCreated)
	if !holderIDMatched {
		reason := holderIDResult.(core.UserIDRejected).Reason
		return ExpiredReservationRow{}, &reason
	}
	ownerIDResult := core.ParseUserID(rawOwnerID)
	ownerIDCreated, ownerIDMatched := ownerIDResult.(core.UserIDCreated)
	if !ownerIDMatched {
		reason := ownerIDResult.(core.UserIDRejected).Reason
		return ExpiredReservationRow{}, &reason
	}
	return ExpiredReservationRow{
		TaskID:        taskIDCreated.Value,
		ReservationID: reservationIDCreated.Value,
		HolderID:      holderIDCreated.Value,
		TaskOwnerID:   ownerIDCreated.Value,
	}, nil
}

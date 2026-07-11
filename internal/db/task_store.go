package db

import (
	"context"
	"errors"

	"github.com/e6qu/sharecrop/internal/attachment"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TaskStore struct {
	pool *pgxpool.Pool
}

func NewTaskStore(pool *pgxpool.Pool) TaskStore {
	return TaskStore{pool: pool}
}

func (store TaskStore) CreateTask(ctx context.Context, seriesID core.TaskSeriesID, taskID core.TaskID, command task.CreateCommand) task.CreateTaskStoreResult {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return task.CreateTaskStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin create task transaction failed")}
	}
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		_ = rollbackErr
	}()

	seriesResult := store.insertSeriesForPlacement(ctx, tx, seriesID, command)
	if rejected, matched := seriesResult.(insertSeriesRejected); matched {
		return task.CreateTaskStoreRejected{Reason: rejected.reason}
	}

	seriesColumns := seriesResult.(insertSeriesAccepted)
	ownerColumns := ownerSQLColumns(command.Owner)
	payloadColumns := payloadSQLColumns(command.Payload)
	rewardColumns := rewardSQLColumns(command.Reward)

	_, err = tx.Exec(ctx, `
		insert into tasks (
			id, series_id, series_position, owner_kind, user_id, team_id, organization_id, title, description,
			task_type, reference_url, reward_kind, reward_credit_amount, participation_policy, assignee_scope, reservation_expires_after_hours,
			state, response_schema_json, data_payload_kind, data_payload_json, created_by_user_id
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18::jsonb, $19, $20::jsonb, $21)
	`, taskID.String(), seriesColumns.seriesID, seriesColumns.position, ownerColumns.kind, ownerColumns.userID, ownerColumns.teamID, ownerColumns.organizationID,
		command.Title.String(), command.Description.String(), command.Type.String(), command.Reference.String(), rewardColumns.kind, rewardColumns.creditAmount, command.Participation.String(), command.AssigneeScope.String(), command.ReservationTTL.Hours(), task.StateDraft.String(), command.ResponseSchema.String(), payloadColumns.kind, payloadColumns.source, command.Actor.ID.String())
	if err != nil {
		return task.CreateTaskStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert task failed")}
	}

	visibilityColumns := visibilitySQLColumns(command.Visibility)
	_, err = tx.Exec(ctx, `
		insert into task_visibility_scopes (task_id, visibility_kind, scope_key, user_id, team_id, organization_id)
		values ($1, $2, $3, $4, $5, $6)
	`, taskID.String(), visibilityColumns.kind, visibilityColumns.scopeKey, visibilityColumns.userID, visibilityColumns.teamID, visibilityColumns.organizationID)
	if err != nil {
		return task.CreateTaskStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert task visibility failed")}
	}

	attachmentsResult := insertTaskAttachments(ctx, tx, taskID, command.Attachments)
	if rejected, matched := attachmentsResult.(insertAttachmentsRejected); matched {
		return task.CreateTaskStoreRejected{Reason: rejected.reason}
	}

	if err := tx.Commit(ctx); err != nil {
		return task.CreateTaskStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit create task transaction failed")}
	}

	found := store.FindTask(ctx, taskID)
	value, matched := found.(task.FindTaskStoreAccepted)
	if !matched {
		rejected := found.(task.FindTaskStoreRejected)
		return task.CreateTaskStoreRejected{Reason: rejected.Reason}
	}
	return task.CreateTaskStoreAccepted{Value: value.Value}
}

func (store TaskStore) FindTask(ctx context.Context, taskID core.TaskID) task.FindTaskStoreResult {
	rows, err := store.pool.Query(ctx, taskSelectSQL()+" where tasks.id = $1", taskID.String())
	if err != nil {
		return task.FindTaskStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "find task failed")}
	}
	defer rows.Close()

	valuesResult := scanTaskRows(rows)
	values, matched := valuesResult.(taskRowsAccepted)
	if !matched {
		rejected := valuesResult.(taskRowsRejected)
		return task.FindTaskStoreRejected{Reason: rejected.reason}
	}
	if len(values.values) != 1 {
		return task.FindTaskStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	return task.FindTaskStoreAccepted{Value: values.values[0]}
}

func (store TaskStore) ChangeTaskState(ctx context.Context, taskID core.TaskID, state task.State) task.ChangeTaskStateStoreResult {
	// The invariant checks and the state write run in one transaction, with the
	// task row locked and the UPDATE predicated on the observed prior state, so
	// a concurrent Cancel+Fund or Open+Refund cannot interleave between the
	// checks and the write (orphaning escrow or reopening a cancelled task).
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return task.ChangeTaskStateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin change task state transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var priorState string
	scanErr := tx.QueryRow(ctx, "select state from tasks where id = $1 for update", taskID.String()).Scan(&priorState)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return task.ChangeTaskStateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	if scanErr != nil {
		return task.ChangeTaskStateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "lock task state failed")}
	}

	if state == task.StateOpen {
		fundResult := requireOpenableReward(ctx, tx, taskID)
		if rejected, matched := fundResult.(openableRewardRejected); matched {
			return task.ChangeTaskStateStoreRejected{Reason: rejected.reason}
		}
	}
	if state == task.StateCancelled {
		// Cancelling a funded task that has not been awarded settles it: the
		// allocated credits and every held collectible are returned to their
		// funder before the task is cancelled, so nothing is orphaned.
		if reason := settleFundedTaskOnCancel(ctx, tx, taskID); reason != nil {
			return task.ChangeTaskStateStoreRejected{Reason: *reason}
		}
		// A cancelled task can never be reserved, submitted to, or reviewed
		// again, so every reservation still held on it must be released in the
		// same transaction. Otherwise it dangles forever: the expiry sweep
		// ignores submitted reservations and no other path clears them, leaving
		// the worker reported as still actively holding the task.
		if reason := releaseReservationsOnCancel(ctx, tx, taskID); reason != nil {
			return task.ChangeTaskStateStoreRejected{Reason: *reason}
		}
	}

	commandTag, err := tx.Exec(ctx, `
		update tasks
		set state = $2, state_recorded_at = now()
		where id = $1 and state = $3
	`, taskID.String(), state.String(), priorState)
	if err != nil {
		return task.ChangeTaskStateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "change task state failed")}
	}
	if commandTag.RowsAffected() != 1 {
		return task.ChangeTaskStateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "task state was not changed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return task.ChangeTaskStateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit change task state transaction failed")}
	}

	found := store.FindTask(ctx, taskID)
	value, matched := found.(task.FindTaskStoreAccepted)
	if !matched {
		rejected := found.(task.FindTaskStoreRejected)
		return task.ChangeTaskStateStoreRejected{Reason: rejected.Reason}
	}
	return task.ChangeTaskStateStoreAccepted{Value: value.Value}
}

type openableRewardResult interface {
	openableRewardResult()
}

type openableRewardAccepted struct{}

type openableRewardRejected struct {
	reason core.DomainError
}

func (openableRewardAccepted) openableRewardResult() {}

func (openableRewardRejected) openableRewardResult() {}

func requireOpenableReward(ctx context.Context, tx pgx.Tx, taskID core.TaskID) openableRewardResult {
	var rewardKind string
	var creditAmount int64
	err := tx.QueryRow(ctx, "select reward_kind, coalesce(reward_credit_amount, 0) from tasks where id = $1", taskID.String()).Scan(&rewardKind, &creditAmount)
	if err != nil {
		return openableRewardRejected{reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	if rewardKind == task.RewardKindCredit.String() || rewardKind == task.RewardKindBundle.String() {
		var allocatedAmount int64
		fundErr := tx.QueryRow(ctx, "select credit_amount from task_funds where task_id = $1", taskID.String()).Scan(&allocatedAmount)
		if fundErr != nil {
			return openableRewardRejected{reason: core.NewDomainError(core.ErrorCodeConflict, "credit reward must be funded before opening")}
		}
		if allocatedAmount != creditAmount {
			return openableRewardRejected{reason: core.NewDomainError(core.ErrorCodeConflict, "allocated credits must match the declared credit reward")}
		}
	}
	if rewardKind == task.RewardKindCollectible.String() || rewardKind == task.RewardKindBundle.String() {
		var heldCollectibles int
		err := tx.QueryRow(ctx, "select count(*) from task_fund_collectibles where task_id = $1", taskID.String()).Scan(&heldCollectibles)
		if err != nil {
			return openableRewardRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "check collectible reward funding failed")}
		}
		if heldCollectibles < 1 {
			return openableRewardRejected{reason: core.NewDomainError(core.ErrorCodeConflict, "collectible reward must be funded before opening")}
		}
	}
	return openableRewardAccepted{}
}

// settleFundedTaskOnCancel returns a cancelled task's allocated credits and every
// held collectibles to their funder. It writes a task_refund ledger entry for
// the credits, returns the collectibles to minted, and deletes the stateless
// task_funds / task_fund_collectibles rows. An unfunded task settles to nothing.
func settleFundedTaskOnCancel(ctx context.Context, tx pgx.Tx, taskID core.TaskID) *core.DomainError {
	var rawFunderAccountID string
	var amount int64
	scanErr := tx.QueryRow(ctx, "select funder_account_id::text, credit_amount from task_funds where task_id = $1 for update", taskID.String()).Scan(&rawFunderAccountID, &amount)
	if scanErr != nil && !errors.Is(scanErr, pgx.ErrNoRows) {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "read task fund failed")
		return &reason
	}
	if scanErr == nil {
		entryResult := core.NewLedgerEntryID()
		entryCreated, matched := entryResult.(core.LedgerEntryIDCreated)
		if !matched {
			reason := entryResult.(core.LedgerEntryIDRejected).Reason
			return &reason
		}
		if _, err := tx.Exec(ctx, `
			insert into ledger_entries (id, account_id, kind, amount, task_id)
			values ($1, $2, 'task_refund', $3, $4)
		`, entryCreated.Value.String(), rawFunderAccountID, amount, taskID.String()); err != nil {
			reason := core.NewDomainError(core.ErrorCodeInvalidState, "insert cancel refund ledger entry failed")
			return &reason
		}
		if _, err := tx.Exec(ctx, "delete from task_funds where task_id = $1", taskID.String()); err != nil {
			reason := core.NewDomainError(core.ErrorCodeInvalidState, "clear task fund failed")
			return &reason
		}
	}
	if reason, rejected := refundHeldCollectibleReward(ctx, tx, taskID); rejected {
		return &reason
	}
	return nil
}

// releaseReservationsOnCancel terminates every non-terminal reservation on a
// task that is being cancelled. A reservation in requested/active/submitted
// moves to cancelled_by_requester, the same terminal state the owner-driven
// submission rejection uses (see internal/db/ledger_store.go). Terminal
// reservations are left untouched.
func releaseReservationsOnCancel(ctx context.Context, tx pgx.Tx, taskID core.TaskID) *core.DomainError {
	if _, err := tx.Exec(ctx, `
		update task_reservations
		set state = $2, state_recorded_at = now()
		where task_id = $1 and state in ('requested', 'active', 'submitted')
	`, taskID.String(), task.ReservationStateCancelledByRequester.String()); err != nil {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "release reservations on cancel failed")
		return &reason
	}
	return nil
}

func (store TaskStore) ListTasks(ctx context.Context, scope task.ListScope, filters task.ListFilters, page core.Page) task.ListTasksStoreResult {
	if err := store.releaseExpiredReservations(ctx); err != nil {
		return task.ListTasksStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "release expired reservations failed")}
	}

	queryResult := listQueryForScope(scope, filters, page)
	query, matched := queryResult.(listQueryAccepted)
	if !matched {
		rejected := queryResult.(listQueryRejected)
		return task.ListTasksStoreRejected{Reason: rejected.reason}
	}

	rows, err := store.pool.Query(ctx, query.sql, query.arguments)
	if err != nil {
		return task.ListTasksStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list tasks failed")}
	}
	defer rows.Close()

	valuesResult := scanTaskListItemRows(rows)
	values, matched := valuesResult.(taskListItemRowsAccepted)
	if !matched {
		rejected := valuesResult.(taskListItemRowsRejected)
		return task.ListTasksStoreRejected{Reason: rejected.reason}
	}
	return task.ListTasksStoreAccepted{Values: values.values}
}

func (store TaskStore) CreateReservation(ctx context.Context, reservationID core.TaskReservationID, command task.ReservationCommand) task.CreateReservationStoreResult {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return task.CreateReservationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin create reservation transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, expireReservationsSQL); err != nil {
		return task.CreateReservationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "release expired reservations failed")}
	}

	var policy string
	var state string
	var ttlHours int
	if err := tx.QueryRow(ctx, "select participation_policy, state, reservation_expires_after_hours from tasks where id = $1 for update", command.TaskID.String()).Scan(&policy, &state, &ttlHours); err != nil {
		return task.CreateReservationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	if state != task.StateOpen.String() {
		return task.CreateReservationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only open tasks can be reserved")}
	}
	if blocked := store.taskSeriesBlocksExecution(ctx, command.TaskID); blocked != nil {
		return task.CreateReservationStoreRejected{Reason: *blocked}
	}

	reservationStateResult := reservationInitialState(policy)
	reservationState, reservationStateMatched := reservationStateResult.(reservationInitialStateAccepted)
	if !reservationStateMatched {
		return task.CreateReservationStoreRejected{Reason: reservationStateResult.(reservationInitialStateRejected).reason}
	}

	assigneeColumns := assigneeSQLColumns(command.Assignee)
	var banned bool
	if err := tx.QueryRow(ctx, `
		select exists(
			select 1 from task_implementor_bans
			where task_id = $1 and assignee_kind = $2
			and coalesce(user_id::text, '') = coalesce($3::text, '')
			and coalesce(team_id::text, '') = coalesce($4::text, '')
			and coalesce(organization_id::text, '') = coalesce($5::text, '')
		)
	`, command.TaskID.String(), assigneeColumns.kind, assigneeColumns.userID, assigneeColumns.teamID, assigneeColumns.organizationID).Scan(&banned); err != nil {
		return task.CreateReservationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check task implementor ban failed")}
	}
	if banned {
		return task.CreateReservationStoreRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "implementor is banned from the task")}
	}

	var existing bool
	if err := tx.QueryRow(ctx, `
		select exists(
			select 1 from task_reservations
			where task_id = $1 and assignee_kind = $2
			and coalesce(user_id::text, '') = coalesce($3::text, '')
			and coalesce(team_id::text, '') = coalesce($4::text, '')
			and coalesce(organization_id::text, '') = coalesce($5::text, '')
			and state in ('requested', 'active')
		)
	`, command.TaskID.String(), assigneeColumns.kind, assigneeColumns.userID, assigneeColumns.teamID, assigneeColumns.organizationID).Scan(&existing); err != nil {
		return task.CreateReservationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check existing reservation failed")}
	}
	if existing {
		return task.CreateReservationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "assignee already has an active or pending reservation")}
	}

	_, err = tx.Exec(ctx, `
		insert into task_reservations (
			id, task_id, assignee_kind, user_id, team_id, organization_id, state, requested_by_user_id, expires_at
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, now() + make_interval(hours => $9))
	`, reservationID.String(), command.TaskID.String(), assigneeColumns.kind, assigneeColumns.userID, assigneeColumns.teamID, assigneeColumns.organizationID, reservationState.value.String(), command.RequestedBy.String(), ttlHours)
	if err != nil {
		return task.CreateReservationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "task already has an active reservation")}
	}

	if err := tx.Commit(ctx); err != nil {
		return task.CreateReservationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit create reservation transaction failed")}
	}

	result := store.findReservation(ctx, reservationID)
	found, matched := result.(reservationFound)
	if !matched {
		return task.CreateReservationStoreRejected{Reason: result.(reservationMissing).reason}
	}
	return task.CreateReservationStoreAccepted{Value: found.value}
}

func (store TaskStore) ChangeReservationState(ctx context.Context, taskID core.TaskID, reservationID core.TaskReservationID, state task.ReservationState) task.ChangeReservationStateStoreResult {
	if err := store.releaseExpiredReservations(ctx); err != nil {
		return task.ChangeReservationStateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "release expired reservations failed")}
	}

	// Bind the mutation to the owning task so a reservation belonging to another
	// task can never be flipped via a task the actor happens to own (IDOR).
	commandTag, err := store.pool.Exec(ctx, `
		update task_reservations
		set state = $2, state_recorded_at = now()
		where id = $1 and task_id = $3 and state in ('requested', 'active')
	`, reservationID.String(), state.String(), taskID.String())
	if err != nil {
		return task.ChangeReservationStateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "reservation state was not changed")}
	}
	if commandTag.RowsAffected() != 1 {
		return task.ChangeReservationStateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "reservation is not pending or active")}
	}

	result := store.findReservation(ctx, reservationID)
	found, matched := result.(reservationFound)
	if !matched {
		return task.ChangeReservationStateStoreRejected{Reason: result.(reservationMissing).reason}
	}
	return task.ChangeReservationStateStoreAccepted{Value: found.value}
}

func (store TaskStore) ListReservations(ctx context.Context, taskID core.TaskID) task.ListReservationsStoreResult {
	if err := store.releaseExpiredReservations(ctx); err != nil {
		return task.ListReservationsStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "release expired reservations failed")}
	}

	rows, err := store.pool.Query(ctx, reservationSelectSQL()+`
		where task_reservations.task_id = $1
		order by task_reservations.created_at
	`, taskID.String())
	if err != nil {
		return task.ListReservationsStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list reservations failed")}
	}
	defer rows.Close()

	valuesResult := scanReservationRows(rows)
	values, matched := valuesResult.(reservationRowsAccepted)
	if !matched {
		return task.ListReservationsStoreRejected{Reason: valuesResult.(reservationRowsRejected).reason}
	}
	return task.ListReservationsStoreAccepted{Values: values.values}
}

func (store TaskStore) CheckSubmissionEligibility(ctx context.Context, taskID core.TaskID, submitterID core.UserID) task.SubmissionEligibilityStoreResult {
	if err := store.releaseExpiredReservations(ctx); err != nil {
		return task.SubmissionEligibilityRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "release expired reservations failed")}
	}

	var policy string
	err := store.pool.QueryRow(ctx, "select participation_policy from tasks where id = $1", taskID.String()).Scan(&policy)
	if err != nil {
		return task.SubmissionEligibilityRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	if blocked := store.taskSeriesBlocksExecution(ctx, taskID); blocked != nil {
		return task.SubmissionEligibilityRejected{Reason: *blocked}
	}
	var banned bool
	if err := store.pool.QueryRow(ctx, `
		select exists(
			select 1 from task_implementor_bans
			where task_id = $1
			and assignee_kind = 'user'
			and user_id = $2
		)
	`, taskID.String(), submitterID.String()).Scan(&banned); err != nil {
		return task.SubmissionEligibilityRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check task implementor ban failed")}
	}
	if banned {
		return task.SubmissionEligibilityRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "implementor is banned from the task")}
	}
	if policy == task.ParticipationPolicyOpen.String() {
		return task.SubmissionEligible{}
	}

	var activeForSubmitter bool
	if err := store.pool.QueryRow(ctx, `
		select exists(
			select 1
			from task_reservations
			left join team_members
				on task_reservations.assignee_kind in ('organization_team', 'team')
				and task_reservations.team_id = team_members.team_id
				and team_members.user_id = $2
			where task_reservations.task_id = $1
			and task_reservations.state = 'active'
			and (
				(task_reservations.assignee_kind = 'user' and task_reservations.user_id = $2)
				or
				(task_reservations.assignee_kind = 'organization_team' and team_members.user_id = $2)
				or
				(task_reservations.assignee_kind = 'team' and team_members.user_id = $2)
			)
		)
	`, taskID.String(), submitterID.String()).Scan(&activeForSubmitter); err != nil {
		return task.SubmissionEligibilityRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check active reservation failed")}
	}
	if activeForSubmitter {
		return task.SubmissionEligible{}
	}
	return task.SubmissionEligibilityRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "task requires an active reservation for the submitter")}
}

type insertSeriesResult interface {
	insertSeriesResult()
}

type insertSeriesAccepted struct {
	seriesID *string
	position *int
}

type insertSeriesRejected struct {
	reason core.DomainError
}

func (insertSeriesAccepted) insertSeriesResult() {}

func (insertSeriesRejected) insertSeriesResult() {}

func (store TaskStore) insertSeriesForPlacement(ctx context.Context, tx pgx.Tx, seriesID core.TaskSeriesID, command task.CreateCommand) insertSeriesResult {
	switch placement := command.Placement.(type) {
	case task.StandalonePlacement:
		return insertSeriesAccepted{}
	case task.NewSeriesPlacement:
		ownerColumns := ownerSQLColumns(command.Owner)
		_, err := tx.Exec(ctx, `
			insert into task_series (id, owner_kind, user_id, team_id, organization_id, title, created_by_user_id)
			values ($1, $2, $3, $4, $5, $6, $7)
		`, seriesID.String(), ownerColumns.kind, ownerColumns.userID, ownerColumns.teamID, ownerColumns.organizationID, placement.Title.String(), command.Actor.ID.String())
		if err != nil {
			return insertSeriesRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert task series failed")}
		}
		return insertSeriesAccepted{seriesID: stringPointer(seriesID.String()), position: intPointer(placement.Position.Int())}
	case task.ExistingSeriesPlacement:
		return insertSeriesAccepted{seriesID: stringPointer(placement.SeriesID.String()), position: intPointer(placement.Position.Int())}
	default:
		return insertSeriesRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task series placement is invalid")}
	}
}

func insertTaskAttachments(ctx context.Context, tx pgx.Tx, taskID core.TaskID, attachments []attachment.Attachment) insertAttachmentsResult {
	for index := range attachments {
		value := attachments[index]
		_, err := tx.Exec(ctx, `
			insert into task_attachments (task_id, attachment_index, filename, content_type, content)
			values ($1, $2, $3, $4, $5)
		`, taskID.String(), index, value.Name.String(), value.ContentType.String(), value.Content.Bytes())
		if err != nil {
			return insertAttachmentsRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert task attachment failed")}
		}
	}
	return insertAttachmentsAccepted{}
}

type ownerSQL struct {
	kind           string
	userID         *string
	teamID         *string
	organizationID *string
}

func ownerSQLColumns(owner task.Owner) ownerSQL {
	switch typed := owner.(type) {
	case task.UserOwner:
		return ownerSQL{kind: task.OwnerKindUser.String(), userID: stringPointer(typed.UserID.String())}
	case task.TeamOwner:
		return ownerSQL{kind: task.OwnerKindTeam.String(), teamID: stringPointer(typed.TeamID.String())}
	case task.OrganizationOwner:
		return ownerSQL{kind: task.OwnerKindOrganization.String(), organizationID: stringPointer(typed.OrganizationID.String())}
	case task.OrganizationTeamOwner:
		return ownerSQL{kind: task.OwnerKindOrganizationTeam.String(), teamID: stringPointer(typed.TeamID.String()), organizationID: stringPointer(typed.OrganizationID.String())}
	default:
		return ownerSQL{}
	}
}

type visibilitySQL struct {
	kind           string
	scopeKey       string
	userID         *string
	teamID         *string
	organizationID *string
}

func visibilitySQLColumns(visibility task.Visibility) visibilitySQL {
	switch typed := visibility.(type) {
	case task.PublicVisibility:
		return visibilitySQL{kind: task.VisibilityKindPublic.String(), scopeKey: "public"}
	case task.UserVisibility:
		return visibilitySQL{kind: task.VisibilityKindUser.String(), scopeKey: typed.UserID.String(), userID: stringPointer(typed.UserID.String())}
	case task.TeamVisibility:
		return visibilitySQL{kind: task.VisibilityKindTeam.String(), scopeKey: typed.TeamID.String(), teamID: stringPointer(typed.TeamID.String())}
	case task.OrganizationVisibility:
		return visibilitySQL{kind: task.VisibilityKindOrganization.String(), scopeKey: typed.OrganizationID.String(), organizationID: stringPointer(typed.OrganizationID.String())}
	case task.OrganizationTeamVisibility:
		return visibilitySQL{kind: task.VisibilityKindOrganizationTeam.String(), scopeKey: typed.OrganizationID.String() + ":" + typed.TeamID.String(), teamID: stringPointer(typed.TeamID.String()), organizationID: stringPointer(typed.OrganizationID.String())}
	default:
		return visibilitySQL{}
	}
}

type payloadSQL struct {
	kind   string
	source *string
}

type assigneeSQL struct {
	kind           string
	userID         *string
	teamID         *string
	organizationID *string
}

func assigneeSQLColumns(assignee task.Assignee) assigneeSQL {
	switch typed := assignee.(type) {
	case task.UserAssignee:
		return assigneeSQL{kind: task.AssigneeScopeUser.String(), userID: stringPointer(typed.UserID.String())}
	case task.OrganizationTeamAssignee:
		return assigneeSQL{kind: task.AssigneeScopeOrganizationTeam.String(), teamID: stringPointer(typed.TeamID.String()), organizationID: stringPointer(typed.OrganizationID.String())}
	case task.TeamAssignee:
		return assigneeSQL{kind: task.AssigneeScopeTeam.String(), teamID: stringPointer(typed.TeamID.String())}
	default:
		return assigneeSQL{}
	}
}

type rewardSQL struct {
	kind         string
	creditAmount *int64
}

func rewardSQLColumns(reward task.RewardSpec) rewardSQL {
	switch typed := reward.(type) {
	case task.NoRewardSpec:
		return rewardSQL{kind: task.RewardKindNone.String()}
	case task.CreditRewardSpec:
		amount := typed.Amount.Int64()
		return rewardSQL{kind: task.RewardKindCredit.String(), creditAmount: &amount}
	case task.CollectibleRewardSpec:
		return rewardSQL{kind: task.RewardKindCollectible.String()}
	case task.BundleRewardSpec:
		amount := typed.Credit.Int64()
		return rewardSQL{kind: task.RewardKindBundle.String(), creditAmount: &amount}
	default:
		return rewardSQL{}
	}
}

func payloadSQLColumns(payload task.DataPayload) payloadSQL {
	switch typed := payload.(type) {
	case task.NoDataPayload:
		return payloadSQL{kind: "none"}
	case task.JSONDataPayload:
		return payloadSQL{kind: "json", source: stringPointer(typed.Source.String())}
	default:
		return payloadSQL{}
	}
}

func stringPointer(value string) *string {
	return &value
}

func intPointer(value int) *int {
	return &value
}

const expireReservationsSQL = `
	update task_reservations
	set state = 'expired', state_recorded_at = now()
	where state in ('requested', 'active') and expires_at <= now()
`

func (store TaskStore) releaseExpiredReservations(ctx context.Context) error {
	_, err := store.pool.Exec(ctx, expireReservationsSQL)
	return err
}

type reservationInitialStateResult interface {
	reservationInitialStateResult()
}

type reservationInitialStateAccepted struct {
	value task.ReservationState
}

type reservationInitialStateRejected struct {
	reason core.DomainError
}

func (reservationInitialStateAccepted) reservationInitialStateResult() {}

func (reservationInitialStateRejected) reservationInitialStateResult() {}

func reservationInitialState(policy string) reservationInitialStateResult {
	switch policy {
	case task.ParticipationPolicyReservationRequired.String():
		return reservationInitialStateAccepted{value: task.ReservationStateActive}
	case task.ParticipationPolicyApprovalRequired.String():
		return reservationInitialStateAccepted{value: task.ReservationStateRequested}
	case task.ParticipationPolicyOpen.String():
		return reservationInitialStateRejected{reason: core.NewDomainError(core.ErrorCodeConflict, "task does not require reservation")}
	default:
		return reservationInitialStateRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task participation policy is invalid")}
	}
}

// taskBaseColumns is the shared task column projection used by both the
// single-task select and the list select, so the column list lives in one place.
const taskBaseColumns = `
		tasks.id::text, tasks.owner_kind, coalesce(tasks.user_id::text, ''), coalesce(tasks.team_id::text, ''),
		coalesce(tasks.organization_id::text, ''), tasks.title, tasks.description, tasks.task_type, tasks.reference_url, tasks.state,
		tasks.reward_kind, coalesce(tasks.reward_credit_amount, 0),
		coalesce((
			select count(*)
			from task_fund_collectibles
			where task_fund_collectibles.task_id = tasks.id
		), 0),
		tasks.participation_policy, tasks.assignee_scope, tasks.reservation_expires_after_hours,
		task_visibility_scopes.visibility_kind, coalesce(task_visibility_scopes.user_id::text, ''),
		coalesce(task_visibility_scopes.team_id::text, ''), coalesce(task_visibility_scopes.organization_id::text, ''),
		coalesce(tasks.series_id::text, ''), coalesce(tasks.series_position, 0), tasks.response_schema_json::text,
		tasks.data_payload_kind, coalesce(tasks.data_payload_json::text, ''), tasks.created_by_user_id::text,
		coalesce((
			select jsonb_agg(
				jsonb_build_object(
					'name', task_attachments.filename,
					'content_type', task_attachments.content_type,
					'content', encode(task_attachments.content, 'base64')
				)
				order by task_attachments.attachment_index
			)
			from task_attachments
			where task_attachments.task_id = tasks.id
		), '[]'::jsonb)::text`

func taskSelectSQL() string {
	return "select " + taskBaseColumns + `
		from tasks
		join task_visibility_scopes on task_visibility_scopes.task_id = tasks.id
	`
}

// taskListSelectSQL extends the base task select with the active reservation
// assignee, exposed on each task list item. The LEFT JOIN keeps tasks without an
// active reservation in the result with empty active-assignee columns.
func taskListSelectSQL() string {
	return "select " + taskBaseColumns + `,
			coalesce(active_reservation.assignee_kind, ''),
			coalesce(active_reservation.user_id::text, ''),
			coalesce(active_reservation.organization_id::text, ''),
			coalesce(active_reservation.team_id::text, '')
		from tasks
		join task_visibility_scopes on task_visibility_scopes.task_id = tasks.id
		left join task_reservations as active_reservation
			on active_reservation.task_id = tasks.id
			and active_reservation.state = 'active'
	`
}

type listQueryResult interface {
	listQueryResult()
}

type listQueryAccepted struct {
	sql       string
	arguments pgx.NamedArgs
}

type listQueryRejected struct {
	reason core.DomainError
}

func (listQueryAccepted) listQueryResult() {}

func (listQueryRejected) listQueryResult() {}

func listQueryForScope(scope task.ListScope, filters task.ListFilters, page core.Page) listQueryResult {
	arguments := pgx.NamedArgs{
		"limit":  page.Limit(),
		"offset": page.Offset(),
	}
	var where string
	switch typed := scope.(type) {
	case task.PublicListScope:
		arguments["visibility_kind"] = task.VisibilityKindPublic.String()
		arguments["include_reserved"] = typed.IncludeReserved
		arguments["viewer_id"] = typed.ViewerID.String()
		where = `
			where task_visibility_scopes.visibility_kind = @visibility_kind
			and (
				@include_reserved::boolean
				or not exists (
					select 1 from task_reservations
					where task_reservations.task_id = tasks.id
					and task_reservations.state = 'active'
				)
				or exists (
					select 1 from task_reservations
					where task_reservations.task_id = tasks.id
					and task_reservations.state = 'active'
					and task_reservations.assignee_kind = 'user'
					and task_reservations.user_id = @viewer_id
				)
				or tasks.created_by_user_id = @viewer_id
			)`
	case task.UserListScope:
		arguments["user_id"] = typed.UserID.String()
		where = " where (task_visibility_scopes.user_id = @user_id or tasks.created_by_user_id = @user_id)"
	case task.OrganizationListScope:
		arguments["organization_id"] = typed.OrganizationID.String()
		where = " where task_visibility_scopes.organization_id = @organization_id"
	case task.TeamListScope:
		arguments["team_id"] = typed.TeamID.String()
		arguments["include_reserved"] = typed.IncludeReserved
		where = `
			where (
				task_visibility_scopes.team_id = @team_id
				or active_reservation.team_id = @team_id
			)
			and (
				@include_reserved::boolean
				or active_reservation.assignee_kind is null
				or active_reservation.team_id = @team_id
			)`
	case task.CreatorListScope:
		arguments["creator_id"] = typed.CreatorID.String()
		arguments["visibility_kind"] = task.VisibilityKindPublic.String()
		where = " where tasks.created_by_user_id = @creator_id and task_visibility_scopes.visibility_kind = @visibility_kind"
	case task.AssigneeListScope:
		arguments["assignee_id"] = typed.AssigneeID.String()
		arguments["visibility_kind"] = task.VisibilityKindPublic.String()
		where = " where active_reservation.assignee_kind = 'user' and active_reservation.user_id = @assignee_id and task_visibility_scopes.visibility_kind = @visibility_kind"
	default:
		return listQueryRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task list scope is invalid")}
	}

	switch stateFilter := filters.State.(type) {
	case task.StateEquals:
		arguments["filter_state"] = stateFilter.Value.String()
		where += " and tasks.state = @filter_state"
	case task.StateIn:
		rawStates := make([]string, len(stateFilter.Values))
		for index := range stateFilter.Values {
			rawStates[index] = stateFilter.Values[index].String()
		}
		arguments["filter_states"] = rawStates
		where += " and tasks.state = some(@filter_states)"
	case task.AnyStateFilter:
	default:
		return listQueryRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task state filter is invalid")}
	}

	switch participationFilter := filters.Participation.(type) {
	case task.ParticipationPolicyEquals:
		arguments["filter_participation_policy"] = participationFilter.Value.String()
		where += " and tasks.participation_policy = @filter_participation_policy"
	case task.AnyParticipationPolicyFilter:
	default:
		return listQueryRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task participation policy filter is invalid")}
	}

	switch searchFilter := filters.Search.(type) {
	case task.SearchContains:
		arguments["filter_query"] = "%" + searchFilter.Value.String() + "%"
		where += " and (tasks.title ilike @filter_query or tasks.id::text ilike @filter_query)"
	case task.NoSearchFilter:
	default:
		return listQueryRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task search filter is invalid")}
	}

	switch typeFilter := filters.Type.(type) {
	case task.TypeEquals:
		arguments["filter_task_type"] = typeFilter.Value.String()
		where += " and tasks.task_type = @filter_task_type"
	case task.AnyTypeFilter:
	default:
		return listQueryRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task type filter is invalid")}
	}

	orderBy := " order by tasks.created_at desc, tasks.id desc"
	switch filters.Sort {
	case task.SortNewest:
		orderBy = " order by tasks.created_at desc, tasks.id desc"
	case task.SortOldest:
		orderBy = " order by tasks.created_at asc, tasks.id asc"
	case task.SortTitleAsc:
		orderBy = " order by lower(tasks.title) asc, tasks.created_at desc, tasks.id desc"
	case task.SortTitleDesc:
		orderBy = " order by lower(tasks.title) desc, tasks.created_at desc, tasks.id desc"
	case task.SortRewardDesc:
		orderBy = " order by coalesce(tasks.reward_credit_amount, 0) desc, tasks.created_at desc, tasks.id desc"
	case task.SortRewardAsc:
		orderBy = " order by coalesce(tasks.reward_credit_amount, 0) asc, tasks.created_at desc, tasks.id desc"
	default:
		return listQueryRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task sort is invalid")}
	}

	return listQueryAccepted{
		sql:       taskListSelectSQL() + where + orderBy + " limit @limit offset @offset",
		arguments: arguments,
	}
}

type taskListItemRowsResult interface {
	taskListItemRowsResult()
}

type taskListItemRowsAccepted struct {
	values []task.ListItem
}

type taskListItemRowsRejected struct {
	reason core.DomainError
}

func (taskListItemRowsAccepted) taskListItemRowsResult() {}

func (taskListItemRowsRejected) taskListItemRowsResult() {}

func scanTaskListItemRows(rows pgx.Rows) taskListItemRowsResult {
	values := make([]task.ListItem, 0)
	for rows.Next() {
		parsed := scanTaskListItemRow(rows)
		accepted, matched := parsed.(taskListItemRowAccepted)
		if !matched {
			rejected := parsed.(taskListItemRowRejected)
			return taskListItemRowsRejected{reason: rejected.reason}
		}
		values = append(values, accepted.value)
	}
	if err := rows.Err(); err != nil {
		return taskListItemRowsRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "read tasks failed")}
	}
	return taskListItemRowsAccepted{values: values}
}

type taskListItemRowResult interface {
	taskListItemRowResult()
}

type taskListItemRowAccepted struct {
	value task.ListItem
}

type taskListItemRowRejected struct {
	reason core.DomainError
}

func (taskListItemRowAccepted) taskListItemRowResult() {}

func (taskListItemRowRejected) taskListItemRowResult() {}

func scanTaskListItemRow(rows pgx.Rows) taskListItemRowResult {
	var row taskBaseRow
	var rawActiveAssigneeKind string
	var rawActiveAssigneeUserID string
	var rawActiveAssigneeOrganizationID string
	var rawActiveAssigneeTeamID string
	if err := rows.Scan(&row.taskID, &row.ownerKind, &row.ownerUserID, &row.ownerTeamID, &row.ownerOrganizationID, &row.title, &row.description, &row.taskType, &row.referenceURL, &row.state, &row.rewardKind, &row.rewardCreditAmount, &row.rewardCollectibleCount, &row.participationPolicy, &row.assigneeScope, &row.reservationTTLHours, &row.visibilityKind, &row.visibilityUserID, &row.visibilityTeamID, &row.visibilityOrganizationID, &row.seriesID, &row.seriesPosition, &row.responseSchema, &row.payloadKind, &row.payload, &row.createdBy, &row.attachments, &rawActiveAssigneeKind, &rawActiveAssigneeUserID, &rawActiveAssigneeOrganizationID, &rawActiveAssigneeTeamID); err != nil {
		return taskListItemRowRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan task failed")}
	}
	taskResult := row.parse()
	taskAccepted, taskMatched := taskResult.(taskRowAccepted)
	if !taskMatched {
		return taskListItemRowRejected{reason: taskResult.(taskRowRejected).reason}
	}
	activeResult := parseActiveAssignee(rawActiveAssigneeKind, rawActiveAssigneeUserID, rawActiveAssigneeOrganizationID, rawActiveAssigneeTeamID)
	activeAccepted, activeMatched := activeResult.(activeAssigneeAccepted)
	if !activeMatched {
		return taskListItemRowRejected{reason: activeResult.(activeAssigneeRejected).reason}
	}
	return taskListItemRowAccepted{value: task.ListItem{Task: taskAccepted.value, ActiveAssignee: activeAccepted.value}}
}

type activeAssigneeResult interface {
	activeAssigneeResult()
}

type activeAssigneeAccepted struct {
	value task.ActiveAssignee
}

type activeAssigneeRejected struct {
	reason core.DomainError
}

func (activeAssigneeAccepted) activeAssigneeResult() {}

func (activeAssigneeRejected) activeAssigneeResult() {}

func parseActiveAssignee(kind string, rawUserID string, rawOrganizationID string, rawTeamID string) activeAssigneeResult {
	switch kind {
	case "":
		return activeAssigneeAccepted{value: task.NoActiveAssignee{}}
	case task.AssigneeScopeUser.String():
		userIDResult := core.ParseUserID(rawUserID)
		userID, matched := userIDResult.(core.UserIDCreated)
		if !matched {
			return activeAssigneeRejected{reason: userIDResult.(core.UserIDRejected).Reason}
		}
		return activeAssigneeAccepted{value: task.ActiveUserAssignee{UserID: userID.Value}}
	case task.AssigneeScopeOrganizationTeam.String():
		organizationIDResult := core.ParseOrganizationID(rawOrganizationID)
		organizationID, organizationMatched := organizationIDResult.(core.OrganizationIDCreated)
		if !organizationMatched {
			return activeAssigneeRejected{reason: organizationIDResult.(core.OrganizationIDRejected).Reason}
		}
		teamIDResult := core.ParseTeamID(rawTeamID)
		teamID, teamMatched := teamIDResult.(core.TeamIDCreated)
		if !teamMatched {
			return activeAssigneeRejected{reason: teamIDResult.(core.TeamIDRejected).Reason}
		}
		return activeAssigneeAccepted{value: task.ActiveOrganizationTeamAssignee{OrganizationID: organizationID.Value, TeamID: teamID.Value}}
	case task.AssigneeScopeTeam.String():
		teamIDResult := core.ParseTeamID(rawTeamID)
		teamID, teamMatched := teamIDResult.(core.TeamIDCreated)
		if !teamMatched {
			return activeAssigneeRejected{reason: teamIDResult.(core.TeamIDRejected).Reason}
		}
		return activeAssigneeAccepted{value: task.ActiveTeamAssignee{TeamID: teamID.Value}}
	default:
		return activeAssigneeRejected{reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "active assignee kind is invalid")}
	}
}

type taskRowsResult interface {
	taskRowsResult()
}

type taskRowsAccepted struct {
	values []task.Task
}

type taskRowsRejected struct {
	reason core.DomainError
}

func (taskRowsAccepted) taskRowsResult() {}

func (taskRowsRejected) taskRowsResult() {}

func scanTaskRows(rows pgx.Rows) taskRowsResult {
	values := make([]task.Task, 0)
	for rows.Next() {
		parsed := scanTaskRow(rows)
		accepted, matched := parsed.(taskRowAccepted)
		if !matched {
			rejected := parsed.(taskRowRejected)
			return taskRowsRejected{reason: rejected.reason}
		}
		values = append(values, accepted.value)
	}
	if err := rows.Err(); err != nil {
		return taskRowsRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "read tasks failed")}
	}
	return taskRowsAccepted{values: values}
}

type taskRowResult interface {
	taskRowResult()
}

type taskRowAccepted struct {
	value task.Task
}

type taskRowRejected struct {
	reason core.DomainError
}

func (taskRowAccepted) taskRowResult() {}

func (taskRowRejected) taskRowResult() {}

// taskBaseRow holds the raw base task columns shared by the single-task select
// and the list select, so the column scan targets and the parse call live in one
// place rather than being duplicated across both scanners.
type taskBaseRow struct {
	taskID                   string
	ownerKind                string
	ownerUserID              string
	ownerTeamID              string
	ownerOrganizationID      string
	title                    string
	description              string
	taskType                 string
	referenceURL             string
	state                    string
	rewardKind               string
	rewardCreditAmount       int64
	rewardCollectibleCount   int
	participationPolicy      string
	assigneeScope            string
	reservationTTLHours      int
	visibilityKind           string
	visibilityUserID         string
	visibilityTeamID         string
	visibilityOrganizationID string
	seriesID                 string
	seriesPosition           int
	responseSchema           string
	payloadKind              string
	payload                  string
	createdBy                string
	attachments              string
}

func (row taskBaseRow) parse() taskRowResult {
	return parseTaskRow(row.taskID, row.ownerKind, row.ownerUserID, row.ownerTeamID, row.ownerOrganizationID, row.title, row.description, row.taskType, row.referenceURL, row.state, row.rewardKind, row.rewardCreditAmount, row.rewardCollectibleCount, row.participationPolicy, row.assigneeScope, row.reservationTTLHours, row.visibilityKind, row.visibilityUserID, row.visibilityTeamID, row.visibilityOrganizationID, row.seriesID, row.seriesPosition, row.responseSchema, row.payloadKind, row.payload, row.createdBy, row.attachments)
}

func scanTaskRow(rows pgx.Rows) taskRowResult {
	var row taskBaseRow
	if err := rows.Scan(&row.taskID, &row.ownerKind, &row.ownerUserID, &row.ownerTeamID, &row.ownerOrganizationID, &row.title, &row.description, &row.taskType, &row.referenceURL, &row.state, &row.rewardKind, &row.rewardCreditAmount, &row.rewardCollectibleCount, &row.participationPolicy, &row.assigneeScope, &row.reservationTTLHours, &row.visibilityKind, &row.visibilityUserID, &row.visibilityTeamID, &row.visibilityOrganizationID, &row.seriesID, &row.seriesPosition, &row.responseSchema, &row.payloadKind, &row.payload, &row.createdBy, &row.attachments); err != nil {
		return taskRowRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan task failed")}
	}
	return row.parse()
}

func parseTaskRow(rawTaskID string, rawOwnerKind string, rawOwnerUserID string, rawOwnerTeamID string, rawOwnerOrganizationID string, rawTitle string, rawDescription string, rawTaskType string, rawReferenceURL string, rawState string, rawRewardKind string, rawRewardCreditAmount int64, rawRewardCollectibleCount int, rawParticipationPolicy string, rawAssigneeScope string, rawReservationTTLHours int, rawVisibilityKind string, rawVisibilityUserID string, rawVisibilityTeamID string, rawVisibilityOrganizationID string, rawSeriesID string, rawSeriesPosition int, rawResponseSchema string, rawPayloadKind string, rawPayload string, rawCreatedBy string, rawAttachments string) taskRowResult {
	taskIDResult := core.ParseTaskID(rawTaskID)
	taskID, taskIDMatched := taskIDResult.(core.TaskIDCreated)
	if !taskIDMatched {
		rejected := taskIDResult.(core.TaskIDRejected)
		return taskRowRejected{reason: rejected.Reason}
	}
	ownerResult := parseTaskOwner(rawOwnerKind, rawOwnerUserID, rawOwnerTeamID, rawOwnerOrganizationID)
	owner, ownerMatched := ownerResult.(taskOwnerAccepted)
	if !ownerMatched {
		rejected := ownerResult.(taskOwnerRejected)
		return taskRowRejected{reason: rejected.reason}
	}
	titleResult := task.NewTitle(rawTitle)
	title, titleMatched := titleResult.(task.TitleAccepted)
	if !titleMatched {
		rejected := titleResult.(task.TitleRejected)
		return taskRowRejected{reason: rejected.Reason}
	}
	descriptionResult := task.NewDescription(rawDescription)
	description, descriptionMatched := descriptionResult.(task.DescriptionAccepted)
	if !descriptionMatched {
		rejected := descriptionResult.(task.DescriptionRejected)
		return taskRowRejected{reason: rejected.Reason}
	}
	taskTypeResult := task.ParseTaskType(rawTaskType)
	taskType, taskTypeMatched := taskTypeResult.(task.TaskTypeAccepted)
	if !taskTypeMatched {
		return taskRowRejected{reason: taskTypeResult.(task.TaskTypeRejected).Reason}
	}
	referenceResult := task.NewReferenceURL(rawReferenceURL)
	reference, referenceMatched := referenceResult.(task.ReferenceURLAccepted)
	if !referenceMatched {
		return taskRowRejected{reason: referenceResult.(task.ReferenceURLRejected).Reason}
	}
	stateResult := task.ParseState(rawState)
	state, stateMatched := stateResult.(task.StateAccepted)
	if !stateMatched {
		rejected := stateResult.(task.StateRejected)
		return taskRowRejected{reason: rejected.Reason}
	}
	rewardResult := parseRewardSpec(rawRewardKind, rawRewardCreditAmount, rawRewardCollectibleCount)
	reward, rewardMatched := rewardResult.(rewardSpecAccepted)
	if !rewardMatched {
		rejected := rewardResult.(rewardSpecRejected)
		return taskRowRejected{reason: rejected.reason}
	}
	participationResult := task.ParseParticipationPolicy(rawParticipationPolicy)
	participation, participationMatched := participationResult.(task.ParticipationPolicyAccepted)
	if !participationMatched {
		rejected := participationResult.(task.ParticipationPolicyRejected)
		return taskRowRejected{reason: rejected.Reason}
	}
	assigneeScopeResult := task.ParseAssigneeScope(rawAssigneeScope)
	assigneeScope, assigneeScopeMatched := assigneeScopeResult.(task.AssigneeScopeAccepted)
	if !assigneeScopeMatched {
		rejected := assigneeScopeResult.(task.AssigneeScopeRejected)
		return taskRowRejected{reason: rejected.Reason}
	}
	ttlResult := task.NewReservationTTL(rawReservationTTLHours)
	ttl, ttlMatched := ttlResult.(task.ReservationTTLAccepted)
	if !ttlMatched {
		rejected := ttlResult.(task.ReservationTTLRejected)
		return taskRowRejected{reason: rejected.Reason}
	}
	visibilityResult := parseTaskVisibility(rawVisibilityKind, rawVisibilityUserID, rawVisibilityTeamID, rawVisibilityOrganizationID)
	visibility, visibilityMatched := visibilityResult.(taskVisibilityAccepted)
	if !visibilityMatched {
		rejected := visibilityResult.(taskVisibilityRejected)
		return taskRowRejected{reason: rejected.reason}
	}
	placementResult := parseSeriesPlacement(rawSeriesID, rawSeriesPosition)
	placement, placementMatched := placementResult.(seriesPlacementAccepted)
	if !placementMatched {
		rejected := placementResult.(seriesPlacementRejected)
		return taskRowRejected{reason: rejected.reason}
	}
	schemaResult := task.NewResponseSchemaSource(rawResponseSchema)
	schemaSource, schemaMatched := schemaResult.(task.ResponseSchemaSourceAccepted)
	if !schemaMatched {
		rejected := schemaResult.(task.ResponseSchemaSourceRejected)
		return taskRowRejected{reason: rejected.Reason}
	}
	payloadResult := parseDataPayload(rawPayloadKind, rawPayload)
	payload, payloadMatched := payloadResult.(dataPayloadAccepted)
	if !payloadMatched {
		rejected := payloadResult.(dataPayloadRejected)
		return taskRowRejected{reason: rejected.reason}
	}
	createdByResult := core.ParseUserID(rawCreatedBy)
	createdBy, createdByMatched := createdByResult.(core.UserIDCreated)
	if !createdByMatched {
		rejected := createdByResult.(core.UserIDRejected)
		return taskRowRejected{reason: rejected.Reason}
	}
	attachmentsResult := parseStoredAttachments(rawAttachments)
	attachments, attachmentsMatched := attachmentsResult.(attachmentsAccepted)
	if !attachmentsMatched {
		return taskRowRejected{reason: attachmentsResult.(attachmentsRejected).reason}
	}
	return taskRowAccepted{value: task.Task{ID: taskID.Value, Owner: owner.value, Title: title.Value, Description: description.Value, Type: taskType.Value, Reference: reference.Value, Reward: reward.value, Participation: participation.Value, AssigneeScope: assigneeScope.Value, ReservationTTL: ttl.Value, State: state.Value, Visibility: visibility.value, Placement: placement.value, ResponseSchema: schemaSource.Value, Payload: payload.value, Attachments: attachments.values, CreatedBy: createdBy.Value}}
}

type rewardSpecResult interface {
	rewardSpecResult()
}

type rewardSpecAccepted struct {
	value task.RewardSpec
}

type rewardSpecRejected struct {
	reason core.DomainError
}

func (rewardSpecAccepted) rewardSpecResult() {}

func (rewardSpecRejected) rewardSpecResult() {}

func parseRewardSpec(rawKind string, rawCreditAmount int64, rawCollectibleCount int) rewardSpecResult {
	switch rawKind {
	case task.RewardKindNone.String():
		if rawCollectibleCount > 0 {
			return collectibleRewardSpec(rawCollectibleCount)
		}
		return rewardSpecAccepted{value: task.NoRewardSpec{}}
	case task.RewardKindCredit.String():
		amountResult := task.NewCreditRewardAmount(rawCreditAmount)
		amount, matched := amountResult.(task.CreditRewardAmountAccepted)
		if !matched {
			return rewardSpecRejected{reason: amountResult.(task.CreditRewardAmountRejected).Reason}
		}
		if rawCollectibleCount > 0 {
			countResult := collectibleRewardSpec(rawCollectibleCount)
			count, countMatched := countResult.(rewardSpecAccepted)
			if !countMatched {
				return countResult
			}
			return rewardSpecAccepted{value: task.BundleRewardSpec{Credit: amount.Value, Collectible: count.value.(task.CollectibleRewardSpec).Count}}
		}
		return rewardSpecAccepted{value: task.CreditRewardSpec{Amount: amount.Value}}
	case task.RewardKindCollectible.String():
		count := rawCollectibleCount
		if count == 0 {
			count = 1
		}
		return collectibleRewardSpec(count)
	case task.RewardKindBundle.String():
		amountResult := task.NewCreditRewardAmount(rawCreditAmount)
		amount, matched := amountResult.(task.CreditRewardAmountAccepted)
		if !matched {
			return rewardSpecRejected{reason: amountResult.(task.CreditRewardAmountRejected).Reason}
		}
		count := rawCollectibleCount
		if count == 0 {
			count = 1
		}
		countResult := collectibleRewardSpec(count)
		collectible, collectibleMatched := countResult.(rewardSpecAccepted)
		if !collectibleMatched {
			return countResult
		}
		return rewardSpecAccepted{value: task.BundleRewardSpec{Credit: amount.Value, Collectible: collectible.value.(task.CollectibleRewardSpec).Count}}
	default:
		return rewardSpecRejected{reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "task reward kind is invalid")}
	}
}

func collectibleRewardSpec(rawCount int) rewardSpecResult {
	countResult := task.NewCollectibleRewardCount(rawCount)
	count, matched := countResult.(task.CollectibleRewardCountAccepted)
	if !matched {
		return rewardSpecRejected{reason: countResult.(task.CollectibleRewardCountRejected).Reason}
	}
	return rewardSpecAccepted{value: task.CollectibleRewardSpec{Count: count.Value}}
}

func reservationSelectSQL() string {
	return `
		select id::text, task_id::text, assignee_kind, coalesce(user_id::text, ''),
			coalesce(team_id::text, ''), coalesce(organization_id::text, ''), state, requested_by_user_id::text
		from task_reservations
	`
}

type reservationLookupResult interface {
	reservationLookupResult()
}

type reservationFound struct {
	value task.Reservation
}

type reservationMissing struct {
	reason core.DomainError
}

func (reservationFound) reservationLookupResult() {}

func (reservationMissing) reservationLookupResult() {}

func (store TaskStore) findReservation(ctx context.Context, reservationID core.TaskReservationID) reservationLookupResult {
	rows, err := store.pool.Query(ctx, reservationSelectSQL()+" where id = $1", reservationID.String())
	if err != nil {
		return reservationMissing{reason: core.NewDomainError(core.ErrorCodeInvalidState, "find reservation failed")}
	}
	defer rows.Close()

	valuesResult := scanReservationRows(rows)
	values, matched := valuesResult.(reservationRowsAccepted)
	if !matched {
		return reservationMissing{reason: valuesResult.(reservationRowsRejected).reason}
	}
	if len(values.values) != 1 {
		return reservationMissing{reason: core.NewDomainError(core.ErrorCodeNotFound, "reservation was not found")}
	}
	return reservationFound{value: values.values[0]}
}

type reservationRowsResult interface {
	reservationRowsResult()
}

type reservationRowsAccepted struct {
	values []task.Reservation
}

type reservationRowsRejected struct {
	reason core.DomainError
}

func (reservationRowsAccepted) reservationRowsResult() {}

func (reservationRowsRejected) reservationRowsResult() {}

func scanReservationRows(rows pgx.Rows) reservationRowsResult {
	values := make([]task.Reservation, 0)
	for rows.Next() {
		parsed := scanReservationRow(rows)
		accepted, matched := parsed.(reservationRowAccepted)
		if !matched {
			return reservationRowsRejected{reason: parsed.(reservationRowRejected).reason}
		}
		values = append(values, accepted.value)
	}
	if err := rows.Err(); err != nil {
		return reservationRowsRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "read reservations failed")}
	}
	return reservationRowsAccepted{values: values}
}

type reservationRowResult interface {
	reservationRowResult()
}

type reservationRowAccepted struct {
	value task.Reservation
}

type reservationRowRejected struct {
	reason core.DomainError
}

func (reservationRowAccepted) reservationRowResult() {}

func (reservationRowRejected) reservationRowResult() {}

func scanReservationRow(rows pgx.Rows) reservationRowResult {
	var rawID string
	var rawTaskID string
	var rawAssigneeKind string
	var rawUserID string
	var rawTeamID string
	var rawOrganizationID string
	var rawState string
	var rawRequestedBy string
	if err := rows.Scan(&rawID, &rawTaskID, &rawAssigneeKind, &rawUserID, &rawTeamID, &rawOrganizationID, &rawState, &rawRequestedBy); err != nil {
		return reservationRowRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan reservation failed")}
	}
	return parseReservationRow(rawID, rawTaskID, rawAssigneeKind, rawUserID, rawTeamID, rawOrganizationID, rawState, rawRequestedBy)
}

func parseReservationRow(rawID string, rawTaskID string, rawAssigneeKind string, rawUserID string, rawTeamID string, rawOrganizationID string, rawState string, rawRequestedBy string) reservationRowResult {
	idResult := core.ParseTaskReservationID(rawID)
	idAccepted, idMatched := idResult.(core.TaskReservationIDCreated)
	if !idMatched {
		return reservationRowRejected{reason: idResult.(core.TaskReservationIDRejected).Reason}
	}
	taskIDResult := core.ParseTaskID(rawTaskID)
	taskIDAccepted, taskIDMatched := taskIDResult.(core.TaskIDCreated)
	if !taskIDMatched {
		return reservationRowRejected{reason: taskIDResult.(core.TaskIDRejected).Reason}
	}
	assigneeResult := parseReservationAssignee(rawAssigneeKind, rawUserID, rawTeamID, rawOrganizationID)
	assigneeAccepted, assigneeMatched := assigneeResult.(reservationAssigneeAccepted)
	if !assigneeMatched {
		return reservationRowRejected{reason: assigneeResult.(reservationAssigneeRejected).reason}
	}
	stateResult := task.ParseReservationState(rawState)
	stateAccepted, stateMatched := stateResult.(task.ReservationStateAccepted)
	if !stateMatched {
		return reservationRowRejected{reason: stateResult.(task.ReservationStateRejected).Reason}
	}
	requestedByResult := core.ParseUserID(rawRequestedBy)
	requestedByAccepted, requestedByMatched := requestedByResult.(core.UserIDCreated)
	if !requestedByMatched {
		return reservationRowRejected{reason: requestedByResult.(core.UserIDRejected).Reason}
	}
	return reservationRowAccepted{value: task.Reservation{ID: idAccepted.Value, TaskID: taskIDAccepted.Value, Assignee: assigneeAccepted.value, State: stateAccepted.Value, RequestedBy: requestedByAccepted.Value}}
}

type reservationAssigneeResult interface {
	reservationAssigneeResult()
}

type reservationAssigneeAccepted struct {
	value task.Assignee
}

type reservationAssigneeRejected struct {
	reason core.DomainError
}

func (reservationAssigneeAccepted) reservationAssigneeResult() {}

func (reservationAssigneeRejected) reservationAssigneeResult() {}

func parseReservationAssignee(rawKind string, rawUserID string, rawTeamID string, rawOrganizationID string) reservationAssigneeResult {
	switch rawKind {
	case task.AssigneeScopeUser.String():
		userIDResult := core.ParseUserID(rawUserID)
		userIDAccepted, matched := userIDResult.(core.UserIDCreated)
		if !matched {
			return reservationAssigneeRejected{reason: userIDResult.(core.UserIDRejected).Reason}
		}
		return reservationAssigneeAccepted{value: task.UserAssignee{UserID: userIDAccepted.Value}}
	case task.AssigneeScopeOrganizationTeam.String():
		organizationIDResult := core.ParseOrganizationID(rawOrganizationID)
		organizationIDAccepted, organizationMatched := organizationIDResult.(core.OrganizationIDCreated)
		if !organizationMatched {
			return reservationAssigneeRejected{reason: organizationIDResult.(core.OrganizationIDRejected).Reason}
		}
		teamIDResult := core.ParseTeamID(rawTeamID)
		teamIDAccepted, teamMatched := teamIDResult.(core.TeamIDCreated)
		if !teamMatched {
			return reservationAssigneeRejected{reason: teamIDResult.(core.TeamIDRejected).Reason}
		}
		return reservationAssigneeAccepted{value: task.OrganizationTeamAssignee{OrganizationID: organizationIDAccepted.Value, TeamID: teamIDAccepted.Value}}
	case task.AssigneeScopeTeam.String():
		teamIDResult := core.ParseTeamID(rawTeamID)
		teamIDAccepted, teamMatched := teamIDResult.(core.TeamIDCreated)
		if !teamMatched {
			return reservationAssigneeRejected{reason: teamIDResult.(core.TeamIDRejected).Reason}
		}
		return reservationAssigneeAccepted{value: task.TeamAssignee{TeamID: teamIDAccepted.Value}}
	default:
		return reservationAssigneeRejected{reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "reservation assignee kind is invalid")}
	}
}

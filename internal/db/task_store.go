package db

import (
	"context"

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
			reward_kind, reward_credit_amount, participation_policy, assignee_scope, reservation_expires_after_hours,
			state, response_schema_json, data_payload_kind, data_payload_json, created_by_user_id
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16::jsonb, $17, $18::jsonb, $19)
	`, taskID.String(), seriesColumns.seriesID, seriesColumns.position, ownerColumns.kind, ownerColumns.userID, ownerColumns.teamID, ownerColumns.organizationID,
		command.Title.String(), command.Description.String(), rewardColumns.kind, rewardColumns.creditAmount, command.Participation.String(), command.AssigneeScope.String(), command.ReservationTTL.Hours(), task.StateDraft.String(), command.ResponseSchema.String(), payloadColumns.kind, payloadColumns.source, command.Actor.ID.String())
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
		return task.FindTaskStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task was not found")}
	}
	return task.FindTaskStoreAccepted{Value: values.values[0]}
}

func (store TaskStore) ChangeTaskState(ctx context.Context, taskID core.TaskID, state task.State) task.ChangeTaskStateStoreResult {
	if state == task.StateOpen {
		escrowResult := store.requireOpenableReward(ctx, taskID)
		if rejected, matched := escrowResult.(openableRewardRejected); matched {
			return task.ChangeTaskStateStoreRejected{Reason: rejected.reason}
		}
	}
	commandTag, err := store.pool.Exec(ctx, `
		update tasks
		set state = $2, state_recorded_at = now()
		where id = $1
	`, taskID.String(), state.String())
	if err != nil {
		return task.ChangeTaskStateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "change task state failed")}
	}
	if commandTag.RowsAffected() != 1 {
		return task.ChangeTaskStateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task state was not changed")}
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

func (store TaskStore) requireOpenableReward(ctx context.Context, taskID core.TaskID) openableRewardResult {
	var rewardKind string
	var creditAmount int64
	err := store.pool.QueryRow(ctx, "select reward_kind, coalesce(reward_credit_amount, 0) from tasks where id = $1", taskID.String()).Scan(&rewardKind, &creditAmount)
	if err != nil {
		return openableRewardRejected{reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	if rewardKind == task.RewardKindCredit.String() || rewardKind == task.RewardKindBundle.String() {
		var heldAmount int64
		var escrowState string
		escrowErr := store.pool.QueryRow(ctx, "select amount, state from task_escrows where task_id = $1", taskID.String()).Scan(&heldAmount, &escrowState)
		if escrowErr != nil {
			return openableRewardRejected{reason: core.NewDomainError(core.ErrorCodeConflict, "credit reward must be funded before opening")}
		}
		if escrowState != "held" || heldAmount != creditAmount {
			return openableRewardRejected{reason: core.NewDomainError(core.ErrorCodeConflict, "held escrow must match the declared credit reward")}
		}
	}
	if rewardKind == task.RewardKindCollectible.String() || rewardKind == task.RewardKindBundle.String() {
		var heldCollectibles int
		err := store.pool.QueryRow(ctx, "select count(*) from task_collectible_rewards where task_id = $1 and state = 'held'", taskID.String()).Scan(&heldCollectibles)
		if err != nil {
			return openableRewardRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "check collectible reward funding failed")}
		}
		if heldCollectibles != 1 {
			return openableRewardRejected{reason: core.NewDomainError(core.ErrorCodeConflict, "collectible reward must be funded before opening")}
		}
	}
	return openableRewardAccepted{}
}

func (store TaskStore) ListTasks(ctx context.Context, scope task.ListScope) task.ListTasksStoreResult {
	if err := store.releaseExpiredReservations(ctx); err != nil {
		return task.ListTasksStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "release expired reservations failed")}
	}

	queryResult := listQueryForScope(scope)
	query, matched := queryResult.(listQueryAccepted)
	if !matched {
		rejected := queryResult.(listQueryRejected)
		return task.ListTasksStoreRejected{Reason: rejected.reason}
	}

	rowsResult := store.queryTaskRows(ctx, query)
	rows, matched := rowsResult.(taskRowsQueried)
	if !matched {
		return task.ListTasksStoreRejected{Reason: rowsResult.(taskRowsQueryRejected).reason}
	}
	defer rows.rows.Close()

	valuesResult := scanTaskRows(rows.rows)
	values, matched := valuesResult.(taskRowsAccepted)
	if !matched {
		rejected := valuesResult.(taskRowsRejected)
		return task.ListTasksStoreRejected{Reason: rejected.reason}
	}
	return task.ListTasksStoreAccepted{Values: values.values}
}

type taskRowsQueryResult interface {
	taskRowsQueryResult()
}

type taskRowsQueried struct {
	rows pgx.Rows
}

type taskRowsQueryRejected struct {
	reason core.DomainError
}

func (taskRowsQueried) taskRowsQueryResult() {}

func (taskRowsQueryRejected) taskRowsQueryResult() {}

func (store TaskStore) queryTaskRows(ctx context.Context, query listQueryAccepted) taskRowsQueryResult {
	switch arguments := query.arguments.(type) {
	case publicListQueryArguments:
		rows, err := store.pool.Query(ctx, query.sql, arguments.visibilityKind, arguments.includeReserved, arguments.viewerID)
		if err != nil {
			return taskRowsQueryRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "list tasks failed")}
		}
		return taskRowsQueried{rows: rows}
	case singleListQueryArgument:
		rows, err := store.pool.Query(ctx, query.sql, arguments.value)
		if err != nil {
			return taskRowsQueryRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "list tasks failed")}
		}
		return taskRowsQueried{rows: rows}
	default:
		return taskRowsQueryRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task list query arguments are invalid")}
	}
}

func (store TaskStore) CreateCapabilityToken(ctx context.Context, tokenID core.TaskCapabilityTokenID, taskID core.TaskID, hash task.CapabilityTokenHash) task.CreateCapabilityTokenStoreResult {
	_, err := store.pool.Exec(ctx, `
		insert into task_capability_tokens (id, task_id, token_hash, state, created_by_user_id)
		select $1, tasks.id, $3, $4, tasks.created_by_user_id
		from tasks
		where tasks.id = $2
	`, tokenID.String(), taskID.String(), hash.String(), task.CapabilityTokenStateActive.String())
	if err != nil {
		return task.CreateCapabilityTokenStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert task capability token failed")}
	}
	return task.CreateCapabilityTokenStoreAccepted{Value: task.CapabilityToken{ID: tokenID, TaskID: taskID, State: task.CapabilityTokenStateActive}}
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

func (store TaskStore) ChangeReservationState(ctx context.Context, reservationID core.TaskReservationID, state task.ReservationState) task.ChangeReservationStateStoreResult {
	if err := store.releaseExpiredReservations(ctx); err != nil {
		return task.ChangeReservationStateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "release expired reservations failed")}
	}

	commandTag, err := store.pool.Exec(ctx, `
		update task_reservations
		set state = $2, state_recorded_at = now()
		where id = $1 and state in ('requested', 'active')
	`, reservationID.String(), state.String())
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

	var activeForUser bool
	if err := store.pool.QueryRow(ctx, `
		select exists(
			select 1 from task_reservations
			where task_id = $1
			and state = 'active'
			and assignee_kind = 'user'
			and user_id = $2
		)
	`, taskID.String(), submitterID.String()).Scan(&activeForUser); err != nil {
		return task.SubmissionEligibilityRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check active reservation failed")}
	}
	if activeForUser {
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

func taskSelectSQL() string {
	return `
		select tasks.id::text, tasks.owner_kind, coalesce(tasks.user_id::text, ''), coalesce(tasks.team_id::text, ''),
			coalesce(tasks.organization_id::text, ''), tasks.title, tasks.description, tasks.state,
			tasks.reward_kind, coalesce(tasks.reward_credit_amount, 0),
			coalesce((
				select count(*)
				from task_collectible_rewards
				where task_collectible_rewards.task_id = tasks.id
				and task_collectible_rewards.state in ('held', 'released')
			), 0),
			tasks.participation_policy, tasks.assignee_scope, tasks.reservation_expires_after_hours,
			task_visibility_scopes.visibility_kind, coalesce(task_visibility_scopes.user_id::text, ''),
			coalesce(task_visibility_scopes.team_id::text, ''), coalesce(task_visibility_scopes.organization_id::text, ''),
			coalesce(tasks.series_id::text, ''), coalesce(tasks.series_position, 0), tasks.response_schema_json::text,
			tasks.data_payload_kind, coalesce(tasks.data_payload_json::text, ''), tasks.created_by_user_id::text
		from tasks
		join task_visibility_scopes on task_visibility_scopes.task_id = tasks.id
	`
}

type listQueryResult interface {
	listQueryResult()
}

type listQueryAccepted struct {
	sql       string
	arguments listQueryArguments
}

type listQueryRejected struct {
	reason core.DomainError
}

func (listQueryAccepted) listQueryResult() {}

func (listQueryRejected) listQueryResult() {}

type listQueryArguments interface {
	listQueryArguments()
}

type publicListQueryArguments struct {
	visibilityKind  string
	includeReserved bool
	viewerID        string
}

type singleListQueryArgument struct {
	value string
}

func (publicListQueryArguments) listQueryArguments() {}

func (singleListQueryArgument) listQueryArguments() {}

func listQueryForScope(scope task.ListScope) listQueryResult {
	switch typed := scope.(type) {
	case task.PublicListScope:
		return listQueryAccepted{sql: taskSelectSQL() + `
			where task_visibility_scopes.visibility_kind = $1
			and (
				$2::boolean
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
					and task_reservations.user_id = $3
				)
				or tasks.created_by_user_id = $3
			)
			order by tasks.created_at desc`, arguments: publicListQueryArguments{visibilityKind: task.VisibilityKindPublic.String(), includeReserved: typed.IncludeReserved, viewerID: typed.ViewerID.String()}}
	case task.UserListScope:
		return listQueryAccepted{sql: taskSelectSQL() + " where task_visibility_scopes.user_id = $1 or tasks.created_by_user_id = $1 order by tasks.created_at desc", arguments: singleListQueryArgument{value: typed.UserID.String()}}
	case task.OrganizationListScope:
		return listQueryAccepted{sql: taskSelectSQL() + " where task_visibility_scopes.organization_id = $1 order by tasks.created_at desc", arguments: singleListQueryArgument{value: typed.OrganizationID.String()}}
	default:
		return listQueryRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task list scope is invalid")}
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

func scanTaskRow(rows pgx.Rows) taskRowResult {
	var rawTaskID string
	var rawOwnerKind string
	var rawOwnerUserID string
	var rawOwnerTeamID string
	var rawOwnerOrganizationID string
	var rawTitle string
	var rawDescription string
	var rawState string
	var rawRewardKind string
	var rawRewardCreditAmount int64
	var rawRewardCollectibleCount int
	var rawParticipationPolicy string
	var rawAssigneeScope string
	var rawReservationTTLHours int
	var rawVisibilityKind string
	var rawVisibilityUserID string
	var rawVisibilityTeamID string
	var rawVisibilityOrganizationID string
	var rawSeriesID string
	var rawSeriesPosition int
	var rawResponseSchema string
	var rawPayloadKind string
	var rawPayload string
	var rawCreatedBy string
	if err := rows.Scan(&rawTaskID, &rawOwnerKind, &rawOwnerUserID, &rawOwnerTeamID, &rawOwnerOrganizationID, &rawTitle, &rawDescription, &rawState, &rawRewardKind, &rawRewardCreditAmount, &rawRewardCollectibleCount, &rawParticipationPolicy, &rawAssigneeScope, &rawReservationTTLHours, &rawVisibilityKind, &rawVisibilityUserID, &rawVisibilityTeamID, &rawVisibilityOrganizationID, &rawSeriesID, &rawSeriesPosition, &rawResponseSchema, &rawPayloadKind, &rawPayload, &rawCreatedBy); err != nil {
		return taskRowRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan task failed")}
	}
	return parseTaskRow(rawTaskID, rawOwnerKind, rawOwnerUserID, rawOwnerTeamID, rawOwnerOrganizationID, rawTitle, rawDescription, rawState, rawRewardKind, rawRewardCreditAmount, rawRewardCollectibleCount, rawParticipationPolicy, rawAssigneeScope, rawReservationTTLHours, rawVisibilityKind, rawVisibilityUserID, rawVisibilityTeamID, rawVisibilityOrganizationID, rawSeriesID, rawSeriesPosition, rawResponseSchema, rawPayloadKind, rawPayload, rawCreatedBy)
}

func parseTaskRow(rawTaskID string, rawOwnerKind string, rawOwnerUserID string, rawOwnerTeamID string, rawOwnerOrganizationID string, rawTitle string, rawDescription string, rawState string, rawRewardKind string, rawRewardCreditAmount int64, rawRewardCollectibleCount int, rawParticipationPolicy string, rawAssigneeScope string, rawReservationTTLHours int, rawVisibilityKind string, rawVisibilityUserID string, rawVisibilityTeamID string, rawVisibilityOrganizationID string, rawSeriesID string, rawSeriesPosition int, rawResponseSchema string, rawPayloadKind string, rawPayload string, rawCreatedBy string) taskRowResult {
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
	return taskRowAccepted{value: task.Task{ID: taskID.Value, Owner: owner.value, Title: title.Value, Description: description.Value, Reward: reward.value, Participation: participation.Value, AssigneeScope: assigneeScope.Value, ReservationTTL: ttl.Value, State: state.Value, Visibility: visibility.value, Placement: placement.value, ResponseSchema: schemaSource.Value, Payload: payload.value, CreatedBy: createdBy.Value}}
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
	default:
		return reservationAssigneeRejected{reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "reservation assignee kind is invalid")}
	}
}

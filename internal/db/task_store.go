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

	_, err = tx.Exec(ctx, `
		insert into tasks (
			id, series_id, series_position, owner_kind, user_id, team_id, organization_id, title, description,
			state, response_schema_json, data_payload_kind, data_payload_json, created_by_user_id
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12, $13::jsonb, $14)
	`, taskID.String(), seriesColumns.seriesID, seriesColumns.position, ownerColumns.kind, ownerColumns.userID, ownerColumns.teamID, ownerColumns.organizationID,
		command.Title.String(), command.Description.String(), task.StateDraft.String(), command.ResponseSchema.String(), payloadColumns.kind, payloadColumns.source, command.Actor.ID.String())
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

func (store TaskStore) ListTasks(ctx context.Context, scope task.ListScope) task.ListTasksStoreResult {
	queryResult := listQueryForScope(scope)
	query, matched := queryResult.(listQueryAccepted)
	if !matched {
		rejected := queryResult.(listQueryRejected)
		return task.ListTasksStoreRejected{Reason: rejected.reason}
	}

	rows, err := store.pool.Query(ctx, query.sql, query.argument)
	if err != nil {
		return task.ListTasksStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list tasks failed")}
	}
	defer rows.Close()

	valuesResult := scanTaskRows(rows)
	values, valuesMatched := valuesResult.(taskRowsAccepted)
	if !valuesMatched {
		rejected := valuesResult.(taskRowsRejected)
		return task.ListTasksStoreRejected{Reason: rejected.reason}
	}
	return task.ListTasksStoreAccepted{Values: values.values}
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

func taskSelectSQL() string {
	return `
		select tasks.id::text, tasks.owner_kind, coalesce(tasks.user_id::text, ''), coalesce(tasks.team_id::text, ''),
			coalesce(tasks.organization_id::text, ''), tasks.title, tasks.description, tasks.state,
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
	sql      string
	argument string
}

type listQueryRejected struct {
	reason core.DomainError
}

func (listQueryAccepted) listQueryResult() {}

func (listQueryRejected) listQueryResult() {}

func listQueryForScope(scope task.ListScope) listQueryResult {
	switch typed := scope.(type) {
	case task.PublicListScope:
		return listQueryAccepted{sql: taskSelectSQL() + " where task_visibility_scopes.visibility_kind = $1 order by tasks.created_at desc", argument: task.VisibilityKindPublic.String()}
	case task.UserListScope:
		return listQueryAccepted{sql: taskSelectSQL() + " where task_visibility_scopes.user_id = $1 order by tasks.created_at desc", argument: typed.UserID.String()}
	case task.OrganizationListScope:
		return listQueryAccepted{sql: taskSelectSQL() + " where task_visibility_scopes.organization_id = $1 order by tasks.created_at desc", argument: typed.OrganizationID.String()}
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
	if err := rows.Scan(&rawTaskID, &rawOwnerKind, &rawOwnerUserID, &rawOwnerTeamID, &rawOwnerOrganizationID, &rawTitle, &rawDescription, &rawState, &rawVisibilityKind, &rawVisibilityUserID, &rawVisibilityTeamID, &rawVisibilityOrganizationID, &rawSeriesID, &rawSeriesPosition, &rawResponseSchema, &rawPayloadKind, &rawPayload, &rawCreatedBy); err != nil {
		return taskRowRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan task failed")}
	}
	return parseTaskRow(rawTaskID, rawOwnerKind, rawOwnerUserID, rawOwnerTeamID, rawOwnerOrganizationID, rawTitle, rawDescription, rawState, rawVisibilityKind, rawVisibilityUserID, rawVisibilityTeamID, rawVisibilityOrganizationID, rawSeriesID, rawSeriesPosition, rawResponseSchema, rawPayloadKind, rawPayload, rawCreatedBy)
}

func parseTaskRow(rawTaskID string, rawOwnerKind string, rawOwnerUserID string, rawOwnerTeamID string, rawOwnerOrganizationID string, rawTitle string, rawDescription string, rawState string, rawVisibilityKind string, rawVisibilityUserID string, rawVisibilityTeamID string, rawVisibilityOrganizationID string, rawSeriesID string, rawSeriesPosition int, rawResponseSchema string, rawPayloadKind string, rawPayload string, rawCreatedBy string) taskRowResult {
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
	return taskRowAccepted{value: task.Task{ID: taskID.Value, Owner: owner.value, Title: title.Value, Description: description.Value, State: state.Value, Visibility: visibility.value, Placement: placement.value, ResponseSchema: schemaSource.Value, Payload: payload.value, CreatedBy: createdBy.Value}}
}

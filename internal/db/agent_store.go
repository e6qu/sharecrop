package db

import (
	"context"
	"errors"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentStore struct {
	db Beginner
}

func NewAgentStore(pool *pgxpool.Pool) AgentStore {
	return NewAgentStoreFromHandle(NewPGX(pool))
}

func NewAgentStoreFromHandle(handle Beginner) AgentStore {
	return AgentStore{db: handle}
}

func (store AgentStore) CreateCredential(ctx context.Context, credential agent.Credential, hash agent.SecretHash) agent.CreateStoreResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return agent.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin create agent credential transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var rawTaskID *string
	if credential.TaskID != nil {
		value := credential.TaskID.String()
		rawTaskID = &value
	}

	columns := workPolicyColumns(credential.WorkPolicy)
	_, err = tx.Exec(ctx, `
		insert into agent_credentials (
			id, user_id, label, token_hash, state, expires_at, task_id,
			work_seeking_state, work_max_tasks_per_day, work_max_concurrent_reservations,
			work_max_credits_per_day, work_min_reward_credits, work_token_budget_tokens, work_token_budget_note
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, credential.ID.String(), credential.UserID.String(), credential.Label.String(), hash.String(), credential.State.String(), credential.ExpiresAt, rawTaskID,
		columns.state, columns.maxTasksPerDay, columns.maxConcurrentReservations, columns.maxCreditsPerDay, columns.minRewardCredits, columns.tokenBudgetTokens, columns.tokenBudgetNote)
	if err != nil {
		return agent.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert agent credential failed")}
	}

	for _, scope := range credential.Scopes.Values() {
		if _, err := tx.Exec(ctx, "insert into agent_credential_scopes (credential_id, scope) values ($1, $2)", credential.ID.String(), scope.String()); err != nil {
			return agent.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert agent credential scope failed")}
		}
	}
	if err := insertWorkTaskTypes(ctx, tx, credential.ID, columns.taskTypes); err != nil {
		return agent.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert agent credential work task type failed")}
	}

	if err := tx.Commit(ctx); err != nil {
		return agent.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit create agent credential transaction failed")}
	}

	return agent.CreateStoreAccepted{}
}

func (store AgentStore) VerifyCredential(ctx context.Context, hash agent.SecretHash) agent.VerifyStoreResult {
	row, scanErr := scanAgentCredentialRow(store.db.QueryRow(ctx, agentCredentialSelectSQL()+`
		where agent_credentials.token_hash = $1
		group by agent_credentials.id
	`, hash.String()))
	if errors.Is(scanErr, ErrNoRows) {
		return agent.VerifyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "agent credential is invalid")}
	}
	if scanErr != nil {
		return agent.VerifyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "verify agent credential failed")}
	}

	parsed := parseAgentCredential(row)
	accepted, matched := parsed.(agentCredentialParsed)
	if !matched {
		return agent.VerifyStoreRejected{Reason: parsed.(agentCredentialParseRejected).reason}
	}
	return agent.VerifyStoreFound{Value: accepted.value}
}

func (store AgentStore) ListCredentials(ctx context.Context, owner core.UserID, page core.Page) agent.ListStoreResult {
	rows, err := store.db.Query(ctx, agentCredentialSelectSQL()+`
		where agent_credentials.user_id = $1
		group by agent_credentials.id
		order by agent_credentials.created_at, agent_credentials.id
		limit $2 offset $3
	`, owner.String(), page.Limit(), page.Offset())
	if err != nil {
		return agent.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list agent credentials failed")}
	}
	defer rows.Close()

	values := make([]agent.Credential, 0)
	for rows.Next() {
		row, scanErr := scanAgentCredentialRow(rows)
		if scanErr != nil {
			return agent.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan agent credential failed")}
		}
		parsed := parseAgentCredential(row)
		accepted, matched := parsed.(agentCredentialParsed)
		if !matched {
			return agent.ListStoreRejected{Reason: parsed.(agentCredentialParseRejected).reason}
		}
		values = append(values, accepted.value)
	}
	if err := rows.Err(); err != nil {
		return agent.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read agent credentials failed")}
	}
	return agent.ListStoreListed{Values: values}
}

func (store AgentStore) RevokeCredential(ctx context.Context, owner core.UserID, id core.AgentCredentialID) agent.RevokeStoreResult {
	tag, err := store.db.Exec(ctx, `
		update agent_credentials
		set state = 'revoked', state_recorded_at = now()
		where id = $1 and user_id = $2 and state = 'active'
	`, id.String(), owner.String())
	if err != nil {
		return agent.RevokeStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "revoke agent credential failed")}
	}
	if tag == 0 {
		return agent.RevokeStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "active agent credential was not found")}
	}

	readResult := store.findCredentialByID(ctx, id)
	found, matched := readResult.(agentCredentialParsed)
	if !matched {
		return agent.RevokeStoreRejected{Reason: readResult.(agentCredentialParseRejected).reason}
	}
	return agent.RevokeStoreRevoked{Value: found.value}
}

// UpdateWorkPolicy replaces a credential's work policy. Only the owner's
// active, unscoped credential qualifies: a task-scoped worker credential
// operates within an already-granted reservation and never carries a policy.
func (store AgentStore) UpdateWorkPolicy(ctx context.Context, owner core.UserID, id core.AgentCredentialID, policy agent.WorkPolicy) agent.UpdateWorkPolicyStoreResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return agent.UpdateWorkPolicyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin work policy transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var state string
	var rawTaskID *string
	scanErr := tx.QueryRow(ctx, "select state, task_id::text from agent_credentials where id = $1 and user_id = $2 for update", id.String(), owner.String()).Scan(&state, &rawTaskID)
	if errors.Is(scanErr, ErrNoRows) {
		return agent.UpdateWorkPolicyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "agent credential was not found")}
	}
	if scanErr != nil {
		return agent.UpdateWorkPolicyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "load agent credential for work policy failed")}
	}
	if state != agent.StateActive.String() {
		return agent.UpdateWorkPolicyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "a revoked agent credential cannot carry a work policy")}
	}
	if rawTaskID != nil {
		return agent.UpdateWorkPolicyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "a task-scoped worker credential cannot carry a work policy")}
	}

	columns := workPolicyColumns(policy)
	if _, err := tx.Exec(ctx, `
		update agent_credentials
		set work_seeking_state = $2, work_max_tasks_per_day = $3, work_max_concurrent_reservations = $4,
			work_max_credits_per_day = $5, work_min_reward_credits = $6, work_token_budget_tokens = $7, work_token_budget_note = $8
		where id = $1
	`, id.String(), columns.state, columns.maxTasksPerDay, columns.maxConcurrentReservations, columns.maxCreditsPerDay, columns.minRewardCredits, columns.tokenBudgetTokens, columns.tokenBudgetNote); err != nil {
		return agent.UpdateWorkPolicyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "update work policy failed")}
	}
	if _, err := tx.Exec(ctx, "delete from agent_credential_work_task_types where credential_id = $1", id.String()); err != nil {
		return agent.UpdateWorkPolicyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "clear work task types failed")}
	}
	if err := insertWorkTaskTypes(ctx, tx, id, columns.taskTypes); err != nil {
		return agent.UpdateWorkPolicyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert work task type failed")}
	}

	if err := tx.Commit(ctx); err != nil {
		return agent.UpdateWorkPolicyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit work policy transaction failed")}
	}

	readResult := store.findCredentialByID(ctx, id)
	found, matched := readResult.(agentCredentialParsed)
	if !matched {
		return agent.UpdateWorkPolicyStoreRejected{Reason: readResult.(agentCredentialParseRejected).reason}
	}
	return agent.UpdateWorkPolicyStoreUpdated{Value: found.value}
}

// ReadWorkActivity reads today's budget consumption for every credential the
// owner holds in one query: today's consumed daily-task and daily-spend
// counters (the same work_day_counters rows the budget charges write) plus the
// count of still-active reservations attributed to the credential. Both
// subqueries are indexed lookups (the counters' primary key and the partial
// active-reservation index), so the read stays cheap for list pages.
func (store AgentStore) ReadWorkActivity(ctx context.Context, owner core.UserID) agent.WorkActivityStoreResult {
	day := utcDay(time.Now())
	rows, err := store.db.Query(ctx, `
		select agent_credentials.id::text,
			coalesce((select used from work_day_counters where key = $2 || agent_credentials.id::text and day = $3), 0),
			coalesce((select used from work_day_counters where key = $4 || agent_credentials.id::text and day = $3), 0),
			(select count(*) from task_reservations where task_reservations.reserved_via_credential_id = agent_credentials.id and task_reservations.state = 'active')
		from agent_credentials
		where agent_credentials.user_id = $1
	`, owner.String(), workDayKeyAgentTasks, day, workDayKeyAgentSpend)
	if err != nil {
		return agent.WorkActivityStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read agent work activity failed")}
	}
	defer rows.Close()

	values := make([]agent.CredentialWorkActivity, 0)
	for rows.Next() {
		var rawID string
		activity := agent.CredentialWorkActivity{}
		if scanErr := rows.Scan(&rawID, &activity.TasksToday, &activity.CreditsSpentToday, &activity.ActiveReservations); scanErr != nil {
			return agent.WorkActivityStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan agent work activity failed")}
		}
		idResult := core.ParseAgentCredentialID(rawID)
		credentialID, idMatched := idResult.(core.AgentCredentialIDCreated)
		if !idMatched {
			return agent.WorkActivityStoreRejected{Reason: idResult.(core.AgentCredentialIDRejected).Reason}
		}
		activity.CredentialID = credentialID.Value
		values = append(values, activity)
	}
	if err := rows.Err(); err != nil {
		return agent.WorkActivityStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read agent work activity rows failed")}
	}
	return agent.WorkActivityStoreListed{Values: values}
}

func (store AgentStore) findCredentialByID(ctx context.Context, id core.AgentCredentialID) agentCredentialParseResult {
	row, scanErr := scanAgentCredentialRow(store.db.QueryRow(ctx, agentCredentialSelectSQL()+`
		where agent_credentials.id = $1
		group by agent_credentials.id
	`, id.String()))
	if scanErr != nil {
		return agentCredentialParseRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "read agent credential failed")}
	}
	return parseAgentCredential(row)
}

func agentCredentialSelectSQL() string {
	return `
		select agent_credentials.id::text, agent_credentials.user_id::text, agent_credentials.label, agent_credentials.state,
			agent_credentials.expires_at, agent_credentials.task_id::text,
			coalesce(array_remove(array_agg(agent_credential_scopes.scope), null), '{}')::text as scopes,
			agent_credentials.work_seeking_state, agent_credentials.work_max_tasks_per_day, agent_credentials.work_max_concurrent_reservations,
			agent_credentials.work_max_credits_per_day, agent_credentials.work_min_reward_credits,
			agent_credentials.work_token_budget_tokens, agent_credentials.work_token_budget_note,
			coalesce((
				select array_remove(array_agg(agent_credential_work_task_types.task_type), null)
				from agent_credential_work_task_types
				where agent_credential_work_task_types.credential_id = agent_credentials.id
			), '{}')::text as work_task_types
		from agent_credentials
		left join agent_credential_scopes on agent_credential_scopes.credential_id = agent_credentials.id
	`
}

// agentCredentialRow carries one scanned agent_credentials row (with its
// aggregated scope and work-task-type children) before domain parsing.
type agentCredentialRow struct {
	id                        string
	userID                    string
	label                     string
	state                     string
	expiresAt                 *time.Time
	taskID                    *string
	scopes                    StringArray
	workSeekingState          string
	maxTasksPerDay            *int64
	maxConcurrentReservations *int64
	maxCreditsPerDay          *int64
	minRewardCredits          *int64
	tokenBudgetTokens         *int64
	tokenBudgetNote           *string
	workTaskTypes             StringArray
}

// scanAgentCredentialRow scans one selected credential row. Rows structurally
// satisfies Row, so both single-row and iterated reads share it.
func scanAgentCredentialRow(source Row) (agentCredentialRow, error) {
	row := agentCredentialRow{}
	err := source.Scan(
		&row.id, &row.userID, &row.label, &row.state, &row.expiresAt, &row.taskID, &row.scopes,
		&row.workSeekingState, &row.maxTasksPerDay, &row.maxConcurrentReservations,
		&row.maxCreditsPerDay, &row.minRewardCredits, &row.tokenBudgetTokens, &row.tokenBudgetNote,
		&row.workTaskTypes,
	)
	return row, err
}

// workPolicyRowColumns is the flattened storage form of an agent.WorkPolicy.
type workPolicyRowColumns struct {
	state                     string
	maxTasksPerDay            *int64
	maxConcurrentReservations *int64
	maxCreditsPerDay          *int64
	minRewardCredits          *int64
	tokenBudgetTokens         *int64
	tokenBudgetNote           *string
	taskTypes                 []task.TaskType
}

func workPolicyColumns(policy agent.WorkPolicy) workPolicyRowColumns {
	columns := workPolicyRowColumns{state: agent.WorkPolicyState(policy).String()}
	enabled, isEnabled := policy.(agent.WorkPolicyEnabled)
	if !isEnabled {
		return columns
	}

	maxTasks := enabled.Allowances.MaxTasksPerDay.Int64()
	columns.maxTasksPerDay = &maxTasks
	if capped, matched := enabled.Allowances.ConcurrentReservations.(agent.ConcurrentReservationsCapped); matched {
		value := capped.Limit.Int64()
		columns.maxConcurrentReservations = &value
	}
	if capped, matched := enabled.Allowances.DailySpend.(agent.DailySpendCapped); matched {
		value := capped.Limit.Int64()
		columns.maxCreditsPerDay = &value
	}
	if floor, matched := enabled.Allowances.RewardFloor.(agent.RewardFloorAtLeast); matched {
		value := floor.Minimum.Int64()
		columns.minRewardCredits = &value
	}
	if advised, matched := enabled.Allowances.TokenBudget.(agent.TokenBudgetAdvised); matched {
		tokens := advised.Tokens.Int64()
		note := advised.Note.String()
		columns.tokenBudgetTokens = &tokens
		columns.tokenBudgetNote = &note
	}
	if limited, matched := enabled.Allowances.TaskTypes.(agent.TaskTypesLimited); matched {
		columns.taskTypes = limited.Values()
	}
	return columns
}

func insertWorkTaskTypes(ctx context.Context, tx Tx, id core.AgentCredentialID, types []task.TaskType) error {
	for _, taskType := range types {
		if _, err := tx.Exec(ctx, "insert into agent_credential_work_task_types (credential_id, task_type) values ($1, $2)", id.String(), taskType.String()); err != nil {
			return err
		}
	}
	return nil
}

type agentCredentialParseResult interface {
	agentCredentialParseResult()
}

type agentCredentialParsed struct {
	value agent.Credential
}

type agentCredentialParseRejected struct {
	reason core.DomainError
}

func (agentCredentialParsed) agentCredentialParseResult() {}

func (agentCredentialParseRejected) agentCredentialParseResult() {}

func parseAgentCredential(row agentCredentialRow) agentCredentialParseResult {
	idResult := core.ParseAgentCredentialID(row.id)
	credentialID, idMatched := idResult.(core.AgentCredentialIDCreated)
	if !idMatched {
		return agentCredentialParseRejected{reason: idResult.(core.AgentCredentialIDRejected).Reason}
	}
	userResult := core.ParseUserID(row.userID)
	userID, userMatched := userResult.(core.UserIDCreated)
	if !userMatched {
		return agentCredentialParseRejected{reason: userResult.(core.UserIDRejected).Reason}
	}
	labelResult := agent.NewLabel(row.label)
	labelAccepted, labelMatched := labelResult.(agent.LabelAccepted)
	if !labelMatched {
		return agentCredentialParseRejected{reason: labelResult.(agent.LabelRejected).Reason}
	}
	stateResult := agent.ParseState(row.state)
	stateAccepted, stateMatched := stateResult.(agent.StateAccepted)
	if !stateMatched {
		return agentCredentialParseRejected{reason: stateResult.(agent.StateRejected).Reason}
	}

	var taskID *core.TaskID
	if row.taskID != nil {
		taskIDResult := core.ParseTaskID(*row.taskID)
		taskIDCreated, taskIDMatched := taskIDResult.(core.TaskIDCreated)
		if !taskIDMatched {
			return agentCredentialParseRejected{reason: taskIDResult.(core.TaskIDRejected).Reason}
		}
		taskID = &taskIDCreated.Value
	}

	scopes := make([]agent.Scope, 0, len(row.scopes))
	for _, rawScope := range row.scopes {
		scopeResult := agent.ParseScope(rawScope)
		scopeAccepted, scopeMatched := scopeResult.(agent.ScopeAccepted)
		if !scopeMatched {
			return agentCredentialParseRejected{reason: scopeResult.(agent.ScopeRejected).Reason}
		}
		scopes = append(scopes, scopeAccepted.Value)
	}

	policyResult := parseWorkPolicyRow(row)
	policyParsed, policyMatched := policyResult.(workPolicyRowParsed)
	if !policyMatched {
		return agentCredentialParseRejected{reason: policyResult.(workPolicyRowRejected).reason}
	}

	return agentCredentialParsed{value: agent.Credential{
		ID:         credentialID.Value,
		UserID:     userID.Value,
		Label:      labelAccepted.Value,
		Scopes:     agent.NewScopeSet(scopes),
		State:      stateAccepted.Value,
		ExpiresAt:  row.expiresAt,
		TaskID:     taskID,
		WorkPolicy: policyParsed.value,
	}}
}

type workPolicyRowParseResult interface {
	workPolicyRowParseResult()
}

type workPolicyRowParsed struct {
	value agent.WorkPolicy
}

type workPolicyRowRejected struct {
	reason core.DomainError
}

func (workPolicyRowParsed) workPolicyRowParseResult() {}

func (workPolicyRowRejected) workPolicyRowParseResult() {}

func parseWorkPolicyRow(row agentCredentialRow) workPolicyRowParseResult {
	stateResult := agent.ParseWorkSeekingState(row.workSeekingState)
	state, stateMatched := stateResult.(agent.WorkSeekingStateAccepted)
	if !stateMatched {
		return workPolicyRowRejected{reason: stateResult.(agent.WorkSeekingStateRejected).Reason}
	}
	if state.Value == agent.WorkSeekingDisabled {
		return workPolicyRowParsed{value: agent.WorkPolicyDisabled{}}
	}

	if row.maxTasksPerDay == nil {
		return workPolicyRowRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "enabled work policy is missing its daily task budget")}
	}
	budgetResult := agent.NewDailyTaskBudget(*row.maxTasksPerDay)
	budget, budgetMatched := budgetResult.(agent.DailyTaskBudgetAccepted)
	if !budgetMatched {
		return workPolicyRowRejected{reason: budgetResult.(agent.DailyTaskBudgetRejected).Reason}
	}

	allowances := agent.WorkAllowances{
		MaxTasksPerDay:         budget.Value,
		ConcurrentReservations: agent.ConcurrentReservationsUnlimited{},
		DailySpend:             agent.DailySpendUnlimited{},
		TaskTypes:              agent.AllTaskTypesAllowed{},
		RewardFloor:            agent.NoRewardFloor{},
		TokenBudget:            agent.NoTokenBudgetAdvisory{},
	}

	if row.maxConcurrentReservations != nil {
		capResult := agent.NewConcurrentReservationCap(*row.maxConcurrentReservations)
		capAccepted, capMatched := capResult.(agent.ConcurrentReservationCapAccepted)
		if !capMatched {
			return workPolicyRowRejected{reason: capResult.(agent.ConcurrentReservationCapRejected).Reason}
		}
		allowances.ConcurrentReservations = agent.ConcurrentReservationsCapped{Limit: capAccepted.Value}
	}
	if row.maxCreditsPerDay != nil {
		amountResult := ledger.NewCreditAmount(*row.maxCreditsPerDay)
		amount, amountMatched := amountResult.(ledger.CreditAmountAccepted)
		if !amountMatched {
			return workPolicyRowRejected{reason: amountResult.(ledger.CreditAmountRejected).Reason}
		}
		allowances.DailySpend = agent.DailySpendCapped{Limit: amount.Value}
	}
	if row.minRewardCredits != nil {
		amountResult := ledger.NewCreditAmount(*row.minRewardCredits)
		amount, amountMatched := amountResult.(ledger.CreditAmountAccepted)
		if !amountMatched {
			return workPolicyRowRejected{reason: amountResult.(ledger.CreditAmountRejected).Reason}
		}
		allowances.RewardFloor = agent.RewardFloorAtLeast{Minimum: amount.Value}
	}
	if row.tokenBudgetTokens != nil {
		tokensResult := agent.NewTokenBudgetTokens(*row.tokenBudgetTokens)
		tokens, tokensMatched := tokensResult.(agent.TokenBudgetTokensAccepted)
		if !tokensMatched {
			return workPolicyRowRejected{reason: tokensResult.(agent.TokenBudgetTokensRejected).Reason}
		}
		note := ""
		if row.tokenBudgetNote != nil {
			note = *row.tokenBudgetNote
		}
		noteResult := agent.NewTokenBudgetNote(note)
		noteAccepted, noteMatched := noteResult.(agent.TokenBudgetNoteAccepted)
		if !noteMatched {
			return workPolicyRowRejected{reason: noteResult.(agent.TokenBudgetNoteRejected).Reason}
		}
		allowances.TokenBudget = agent.TokenBudgetAdvised{Tokens: tokens.Value, Note: noteAccepted.Value}
	}
	if len(row.workTaskTypes) > 0 {
		types := make([]task.TaskType, 0, len(row.workTaskTypes))
		for _, rawType := range row.workTaskTypes {
			typeResult := task.ParseTaskType(rawType)
			typeAccepted, typeMatched := typeResult.(task.TaskTypeAccepted)
			if !typeMatched {
				return workPolicyRowRejected{reason: typeResult.(task.TaskTypeRejected).Reason}
			}
			types = append(types, typeAccepted.Value)
		}
		limitedResult := agent.NewTaskTypesLimited(types)
		limited, limitedMatched := limitedResult.(agent.TaskTypesLimitedAccepted)
		if !limitedMatched {
			return workPolicyRowRejected{reason: limitedResult.(agent.TaskTypesLimitedRejected).Reason}
		}
		allowances.TaskTypes = limited.Value
	}

	return workPolicyRowParsed{value: agent.WorkPolicyEnabled{Allowances: allowances}}
}

package db

import (
	"context"
	"errors"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentStore struct {
	db Beginner
}

func NewAgentStore(pool *pgxpool.Pool) AgentStore {
	return AgentStore{db: NewPGX(pool)}
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

	_, err = tx.Exec(ctx, `
		insert into agent_credentials (id, user_id, label, token_hash, state, expires_at, task_id)
		values ($1, $2, $3, $4, $5, $6, $7)
	`, credential.ID.String(), credential.UserID.String(), credential.Label.String(), hash.String(), credential.State.String(), credential.ExpiresAt, rawTaskID)
	if err != nil {
		return agent.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert agent credential failed")}
	}

	for _, scope := range credential.Scopes.Values() {
		if _, err := tx.Exec(ctx, "insert into agent_credential_scopes (credential_id, scope) values ($1, $2)", credential.ID.String(), scope.String()); err != nil {
			return agent.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert agent credential scope failed")}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return agent.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit create agent credential transaction failed")}
	}

	return agent.CreateStoreAccepted{}
}

func (store AgentStore) VerifyCredential(ctx context.Context, hash agent.SecretHash) agent.VerifyStoreResult {
	var rawID string
	var rawUserID string
	var label string
	var state string
	var expiresAt *time.Time
	var rawTaskID *string
	var rawScopes []string
	scanErr := store.db.QueryRow(ctx, agentCredentialSelectSQL()+`
		where agent_credentials.token_hash = $1
		group by agent_credentials.id
	`, hash.String()).Scan(&rawID, &rawUserID, &label, &state, &expiresAt, &rawTaskID, &rawScopes)
	if errors.Is(scanErr, ErrNoRows) {
		return agent.VerifyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "agent credential is invalid")}
	}
	if scanErr != nil {
		return agent.VerifyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "verify agent credential failed")}
	}

	parsed := parseAgentCredential(rawID, rawUserID, label, state, expiresAt, rawTaskID, rawScopes)
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
		var rawID string
		var rawUserID string
		var label string
		var state string
		var expiresAt *time.Time
		var rawTaskID *string
		var rawScopes []string
		if err := rows.Scan(&rawID, &rawUserID, &label, &state, &expiresAt, &rawTaskID, &rawScopes); err != nil {
			return agent.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan agent credential failed")}
		}
		parsed := parseAgentCredential(rawID, rawUserID, label, state, expiresAt, rawTaskID, rawScopes)
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

	var rawID string
	var rawUserID string
	var label string
	var state string
	var expiresAt *time.Time
	var rawTaskID *string
	var rawScopes []string
	scanErr := store.db.QueryRow(ctx, agentCredentialSelectSQL()+`
		where agent_credentials.id = $1
		group by agent_credentials.id
	`, id.String()).Scan(&rawID, &rawUserID, &label, &state, &expiresAt, &rawTaskID, &rawScopes)
	if scanErr != nil {
		return agent.RevokeStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read revoked agent credential failed")}
	}

	parsed := parseAgentCredential(rawID, rawUserID, label, state, expiresAt, rawTaskID, rawScopes)
	accepted, matched := parsed.(agentCredentialParsed)
	if !matched {
		return agent.RevokeStoreRejected{Reason: parsed.(agentCredentialParseRejected).reason}
	}
	return agent.RevokeStoreRevoked{Value: accepted.value}
}

func agentCredentialSelectSQL() string {
	return `
		select agent_credentials.id::text, agent_credentials.user_id::text, agent_credentials.label, agent_credentials.state,
			agent_credentials.expires_at, agent_credentials.task_id::text,
			coalesce(array_remove(array_agg(agent_credential_scopes.scope), null), '{}') as scopes
		from agent_credentials
		left join agent_credential_scopes on agent_credential_scopes.credential_id = agent_credentials.id
	`
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

func parseAgentCredential(rawID string, rawUserID string, label string, rawState string, expiresAt *time.Time, rawTaskID *string, rawScopes []string) agentCredentialParseResult {
	idResult := core.ParseAgentCredentialID(rawID)
	credentialID, idMatched := idResult.(core.AgentCredentialIDCreated)
	if !idMatched {
		return agentCredentialParseRejected{reason: idResult.(core.AgentCredentialIDRejected).Reason}
	}
	userResult := core.ParseUserID(rawUserID)
	userID, userMatched := userResult.(core.UserIDCreated)
	if !userMatched {
		return agentCredentialParseRejected{reason: userResult.(core.UserIDRejected).Reason}
	}
	labelResult := agent.NewLabel(label)
	labelAccepted, labelMatched := labelResult.(agent.LabelAccepted)
	if !labelMatched {
		return agentCredentialParseRejected{reason: labelResult.(agent.LabelRejected).Reason}
	}
	stateResult := agent.ParseState(rawState)
	stateAccepted, stateMatched := stateResult.(agent.StateAccepted)
	if !stateMatched {
		return agentCredentialParseRejected{reason: stateResult.(agent.StateRejected).Reason}
	}

	var taskID *core.TaskID
	if rawTaskID != nil {
		taskIDResult := core.ParseTaskID(*rawTaskID)
		taskIDCreated, taskIDMatched := taskIDResult.(core.TaskIDCreated)
		if !taskIDMatched {
			return agentCredentialParseRejected{reason: taskIDResult.(core.TaskIDRejected).Reason}
		}
		taskID = &taskIDCreated.Value
	}

	scopes := make([]agent.Scope, 0, len(rawScopes))
	for _, rawScope := range rawScopes {
		scopeResult := agent.ParseScope(rawScope)
		scopeAccepted, scopeMatched := scopeResult.(agent.ScopeAccepted)
		if !scopeMatched {
			return agentCredentialParseRejected{reason: scopeResult.(agent.ScopeRejected).Reason}
		}
		scopes = append(scopes, scopeAccepted.Value)
	}

	return agentCredentialParsed{value: agent.Credential{
		ID:        credentialID.Value,
		UserID:    userID.Value,
		Label:     labelAccepted.Value,
		Scopes:    agent.NewScopeSet(scopes),
		State:     stateAccepted.Value,
		ExpiresAt: expiresAt,
		TaskID:    taskID,
	}}
}

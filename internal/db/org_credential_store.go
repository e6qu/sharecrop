package db

import (
	"context"
	"errors"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/orgcred"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrgCredentialStore struct {
	db Beginner
}

func NewOrgCredentialStore(pool *pgxpool.Pool) OrgCredentialStore {
	return NewOrgCredentialStoreFromHandle(NewPGX(pool))
}

func NewOrgCredentialStoreFromHandle(handle Beginner) OrgCredentialStore {
	return OrgCredentialStore{db: handle}
}

func (store OrgCredentialStore) CreateCredential(ctx context.Context, credential orgcred.Credential, hash orgcred.SecretHash) orgcred.CreateStoreResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return orgcred.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin create org credential transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		insert into org_credentials (id, organization_id, label, token_hash, state, expires_at)
		values ($1, $2, $3, $4, $5, $6)
	`, credential.ID.String(), credential.OrganizationID.String(), credential.Label.String(), hash.String(), credential.State.String(), credential.ExpiresAt)
	if err != nil {
		return orgcred.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert org credential failed")}
	}

	for _, scope := range credential.Scopes.Values() {
		if _, err := tx.Exec(ctx, "insert into org_credential_scopes (credential_id, scope) values ($1, $2)", credential.ID.String(), scope.String()); err != nil {
			return orgcred.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert org credential scope failed")}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return orgcred.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit create org credential transaction failed")}
	}

	return orgcred.CreateStoreAccepted{}
}

func (store OrgCredentialStore) VerifyCredential(ctx context.Context, hash orgcred.SecretHash) orgcred.VerifyStoreResult {
	var rawID string
	var rawOrganizationID string
	var label string
	var state string
	var expiresAt *time.Time
	var rawScopes StringArray
	scanErr := store.db.QueryRow(ctx, orgCredentialSelectSQL()+`
		where org_credentials.token_hash = $1
		group by org_credentials.id
	`, hash.String()).Scan(&rawID, &rawOrganizationID, &label, &state, &expiresAt, &rawScopes)
	if errors.Is(scanErr, ErrNoRows) {
		return orgcred.VerifyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "org credential is invalid")}
	}
	if scanErr != nil {
		return orgcred.VerifyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "verify org credential failed")}
	}

	parsed := parseOrgCredential(rawID, rawOrganizationID, label, state, expiresAt, rawScopes)
	accepted, matched := parsed.(orgCredentialParsed)
	if !matched {
		return orgcred.VerifyStoreRejected{Reason: parsed.(orgCredentialParseRejected).reason}
	}
	return orgcred.VerifyStoreFound{Value: accepted.value}
}

func (store OrgCredentialStore) ListCredentials(ctx context.Context, organizationID core.OrganizationID, page core.Page) orgcred.ListStoreResult {
	rows, err := store.db.Query(ctx, orgCredentialSelectSQL()+`
		where org_credentials.organization_id = $1
		group by org_credentials.id
		order by org_credentials.created_at, org_credentials.id
		limit $2 offset $3
	`, organizationID.String(), page.Limit(), page.Offset())
	if err != nil {
		return orgcred.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list org credentials failed")}
	}
	defer rows.Close()

	values := make([]orgcred.Credential, 0)
	for rows.Next() {
		var rawID string
		var rawOrganizationID string
		var label string
		var state string
		var expiresAt *time.Time
		var rawScopes StringArray
		if err := rows.Scan(&rawID, &rawOrganizationID, &label, &state, &expiresAt, &rawScopes); err != nil {
			return orgcred.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan org credential failed")}
		}
		parsed := parseOrgCredential(rawID, rawOrganizationID, label, state, expiresAt, rawScopes)
		accepted, matched := parsed.(orgCredentialParsed)
		if !matched {
			return orgcred.ListStoreRejected{Reason: parsed.(orgCredentialParseRejected).reason}
		}
		values = append(values, accepted.value)
	}
	if err := rows.Err(); err != nil {
		return orgcred.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read org credentials failed")}
	}
	return orgcred.ListStoreListed{Values: values}
}

func (store OrgCredentialStore) RevokeCredential(ctx context.Context, organizationID core.OrganizationID, id core.OrgCredentialID) orgcred.RevokeStoreResult {
	tag, err := store.db.Exec(ctx, `
		update org_credentials
		set state = 'revoked', state_recorded_at = now()
		where id = $1 and organization_id = $2 and state = 'active'
	`, id.String(), organizationID.String())
	if err != nil {
		return orgcred.RevokeStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "revoke org credential failed")}
	}
	if tag == 0 {
		return orgcred.RevokeStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "active org credential was not found")}
	}

	var rawID string
	var rawOrganizationID string
	var label string
	var state string
	var expiresAt *time.Time
	var rawScopes StringArray
	scanErr := store.db.QueryRow(ctx, orgCredentialSelectSQL()+`
		where org_credentials.id = $1
		group by org_credentials.id
	`, id.String()).Scan(&rawID, &rawOrganizationID, &label, &state, &expiresAt, &rawScopes)
	if scanErr != nil {
		return orgcred.RevokeStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read revoked org credential failed")}
	}

	parsed := parseOrgCredential(rawID, rawOrganizationID, label, state, expiresAt, rawScopes)
	accepted, matched := parsed.(orgCredentialParsed)
	if !matched {
		return orgcred.RevokeStoreRejected{Reason: parsed.(orgCredentialParseRejected).reason}
	}
	return orgcred.RevokeStoreRevoked{Value: accepted.value}
}

func orgCredentialSelectSQL() string {
	return `
		select org_credentials.id::text, org_credentials.organization_id::text, org_credentials.label, org_credentials.state,
			org_credentials.expires_at,
			coalesce(array_remove(array_agg(org_credential_scopes.scope), null), '{}')::text as scopes
		from org_credentials
		left join org_credential_scopes on org_credential_scopes.credential_id = org_credentials.id
	`
}

type orgCredentialParseResult interface {
	orgCredentialParseResult()
}

type orgCredentialParsed struct {
	value orgcred.Credential
}

type orgCredentialParseRejected struct {
	reason core.DomainError
}

func (orgCredentialParsed) orgCredentialParseResult() {}

func (orgCredentialParseRejected) orgCredentialParseResult() {}

func parseOrgCredential(rawID string, rawOrganizationID string, label string, rawState string, expiresAt *time.Time, rawScopes []string) orgCredentialParseResult {
	idResult := core.ParseOrgCredentialID(rawID)
	credentialID, idMatched := idResult.(core.OrgCredentialIDCreated)
	if !idMatched {
		return orgCredentialParseRejected{reason: idResult.(core.OrgCredentialIDRejected).Reason}
	}
	organizationResult := core.ParseOrganizationID(rawOrganizationID)
	organizationID, organizationMatched := organizationResult.(core.OrganizationIDCreated)
	if !organizationMatched {
		return orgCredentialParseRejected{reason: organizationResult.(core.OrganizationIDRejected).Reason}
	}
	labelResult := agent.NewLabel(label)
	labelAccepted, labelMatched := labelResult.(agent.LabelAccepted)
	if !labelMatched {
		return orgCredentialParseRejected{reason: labelResult.(agent.LabelRejected).Reason}
	}
	stateResult := agent.ParseState(rawState)
	stateAccepted, stateMatched := stateResult.(agent.StateAccepted)
	if !stateMatched {
		return orgCredentialParseRejected{reason: stateResult.(agent.StateRejected).Reason}
	}

	scopes := make([]agent.Scope, 0, len(rawScopes))
	for _, rawScope := range rawScopes {
		scopeResult := agent.ParseScope(rawScope)
		scopeAccepted, scopeMatched := scopeResult.(agent.ScopeAccepted)
		if !scopeMatched {
			return orgCredentialParseRejected{reason: scopeResult.(agent.ScopeRejected).Reason}
		}
		scopes = append(scopes, scopeAccepted.Value)
	}

	return orgCredentialParsed{value: orgcred.Credential{
		ID:             credentialID.Value,
		OrganizationID: organizationID.Value,
		Label:          labelAccepted.Value,
		Scopes:         agent.NewScopeSet(scopes),
		State:          stateAccepted.Value,
		ExpiresAt:      expiresAt,
	}}
}

package wasmdemo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/orgcred"
)

// OrgCredentialBrowserStore implements orgcred.Store against BrowserStorage,
// so the real orgcred.Service (the same code cmd/sharecrop runs against
// Postgres) can serve the browser demo directly.
type OrgCredentialBrowserStore struct {
	storage BrowserStorage
}

func NewOrgCredentialBrowserStore(storage BrowserStorage) OrgCredentialBrowserStore {
	return OrgCredentialBrowserStore{storage: storage}
}

type storedOrgCredential struct {
	ID             string   `json:"id"`
	OrganizationID string   `json:"organization_id"`
	Label          string   `json:"label"`
	State          string   `json:"state"`
	ExpiresAt      int64    `json:"expires_at_unix,omitempty"`
	Scopes         []string `json:"scopes"`
}

func orgCredentialKey(id string) string       { return "orgcred:credential:" + id }
func orgCredentialHashKey(hash string) string { return "orgcred:credential_hash:" + hash }
func orgCredentialIndexKey(organizationID string) string {
	return "orgcred:credential_index:" + organizationID
}

func putStoredOrgCredentialJSON(storage BrowserStorage, rawKey string, record storedOrgCredential) bool {
	encoded, err := json.Marshal(record)
	if err != nil {
		return false
	}
	return putStorageString(storage, rawKey, string(encoded))
}

func getStoredOrgCredentialJSON(storage BrowserStorage, rawKey string) (storedOrgCredential, bool, bool) {
	raw, found, ok := getStorageString(storage, rawKey)
	if !ok || !found {
		return storedOrgCredential{}, found, ok
	}
	var record storedOrgCredential
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return storedOrgCredential{}, false, false
	}
	return record, true, true
}

func (store OrgCredentialBrowserStore) CreateCredential(_ context.Context, credential orgcred.Credential, hash orgcred.SecretHash) orgcred.CreateStoreResult {
	var expiresAt int64
	if credential.ExpiresAt != nil {
		expiresAt = credential.ExpiresAt.UnixNano()
	}
	scopeValues := credential.Scopes.Values()
	rawScopes := make([]string, len(scopeValues))
	for index, scope := range scopeValues {
		rawScopes[index] = scope.String()
	}

	record := storedOrgCredential{
		ID:             credential.ID.String(),
		OrganizationID: credential.OrganizationID.String(),
		Label:          credential.Label.String(),
		State:          credential.State.String(),
		ExpiresAt:      expiresAt,
		Scopes:         rawScopes,
	}
	if !putStoredOrgCredentialJSON(store.storage, orgCredentialKey(record.ID), record) {
		return orgcred.CreateStoreRejected{Reason: invalidState("insert org credential failed")}
	}
	if !putStorageString(store.storage, orgCredentialHashKey(hash.String()), record.ID) {
		return orgcred.CreateStoreRejected{Reason: invalidState("insert org credential hash index failed")}
	}
	indexResult := appendStringIndex(store.storage, orgCredentialIndexKey(record.OrganizationID), record.ID, "org credential")
	if _, matched := indexResult.(stringIndexStored); !matched {
		return orgcred.CreateStoreRejected{Reason: invalidState("update org credential index failed")}
	}
	return orgcred.CreateStoreAccepted{}
}

func (store OrgCredentialBrowserStore) loadCredential(id string) (orgcred.Credential, bool, *core.DomainError) {
	record, found, ok := getStoredOrgCredentialJSON(store.storage, orgCredentialKey(id))
	if !ok {
		reason := invalidState("org credential lookup failed")
		return orgcred.Credential{}, false, &reason
	}
	if !found {
		return orgcred.Credential{}, false, nil
	}

	idResult := core.ParseOrgCredentialID(record.ID)
	credentialID, idMatched := idResult.(core.OrgCredentialIDCreated)
	if !idMatched {
		reason := idResult.(core.OrgCredentialIDRejected).Reason
		return orgcred.Credential{}, false, &reason
	}
	organizationResult := core.ParseOrganizationID(record.OrganizationID)
	organizationID, organizationMatched := organizationResult.(core.OrganizationIDCreated)
	if !organizationMatched {
		reason := organizationResult.(core.OrganizationIDRejected).Reason
		return orgcred.Credential{}, false, &reason
	}
	labelResult := agent.NewLabel(record.Label)
	labelAccepted, labelMatched := labelResult.(agent.LabelAccepted)
	if !labelMatched {
		reason := labelResult.(agent.LabelRejected).Reason
		return orgcred.Credential{}, false, &reason
	}
	stateResult := agent.ParseState(record.State)
	stateAccepted, stateMatched := stateResult.(agent.StateAccepted)
	if !stateMatched {
		reason := stateResult.(agent.StateRejected).Reason
		return orgcred.Credential{}, false, &reason
	}
	var expiresAt *time.Time
	if record.ExpiresAt != 0 {
		value := time.Unix(0, record.ExpiresAt).UTC()
		expiresAt = &value
	}
	scopes := make([]agent.Scope, 0, len(record.Scopes))
	for _, raw := range record.Scopes {
		scopeResult := agent.ParseScope(raw)
		scopeAccepted, scopeMatched := scopeResult.(agent.ScopeAccepted)
		if !scopeMatched {
			reason := scopeResult.(agent.ScopeRejected).Reason
			return orgcred.Credential{}, false, &reason
		}
		scopes = append(scopes, scopeAccepted.Value)
	}

	return orgcred.Credential{
		ID:             credentialID.Value,
		OrganizationID: organizationID.Value,
		Label:          labelAccepted.Value,
		Scopes:         agent.NewScopeSet(scopes),
		State:          stateAccepted.Value,
		ExpiresAt:      expiresAt,
	}, true, nil
}

func (store OrgCredentialBrowserStore) VerifyCredential(_ context.Context, hash orgcred.SecretHash) orgcred.VerifyStoreResult {
	id, found, ok := getStorageString(store.storage, orgCredentialHashKey(hash.String()))
	if !ok {
		return orgcred.VerifyStoreRejected{Reason: invalidState("verify org credential failed")}
	}
	if !found {
		return orgcred.VerifyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "org credential is invalid")}
	}
	credential, credentialFound, rejected := store.loadCredential(id)
	if rejected != nil {
		return orgcred.VerifyStoreRejected{Reason: *rejected}
	}
	if !credentialFound {
		return orgcred.VerifyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "org credential is invalid")}
	}
	return orgcred.VerifyStoreFound{Value: credential}
}

func (store OrgCredentialBrowserStore) ListCredentials(_ context.Context, organizationID core.OrganizationID, page core.Page) orgcred.ListStoreResult {
	indexResult := loadStringIndex(store.storage, orgCredentialIndexKey(organizationID.String()), "org credential")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		return orgcred.ListStoreRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
	}

	start := page.Offset()
	if start > len(loaded.values) {
		start = len(loaded.values)
	}
	end := start + page.Limit()
	if end > len(loaded.values) {
		end = len(loaded.values)
	}

	values := make([]orgcred.Credential, 0, end-start)
	for _, id := range loaded.values[start:end] {
		credential, found, rejected := store.loadCredential(id)
		if rejected != nil {
			return orgcred.ListStoreRejected{Reason: *rejected}
		}
		if found {
			values = append(values, credential)
		}
	}
	return orgcred.ListStoreListed{Values: values}
}

func (store OrgCredentialBrowserStore) RevokeCredential(_ context.Context, organizationID core.OrganizationID, id core.OrgCredentialID) orgcred.RevokeStoreResult {
	credential, found, rejected := store.loadCredential(id.String())
	if rejected != nil {
		return orgcred.RevokeStoreRejected{Reason: *rejected}
	}
	if !found || credential.OrganizationID != organizationID || credential.State != agent.StateActive {
		return orgcred.RevokeStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "active org credential was not found")}
	}
	credential.State = agent.StateRevoked

	record, _, ok := getStoredOrgCredentialJSON(store.storage, orgCredentialKey(id.String()))
	if !ok {
		return orgcred.RevokeStoreRejected{Reason: invalidState("revoke org credential failed")}
	}
	record.State = agent.StateRevoked.String()
	if !putStoredOrgCredentialJSON(store.storage, orgCredentialKey(id.String()), record) {
		return orgcred.RevokeStoreRejected{Reason: invalidState("revoke org credential failed")}
	}
	return orgcred.RevokeStoreRevoked{Value: credential}
}

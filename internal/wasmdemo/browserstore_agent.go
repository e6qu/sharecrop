package wasmdemo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
)

// AgentBrowserStore implements agent.Store against BrowserStorage, so the
// real agent.Service (the same code cmd/sharecrop runs against Postgres)
// can serve the browser demo directly.
type AgentBrowserStore struct {
	storage BrowserStorage
}

func NewAgentBrowserStore(storage BrowserStorage) AgentBrowserStore {
	return AgentBrowserStore{storage: storage}
}

type storedAgentCredential struct {
	ID        string   `json:"id"`
	UserID    string   `json:"user_id"`
	Label     string   `json:"label"`
	Hash      string   `json:"hash"`
	State     string   `json:"state"`
	ExpiresAt int64    `json:"expires_at_unix,omitempty"`
	TaskID    string   `json:"task_id,omitempty"`
	Scopes    []string `json:"scopes"`
}

func agentCredentialKey(id string) string       { return "agent:credential:" + id }
func agentCredentialHashKey(hash string) string { return "agent:credential_hash:" + hash }
func agentCredentialIndexKey(userID string) string {
	return "agent:credential_index:" + userID
}

func putStoredAgentCredentialJSON(storage BrowserStorage, rawKey string, record storedAgentCredential) bool {
	encoded, err := json.Marshal(record)
	if err != nil {
		return false
	}
	return putStorageString(storage, rawKey, string(encoded))
}

func getStoredAgentCredentialJSON(storage BrowserStorage, rawKey string) (storedAgentCredential, bool, bool) {
	raw, found, ok := getStorageString(storage, rawKey)
	if !ok || !found {
		return storedAgentCredential{}, found, ok
	}
	var record storedAgentCredential
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return storedAgentCredential{}, false, false
	}
	return record, true, true
}

func (store AgentBrowserStore) CreateCredential(_ context.Context, credential agent.Credential, hash agent.SecretHash) agent.CreateStoreResult {
	rawTaskID := ""
	if credential.TaskID != nil {
		rawTaskID = credential.TaskID.String()
	}
	var expiresAt int64
	if credential.ExpiresAt != nil {
		expiresAt = credential.ExpiresAt.UnixNano()
	}
	scopeValues := credential.Scopes.Values()
	rawScopes := make([]string, len(scopeValues))
	for index, scope := range scopeValues {
		rawScopes[index] = scope.String()
	}

	record := storedAgentCredential{
		ID:        credential.ID.String(),
		UserID:    credential.UserID.String(),
		Label:     credential.Label.String(),
		Hash:      hash.String(),
		State:     credential.State.String(),
		ExpiresAt: expiresAt,
		TaskID:    rawTaskID,
		Scopes:    rawScopes,
	}
	if !putStoredAgentCredentialJSON(store.storage, agentCredentialKey(record.ID), record) {
		return agent.CreateStoreRejected{Reason: invalidState("insert agent credential failed")}
	}
	if !putStorageString(store.storage, agentCredentialHashKey(hash.String()), record.ID) {
		return agent.CreateStoreRejected{Reason: invalidState("insert agent credential hash index failed")}
	}
	indexResult := appendStringIndex(store.storage, agentCredentialIndexKey(record.UserID), record.ID, "agent credential")
	if _, matched := indexResult.(stringIndexStored); !matched {
		return agent.CreateStoreRejected{Reason: invalidState("update agent credential index failed")}
	}
	return agent.CreateStoreAccepted{}
}

func (store AgentBrowserStore) loadCredential(id string) (agent.Credential, bool, *core.DomainError) {
	record, found, ok := getStoredAgentCredentialJSON(store.storage, agentCredentialKey(id))
	if !ok {
		reason := invalidState("agent credential lookup failed")
		return agent.Credential{}, false, &reason
	}
	if !found {
		return agent.Credential{}, false, nil
	}

	idResult := core.ParseAgentCredentialID(record.ID)
	credentialID, idMatched := idResult.(core.AgentCredentialIDCreated)
	if !idMatched {
		reason := idResult.(core.AgentCredentialIDRejected).Reason
		return agent.Credential{}, false, &reason
	}
	userResult := core.ParseUserID(record.UserID)
	userID, userMatched := userResult.(core.UserIDCreated)
	if !userMatched {
		reason := userResult.(core.UserIDRejected).Reason
		return agent.Credential{}, false, &reason
	}
	labelResult := agent.NewLabel(record.Label)
	labelAccepted, labelMatched := labelResult.(agent.LabelAccepted)
	if !labelMatched {
		reason := labelResult.(agent.LabelRejected).Reason
		return agent.Credential{}, false, &reason
	}
	stateResult := agent.ParseState(record.State)
	stateAccepted, stateMatched := stateResult.(agent.StateAccepted)
	if !stateMatched {
		reason := stateResult.(agent.StateRejected).Reason
		return agent.Credential{}, false, &reason
	}
	var expiresAt *time.Time
	if record.ExpiresAt != 0 {
		value := time.Unix(0, record.ExpiresAt).UTC()
		expiresAt = &value
	}
	var taskID *core.TaskID
	if record.TaskID != "" {
		taskIDResult := core.ParseTaskID(record.TaskID)
		taskIDCreated, taskIDMatched := taskIDResult.(core.TaskIDCreated)
		if !taskIDMatched {
			reason := taskIDResult.(core.TaskIDRejected).Reason
			return agent.Credential{}, false, &reason
		}
		taskID = &taskIDCreated.Value
	}
	scopes := make([]agent.Scope, 0, len(record.Scopes))
	for _, raw := range record.Scopes {
		scopeResult := agent.ParseScope(raw)
		scopeAccepted, scopeMatched := scopeResult.(agent.ScopeAccepted)
		if !scopeMatched {
			reason := scopeResult.(agent.ScopeRejected).Reason
			return agent.Credential{}, false, &reason
		}
		scopes = append(scopes, scopeAccepted.Value)
	}

	return agent.Credential{
		ID:        credentialID.Value,
		UserID:    userID.Value,
		Label:     labelAccepted.Value,
		Scopes:    agent.NewScopeSet(scopes),
		State:     stateAccepted.Value,
		ExpiresAt: expiresAt,
		TaskID:    taskID,
	}, true, nil
}

func (store AgentBrowserStore) VerifyCredential(_ context.Context, hash agent.SecretHash) agent.VerifyStoreResult {
	id, found, ok := getStorageString(store.storage, agentCredentialHashKey(hash.String()))
	if !ok {
		return agent.VerifyStoreRejected{Reason: invalidState("verify agent credential failed")}
	}
	if !found {
		return agent.VerifyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "agent credential is invalid")}
	}
	credential, credentialFound, rejected := store.loadCredential(id)
	if rejected != nil {
		return agent.VerifyStoreRejected{Reason: *rejected}
	}
	if !credentialFound {
		return agent.VerifyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "agent credential is invalid")}
	}
	return agent.VerifyStoreFound{Value: credential}
}

func (store AgentBrowserStore) ListCredentials(_ context.Context, owner core.UserID, page core.Page) agent.ListStoreResult {
	indexResult := loadStringIndex(store.storage, agentCredentialIndexKey(owner.String()), "agent credential")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		return agent.ListStoreRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
	}

	start := page.Offset()
	if start > len(loaded.values) {
		start = len(loaded.values)
	}
	end := start + page.Limit()
	if end > len(loaded.values) {
		end = len(loaded.values)
	}

	values := make([]agent.Credential, 0, end-start)
	for _, id := range loaded.values[start:end] {
		credential, found, rejected := store.loadCredential(id)
		if rejected != nil {
			return agent.ListStoreRejected{Reason: *rejected}
		}
		if found {
			values = append(values, credential)
		}
	}
	return agent.ListStoreListed{Values: values}
}

func (store AgentBrowserStore) RevokeCredential(_ context.Context, owner core.UserID, id core.AgentCredentialID) agent.RevokeStoreResult {
	credential, found, rejected := store.loadCredential(id.String())
	if rejected != nil {
		return agent.RevokeStoreRejected{Reason: *rejected}
	}
	if !found || credential.UserID != owner || credential.State != agent.StateActive {
		return agent.RevokeStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "active agent credential was not found")}
	}
	credential.State = agent.StateRevoked

	record, _, ok := getStoredAgentCredentialJSON(store.storage, agentCredentialKey(id.String()))
	if !ok {
		return agent.RevokeStoreRejected{Reason: invalidState("revoke agent credential failed")}
	}
	record.State = agent.StateRevoked.String()
	if !putStoredAgentCredentialJSON(store.storage, agentCredentialKey(id.String()), record) {
		return agent.RevokeStoreRejected{Reason: invalidState("revoke agent credential failed")}
	}
	return agent.RevokeStoreRevoked{Value: credential}
}

package agent

import (
	"context"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

type Store interface {
	CreateCredential(context.Context, Credential, SecretHash) CreateStoreResult
	VerifyCredential(context.Context, SecretHash) VerifyStoreResult
	ListCredentials(context.Context, core.UserID, core.Page) ListStoreResult
	RevokeCredential(context.Context, core.UserID, core.AgentCredentialID) RevokeStoreResult
}

type Service struct {
	store Store
}

func NewService(store Store) Service {
	return Service{store: store}
}

type CreateResult interface {
	createResult()
}

type CredentialCreated struct {
	Value  Credential
	Secret SecretPlain
}

type CreateRejected struct {
	Reason core.DomainError
}

func (CredentialCreated) createResult() {}

func (CreateRejected) createResult() {}

// Create mints a new credential. expiresAt is nil for a non-expiring
// credential; taskID is nil for a credential usable against every task the
// scopes otherwise allow, or set to restrict it to exactly one task
// (see Credential.MatchesTask).
func (service Service) Create(ctx context.Context, owner core.UserID, label Label, scopes ScopeSet, expiresAt *time.Time, taskID *core.TaskID) CreateResult {
	if scopes.IsEmpty() {
		return CreateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "at least one agent scope is required")}
	}

	idResult := core.NewAgentCredentialID()
	idCreated, idMatched := idResult.(core.AgentCredentialIDCreated)
	if !idMatched {
		return CreateRejected{Reason: idResult.(core.AgentCredentialIDRejected).Reason}
	}

	secretResult := NewSecretPlain()
	secretCreated, secretMatched := secretResult.(SecretPlainAccepted)
	if !secretMatched {
		return CreateRejected{Reason: secretResult.(SecretPlainRejected).Reason}
	}

	credential := Credential{
		ID:        idCreated.Value,
		UserID:    owner,
		Label:     label,
		Scopes:    scopes,
		State:     StateActive,
		ExpiresAt: expiresAt,
		TaskID:    taskID,
	}

	storeResult := service.store.CreateCredential(ctx, credential, secretCreated.Value.Hash())
	if rejected, matched := storeResult.(CreateStoreRejected); matched {
		return CreateRejected{Reason: rejected.Reason}
	}

	return CredentialCreated{Value: credential, Secret: secretCreated.Value}
}

// taskWorkerCredentialTTL is the default lifetime of an auto-issued
// task-scoped credential (see IssueTaskWorkerCredential). 30 days comfortably
// covers typical task turnaround without an indefinitely-valid secret.
const taskWorkerCredentialTTL = 30 * 24 * time.Hour

// IssueTaskWorkerCredential mints a credential restricted to exactly one
// task, scoped to just what's needed to read that task and submit a
// response — narrow enough to safely hand to an agent. It satisfies
// task.TaskCredentialIssuer structurally (Go interfaces are implicit), so
// task.Service can depend on that interface without importing this package.
// Best-effort by design: a failure reports ok=false rather than an error,
// since minting this convenience credential should never block the
// reservation/approval flow it's attached to.
func (service Service) IssueTaskWorkerCredential(ctx context.Context, owner core.UserID, taskID core.TaskID) (secret string, ok bool) {
	label, labelMatched := NewLabel("Task worker token").(LabelAccepted)
	if !labelMatched {
		return "", false
	}
	scopes := NewScopeSet([]Scope{ScopeTasksRead, ScopeSubmissionsWrite, ScopeSubmissionsRead})
	expiresAt := time.Now().Add(taskWorkerCredentialTTL)

	result := service.Create(ctx, owner, label.Value, scopes, &expiresAt, &taskID)
	created, matched := result.(CredentialCreated)
	if !matched {
		return "", false
	}
	return created.Secret.String(), true
}

type VerifyResult interface {
	verifyResult()
}

type CredentialVerified struct {
	Subject    auth.UserSubject
	Credential Credential
}

type VerifyRejected struct {
	Reason core.DomainError
}

func (CredentialVerified) verifyResult() {}

func (VerifyRejected) verifyResult() {}

func (service Service) Verify(ctx context.Context, secret SecretPlain) VerifyResult {
	storeResult := service.store.VerifyCredential(ctx, secret.Hash())
	found, matched := storeResult.(VerifyStoreFound)
	if !matched {
		return VerifyRejected{Reason: storeResult.(VerifyStoreRejected).Reason}
	}
	if found.Value.State != StateActive {
		return VerifyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "agent credential is revoked")}
	}
	if found.Value.IsExpired(time.Now()) {
		return VerifyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "agent credential has expired")}
	}

	return CredentialVerified{
		Subject:    auth.UserSubject{ID: found.Value.UserID},
		Credential: found.Value,
	}
}

type ListResult interface {
	listResult()
}

type CredentialsListed struct {
	Values []Credential
}

type ListRejected struct {
	Reason core.DomainError
}

func (CredentialsListed) listResult() {}

func (ListRejected) listResult() {}

func (service Service) List(ctx context.Context, owner core.UserID, page core.Page) ListResult {
	storeResult := service.store.ListCredentials(ctx, owner, page)
	listed, matched := storeResult.(ListStoreListed)
	if !matched {
		return ListRejected{Reason: storeResult.(ListStoreRejected).Reason}
	}
	return CredentialsListed{Values: listed.Values}
}

type RevokeResult interface {
	revokeResult()
}

type CredentialRevoked struct {
	Value Credential
}

type RevokeRejected struct {
	Reason core.DomainError
}

func (CredentialRevoked) revokeResult() {}

func (RevokeRejected) revokeResult() {}

func (service Service) Revoke(ctx context.Context, owner core.UserID, id core.AgentCredentialID) RevokeResult {
	storeResult := service.store.RevokeCredential(ctx, owner, id)
	revoked, matched := storeResult.(RevokeStoreRevoked)
	if !matched {
		return RevokeRejected{Reason: storeResult.(RevokeStoreRejected).Reason}
	}
	return CredentialRevoked{Value: revoked.Value}
}

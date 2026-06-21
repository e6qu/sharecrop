package agent

import (
	"context"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

type Store interface {
	CreateCredential(context.Context, Credential, SecretHash) CreateStoreResult
	VerifyCredential(context.Context, SecretHash) VerifyStoreResult
	ListCredentials(context.Context, core.UserID) ListStoreResult
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

func (service Service) Create(ctx context.Context, owner core.UserID, label Label, scopes ScopeSet) CreateResult {
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
		ID:     idCreated.Value,
		UserID: owner,
		Label:  label,
		Scopes: scopes,
		State:  StateActive,
	}

	storeResult := service.store.CreateCredential(ctx, credential, secretCreated.Value.Hash())
	if rejected, matched := storeResult.(CreateStoreRejected); matched {
		return CreateRejected{Reason: rejected.Reason}
	}

	return CredentialCreated{Value: credential, Secret: secretCreated.Value}
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

func (service Service) List(ctx context.Context, owner core.UserID) ListResult {
	storeResult := service.store.ListCredentials(ctx, owner)
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

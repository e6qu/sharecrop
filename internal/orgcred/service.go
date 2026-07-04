package orgcred

import (
	"context"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

type Store interface {
	CreateCredential(context.Context, Credential, SecretHash) CreateStoreResult
	VerifyCredential(context.Context, SecretHash) VerifyStoreResult
	ListCredentials(context.Context, core.OrganizationID, core.Page) ListStoreResult
	RevokeCredential(context.Context, core.OrganizationID, core.OrgCredentialID) RevokeStoreResult
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

// Create mints a new org-wide credential. expiresAt is nil for a
// non-expiring credential.
func (service Service) Create(ctx context.Context, organizationID core.OrganizationID, label agent.Label, scopes agent.ScopeSet, expiresAt *time.Time) CreateResult {
	if scopes.IsEmpty() {
		return CreateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "at least one scope is required")}
	}

	idResult := core.NewOrgCredentialID()
	idCreated, idMatched := idResult.(core.OrgCredentialIDCreated)
	if !idMatched {
		return CreateRejected{Reason: idResult.(core.OrgCredentialIDRejected).Reason}
	}

	secretResult := NewSecretPlain()
	secretCreated, secretMatched := secretResult.(SecretPlainAccepted)
	if !secretMatched {
		return CreateRejected{Reason: secretResult.(SecretPlainRejected).Reason}
	}

	credential := Credential{
		ID:             idCreated.Value,
		OrganizationID: organizationID,
		Label:          label,
		Scopes:         scopes,
		State:          agent.StateActive,
		ExpiresAt:      expiresAt,
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
	Subject    auth.OrgSubject
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
	if found.Value.State != agent.StateActive {
		return VerifyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "org credential is revoked")}
	}
	if found.Value.IsExpired(time.Now()) {
		return VerifyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "org credential has expired")}
	}

	return CredentialVerified{
		Subject:    auth.OrgSubject{ID: found.Value.OrganizationID},
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

func (service Service) List(ctx context.Context, organizationID core.OrganizationID, page core.Page) ListResult {
	storeResult := service.store.ListCredentials(ctx, organizationID, page)
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

func (service Service) Revoke(ctx context.Context, organizationID core.OrganizationID, id core.OrgCredentialID) RevokeResult {
	storeResult := service.store.RevokeCredential(ctx, organizationID, id)
	revoked, matched := storeResult.(RevokeStoreRevoked)
	if !matched {
		return RevokeRejected{Reason: storeResult.(RevokeStoreRejected).Reason}
	}
	return CredentialRevoked{Value: revoked.Value}
}

package auth

import "github.com/e6qu/sharecrop/internal/core"

type StoreUserResult interface {
	storeUserResult()
}

type StoreUserAccepted struct{}

type StoreUserRejected struct {
	Reason core.DomainError
}

func (StoreUserAccepted) storeUserResult() {}

func (StoreUserRejected) storeUserResult() {}

type ExternalIdentityResult interface{ externalIdentityResult() }

// ExternalIdentity is the stable issuer/subject pair from a verified OpenID
// Connect ID token. It is deliberately independent of mutable profile claims.
type ExternalIdentity struct {
	Issuer  string
	Subject string
}

type ExternalIdentityFound struct{ UserID core.UserID }
type ExternalIdentityRejected struct{ Reason core.DomainError }

func (ExternalIdentityFound) externalIdentityResult()    {}
func (ExternalIdentityRejected) externalIdentityResult() {}

type CredentialLookupResult interface {
	credentialLookupResult()
}

type CredentialFound struct {
	Record CredentialRecord
}

type CredentialMissing struct{}

type CredentialLookupRejected struct {
	Reason core.DomainError
}

func (CredentialFound) credentialLookupResult() {}

func (CredentialMissing) credentialLookupResult() {}

func (CredentialLookupRejected) credentialLookupResult() {}

type UserDirectoryEntry struct {
	ID     core.UserID
	Email  EmailAddress
	Status string
}

type UserDirectoryResult interface {
	userDirectoryResult()
}

type UsersListed struct {
	Values []UserDirectoryEntry
}

type UserDirectoryRejected struct {
	Reason core.DomainError
}

func (UsersListed) userDirectoryResult() {}

func (UserDirectoryRejected) userDirectoryResult() {}

type StoreGuestResult interface {
	storeGuestResult()
}

type StoreGuestAccepted struct{}

type StoreGuestRejected struct {
	Reason core.DomainError
}

func (StoreGuestAccepted) storeGuestResult() {}

func (StoreGuestRejected) storeGuestResult() {}

type StoreRefreshTokenResult interface {
	storeRefreshTokenResult()
}

type StoreRefreshTokenAccepted struct{}

type StoreRefreshTokenRejected struct {
	Reason core.DomainError
}

func (StoreRefreshTokenAccepted) storeRefreshTokenResult() {}

func (StoreRefreshTokenRejected) storeRefreshTokenResult() {}

type ValidateRefreshTokenResult interface {
	validateRefreshTokenResult()
}

type RefreshTokenActive struct{}

type RefreshTokenInactive struct{}

type ValidateRefreshTokenRejected struct {
	Reason core.DomainError
}

func (RefreshTokenActive) validateRefreshTokenResult() {}

func (RefreshTokenInactive) validateRefreshTokenResult() {}

func (ValidateRefreshTokenRejected) validateRefreshTokenResult() {}

type ConsumeRefreshTokenResult interface {
	consumeRefreshTokenResult()
}

type RefreshTokenConsumed struct {
	Subject Subject
	Family  core.RefreshTokenID
}

type RefreshTokenNotConsumed struct{}

// RefreshTokenReuseDetected reports that a refresh token that was already
// consumed or revoked was presented again. This signals possible token theft,
// so the store revokes every remaining active token in the same family.
type RefreshTokenReuseDetected struct{}

type ConsumeRefreshTokenRejected struct {
	Reason core.DomainError
}

func (RefreshTokenConsumed) consumeRefreshTokenResult() {}

func (RefreshTokenNotConsumed) consumeRefreshTokenResult() {}

func (RefreshTokenReuseDetected) consumeRefreshTokenResult() {}

func (ConsumeRefreshTokenRejected) consumeRefreshTokenResult() {}

type AccountTokenStoreResult interface {
	accountTokenStoreResult()
}

type AccountTokenStored struct{}

type AccountTokenStoreRejected struct {
	Reason core.DomainError
}

func (AccountTokenStored) accountTokenStoreResult() {}

func (AccountTokenStoreRejected) accountTokenStoreResult() {}

type AccountTokenConsumeResult interface {
	accountTokenConsumeResult()
}

type AccountTokenConsumed struct {
	UserID core.UserID
}

type AccountTokenNotConsumed struct{}

type AccountTokenConsumeRejected struct {
	Reason core.DomainError
}

func (AccountTokenConsumed) accountTokenConsumeResult() {}

func (AccountTokenNotConsumed) accountTokenConsumeResult() {}

func (AccountTokenConsumeRejected) accountTokenConsumeResult() {}

type AccountMutationResult interface {
	accountMutationResult()
}

type AccountMutationAccepted struct{}

type AccountMutationRejected struct {
	Reason core.DomainError
}

func (AccountMutationAccepted) accountMutationResult() {}

func (AccountMutationRejected) accountMutationResult() {}

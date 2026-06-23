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

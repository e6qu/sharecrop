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
}

type RefreshTokenNotConsumed struct{}

type ConsumeRefreshTokenRejected struct {
	Reason core.DomainError
}

func (RefreshTokenConsumed) consumeRefreshTokenResult() {}

func (RefreshTokenNotConsumed) consumeRefreshTokenResult() {}

func (ConsumeRefreshTokenRejected) consumeRefreshTokenResult() {}

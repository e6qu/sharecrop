package auth

import (
	"net/mail"
	"strings"

	"github.com/e6qu/sharecrop/internal/core"
)

type EmailAddress struct {
	value string
}

type PasswordSecret struct {
	value string
}

type EmailAddressResult interface {
	emailAddressResult()
}

type EmailAddressAccepted struct {
	Value EmailAddress
}

type EmailAddressRejected struct {
	Reason core.DomainError
}

func (EmailAddressAccepted) emailAddressResult() {}

func (EmailAddressRejected) emailAddressResult() {}

func NewEmailAddress(raw string) EmailAddressResult {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return EmailAddressRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "email address is required")}
	}

	parsed, err := mail.ParseAddress(trimmed)
	if err != nil || parsed.Address != trimmed {
		return EmailAddressRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "email address is invalid")}
	}

	return EmailAddressAccepted{Value: EmailAddress{value: strings.ToLower(trimmed)}}
}

func (email EmailAddress) String() string {
	return email.value
}

type PasswordSecretResult interface {
	passwordSecretResult()
}

type PasswordSecretAccepted struct {
	Value PasswordSecret
}

type PasswordSecretRejected struct {
	Reason core.DomainError
}

func (PasswordSecretAccepted) passwordSecretResult() {}

func (PasswordSecretRejected) passwordSecretResult() {}

func NewPasswordSecret(raw string) PasswordSecretResult {
	if len(raw) < 12 {
		return PasswordSecretRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "password must contain at least 12 bytes")}
	}

	return PasswordSecretAccepted{Value: PasswordSecret{value: raw}}
}

func (secret PasswordSecret) String() string {
	return secret.value
}

// DisplayName is a user's required human-readable name, shown wherever the
// product names an actor or counterparty (task creators, submitters,
// reservation holders, comment authors, notification actors).
type DisplayName struct {
	value string
}

// displayNameMaxLength bounds a display name in bytes; the derivation
// truncates to it and the constructor rejects beyond it.
const displayNameMaxLength = 120

// fallbackDisplayName names a user whose email local part is empty after
// trimming (derivation never produces an empty name).
const fallbackDisplayName = "member"

type DisplayNameResult interface {
	displayNameResult()
}

type DisplayNameAccepted struct {
	Value DisplayName
}

type DisplayNameRejected struct {
	Reason core.DomainError
}

func (DisplayNameAccepted) displayNameResult() {}

func (DisplayNameRejected) displayNameResult() {}

func NewDisplayName(raw string) DisplayNameResult {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return DisplayNameRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "display name is required")}
	}
	if len(trimmed) > displayNameMaxLength {
		return DisplayNameRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "display name is too long")}
	}
	return DisplayNameAccepted{Value: DisplayName{value: trimmed}}
}

// DeriveDisplayNameFromEmail builds the default display name from an email
// address: the local part before '@', trimmed, truncated to the length bound,
// with a fixed fallback when nothing remains. It is total - a validated email
// always yields a valid display name.
func DeriveDisplayNameFromEmail(email EmailAddress) DisplayName {
	local := email.String()
	if at := strings.Index(local, "@"); at >= 0 {
		local = local[:at]
	}
	local = strings.TrimSpace(local)
	if len(local) > displayNameMaxLength {
		local = local[:displayNameMaxLength]
	}
	local = strings.TrimSpace(local)
	if local == "" {
		return DisplayName{value: fallbackDisplayName}
	}
	return DisplayName{value: local}
}

func (name DisplayName) String() string {
	return name.value
}

// DisplayNameChoice says how a new account's display name is set: provided
// explicitly by the caller, or derived from the email address.
type DisplayNameChoice interface {
	displayNameChoice()
}

type DeriveDisplayName struct{}

type ProvidedDisplayName struct {
	Value DisplayName
}

func (DeriveDisplayName) displayNameChoice() {}

func (ProvidedDisplayName) displayNameChoice() {}

// EmailVerificationState is the explicit email-verification lifecycle of an
// account. The 100-credit signup grant is written when the account first
// becomes verified, never at registration, so an unverified account keeps a
// zero balance (a sybil gate). email_verified_at stays as the event-time
// fact; this state is the flag.
type EmailVerificationState struct {
	value string
}

var (
	EmailUnverified = EmailVerificationState{value: "unverified"}
	EmailVerified   = EmailVerificationState{value: "verified"}
)

type EmailVerificationStateResult interface {
	emailVerificationStateResult()
}

type EmailVerificationStateAccepted struct {
	Value EmailVerificationState
}

type EmailVerificationStateRejected struct {
	Reason core.DomainError
}

func (EmailVerificationStateAccepted) emailVerificationStateResult() {}

func (EmailVerificationStateRejected) emailVerificationStateResult() {}

func ParseEmailVerificationState(raw string) EmailVerificationStateResult {
	switch raw {
	case EmailUnverified.value:
		return EmailVerificationStateAccepted{Value: EmailUnverified}
	case EmailVerified.value:
		return EmailVerificationStateAccepted{Value: EmailVerified}
	default:
		return EmailVerificationStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "email verification state is invalid")}
	}
}

func (state EmailVerificationState) String() string {
	return state.value
}

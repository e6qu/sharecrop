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

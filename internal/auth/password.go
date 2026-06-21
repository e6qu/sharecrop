package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"crypto/pbkdf2"

	"github.com/e6qu/sharecrop/internal/core"
)

const passwordSaltBytes = 32
const passwordKeyBytes = 32
const passwordIterations = 210_000

type PasswordHash struct {
	value string
}

type PasswordHashResult interface {
	passwordHashResult()
}

type PasswordHashCreated struct {
	Value PasswordHash
}

type PasswordHashRejected struct {
	Reason core.DomainError
}

func (PasswordHashCreated) passwordHashResult() {}

func (PasswordHashRejected) passwordHashResult() {}

func HashPassword(secret PasswordSecret) PasswordHashResult {
	salt := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return PasswordHashRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "password salt generation failed")}
	}

	key, err := pbkdf2.Key(sha256.New, secret.String(), salt, passwordIterations, passwordKeyBytes)
	if err != nil {
		return PasswordHashRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "password hash derivation failed")}
	}

	value := fmt.Sprintf(
		"pbkdf2-sha256$%d$%s$%s",
		passwordIterations,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
	return PasswordHashCreated{Value: PasswordHash{value: value}}
}

func ParsePasswordHash(raw string) PasswordHashResult {
	parts := strings.Split(raw, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return PasswordHashRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "password hash format is invalid")}
	}

	if _, err := strconv.Atoi(parts[1]); err != nil {
		return PasswordHashRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "password hash iteration count is invalid")}
	}

	if _, err := base64.RawStdEncoding.DecodeString(parts[2]); err != nil {
		return PasswordHashRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "password hash salt is invalid")}
	}

	if _, err := base64.RawStdEncoding.DecodeString(parts[3]); err != nil {
		return PasswordHashRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "password hash key is invalid")}
	}

	return PasswordHashCreated{Value: PasswordHash{value: raw}}
}

func VerifyPassword(secret PasswordSecret, stored PasswordHash) PasswordVerification {
	parts := strings.Split(stored.value, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return PasswordRejected{}
	}

	iterations, err := strconv.Atoi(parts[1])
	if err != nil {
		return PasswordRejected{}
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return PasswordRejected{}
	}

	expected, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return PasswordRejected{}
	}

	actual, err := pbkdf2.Key(sha256.New, secret.String(), salt, iterations, len(expected))
	if err != nil {
		return PasswordRejected{}
	}

	if subtle.ConstantTimeCompare(actual, expected) == 1 {
		return PasswordAccepted{}
	}

	return PasswordRejected{}
}

func (hash PasswordHash) String() string {
	return hash.value
}

type PasswordVerification interface {
	passwordVerification()
}

type PasswordAccepted struct{}

type PasswordRejected struct{}

func (PasswordAccepted) passwordVerification() {}

func (PasswordRejected) passwordVerification() {}

package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"

	"github.com/e6qu/sharecrop/internal/core"
)

// Argon2id parameters follow the OWASP Password Storage Cheat Sheet's
// recommended configuration (m = 19 MiB, t = 2, p = 1). Argon2id is OWASP's
// first-choice password hashing algorithm; internal/auth is the boundary
// package that owns this dependency (see PLAN.md).
const (
	passwordSaltBytes   = 16
	passwordKeyBytes    = 32
	passwordMemoryKiB   = 19456
	passwordTimeCost    = 2
	passwordParallelism = 1
)

const passwordHashScheme = "argon2id"

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

	key := argon2.IDKey([]byte(secret.String()), salt, passwordTimeCost, passwordMemoryKiB, passwordParallelism, passwordKeyBytes)

	value := fmt.Sprintf(
		"%s$%d$%d$%d$%s$%s",
		passwordHashScheme,
		passwordMemoryKiB,
		passwordTimeCost,
		passwordParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
	return PasswordHashCreated{Value: PasswordHash{value: value}}
}

// parsedHash is the decoded shape of a stored Argon2id hash string.
type parsedHash struct {
	memoryKiB   uint32
	timeCost    uint32
	parallelism uint8
	salt        []byte
	key         []byte
}

func parseHashString(raw string) (parsedHash, bool) {
	parts := strings.Split(raw, "$")
	if len(parts) != 6 || parts[0] != passwordHashScheme {
		return parsedHash{}, false
	}

	memory, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return parsedHash{}, false
	}
	timeCost, err := strconv.ParseUint(parts[2], 10, 32)
	if err != nil {
		return parsedHash{}, false
	}
	parallelism, err := strconv.ParseUint(parts[3], 10, 8)
	if err != nil {
		return parsedHash{}, false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return parsedHash{}, false
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return parsedHash{}, false
	}

	return parsedHash{
		memoryKiB:   uint32(memory),
		timeCost:    uint32(timeCost),
		parallelism: uint8(parallelism),
		salt:        salt,
		key:         key,
	}, true
}

func ParsePasswordHash(raw string) PasswordHashResult {
	if _, ok := parseHashString(raw); !ok {
		return PasswordHashRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "password hash format is invalid")}
	}
	return PasswordHashCreated{Value: PasswordHash{value: raw}}
}

func VerifyPassword(secret PasswordSecret, stored PasswordHash) PasswordVerification {
	parsed, ok := parseHashString(stored.value)
	if !ok {
		return PasswordRejected{}
	}

	actual := argon2.IDKey([]byte(secret.String()), parsed.salt, parsed.timeCost, parsed.memoryKiB, parsed.parallelism, uint32(len(parsed.key)))

	if subtle.ConstantTimeCompare(actual, parsed.key) == 1 {
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

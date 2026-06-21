package task

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"github.com/e6qu/sharecrop/internal/core"
)

type CapabilityTokenPlain struct {
	value string
}

type CapabilityTokenHash struct {
	value string
}

type CapabilityTokenPlainResult interface {
	capabilityTokenPlainResult()
}

type CapabilityTokenPlainAccepted struct {
	Value CapabilityTokenPlain
}

type CapabilityTokenPlainRejected struct {
	Reason core.DomainError
}

func (CapabilityTokenPlainAccepted) capabilityTokenPlainResult() {}

func (CapabilityTokenPlainRejected) capabilityTokenPlainResult() {}

func NewCapabilityTokenPlain() CapabilityTokenPlainResult {
	bytes := make([]byte, 32)
	readCount, err := rand.Read(bytes)
	if err != nil {
		return CapabilityTokenPlainRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "generate task capability token failed")}
	}
	if readCount != len(bytes) {
		return CapabilityTokenPlainRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "generate task capability token was short")}
	}
	return CapabilityTokenPlainAccepted{Value: CapabilityTokenPlain{value: base64.RawURLEncoding.EncodeToString(bytes)}}
}

func ParseCapabilityTokenPlain(raw string) CapabilityTokenPlainResult {
	if raw == "" {
		return CapabilityTokenPlainRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task capability token is required")}
	}
	if _, err := base64.RawURLEncoding.DecodeString(raw); err != nil {
		return CapabilityTokenPlainRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task capability token is invalid")}
	}
	return CapabilityTokenPlainAccepted{Value: CapabilityTokenPlain{value: raw}}
}

func (token CapabilityTokenPlain) String() string {
	return token.value
}

func (token CapabilityTokenPlain) Hash() CapabilityTokenHash {
	digest := sha256.Sum256([]byte(token.value))
	return CapabilityTokenHash{value: hex.EncodeToString(digest[:])}
}

func (hash CapabilityTokenHash) String() string {
	return hash.value
}

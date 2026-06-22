package submission

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/e6qu/sharecrop/internal/core"
)

type ResponseSource struct {
	value string
}

type ResponseSourceResult interface {
	responseSourceResult()
}

type ResponseSourceAccepted struct {
	Value ResponseSource
}

type ResponseSourceRejected struct {
	Reason core.DomainError
}

func (ResponseSourceAccepted) responseSourceResult() {}

func (ResponseSourceRejected) responseSourceResult() {}

func NewResponseSource(raw string) ResponseSourceResult {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ResponseSourceRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "submission response JSON is required")}
	}
	return ResponseSourceAccepted{Value: ResponseSource{value: trimmed}}
}

func (source ResponseSource) String() string {
	return source.value
}

type ReceiptTokenPlain struct {
	value string
}

type ReceiptTokenHash struct {
	value string
}

type ReceiptTokenPlainResult interface {
	receiptTokenPlainResult()
}

type ReceiptTokenPlainAccepted struct {
	Value ReceiptTokenPlain
}

type ReceiptTokenPlainRejected struct {
	Reason core.DomainError
}

func (ReceiptTokenPlainAccepted) receiptTokenPlainResult() {}

func (ReceiptTokenPlainRejected) receiptTokenPlainResult() {}

func NewReceiptTokenPlain() ReceiptTokenPlainResult {
	bytes := make([]byte, 32)
	readCount, err := rand.Read(bytes)
	if err != nil {
		return ReceiptTokenPlainRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "generate submission receipt token failed")}
	}
	if readCount != len(bytes) {
		return ReceiptTokenPlainRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "generate submission receipt token was short")}
	}
	return ReceiptTokenPlainAccepted{Value: ReceiptTokenPlain{value: base64.RawURLEncoding.EncodeToString(bytes)}}
}

func ParseReceiptTokenPlain(raw string) ReceiptTokenPlainResult {
	if raw == "" {
		return ReceiptTokenPlainRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "submission receipt token is required")}
	}
	if _, err := base64.RawURLEncoding.DecodeString(raw); err != nil {
		return ReceiptTokenPlainRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "submission receipt token is invalid")}
	}
	return ReceiptTokenPlainAccepted{Value: ReceiptTokenPlain{value: raw}}
}

func (token ReceiptTokenPlain) String() string {
	return token.value
}

func (token ReceiptTokenPlain) Hash() ReceiptTokenHash {
	digest := sha256.Sum256([]byte(token.value))
	return ReceiptTokenHash{value: hex.EncodeToString(digest[:])}
}

func (hash ReceiptTokenHash) String() string {
	return hash.value
}

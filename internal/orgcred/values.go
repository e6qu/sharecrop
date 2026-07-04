package orgcred

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/e6qu/sharecrop/internal/core"
)

// secretPrefix distinguishes an org-wide credential's secret from an
// agent.Credential's ("scrop_agent_") by prefix alone, so a bearer token can
// be dispatched to the right verifier without trying both.
const secretPrefix = "scrop_org_"

// SecretPlain is the opaque org credential shown once at creation.
type SecretPlain struct {
	value string
}

type SecretHash struct {
	value string
}

type SecretPlainResult interface {
	secretPlainResult()
}

type SecretPlainAccepted struct {
	Value SecretPlain
}

type SecretPlainRejected struct {
	Reason core.DomainError
}

func (SecretPlainAccepted) secretPlainResult() {}

func (SecretPlainRejected) secretPlainResult() {}

func NewSecretPlain() SecretPlainResult {
	bytes := make([]byte, 32)
	readCount, err := rand.Read(bytes)
	if err != nil {
		return SecretPlainRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "generate org credential failed")}
	}
	if readCount != len(bytes) {
		return SecretPlainRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "generate org credential was short")}
	}
	return SecretPlainAccepted{Value: SecretPlain{value: secretPrefix + base64.RawURLEncoding.EncodeToString(bytes)}}
}

// HasSecretPrefix reports whether a bearer token looks like an org
// credential secret, letting a resolver dispatch without first trying (and
// failing) other verifiers.
func HasSecretPrefix(raw string) bool {
	return strings.HasPrefix(raw, secretPrefix)
}

func ParseSecretPlain(raw string) SecretPlainResult {
	if !strings.HasPrefix(raw, secretPrefix) {
		return SecretPlainRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "org credential is invalid")}
	}
	encoded := strings.TrimPrefix(raw, secretPrefix)
	if _, err := base64.RawURLEncoding.DecodeString(encoded); err != nil {
		return SecretPlainRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "org credential is invalid")}
	}
	return SecretPlainAccepted{Value: SecretPlain{value: raw}}
}

func (secret SecretPlain) String() string {
	return secret.value
}

func (secret SecretPlain) Hash() SecretHash {
	digest := sha256.Sum256([]byte(secret.value))
	return SecretHash{value: hex.EncodeToString(digest[:])}
}

func (hash SecretHash) String() string {
	return hash.value
}

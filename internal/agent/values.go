package agent

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/e6qu/sharecrop/internal/core"
)

const secretPrefix = "scrop_agent_"

// Scope is a typed capability granted to an agent credential.
type Scope struct {
	value string
}

var (
	ScopeTasksRead           = Scope{value: "tasks_read"}
	ScopeTasksWrite          = Scope{value: "tasks_write"}
	ScopeSubmissionsWrite    = Scope{value: "submissions_write"}
	ScopeSubmissionsRead     = Scope{value: "submissions_read"}
	ScopeSubmissionsReview   = Scope{value: "submissions_review"}
	ScopeOrgRead             = Scope{value: "org_read"}
	ScopeOrgManage           = Scope{value: "org_manage"}
	ScopeCollectiblesRead    = Scope{value: "collectibles_read"}
	ScopeCollectiblesManage  = Scope{value: "collectibles_manage"}
	ScopeNotificationsRead   = Scope{value: "notifications_read"}
	ScopeNotificationsManage = Scope{value: "notifications_manage"}
	ScopeUsersRead           = Scope{value: "users_read"}
	ScopeLedgerRead          = Scope{value: "ledger_read"}
	ScopeModerationRead      = Scope{value: "moderation_read"}
	ScopeModerationManage    = Scope{value: "moderation_manage"}
	ScopePrivacyRead         = Scope{value: "privacy_read"}
	ScopePrivacyManage       = Scope{value: "privacy_manage"}
	ScopePlatformAdmin       = Scope{value: "platform_admin"}
	ScopeCredentialsManage   = Scope{value: "credentials_manage"}
)

// allScopes lists every legal scope, used by ParseScope so adding a new
// scope means adding one entry here rather than a new switch case.
var allScopes = []Scope{
	ScopeTasksRead, ScopeTasksWrite, ScopeSubmissionsWrite, ScopeSubmissionsRead, ScopeSubmissionsReview,
	ScopeOrgRead, ScopeOrgManage,
	ScopeCollectiblesRead, ScopeCollectiblesManage,
	ScopeNotificationsRead, ScopeNotificationsManage,
	ScopeUsersRead,
	ScopeLedgerRead,
	ScopeModerationRead, ScopeModerationManage,
	ScopePrivacyRead, ScopePrivacyManage,
	ScopePlatformAdmin,
	ScopeCredentialsManage,
}

type ScopeResult interface {
	scopeResult()
}

type ScopeAccepted struct {
	Value Scope
}

type ScopeRejected struct {
	Reason core.DomainError
}

func (ScopeAccepted) scopeResult() {}

func (ScopeRejected) scopeResult() {}

func ParseScope(raw string) ScopeResult {
	for _, scope := range allScopes {
		if scope.value == raw {
			return ScopeAccepted{Value: scope}
		}
	}
	return ScopeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "agent scope is invalid")}
}

func (scope Scope) String() string {
	return scope.value
}

// State is the lifecycle of an agent credential.
type State struct {
	value string
}

var (
	StateActive  = State{value: "active"}
	StateRevoked = State{value: "revoked"}
)

type StateResult interface {
	stateResult()
}

type StateAccepted struct {
	Value State
}

type StateRejected struct {
	Reason core.DomainError
}

func (StateAccepted) stateResult() {}

func (StateRejected) stateResult() {}

func ParseState(raw string) StateResult {
	switch raw {
	case StateActive.value:
		return StateAccepted{Value: StateActive}
	case StateRevoked.value:
		return StateAccepted{Value: StateRevoked}
	default:
		return StateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "agent credential state is invalid")}
	}
}

func (state State) String() string {
	return state.value
}

// Label is a human-readable name for an agent credential.
type Label struct {
	value string
}

type LabelResult interface {
	labelResult()
}

type LabelAccepted struct {
	Value Label
}

type LabelRejected struct {
	Reason core.DomainError
}

func (LabelAccepted) labelResult() {}

func (LabelRejected) labelResult() {}

func NewLabel(raw string) LabelResult {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return LabelRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "agent credential label is required")}
	}
	if len(trimmed) > 120 {
		return LabelRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "agent credential label is too long")}
	}
	return LabelAccepted{Value: Label{value: trimmed}}
}

func (label Label) String() string {
	return label.value
}

// SecretPlain is the opaque agent credential shown once at creation.
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
		return SecretPlainRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "generate agent credential failed")}
	}
	if readCount != len(bytes) {
		return SecretPlainRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "generate agent credential was short")}
	}
	return SecretPlainAccepted{Value: SecretPlain{value: secretPrefix + base64.RawURLEncoding.EncodeToString(bytes)}}
}

func ParseSecretPlain(raw string) SecretPlainResult {
	if !strings.HasPrefix(raw, secretPrefix) {
		return SecretPlainRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "agent credential is invalid")}
	}
	encoded := strings.TrimPrefix(raw, secretPrefix)
	if _, err := base64.RawURLEncoding.DecodeString(encoded); err != nil {
		return SecretPlainRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "agent credential is invalid")}
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

// SecretHashFromString reconstructs a SecretHash from its stored string form,
// for storage adapters (including the WASI store bridge) that carry it as a
// string. It does not hash a plaintext secret (that is SecretPlain.Hash).
func SecretHashFromString(raw string) SecretHash {
	return SecretHash{value: raw}
}

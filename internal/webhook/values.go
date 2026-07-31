// Package webhook is the guest-safe domain package for outbound webhook
// subscriptions and their delivery read model. Subscription management (create,
// list, revoke, list deliveries) runs everywhere the app mux runs, including
// the WASI guest, through the bridged Store. The host-only delivery pump and
// dispatcher live in internal/webhookdispatch over struct methods on the
// concrete db store, never through this package's Store interface.
package webhook

import (
	"crypto/rand"
	"encoding/base64"
	"net/url"
	"strings"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
)

// secretPrefix distinguishes a webhook signing secret from the other opaque
// Sharecrop secrets ("scrop_agent_", "scrop_org_") by prefix alone.
const secretPrefix = "scrop_whsec_"

// EndpointURL is a validated webhook receiver address: an absolute https URL
// with a non-empty host and no userinfo. The constructor checks form only;
// the dispatcher's dial policy separately rejects non-public addresses at
// delivery time.
type EndpointURL struct {
	value string
}

type EndpointURLResult interface {
	endpointURLResult()
}

type EndpointURLAccepted struct {
	Value EndpointURL
}

type EndpointURLRejected struct {
	Reason core.DomainError
}

func (EndpointURLAccepted) endpointURLResult() {}

func (EndpointURLRejected) endpointURLResult() {}

func NewEndpointURL(raw string) EndpointURLResult {
	parsed, err := url.Parse(raw)
	if err != nil {
		return EndpointURLRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "webhook url is not a valid URL")}
	}
	if parsed.Scheme != "https" {
		return EndpointURLRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "webhook url must use https")}
	}
	if parsed.Host == "" || parsed.Hostname() == "" {
		return EndpointURLRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "webhook url must have a host")}
	}
	if parsed.User != nil {
		return EndpointURLRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "webhook url must not carry userinfo")}
	}
	return EndpointURLAccepted{Value: EndpointURL{value: parsed.String()}}
}

func (endpoint EndpointURL) String() string {
	return endpoint.value
}

// Secret is the per-subscription HMAC signing secret, shown once at creation.
//
// Unlike agent and org credential secrets, a webhook secret is stored AS
// WRITTEN, never hashed: the dispatcher must compute an HMAC-SHA256 over each
// delivery body with the original secret bytes, which is impossible from a
// hash. The Store documents the same requirement.
type Secret struct {
	value string
}

type SecretResult interface {
	secretResult()
}

type SecretAccepted struct {
	Value Secret
}

type SecretRejected struct {
	Reason core.DomainError
}

func (SecretAccepted) secretResult() {}

func (SecretRejected) secretResult() {}

// NewSecret generates a fresh signing secret: the scrop_whsec_ prefix plus 43
// base64url characters of crypto/rand entropy (32 bytes, unpadded).
func NewSecret() SecretResult {
	bytes := make([]byte, 32)
	readCount, err := rand.Read(bytes)
	if err != nil {
		return SecretRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "generate webhook secret failed")}
	}
	if readCount != len(bytes) {
		return SecretRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "generate webhook secret was short")}
	}
	return SecretAccepted{Value: Secret{value: secretPrefix + base64.RawURLEncoding.EncodeToString(bytes)}}
}

// HasSecretPrefix reports whether a raw string looks like a webhook signing
// secret.
func HasSecretPrefix(raw string) bool {
	return strings.HasPrefix(raw, secretPrefix)
}

// ParseSecret reconstructs a Secret from its stored string form, for storage
// adapters (including the WASI store bridge) that carry it as a string.
func ParseSecret(raw string) SecretResult {
	if !strings.HasPrefix(raw, secretPrefix) {
		return SecretRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "webhook secret is invalid")}
	}
	encoded := strings.TrimPrefix(raw, secretPrefix)
	if _, err := base64.RawURLEncoding.DecodeString(encoded); err != nil {
		return SecretRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "webhook secret is invalid")}
	}
	return SecretAccepted{Value: Secret{value: raw}}
}

func (secret Secret) String() string {
	return secret.value
}

// State is the lifecycle of a webhook subscription.
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
		return StateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "webhook subscription state is invalid")}
	}
}

func (state State) String() string {
	return state.value
}

// KindFilter is a validated, deduplicated, non-empty set of domain event
// kinds a subscription listens for.
type KindFilter struct {
	kinds []event.Kind
}

type KindFilterResult interface {
	kindFilterResult()
}

type KindFilterAccepted struct {
	Value KindFilter
}

type KindFilterRejected struct {
	Reason core.DomainError
}

func (KindFilterAccepted) kindFilterResult() {}

func (KindFilterRejected) kindFilterResult() {}

func NewKindFilter(kinds []event.Kind) KindFilterResult {
	seen := map[string]bool{}
	deduplicated := make([]event.Kind, 0, len(kinds))
	for _, kind := range kinds {
		if seen[kind.String()] {
			continue
		}
		seen[kind.String()] = true
		deduplicated = append(deduplicated, kind)
	}
	if len(deduplicated) == 0 {
		return KindFilterRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "webhook subscription requires at least one event kind")}
	}
	return KindFilterAccepted{Value: KindFilter{kinds: deduplicated}}
}

func (filter KindFilter) Values() []event.Kind {
	values := make([]event.Kind, len(filter.kinds))
	copy(values, filter.kinds)
	return values
}

// Contains reports whether the filter includes the given kind. The report is
// a plain boundary predicate, not domain state.
func (filter KindFilter) Contains(kind event.Kind) bool {
	for _, candidate := range filter.kinds {
		if candidate == kind {
			return true
		}
	}
	return false
}

// Owner is who a subscription belongs to: a user (their personal feed) or an
// organization (that organization's events).
type Owner interface {
	webhookOwner()
}

type OwnerUser struct {
	ID core.UserID
}

type OwnerOrganization struct {
	ID core.OrganizationID
}

func (OwnerUser) webhookOwner() {}

func (OwnerOrganization) webhookOwner() {}

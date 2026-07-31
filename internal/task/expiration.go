package task

import (
	"time"

	"github.com/e6qu/sharecrop/internal/core"
)

// ExpirationPolicy is a task's expiration behavior: NoExpiration means the
// task stays open until an explicit transition; ExpiresAt names the instant
// after which the lifecycle runner moves an open task to the expired state
// and refunds its escrow.
type ExpirationPolicy interface {
	expirationPolicy()
}

type NoExpiration struct{}

type ExpiresAt struct {
	Instant time.Time
}

func (NoExpiration) expirationPolicy() {}

func (ExpiresAt) expirationPolicy() {}

type ExpirationPolicyResult interface {
	expirationPolicyResult()
}

type ExpirationPolicyAccepted struct {
	Value ExpirationPolicy
}

type ExpirationPolicyRejected struct {
	Reason core.DomainError
}

func (ExpirationPolicyAccepted) expirationPolicyResult() {}

func (ExpirationPolicyRejected) expirationPolicyResult() {}

// NewExpiresAt validates a task-creation expiration instant: it must lie in
// the future relative to now. Stored tasks loaded later may legitimately
// carry past instants (that is what the expiry sweep looks for), so this
// constructor is for the creation boundary only.
func NewExpiresAt(instant time.Time, now time.Time) ExpirationPolicyResult {
	if !instant.After(now) {
		return ExpirationPolicyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task expiration must be in the future")}
	}
	return ExpirationPolicyAccepted{Value: ExpiresAt{Instant: instant.UTC()}}
}

// ParseExpirationPolicy parses the boundary form used by REST and MCP: an
// empty string means no expiration; anything else must be a valid RFC3339
// instant in the future.
func ParseExpirationPolicy(raw string, now time.Time) ExpirationPolicyResult {
	if raw == "" {
		return ExpirationPolicyAccepted{Value: NoExpiration{}}
	}
	instant, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return ExpirationPolicyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "expires_at must be an RFC3339 timestamp")}
	}
	return NewExpiresAt(instant, now)
}

// ExpirationInstantString renders the policy back to its boundary form: the
// RFC3339 instant, or the empty string for NoExpiration.
func ExpirationInstantString(policy ExpirationPolicy) string {
	if typed, matched := policy.(ExpiresAt); matched {
		return typed.Instant.UTC().Format(time.RFC3339)
	}
	return ""
}

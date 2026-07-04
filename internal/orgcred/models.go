package orgcred

import (
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
)

// Credential is an opaque, scoped access token that acts as an organization
// itself, not through an individual member. Label/Scopes/State reuse
// agent.Label/agent.ScopeSet/agent.State directly since those carry no
// user-specific semantics — an org credential and a user credential share
// the same scope vocabulary and lifecycle states.
type Credential struct {
	ID             core.OrgCredentialID
	OrganizationID core.OrganizationID
	Label          agent.Label
	Scopes         agent.ScopeSet
	State          agent.State
	ExpiresAt      *time.Time
}

// IsExpired reports whether the credential's expiration, if set, is in the past.
func (credential Credential) IsExpired(now time.Time) bool {
	return credential.ExpiresAt != nil && credential.ExpiresAt.Before(now)
}

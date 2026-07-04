// Package orgactor answers one narrow question: does this actor's
// organization-wide credential match a given organization? Both
// internal/authz (task/series/submission ownership) and internal/org (its
// own membership/team endpoints) need this exact same actor-kind dispatch,
// but internal/authz already imports internal/org for org.Permission/
// org.PermissionChecker, so internal/org cannot import internal/authz back
// without a cycle. Extracting the shared check here — a leaf package with
// no dependency on either — lets both call the same logic instead of each
// hand-rolling its own copy.
package orgactor

import (
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

// Result is how an actor relates to a given organization.
type Result int

const (
	// NotApplicable means actor isn't an organization-wide credential at
	// all (e.g. a UserSubject or GuestSubject); the caller should fall
	// through to its own per-user check.
	NotApplicable Result = iota
	// Match means actor is that exact organization's own credential —
	// full parity with an org-admin member, no per-user check needed.
	Match
	// Mismatch means actor is an organization-wide credential, but for a
	// different organization; access is denied outright, since an
	// org-wide credential has no per-user permission fallback.
	Mismatch
)

// Check reports how actor relates to organizationID.
func Check(actor auth.Subject, organizationID core.OrganizationID) Result {
	orgActor, isOrg := actor.(auth.OrgSubject)
	if !isOrg {
		return NotApplicable
	}
	if orgActor.ID == organizationID {
		return Match
	}
	return Mismatch
}

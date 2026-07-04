// Package authz centralizes the "does this actor have organization-level
// access" check that task, series, and submission authorization each need:
// an org-wide credential (auth.OrgSubject) gets unconditional access to its
// own organization's resources (full parity with an org-admin member, no
// per-member permission lookup — the token is the org), while a user
// delegates to the existing per-member permission check. It does not
// replace org.CheckPermission, which stays the single source of truth for
// per-member role/permission logic; authz only decides which actor kinds
// may reach it and how an org-wide credential bypasses it.
package authz

import (
	"context"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/orgactor"
)

// Decision is the outcome of an authorization check.
type Decision interface {
	decision()
}

type Granted struct{}

type Denied struct {
	Reason core.DomainError
}

func (Granted) decision() {}

func (Denied) decision() {}

// OrganizationPermissionChecker is the narrow slice of org.Service that
// RequireOrganizationAccess needs; task.Service and submission.Service's own
// OrganizationPermissions interfaces already satisfy this structurally.
type OrganizationPermissionChecker interface {
	CheckOrganizationPermission(context.Context, core.OrganizationID, core.UserID, org.Permission) org.PermissionCheck
}

// RequireOrganizationAccess grants access to a resource owned by
// organizationID either because actor is that exact organization's own
// credential (auth.OrgSubject — full parity with an org-admin member,
// checked with no per-member permission lookup since the token is the org
// itself), or because actor is a user who holds permission via checker. Any
// other actor kind, or an org-subject/actor-kind mismatch, is Denied with
// deniedReason under deniedCode (callers vary this — e.g. task callers use
// core.ErrorCodePermissionDenied, series callers use
// core.ErrorCodeInvalidState — since the resulting DomainError code drives
// the HTTP status code, so it must match each caller's existing contract).
// A user lacking permission is Denied with the checker's own reason as-is.
func RequireOrganizationAccess(ctx context.Context, actor auth.Subject, organizationID core.OrganizationID, checker OrganizationPermissionChecker, permission org.Permission, deniedCode core.ErrorCode, deniedReason string) Decision {
	switch orgactor.Check(actor, organizationID) {
	case orgactor.Match:
		return Granted{}
	case orgactor.Mismatch:
		return Denied{Reason: core.NewDomainError(deniedCode, deniedReason)}
	}
	userActor, isUser := actor.(auth.UserSubject)
	if !isUser {
		return Denied{Reason: core.NewDomainError(deniedCode, deniedReason)}
	}
	check := checker.CheckOrganizationPermission(ctx, organizationID, userActor.ID, permission)
	if rejected, matched := check.(org.PermissionDenied); matched {
		return Denied{Reason: rejected.Reason}
	}
	return Granted{}
}

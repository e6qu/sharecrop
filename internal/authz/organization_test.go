package authz

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
)

type stubChecker struct {
	result org.PermissionCheck
}

func (checker stubChecker) CheckOrganizationPermission(context.Context, core.OrganizationID, core.UserID, org.Permission) org.PermissionCheck {
	return checker.result
}

func newOrganizationID(t *testing.T) core.OrganizationID {
	t.Helper()
	created, matched := core.NewOrganizationID().(core.OrganizationIDCreated)
	if !matched {
		t.Fatalf("organization id rejected")
	}
	return created.Value
}

func newUserID(t *testing.T) core.UserID {
	t.Helper()
	created, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("user id rejected")
	}
	return created.Value
}

func TestRequireOrganizationAccessGrantsMatchingOrgSubject(t *testing.T) {
	organizationID := newOrganizationID(t)
	actor := auth.OrgSubject{ID: organizationID}
	checker := stubChecker{result: org.PermissionDenied{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "should not be consulted")}}

	decision := RequireOrganizationAccess(context.Background(), actor, organizationID, checker, org.PermissionCreateOrganizationTask, core.ErrorCodePermissionDenied, "denied")
	if _, granted := decision.(Granted); !granted {
		t.Fatalf("matching org subject was denied: %#v", decision)
	}
}

func TestRequireOrganizationAccessDeniesMismatchedOrgSubjectWithoutConsultingChecker(t *testing.T) {
	actor := auth.OrgSubject{ID: newOrganizationID(t)}
	otherOrganizationID := newOrganizationID(t)
	checker := stubChecker{result: org.PermissionGranted{}}

	decision := RequireOrganizationAccess(context.Background(), actor, otherOrganizationID, checker, org.PermissionCreateOrganizationTask, core.ErrorCodePermissionDenied, "denied")
	denied, isDenied := decision.(Denied)
	if !isDenied {
		t.Fatalf("mismatched org subject was granted: %#v", decision)
	}
	if denied.Reason.Description() != "denied" {
		t.Fatalf("reason = %q, want %q", denied.Reason.Description(), "denied")
	}
}

// TestRequireOrganizationAccessUsesCallerSuppliedErrorCode guards the reason
// this function takes an explicit deniedCode rather than hardcoding one:
// different callers map denial to different HTTP statuses (task callers use
// PermissionDenied -> 403, series callers use InvalidState -> 409), and a
// hardcoded code would silently change one of those statuses.
func TestRequireOrganizationAccessUsesCallerSuppliedErrorCode(t *testing.T) {
	actor := auth.OrgSubject{ID: newOrganizationID(t)}
	otherOrganizationID := newOrganizationID(t)

	decision := RequireOrganizationAccess(context.Background(), actor, otherOrganizationID, stubChecker{}, org.PermissionCreateOrganizationTask, core.ErrorCodeInvalidState, "denied")
	denied, isDenied := decision.(Denied)
	if !isDenied {
		t.Fatalf("mismatched org subject was granted: %#v", decision)
	}
	if denied.Reason.Code() != core.ErrorCodeInvalidState {
		t.Fatalf("code = %v, want ErrorCodeInvalidState", denied.Reason.Code())
	}
}

func TestRequireOrganizationAccessDelegatesToCheckerForUserSubject(t *testing.T) {
	organizationID := newOrganizationID(t)
	actor := auth.UserSubject{ID: newUserID(t)}

	granted := RequireOrganizationAccess(context.Background(), actor, organizationID, stubChecker{result: org.PermissionGranted{}}, org.PermissionCreateOrganizationTask, core.ErrorCodePermissionDenied, "denied")
	if _, isGranted := granted.(Granted); !isGranted {
		t.Fatalf("granted checker result was denied: %#v", granted)
	}

	denied := RequireOrganizationAccess(context.Background(), actor, organizationID, stubChecker{result: org.PermissionDenied{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "no role")}}, org.PermissionCreateOrganizationTask, core.ErrorCodePermissionDenied, "denied")
	rejected, isDenied := denied.(Denied)
	if !isDenied {
		t.Fatalf("denied checker result was granted: %#v", denied)
	}
	if rejected.Reason.Description() != "no role" {
		t.Fatalf("reason = %q, want the checker's own reason", rejected.Reason.Description())
	}
}

func TestRequireOrganizationAccessDeniesOtherActorKinds(t *testing.T) {
	organizationID := newOrganizationID(t)
	created, matched := core.NewGuestID().(core.GuestIDCreated)
	if !matched {
		t.Fatalf("guest id rejected")
	}
	actor := auth.GuestSubject{ID: created.Value}

	decision := RequireOrganizationAccess(context.Background(), actor, organizationID, stubChecker{result: org.PermissionGranted{}}, org.PermissionCreateOrganizationTask, core.ErrorCodePermissionDenied, "denied")
	if _, isDenied := decision.(Denied); !isDenied {
		t.Fatalf("guest subject was granted: %#v", decision)
	}
}

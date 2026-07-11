// Package orgtest holds test-support helpers for org models, shared by the org
// bridge's codec tests and the integration dual-run test so the two do not carry
// duplicate comparisons.
package orgtest

import (
	"fmt"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
)

// ---- builders shared by the codec tests and the integration dual-run test ----

// NewUserID mints a fresh user id or fails the test.
func NewUserID(t testing.TB) core.UserID {
	t.Helper()
	created, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("user id rejected")
	}
	return created.Value
}

// NewOrganizationID mints a fresh organization id or fails the test.
func NewOrganizationID(t testing.TB) core.OrganizationID {
	t.Helper()
	created, matched := core.NewOrganizationID().(core.OrganizationIDCreated)
	if !matched {
		t.Fatalf("organization id rejected")
	}
	return created.Value
}

// NewMembershipID mints a fresh organization membership id or fails the test.
func NewMembershipID(t testing.TB) core.OrganizationMembershipID {
	t.Helper()
	created, matched := core.NewOrganizationMembershipID().(core.OrganizationMembershipIDCreated)
	if !matched {
		t.Fatalf("membership id rejected")
	}
	return created.Value
}

// NewTeamID mints a fresh team id or fails the test.
func NewTeamID(t testing.TB) core.TeamID {
	t.Helper()
	created, matched := core.NewTeamID().(core.TeamIDCreated)
	if !matched {
		t.Fatalf("team id rejected")
	}
	return created.Value
}

// OrganizationName builds a validated organization name or fails the test.
func OrganizationName(t testing.TB, raw string) org.OrganizationName {
	t.Helper()
	accepted, matched := org.NewOrganizationName(raw).(org.OrganizationNameAccepted)
	if !matched {
		t.Fatalf("organization name rejected")
	}
	return accepted.Value
}

// TeamName builds a validated team name or fails the test.
func TeamName(t testing.TB, raw string) org.TeamName {
	t.Helper()
	accepted, matched := org.NewTeamName(raw).(org.TeamNameAccepted)
	if !matched {
		t.Fatalf("team name rejected")
	}
	return accepted.Value
}

// RolesKey renders a role slice as a stable comma-joined string for comparison.
func RolesKey(roles []org.Role) string {
	return rolesKey(roles)
}

// OrganizationDiff returns a description of the first field in which two
// organizations differ, or "" if they are equal.
func OrganizationDiff(got, want org.Organization) string {
	switch {
	case got.ID != want.ID:
		return fmt.Sprintf("id: %s != %s", got.ID, want.ID)
	case got.Name.String() != want.Name.String():
		return fmt.Sprintf("name: %s != %s", got.Name, want.Name)
	case got.CreatedBy != want.CreatedBy:
		return fmt.Sprintf("created_by: %s != %s", got.CreatedBy, want.CreatedBy)
	default:
		return ""
	}
}

// MemberDiff returns a description of the first field in which two organization
// members differ, or "" if they are equal.
func MemberDiff(got, want org.OrganizationMember) string {
	switch {
	case got.ID != want.ID:
		return fmt.Sprintf("id: %s != %s", got.ID, want.ID)
	case got.OrganizationID != want.OrganizationID:
		return fmt.Sprintf("organization_id: %s != %s", got.OrganizationID, want.OrganizationID)
	case got.UserID != want.UserID:
		return fmt.Sprintf("user_id: %s != %s", got.UserID, want.UserID)
	case got.Status.String() != want.Status.String():
		return fmt.Sprintf("status: %s != %s", got.Status, want.Status)
	case rolesKey(got.Roles) != rolesKey(want.Roles):
		return fmt.Sprintf("roles: %s != %s", rolesKey(got.Roles), rolesKey(want.Roles))
	default:
		return ""
	}
}

// TeamDiff returns a description of the first field in which two teams differ,
// or "" if they are equal (including the owner tagged union).
func TeamDiff(got, want org.Team) string {
	switch {
	case got.ID != want.ID:
		return fmt.Sprintf("id: %s != %s", got.ID, want.ID)
	case got.Name.String() != want.Name.String():
		return fmt.Sprintf("name: %s != %s", got.Name, want.Name)
	case got.CreatedBy != want.CreatedBy:
		return fmt.Sprintf("created_by: %s != %s", got.CreatedBy, want.CreatedBy)
	case ownerKey(got.Owner) != ownerKey(want.Owner):
		return fmt.Sprintf("owner: %s != %s", ownerKey(got.Owner), ownerKey(want.Owner))
	default:
		return ""
	}
}

func rolesKey(roles []org.Role) string {
	parts := make([]string, 0, len(roles))
	for index := range roles {
		parts = append(parts, roles[index].String())
	}
	return strings.Join(parts, ",")
}

func ownerKey(owner org.TeamOwner) string {
	switch typed := owner.(type) {
	case org.OrganizationOwnedTeam:
		return "organization:" + typed.OrganizationID.String()
	case org.UserOwnedTeam:
		return "user:" + typed.OwnerUserID.String()
	default:
		return "unknown"
	}
}

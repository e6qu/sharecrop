package orgbridge

import (
	"testing"

	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/org/orgtest"
)

func sampleMember(t *testing.T) org.OrganizationMember {
	t.Helper()
	return org.OrganizationMember{
		ID:             orgtest.NewMembershipID(t),
		OrganizationID: orgtest.NewOrganizationID(t),
		UserID:         orgtest.NewUserID(t),
		Status:         org.MembershipStatusActive,
		Roles:          []org.Role{org.RoleAdmin, org.RoleReviewer},
	}
}

func TestOrganizationRoundTrip(t *testing.T) {
	original := org.Organization{ID: orgtest.NewOrganizationID(t), Name: orgtest.OrganizationName(t, "Acme Co"), CreatedBy: orgtest.NewUserID(t)}
	restored, err := decodeOrganization(encodeOrganization(original))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if diff := orgtest.OrganizationDiff(restored, original); diff != "" {
		t.Errorf("organization mismatch: %s", diff)
	}
}

func TestMemberRoundTrip(t *testing.T) {
	original := sampleMember(t)
	restored, err := decodeMember(encodeMember(original))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if diff := orgtest.MemberDiff(restored, original); diff != "" {
		t.Errorf("member mismatch: %s", diff)
	}
}

func TestTeamRoundTripBothOwners(t *testing.T) {
	orgOwned := org.Team{
		ID:        orgtest.NewTeamID(t),
		Owner:     org.OrganizationOwnedTeam{OrganizationID: orgtest.NewOrganizationID(t)},
		Name:      orgtest.TeamName(t, "Platform"),
		CreatedBy: orgtest.NewUserID(t),
	}
	userOwned := org.Team{
		ID:        orgtest.NewTeamID(t),
		Owner:     org.UserOwnedTeam{OwnerUserID: orgtest.NewUserID(t)},
		Name:      orgtest.TeamName(t, "Solo Squad"),
		CreatedBy: orgtest.NewUserID(t),
	}
	for _, original := range []org.Team{orgOwned, userOwned} {
		restored, err := decodeTeam(encodeTeam(original))
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if diff := orgtest.TeamDiff(restored, original); diff != "" {
			t.Errorf("team mismatch: %s", diff)
		}
	}
}

func TestResultRoundTrips(t *testing.T) {
	member := sampleMember(t)

	provisioned, err := decodeProvisionMemberResult(encodeProvisionMemberResult(org.MemberProvisioned{Value: member}))
	if err != nil {
		t.Fatalf("decode provision: %v", err)
	}
	provisionedValue, matched := provisioned.(org.MemberProvisioned)
	if !matched {
		t.Fatalf("provision result = %T, want MemberProvisioned", provisioned)
	}
	if diff := orgtest.MemberDiff(provisionedValue.Value, member); diff != "" {
		t.Errorf("provisioned member mismatch: %s", diff)
	}

	rolesResult, err := decodeMemberRolesResult(encodeMemberRolesResult(org.MemberRolesFound{Roles: []org.Role{org.RoleOwner}}))
	if err != nil {
		t.Fatalf("decode member roles: %v", err)
	}
	found, matched := rolesResult.(org.MemberRolesFound)
	if !matched || orgtest.RolesKey(found.Roles) != "owner" {
		t.Errorf("member roles did not round-trip: %+v", rolesResult)
	}

	missing, err := decodeMemberRolesResult(encodeMemberRolesResult(org.MemberRolesMissing{}))
	if err != nil {
		t.Fatalf("decode missing roles: %v", err)
	}
	if _, matched := missing.(org.MemberRolesMissing); !matched {
		t.Errorf("missing roles did not round-trip: %T", missing)
	}

	members, err := decodeListMembersResult(encodeListMembersResult(org.MembersListed{Values: []org.OrganizationMember{member}}))
	if err != nil {
		t.Fatalf("decode members: %v", err)
	}
	if listed, matched := members.(org.MembersListed); !matched || len(listed.Values) != 1 {
		t.Errorf("members list did not round-trip: %+v", members)
	}
}

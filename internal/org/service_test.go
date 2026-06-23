package org

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

func TestServiceCreatesOrganizationWithOwnerMembership(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store)
	actor := testUserSubject(t)
	name := acceptedOrganizationName(t, "Sharecrop Labs")

	result := service.CreateOrganization(context.Background(), actor, name)
	created, matched := result.(OrganizationCreated)
	if !matched {
		t.Fatalf("result = %T, want OrganizationCreated", result)
	}

	rolesResult := store.FindMemberRoles(context.Background(), created.Value.ID, actor.ID)
	rolesFound, rolesMatched := rolesResult.(MemberRolesFound)
	if !rolesMatched {
		t.Fatalf("roles result = %T, want MemberRolesFound", rolesResult)
	}

	if _, granted := CheckPermission(rolesFound.Roles, PermissionManageMembers).(PermissionGranted); !granted {
		t.Fatalf("owner roles did not grant member management")
	}
}

func TestServiceDeniesProvisionWithoutPermission(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store)
	organization := testOrganizationID(t)
	actor := testUserSubject(t)
	email := acceptedEmail(t, "member@example.com")

	result := service.ProvisionMember(context.Background(), actor, organization, email, []Role{RoleMember})
	if _, matched := result.(ProvisionMemberRejected); !matched {
		t.Fatalf("result = %T, want ProvisionMemberRejected", result)
	}
}

type memoryStore struct {
	organizations map[string]Organization
	members       map[string]OrganizationMember
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		organizations: make(map[string]Organization),
		members:       make(map[string]OrganizationMember),
	}
}

func (store *memoryStore) CreateOrganization(_ context.Context, organizationID core.OrganizationID, name OrganizationName, createdBy core.UserID, membershipID core.OrganizationMembershipID) CreateOrganizationStoreResult {
	store.organizations[organizationID.String()] = Organization{ID: organizationID, Name: name, CreatedBy: createdBy}
	store.members[organizationID.String()+":"+createdBy.String()] = OrganizationMember{
		ID:             membershipID,
		OrganizationID: organizationID,
		UserID:         createdBy,
		Status:         MembershipStatusActive,
		Roles:          []Role{RoleOwner},
	}
	return CreateOrganizationStoreAccepted{}
}

func (store *memoryStore) ListOrganizationsForUser(_ context.Context, userID core.UserID, _ core.Page) ListOrganizationsResult {
	values := make([]Organization, 0)
	for _, organization := range store.organizations {
		if _, matched := store.members[organization.ID.String()+":"+userID.String()]; matched {
			values = append(values, organization)
		}
	}
	return OrganizationsListed{Values: values}
}

func (store *memoryStore) FindMemberRoles(_ context.Context, organizationID core.OrganizationID, userID core.UserID) MemberRolesResult {
	member, matched := store.members[organizationID.String()+":"+userID.String()]
	if !matched {
		return MemberRolesMissing{}
	}
	return MemberRolesFound{Roles: member.Roles}
}

func (store *memoryStore) ProvisionMember(_ context.Context, membershipID core.OrganizationMembershipID, organizationID core.OrganizationID, email auth.EmailAddress, roles []Role) ProvisionMemberStoreResult {
	userID := testUserIDFromEmail(email)
	member := OrganizationMember{ID: membershipID, OrganizationID: organizationID, UserID: userID, Status: MembershipStatusActive, Roles: roles}
	store.members[organizationID.String()+":"+userID.String()] = member
	return MemberProvisioned{Value: member}
}

func (store *memoryStore) DeactivateMember(_ context.Context, organizationID core.OrganizationID, userID core.UserID) DeactivateMemberStoreResult {
	member, matched := store.members[organizationID.String()+":"+userID.String()]
	if !matched {
		return DeactivateMemberStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "member missing")}
	}
	member.Status = MembershipStatusDeactivated
	store.members[organizationID.String()+":"+userID.String()] = member
	return MemberDeactivated{}
}

func (store *memoryStore) CreateOrganizationTeam(context.Context, core.TeamID, core.OrganizationID, TeamName, core.UserID) CreateTeamStoreResult {
	return CreateTeamStoreAccepted{}
}

func (store *memoryStore) CreateStandaloneTeam(context.Context, core.TeamID, core.UserID, TeamName) CreateTeamStoreResult {
	return CreateTeamStoreAccepted{}
}

func (store *memoryStore) AddTeamMember(context.Context, core.TeamID, core.UserID) AddTeamMemberStoreResult {
	return TeamMemberAdded{}
}

func (store *memoryStore) ListOrganizationTeams(context.Context, core.OrganizationID, core.UserID, core.Page) TeamListResult {
	return TeamsListed{Values: []Team{}}
}

func (store *memoryStore) ListStandaloneTeams(context.Context, core.UserID, core.Page) TeamListResult {
	return TeamsListed{Values: []Team{}}
}

func acceptedOrganizationName(t *testing.T, raw string) OrganizationName {
	t.Helper()
	result := NewOrganizationName(raw)
	accepted, matched := result.(OrganizationNameAccepted)
	if !matched {
		t.Fatalf("organization name result = %T, want OrganizationNameAccepted", result)
	}
	return accepted.Value
}

func acceptedEmail(t *testing.T, raw string) auth.EmailAddress {
	t.Helper()
	result := auth.NewEmailAddress(raw)
	accepted, matched := result.(auth.EmailAddressAccepted)
	if !matched {
		t.Fatalf("email result = %T, want EmailAddressAccepted", result)
	}
	return accepted.Value
}

func testUserSubject(t *testing.T) auth.UserSubject {
	t.Helper()
	return auth.UserSubject{ID: testUserID(t)}
}

func testUserID(t *testing.T) core.UserID {
	t.Helper()
	result := core.NewUserID()
	created, matched := result.(core.UserIDCreated)
	if !matched {
		t.Fatalf("user id result = %T, want UserIDCreated", result)
	}
	return created.Value
}

func testOrganizationID(t *testing.T) core.OrganizationID {
	t.Helper()
	result := core.NewOrganizationID()
	created, matched := result.(core.OrganizationIDCreated)
	if !matched {
		t.Fatalf("organization id result = %T, want OrganizationIDCreated", result)
	}
	return created.Value
}

func testUserIDFromEmail(auth.EmailAddress) core.UserID {
	result := core.NewUserID()
	created := result.(core.UserIDCreated)
	return created.Value
}

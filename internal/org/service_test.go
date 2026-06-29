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

func TestServicePassesSelectorQueriesToStore(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store)
	actor := testUserSubject(t)
	organization := testOrganizationID(t)
	page := acceptedPage(t, 20, 40)

	service.ListOrganizations(context.Background(), actor, "lattice", page)
	if store.lastOrganizationQuery != "lattice" || !samePage(store.lastOrganizationPage, page) {
		t.Fatalf("organization selector query/page = %q/%+v, want lattice/%+v", store.lastOrganizationQuery, store.lastOrganizationPage, page)
	}

	service.ListStandaloneTeams(context.Background(), actor, "field", page)
	if store.lastStandaloneTeamQuery != "field" || !samePage(store.lastStandaloneTeamPage, page) {
		t.Fatalf("standalone team selector query/page = %q/%+v, want field/%+v", store.lastStandaloneTeamQuery, store.lastStandaloneTeamPage, page)
	}

	service.ListOrganizationTeams(context.Background(), actor, organization, "survey", page)
	if store.lastOrgTeamQuery != "survey" || !samePage(store.lastOrgTeamPage, page) {
		t.Fatalf("organization team selector query/page = %q/%+v, want survey/%+v", store.lastOrgTeamQuery, store.lastOrgTeamPage, page)
	}
}

type memoryStore struct {
	organizations           map[string]Organization
	members                 map[string]OrganizationMember
	lastOrganizationQuery   string
	lastOrganizationPage    core.Page
	lastStandaloneTeamQuery string
	lastStandaloneTeamPage  core.Page
	lastOrgTeamQuery        string
	lastOrgTeamPage         core.Page
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

func (store *memoryStore) ListOrganizationsForUser(_ context.Context, userID core.UserID, query string, page core.Page) ListOrganizationsResult {
	store.lastOrganizationQuery = query
	store.lastOrganizationPage = page
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

func (store *memoryStore) UpdateMemberRoles(_ context.Context, organizationID core.OrganizationID, userID core.UserID, roles []Role) UpdateMemberRolesStoreResult {
	member, matched := store.members[organizationID.String()+":"+userID.String()]
	if !matched {
		return UpdateMemberRolesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "member missing")}
	}
	member.Roles = roles
	store.members[organizationID.String()+":"+userID.String()] = member
	return MemberRolesUpdated{Value: member}
}

func (store *memoryStore) CreateOrganizationTeam(context.Context, core.TeamID, core.OrganizationID, TeamName, core.UserID) CreateTeamStoreResult {
	return CreateTeamStoreAccepted{}
}

func (store *memoryStore) CreateStandaloneTeam(context.Context, core.TeamID, core.UserID, TeamName) CreateTeamStoreResult {
	return CreateTeamStoreAccepted{}
}

func (store *memoryStore) ListMembers(context.Context, core.OrganizationID, core.Page) ListMembersResult {
	return MembersListed{Values: []OrganizationMember{}}
}

func (store *memoryStore) AddTeamMember(context.Context, core.TeamID, core.UserID) AddTeamMemberStoreResult {
	return TeamMemberAdded{}
}

func (store *memoryStore) AddTeamMemberByEmail(context.Context, core.TeamID, auth.EmailAddress) AddTeamMemberStoreResult {
	return TeamMemberAdded{}
}

func (store *memoryStore) ListOrganizationTeams(_ context.Context, _ core.OrganizationID, _ core.UserID, query string, page core.Page) TeamListResult {
	store.lastOrgTeamQuery = query
	store.lastOrgTeamPage = page
	return TeamsListed{Values: []Team{}}
}

func (store *memoryStore) ListStandaloneTeams(_ context.Context, _ core.UserID, query string, page core.Page) TeamListResult {
	store.lastStandaloneTeamQuery = query
	store.lastStandaloneTeamPage = page
	return TeamsListed{Values: []Team{}}
}

func (store *memoryStore) FindTeam(context.Context, core.TeamID) FindTeamResult {
	return TeamMissing{Reason: core.NewDomainError(core.ErrorCodeNotFound, "team not found")}
}

func (store *memoryStore) ListTeamMembers(context.Context, core.TeamID) TeamMembersResult {
	return TeamMembersListed{Values: []core.UserID{}}
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

func acceptedPage(t *testing.T, limit int, offset int) core.Page {
	t.Helper()
	result := core.NewPage(limit, offset)
	accepted, matched := result.(core.PageAccepted)
	if !matched {
		t.Fatalf("page result = %T, want PageAccepted", result)
	}
	return accepted.Value
}

func samePage(left core.Page, right core.Page) bool {
	return left.Limit() == right.Limit() && left.Offset() == right.Offset()
}

func testUserIDFromEmail(auth.EmailAddress) core.UserID {
	result := core.NewUserID()
	created := result.(core.UserIDCreated)
	return created.Value
}

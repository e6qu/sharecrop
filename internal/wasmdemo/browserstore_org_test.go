package wasmdemo

import (
	"context"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/org"
)

func testOrgName(t *testing.T, raw string) org.OrganizationName {
	t.Helper()
	result := org.NewOrganizationName(raw)
	accepted, matched := result.(org.OrganizationNameAccepted)
	if !matched {
		t.Fatalf("new organization name %q failed", raw)
	}
	return accepted.Value
}

func testTeamName(t *testing.T, raw string) org.TeamName {
	t.Helper()
	result := org.NewTeamName(raw)
	accepted, matched := result.(org.TeamNameAccepted)
	if !matched {
		t.Fatalf("new team name %q failed", raw)
	}
	return accepted.Value
}

// newOrgTestEnv wires an OrgBrowserStore and an AuthBrowserStore against the
// SAME underlying storage, since org membership provisioning looks up users
// by email through auth's own index.
func newOrgTestEnv(t *testing.T) (org.Service, auth.Service, *counterLedgerIDs) {
	t.Helper()
	storage := newTestBrowserStorage()
	ids := &counterLedgerIDs{}
	orgService := org.NewService(NewOrgBrowserStore(storage, ids))
	authServiceResult := auth.NewService(NewAuthBrowserStore(storage, ids), testAccessTokenSecret(t), fixedTestClock{now: time.Now()})
	authService := authServiceResult.(auth.ServiceCreated).Value
	return orgService, authService, ids
}

func TestOrgBrowserStoreCreateAndListOrganizations(t *testing.T) {
	orgService, _, _ := newOrgTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}

	createResult := orgService.CreateOrganization(ctx, owner, testOrgName(t, "Field Labs"))
	created, matched := createResult.(org.OrganizationCreated)
	if !matched {
		t.Fatalf("create organization: want OrganizationCreated, got %#v", createResult)
	}

	listResult := orgService.ListOrganizations(ctx, owner, "", testPage(t, 10, 0))
	listed, matched := listResult.(org.OrganizationsListed)
	if !matched {
		t.Fatalf("list organizations: want OrganizationsListed, got %#v", listResult)
	}
	if len(listed.Values) != 1 || listed.Values[0].ID != created.Value.ID {
		t.Fatalf("listed organizations = %+v, want just %v", listed.Values, created.Value.ID)
	}
}

func TestOrgBrowserStoreOwnerHasManageMembersPermission(t *testing.T) {
	orgService, _, _ := newOrgTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}

	created := orgService.CreateOrganization(ctx, owner, testOrgName(t, "Field Labs")).(org.OrganizationCreated)

	check := orgService.CheckOrganizationPermission(ctx, created.Value.ID, owner.ID, org.PermissionManageMembers)
	if _, matched := check.(org.PermissionGranted); !matched {
		t.Fatalf("owner permission check: want PermissionGranted, got %#v", check)
	}
}

func TestOrgBrowserStoreProvisionMemberByEmail(t *testing.T) {
	orgService, authService, _ := newOrgTestEnv(t)
	ctx := context.Background()
	ownerEmail := testEmail(t, "owner@example.com")
	ownerRegistered := authService.Register(ctx, ownerEmail, testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted)
	owner := auth.UserSubject{ID: ownerRegistered.Subject.ID}

	memberEmail := testEmail(t, "member@example.com")
	memberRegistered := authService.Register(ctx, memberEmail, testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted)

	organization := orgService.CreateOrganization(ctx, owner, testOrgName(t, "Field Labs")).(org.OrganizationCreated)

	provisionResult := orgService.ProvisionMember(ctx, owner, organization.Value.ID, memberEmail, []org.Role{org.RoleMember})
	provisioned, matched := provisionResult.(org.MemberProvisioned)
	if !matched {
		t.Fatalf("provision member: want MemberProvisioned, got %#v", provisionResult)
	}
	if provisioned.Value.UserID != memberRegistered.Subject.ID {
		t.Fatalf("provisioned member user id = %v, want %v", provisioned.Value.UserID, memberRegistered.Subject.ID)
	}

	listResult := orgService.ListMembers(ctx, owner, organization.Value.ID, testPage(t, 10, 0))
	listed, matched := listResult.(org.MembersListed)
	if !matched {
		t.Fatalf("list members: want MembersListed, got %#v", listResult)
	}
	if len(listed.Values) != 2 {
		t.Fatalf("listed members count = %d, want 2 (owner + provisioned member)", len(listed.Values))
	}
}

func TestOrgBrowserStoreProvisionMemberRejectsDuplicate(t *testing.T) {
	orgService, authService, _ := newOrgTestEnv(t)
	ctx := context.Background()
	ownerEmail := testEmail(t, "owner@example.com")
	ownerRegistered := authService.Register(ctx, ownerEmail, testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted)
	owner := auth.UserSubject{ID: ownerRegistered.Subject.ID}

	memberEmail := testEmail(t, "member@example.com")
	authService.Register(ctx, memberEmail, testPassword(t, "correct horse battery staple"))

	organization := orgService.CreateOrganization(ctx, owner, testOrgName(t, "Field Labs")).(org.OrganizationCreated)
	orgService.ProvisionMember(ctx, owner, organization.Value.ID, memberEmail, []org.Role{org.RoleMember})

	duplicateResult := orgService.ProvisionMember(ctx, owner, organization.Value.ID, memberEmail, []org.Role{org.RoleMember})
	if _, matched := duplicateResult.(org.ProvisionMemberRejected); !matched {
		t.Fatalf("provision already-member email: want ProvisionMemberRejected, got %#v", duplicateResult)
	}
}

func TestOrgBrowserStoreProvisionMemberRejectsReprovisioningDeactivatedMember(t *testing.T) {
	orgService, authService, _ := newOrgTestEnv(t)
	ctx := context.Background()
	ownerEmail := testEmail(t, "owner@example.com")
	ownerRegistered := authService.Register(ctx, ownerEmail, testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted)
	owner := auth.UserSubject{ID: ownerRegistered.Subject.ID}

	memberEmail := testEmail(t, "member@example.com")
	memberRegistered := authService.Register(ctx, memberEmail, testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted)

	organization := orgService.CreateOrganization(ctx, owner, testOrgName(t, "Field Labs")).(org.OrganizationCreated)
	orgService.ProvisionMember(ctx, owner, organization.Value.ID, memberEmail, []org.Role{org.RoleMember})
	orgService.DeactivateMember(ctx, owner, organization.Value.ID, memberRegistered.Subject.ID)

	// Even a deactivated membership can't be re-provisioned - it must be
	// reactivated via role updates, matching the real store's uniqueness
	// constraint (one membership row per (organization, user), whatever status).
	reprovisionResult := orgService.ProvisionMember(ctx, owner, organization.Value.ID, memberEmail, []org.Role{org.RoleMember})
	if _, matched := reprovisionResult.(org.ProvisionMemberRejected); !matched {
		t.Fatalf("re-provision deactivated member: want ProvisionMemberRejected, got %#v", reprovisionResult)
	}
}

func TestOrgBrowserStoreProvisionMemberRejectsUnknownEmail(t *testing.T) {
	orgService, authService, _ := newOrgTestEnv(t)
	ctx := context.Background()
	ownerEmail := testEmail(t, "owner@example.com")
	ownerRegistered := authService.Register(ctx, ownerEmail, testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted)
	owner := auth.UserSubject{ID: ownerRegistered.Subject.ID}
	organization := orgService.CreateOrganization(ctx, owner, testOrgName(t, "Field Labs")).(org.OrganizationCreated)

	provisionResult := orgService.ProvisionMember(ctx, owner, organization.Value.ID, testEmail(t, "nobody@example.com"), []org.Role{org.RoleMember})
	if _, matched := provisionResult.(org.ProvisionMemberRejected); !matched {
		t.Fatalf("provision unknown email: want ProvisionMemberRejected, got %#v", provisionResult)
	}
}

func TestOrgBrowserStoreDeactivateMemberRemovesPermission(t *testing.T) {
	orgService, authService, _ := newOrgTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: authService.Register(ctx, testEmail(t, "owner@example.com"), testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted).Subject.ID}
	memberRegistered := authService.Register(ctx, testEmail(t, "member@example.com"), testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted)

	organization := orgService.CreateOrganization(ctx, owner, testOrgName(t, "Field Labs")).(org.OrganizationCreated)
	orgService.ProvisionMember(ctx, owner, organization.Value.ID, testEmail(t, "member@example.com"), []org.Role{org.RoleMember})

	deactivateResult := orgService.DeactivateMember(ctx, owner, organization.Value.ID, memberRegistered.Subject.ID)
	if _, matched := deactivateResult.(org.MemberDeactivationAccepted); !matched {
		t.Fatalf("deactivate member: want MemberDeactivationAccepted, got %#v", deactivateResult)
	}

	check := orgService.CheckOrganizationPermission(ctx, organization.Value.ID, memberRegistered.Subject.ID, org.PermissionCreateOrganizationTask)
	if _, matched := check.(org.PermissionDenied); !matched {
		t.Fatalf("deactivated member permission check: want PermissionDenied, got %#v", check)
	}
}

func TestOrgBrowserStoreUpdateMemberRoles(t *testing.T) {
	orgService, authService, _ := newOrgTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: authService.Register(ctx, testEmail(t, "owner@example.com"), testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted).Subject.ID}
	memberRegistered := authService.Register(ctx, testEmail(t, "member@example.com"), testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted)

	organization := orgService.CreateOrganization(ctx, owner, testOrgName(t, "Field Labs")).(org.OrganizationCreated)
	orgService.ProvisionMember(ctx, owner, organization.Value.ID, testEmail(t, "member@example.com"), []org.Role{org.RoleMember})

	updateResult := orgService.UpdateMemberRoles(ctx, owner, organization.Value.ID, memberRegistered.Subject.ID, []org.Role{org.RoleAdmin})
	updated, matched := updateResult.(org.MemberRolesUpdatedResult)
	if !matched {
		t.Fatalf("update member roles: want MemberRolesUpdatedResult, got %#v", updateResult)
	}
	if len(updated.Value.Roles) != 1 || updated.Value.Roles[0] != org.RoleAdmin {
		t.Fatalf("updated roles = %+v, want [admin]", updated.Value.Roles)
	}

	check := orgService.CheckOrganizationPermission(ctx, organization.Value.ID, memberRegistered.Subject.ID, org.PermissionManageTeams)
	if _, matched := check.(org.PermissionGranted); !matched {
		t.Fatalf("admin role permission check: want PermissionGranted, got %#v", check)
	}
}

func TestOrgBrowserStoreOrganizationTeamLifecycle(t *testing.T) {
	orgService, authService, _ := newOrgTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: authService.Register(ctx, testEmail(t, "owner@example.com"), testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted).Subject.ID}
	memberRegistered := authService.Register(ctx, testEmail(t, "member@example.com"), testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted)

	organization := orgService.CreateOrganization(ctx, owner, testOrgName(t, "Field Labs")).(org.OrganizationCreated)

	createTeamResult := orgService.CreateOrganizationTeam(ctx, owner, organization.Value.ID, testTeamName(t, "Field crew"))
	teamCreated, matched := createTeamResult.(org.TeamCreated)
	if !matched {
		t.Fatalf("create organization team: want TeamCreated, got %#v", createTeamResult)
	}

	addResult := orgService.AddTeamMember(ctx, owner, teamCreated.Value.ID, testEmail(t, "member@example.com"))
	if _, matched := addResult.(org.TeamMemberAddedResult); !matched {
		t.Fatalf("add team member: want TeamMemberAddedResult, got %#v", addResult)
	}

	getResult := orgService.GetTeam(ctx, owner, teamCreated.Value.ID)
	got, matched := getResult.(org.TeamGot)
	if !matched {
		t.Fatalf("get team: want TeamGot, got %#v", getResult)
	}
	if len(got.Members) != 1 || got.Members[0] != memberRegistered.Subject.ID {
		t.Fatalf("team members = %+v, want [%v]", got.Members, memberRegistered.Subject.ID)
	}

	listResult := orgService.ListOrganizationTeams(ctx, owner, organization.Value.ID, "", testPage(t, 10, 0))
	listed, matched := listResult.(org.OrganizationTeamsListed)
	if !matched {
		t.Fatalf("list organization teams: want OrganizationTeamsListed, got %#v", listResult)
	}
	if len(listed.Values) != 1 || listed.Values[0].ID != teamCreated.Value.ID {
		t.Fatalf("listed teams = %+v, want just %v", listed.Values, teamCreated.Value.ID)
	}
}

func TestOrgBrowserStoreStandaloneTeamLifecycle(t *testing.T) {
	orgService, authService, _ := newOrgTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: authService.Register(ctx, testEmail(t, "owner@example.com"), testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted).Subject.ID}

	createResult := orgService.CreateStandaloneTeam(ctx, owner, testTeamName(t, "My crew"))
	created, matched := createResult.(org.TeamCreated)
	if !matched {
		t.Fatalf("create standalone team: want TeamCreated, got %#v", createResult)
	}

	listResult := orgService.ListStandaloneTeams(ctx, owner, "", testPage(t, 10, 0))
	listed, matched := listResult.(org.OrganizationTeamsListed)
	if !matched {
		t.Fatalf("list standalone teams: want OrganizationTeamsListed, got %#v", listResult)
	}
	if len(listed.Values) != 1 || listed.Values[0].ID != created.Value.ID {
		t.Fatalf("listed standalone teams = %+v, want just %v", listed.Values, created.Value.ID)
	}

	// A non-member, non-owner outsider cannot view the team.
	outsider := auth.UserSubject{ID: authService.Register(ctx, testEmail(t, "outsider@example.com"), testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted).Subject.ID}
	getResult := orgService.GetTeam(ctx, outsider, created.Value.ID)
	if _, matched := getResult.(org.GetTeamRejected); !matched {
		t.Fatalf("outsider get team: want GetTeamRejected, got %#v", getResult)
	}
}

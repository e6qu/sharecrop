//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/org/orgtest"
	"github.com/e6qu/sharecrop/internal/wasibridge/orgbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

// TestOrgBridgeDualRun drives the organization store through the compiled wasip1
// guest + host bridge against real Postgres: create org, provision + update +
// deactivate a member, create a team and add a member, and every read path
// (list orgs, member roles, list members, list teams, find team, list team
// members) is checked against a direct db call. Teams exercise the TeamOwner
// tagged union.
func TestOrgBridgeDualRun(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	dbStore := db.NewOrgStore(pool)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-store-guest")
	if err != nil {
		t.Fatalf("compile store guest: %v", err)
	}
	host, err := rpc.NewHost(ctx, guestWASM, func(ctx context.Context, method string, args []byte) ([]byte, error) {
		return orgbridge.Dispatch(ctx, dbStore, method, args)
	})
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close(ctx) })
	bridgeStore := orgbridge.NewGuestStore(func(method string, args []byte) ([]byte, error) {
		return host.Call(ctx, method, args)
	})

	creator := createUser(t, pool, "org-creator")
	member := createUser(t, pool, "org-member")
	memberEmail := userEmail(t, "org-member", member)
	page := requirePage(t, 50, 0)

	orgID := orgtest.NewOrganizationID(t)
	name := orgtest.OrganizationName(t, "Bridge Org")

	t.Run("create organization then list matches a direct call", func(t *testing.T) {
		result := bridgeStore.CreateOrganization(ctx, orgID, name, creator, orgtest.NewMembershipID(t))
		if _, matched := result.(org.CreateOrganizationStoreAccepted); !matched {
			t.Fatalf("bridge CreateOrganization = %T, want accepted", result)
		}
		viaBridge := requireOrgsListed(t, bridgeStore.ListOrganizationsForUser(ctx, creator, "", page))
		direct := requireOrgsListed(t, dbStore.ListOrganizationsForUser(ctx, creator, "", page))
		if len(viaBridge) != len(direct) || len(viaBridge) != 1 {
			t.Fatalf("org counts: bridge %d, direct %d, want 1", len(viaBridge), len(direct))
		}
		if diff := orgtest.OrganizationDiff(viaBridge[0], direct[0]); diff != "" {
			t.Errorf("organization mismatch: %s", diff)
		}
	})

	t.Run("creator holds the owner role", func(t *testing.T) {
		viaBridge := requireRolesFound(t, bridgeStore.FindMemberRoles(ctx, orgID, creator))
		direct := requireRolesFound(t, dbStore.FindMemberRoles(ctx, orgID, creator))
		if orgtest.RolesKey(viaBridge) != orgtest.RolesKey(direct) {
			t.Errorf("creator roles: bridge %s, direct %s", orgtest.RolesKey(viaBridge), orgtest.RolesKey(direct))
		}
	})

	t.Run("provision then update a member's roles", func(t *testing.T) {
		provision := bridgeStore.ProvisionMember(ctx, orgtest.NewMembershipID(t), orgID, memberEmail, []org.Role{org.RoleMember})
		provisioned, matched := provision.(org.MemberProvisioned)
		if !matched {
			t.Fatalf("bridge ProvisionMember = %T, want MemberProvisioned", provision)
		}
		if provisioned.Value.UserID != member {
			t.Errorf("provisioned member = %s, want %s", provisioned.Value.UserID, member)
		}

		update := bridgeStore.UpdateMemberRoles(ctx, orgID, member, []org.Role{org.RoleAdmin, org.RoleReviewer})
		updated, matched := update.(org.MemberRolesUpdated)
		if !matched {
			t.Fatalf("bridge UpdateMemberRoles = %T, want MemberRolesUpdated", update)
		}
		if orgtest.RolesKey(updated.Value.Roles) != "admin,reviewer" {
			t.Errorf("updated roles = %s, want admin,reviewer", orgtest.RolesKey(updated.Value.Roles))
		}

		viaBridge := requireMembersListed(t, bridgeStore.ListMembers(ctx, orgID, page))
		direct := requireMembersListed(t, dbStore.ListMembers(ctx, orgID, page))
		assertMemberSetsEqual(t, viaBridge, direct)
	})

	teamID := orgtest.NewTeamID(t)

	t.Run("create an organization team and add a member", func(t *testing.T) {
		if _, matched := bridgeStore.CreateOrganizationTeam(ctx, teamID, orgID, orgtest.TeamName(t, "Bridge Team"), creator).(org.CreateTeamStoreAccepted); !matched {
			t.Fatalf("bridge CreateOrganizationTeam did not accept")
		}
		if _, matched := bridgeStore.AddTeamMember(ctx, teamID, member).(org.TeamMemberAdded); !matched {
			t.Fatalf("bridge AddTeamMember did not accept")
		}

		viaBridge := requireTeamsListed(t, bridgeStore.ListOrganizationTeams(ctx, orgID, creator, "", page))
		direct := requireTeamsListed(t, dbStore.ListOrganizationTeams(ctx, orgID, creator, "", page))
		if len(viaBridge) != len(direct) || len(viaBridge) != 1 {
			t.Fatalf("team counts: bridge %d, direct %d, want 1", len(viaBridge), len(direct))
		}
		if diff := orgtest.TeamDiff(viaBridge[0], direct[0]); diff != "" {
			t.Errorf("team mismatch: %s", diff)
		}
	})

	t.Run("find team and list its members match a direct call", func(t *testing.T) {
		viaBridge, matched := bridgeStore.FindTeam(ctx, teamID).(org.TeamFound)
		if !matched {
			t.Fatalf("bridge FindTeam did not find the team")
		}
		direct, matched := dbStore.FindTeam(ctx, teamID).(org.TeamFound)
		if !matched {
			t.Fatalf("direct FindTeam did not find the team")
		}
		if diff := orgtest.TeamDiff(viaBridge.Value, direct.Value); diff != "" {
			t.Errorf("found team mismatch: %s", diff)
		}
		if _, isOrgOwned := viaBridge.Value.Owner.(org.OrganizationOwnedTeam); !isOrgOwned {
			t.Errorf("team owner = %T, want OrganizationOwnedTeam", viaBridge.Value.Owner)
		}

		bridgeMembers := requireTeamMembers(t, bridgeStore.ListTeamMembers(ctx, teamID))
		directMembers := requireTeamMembers(t, dbStore.ListTeamMembers(ctx, teamID))
		if len(bridgeMembers) != len(directMembers) {
			t.Errorf("team member counts: bridge %d, direct %d", len(bridgeMembers), len(directMembers))
		}
	})

	t.Run("deactivate the member through the bridge", func(t *testing.T) {
		if _, matched := bridgeStore.DeactivateMember(ctx, orgID, member).(org.MemberDeactivated); !matched {
			t.Fatalf("bridge DeactivateMember did not accept")
		}
	})
}

func userEmail(t *testing.T, prefix string, userID core.UserID) auth.EmailAddress {
	t.Helper()
	accepted, matched := auth.NewEmailAddress(prefix + "-" + userID.String() + "@example.com").(auth.EmailAddressAccepted)
	if !matched {
		t.Fatalf("email rejected")
	}
	return accepted.Value
}

func requireOrgsListed(t *testing.T, result org.ListOrganizationsResult) []org.Organization {
	t.Helper()
	listed, matched := result.(org.OrganizationsListed)
	if !matched {
		t.Fatalf("list organizations result = %T, want listed", result)
	}
	return listed.Values
}

func requireRolesFound(t *testing.T, result org.MemberRolesResult) []org.Role {
	t.Helper()
	found, matched := result.(org.MemberRolesFound)
	if !matched {
		t.Fatalf("member roles result = %T, want found", result)
	}
	return found.Roles
}

func requireMembersListed(t *testing.T, result org.ListMembersResult) []org.OrganizationMember {
	t.Helper()
	listed, matched := result.(org.MembersListed)
	if !matched {
		t.Fatalf("list members result = %T, want listed", result)
	}
	return listed.Values
}

func requireTeamsListed(t *testing.T, result org.TeamListResult) []org.Team {
	t.Helper()
	listed, matched := result.(org.TeamsListed)
	if !matched {
		t.Fatalf("team list result = %T, want listed", result)
	}
	return listed.Values
}

func requireTeamMembers(t *testing.T, result org.TeamMembersResult) []core.UserID {
	t.Helper()
	listed, matched := result.(org.TeamMembersListed)
	if !matched {
		t.Fatalf("team members result = %T, want listed", result)
	}
	return listed.Values
}

func assertMemberSetsEqual(t *testing.T, got, want []org.OrganizationMember) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("member counts: bridge %d, direct %d", len(got), len(want))
	}
	for index := range want {
		if diff := orgtest.MemberDiff(got[index], want[index]); diff != "" {
			t.Errorf("member %d mismatch: %s", index, diff)
		}
	}
}

package mcp

import (
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/agent"
)

const (
	toolCreateOrganization     = "sharecrop.create_organization"
	toolListOrganizations      = "sharecrop.list_organizations"
	toolListOrgMembers         = "sharecrop.list_organization_members"
	toolProvisionOrgMember     = "sharecrop.provision_organization_member"
	toolDeactivateOrgMember    = "sharecrop.deactivate_organization_member"
	toolUpdateOrgMemberRoles   = "sharecrop.update_organization_member_roles"
	toolCreateOrganizationTeam = "sharecrop.create_organization_team"
	toolListOrganizationTeams  = "sharecrop.list_organization_teams"
	toolCreateStandaloneTeam   = "sharecrop.create_standalone_team"
	toolListStandaloneTeams    = "sharecrop.list_standalone_teams"
	toolGetTeam                = "sharecrop.get_team"
	toolGetTeamWork            = "sharecrop.get_team_work"
	toolAddTeamMember          = "sharecrop.add_team_member"

	toolCreateOrgCredential = "sharecrop.create_org_credential"
	toolListOrgCredentials  = "sharecrop.list_org_credentials"
	toolRevokeOrgCredential = "sharecrop.revoke_org_credential"
)

func orgToolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			Name:        toolCreateOrganization,
			Description: "Create a new organization owned by the agent's user, who becomes its owner.",
			Scope:       agent.ScopeOrgManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`),
		},
		{
			Name:        toolListOrganizations,
			Description: "List organizations the agent's user belongs to. query optionally filters by name.",
			Scope:       agent.ScopeOrgRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`),
		},
		{
			Name:        toolListOrgMembers,
			Description: "List an organization's members and their roles.",
			Scope:       agent.ScopeOrgRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"organization_id":{"type":"string"}},"required":["organization_id"]}`),
		},
		{
			Name:        toolProvisionOrgMember,
			Description: "Invite or provision a member into an organization by email, with the given roles (owner, admin, member, billing, reviewer, public_publisher).",
			Scope:       agent.ScopeOrgManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"organization_id":{"type":"string"},"email":{"type":"string"},"roles":{"type":"array","items":{"type":"string"}}},"required":["organization_id","email","roles"]}`),
		},
		{
			Name:        toolDeactivateOrgMember,
			Description: "Deactivate a member's access to an organization.",
			Scope:       agent.ScopeOrgManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"organization_id":{"type":"string"},"user_id":{"type":"string"}},"required":["organization_id","user_id"]}`),
		},
		{
			Name:        toolUpdateOrgMemberRoles,
			Description: "Replace a member's roles within an organization.",
			Scope:       agent.ScopeOrgManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"organization_id":{"type":"string"},"user_id":{"type":"string"},"roles":{"type":"array","items":{"type":"string"}}},"required":["organization_id","user_id","roles"]}`),
		},
		{
			Name:        toolCreateOrganizationTeam,
			Description: "Create a team owned by an organization.",
			Scope:       agent.ScopeOrgManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"organization_id":{"type":"string"},"name":{"type":"string"}},"required":["organization_id","name"]}`),
		},
		{
			Name:        toolListOrganizationTeams,
			Description: "List an organization's teams. query optionally filters by name.",
			Scope:       agent.ScopeOrgRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"organization_id":{"type":"string"},"query":{"type":"string"}},"required":["organization_id"]}`),
		},
		{
			Name:        toolCreateStandaloneTeam,
			Description: "Create a standalone team owned by the agent's user directly, not by an organization.",
			Scope:       agent.ScopeOrgManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`),
		},
		{
			Name:        toolListStandaloneTeams,
			Description: "List standalone (non-organization) teams. query optionally filters by name.",
			Scope:       agent.ScopeOrgRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`),
		},
		{
			Name:        toolGetTeam,
			Description: "Get a team's detail and member list.",
			Scope:       agent.ScopeOrgRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"team_id":{"type":"string"}},"required":["team_id"]}`),
		},
		{
			Name:        toolGetTeamWork,
			Description: "List tasks assigned to or reserved by a team.",
			Scope:       agent.ScopeOrgRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"team_id":{"type":"string"}},"required":["team_id"]}`),
		},
		{
			Name:        toolAddTeamMember,
			Description: "Add a member to a team by email.",
			Scope:       agent.ScopeOrgManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"team_id":{"type":"string"},"email":{"type":"string"}},"required":["team_id","email"]}`),
		},
		{
			Name:        toolCreateOrgCredential,
			Description: "Mint a new organization-wide credential, which acts with full parity to an org-admin member. Requires PermissionManageMembers on the organization.",
			Scope:       agent.ScopeCredentialsManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"organization_id":{"type":"string"},"label":{"type":"string"},"scopes":{"type":"array","items":{"type":"string"}},"expires_at":{"type":"string"}},"required":["organization_id","label","scopes"]}`),
		},
		{
			Name:        toolListOrgCredentials,
			Description: "List an organization's org-wide credentials.",
			Scope:       agent.ScopeCredentialsManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"organization_id":{"type":"string"}},"required":["organization_id"]}`),
		},
		{
			Name:        toolRevokeOrgCredential,
			Description: "Revoke one of an organization's org-wide credentials.",
			Scope:       agent.ScopeCredentialsManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"organization_id":{"type":"string"},"credential_id":{"type":"string"}},"required":["organization_id","credential_id"]}`),
		},
	}
}

//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestMCPOrganizationAndTeamLifecycle(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-org-lifecycle-owner")
	memberEmail := "mcp-org-lifecycle-member-" + uniqueTestSuffix(t) + "@example.com"
	registerUserWithEmail(t, server, memberEmail)
	ownerAgent := createAgentCredential(t, server, owner.AccessToken, []string{"org_read", "org_manage"})
	session := initializeMCPSession(t, server, ownerAgent)

	created := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `1`, "sharecrop.create_organization", `{"name":"MCP Tool Org"}`)))
	var organization struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(created), &organization); err != nil {
		t.Fatalf("decode create_organization: %v (%s)", err, created)
	}

	listed := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `2`, "sharecrop.list_organizations", `{}`)))
	if !strings.Contains(listed, organization.ID) {
		t.Fatalf("list_organizations missing the created org: %s", listed)
	}

	team := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `3`, "sharecrop.create_organization_team", `{"organization_id":"`+organization.ID+`","name":"MCP Tool Team"}`)))
	var teamResult struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(team), &teamResult); err != nil {
		t.Fatalf("decode create_organization_team: %v (%s)", err, team)
	}

	teams := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `4`, "sharecrop.list_organization_teams", `{"organization_id":"`+organization.ID+`"}`)))
	if !strings.Contains(teams, teamResult.ID) {
		t.Fatalf("list_organization_teams missing the created team: %s", teams)
	}

	provisioned := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `5`, "sharecrop.provision_organization_member", `{"organization_id":"`+organization.ID+`","email":"`+memberEmail+`","roles":["member"]}`)))
	var provisionedMember struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal([]byte(provisioned), &provisionedMember); err != nil {
		t.Fatalf("decode provision_organization_member: %v (%s)", err, provisioned)
	}

	members := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `6`, "sharecrop.list_organization_members", `{"organization_id":"`+organization.ID+`"}`)))
	if !strings.Contains(members, provisionedMember.UserID) {
		t.Fatalf("list_organization_members missing the provisioned member: %s", members)
	}

	updatedRoles := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `7`, "sharecrop.update_organization_member_roles", `{"organization_id":"`+organization.ID+`","user_id":"`+provisionedMember.UserID+`","roles":["member","reviewer"]}`)))
	if !strings.Contains(updatedRoles, "reviewer") {
		t.Fatalf("update_organization_member_roles missing the new role: %s", updatedRoles)
	}

	deactivated := decodeRPC(t, mcpCall(t, server, ownerAgent, session, `8`, "sharecrop.deactivate_organization_member", `{"organization_id":"`+organization.ID+`","user_id":"`+provisionedMember.UserID+`"}`))
	if deactivated.Error != nil {
		t.Fatalf("deactivate_organization_member returned a protocol error: %+v", deactivated.Error)
	}

	teamDetail := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `9`, "sharecrop.get_team", `{"team_id":"`+teamResult.ID+`"}`)))
	if !strings.Contains(teamDetail, teamResult.ID) {
		t.Fatalf("get_team missing team id: %s", teamDetail)
	}

	teamWork := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `10`, "sharecrop.get_team_work", `{"team_id":"`+teamResult.ID+`"}`)))
	if !strings.Contains(teamWork, `"tasks"`) {
		t.Fatalf("get_team_work missing tasks key: %s", teamWork)
	}

	standaloneTeam := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `11`, "sharecrop.create_standalone_team", `{"name":"MCP Standalone Team"}`)))
	var standalone struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(standaloneTeam), &standalone); err != nil {
		t.Fatalf("decode create_standalone_team: %v (%s)", err, standaloneTeam)
	}
	standaloneTeams := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `12`, "sharecrop.list_standalone_teams", `{}`)))
	if !strings.Contains(standaloneTeams, standalone.ID) {
		t.Fatalf("list_standalone_teams missing the created team: %s", standaloneTeams)
	}
}

// TestMCPGetTeamAndAddTeamMemberAcceptOrgCredential is the org-token-parity
// regression test for the two org/team tools that accept an org-wide
// credential over MCP, matching REST's requireUserOrOrgSubject-gated
// get_team/add_team_member handlers exactly.
func TestMCPGetTeamAndAddTeamMemberAcceptOrgCredential(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-team-org-owner")
	orgResponse := postJSONWithBearer(t, server.URL+"/api/organizations", []byte(`{"name":"MCP Team Org"}`), owner.AccessToken)
	defer orgResponse.Body.Close()
	assertStatus(t, orgResponse, http.StatusCreated)
	organization := decodeOrganizationHTTPResponse(t, orgResponse)

	credentialResponse := postJSONWithBearer(t, server.URL+"/api/organizations/"+organization.ID+"/credentials", []byte(`{"label":"MCP team org token","scopes":["org_read","org_manage"],"expires_at":""}`), owner.AccessToken)
	defer credentialResponse.Body.Close()
	assertStatus(t, credentialResponse, http.StatusCreated)
	orgCredential := decodeOrgCredentialCreatedHTTPResponse(t, credentialResponse)
	orgSession := initializeMCPSession(t, server, orgCredential.Secret)

	teamResponse := postJSONWithBearer(t, server.URL+"/api/organizations/"+organization.ID+"/teams", []byte(`{"name":"MCP Team Org Team"}`), owner.AccessToken)
	defer teamResponse.Body.Close()
	assertStatus(t, teamResponse, http.StatusCreated)
	var team struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(teamResponse.Body).Decode(&team); err != nil {
		t.Fatalf("decode team response: %v", err)
	}

	// The org token gets team detail over MCP: full parity.
	got := toolText(t, decodeRPC(t, mcpCall(t, server, orgCredential.Secret, orgSession, `1`, "sharecrop.get_team", `{"team_id":"`+team.ID+`"}`)))
	if !strings.Contains(got, team.ID) {
		t.Fatalf("get_team via org credential missing team id: %s", got)
	}

	// The org token adds a team member over MCP: full parity.
	newMember := registerUser(t, server, "mcp-team-org-newmember")
	added := decodeRPC(t, mcpCall(t, server, orgCredential.Secret, orgSession, `2`, "sharecrop.add_team_member", `{"team_id":"`+team.ID+`","email":"`+newMember.SubjectID+`@nonexistent.example"}`))
	if added.Error != nil {
		t.Fatalf("add_team_member via org credential returned a protocol error: %+v", added.Error)
	}
}

func TestMCPOrgCredentialSelfManagementTools(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-orgcred-owner")
	orgResponse := postJSONWithBearer(t, server.URL+"/api/organizations", []byte(`{"name":"MCP OrgCred Org"}`), owner.AccessToken)
	defer orgResponse.Body.Close()
	assertStatus(t, orgResponse, http.StatusCreated)
	organization := decodeOrganizationHTTPResponse(t, orgResponse)

	ownerAgent := createAgentCredential(t, server, owner.AccessToken, []string{"credentials_manage"})
	session := initializeMCPSession(t, server, ownerAgent)

	created := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `1`, "sharecrop.create_org_credential", `{"organization_id":"`+organization.ID+`","label":"MCP-minted org token","scopes":["tasks_read"],"expires_at":""}`)))
	var createdPayload struct {
		Credential struct {
			ID string `json:"id"`
		} `json:"credential"`
		Secret string `json:"secret"`
	}
	if err := json.Unmarshal([]byte(created), &createdPayload); err != nil {
		t.Fatalf("decode create_org_credential: %v (%s)", err, created)
	}
	if createdPayload.Secret == "" {
		t.Fatalf("create_org_credential did not return a secret")
	}

	listed := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `2`, "sharecrop.list_org_credentials", `{"organization_id":"`+organization.ID+`"}`)))
	if !strings.Contains(listed, createdPayload.Credential.ID) {
		t.Fatalf("list_org_credentials missing the created credential: %s", listed)
	}

	revoked := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `3`, "sharecrop.revoke_org_credential", `{"organization_id":"`+organization.ID+`","credential_id":"`+createdPayload.Credential.ID+`"}`)))
	if !strings.Contains(revoked, `"state":"revoked"`) {
		t.Fatalf("revoke_org_credential did not report revoked state: %s", revoked)
	}
}

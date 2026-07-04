//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestMCPCollectiblesLifecycle(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-collectible-owner")
	recipient := registerUser(t, server, "mcp-collectible-recipient")
	ownerAgent := createAgentCredential(t, server, owner.AccessToken, []string{"collectibles_read", "collectibles_manage", "tasks_write", "tasks_read"})
	session := initializeMCPSession(t, server, ownerAgent)

	catalog := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `1`, "sharecrop.collectible_catalog", `{}`)))
	var catalogPayload struct {
		Entries []struct {
			Slug string `json:"slug"`
		} `json:"entries"`
	}
	if err := json.Unmarshal([]byte(catalog), &catalogPayload); err != nil {
		t.Fatalf("decode collectible_catalog: %v (%s)", err, catalog)
	}
	if len(catalogPayload.Entries) == 0 {
		t.Fatalf("collectible_catalog returned no entries")
	}

	minted := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `2`, "sharecrop.mint_collectible", `{"name":"MCP Badge","kind":"badge","transfer_policy":"transferable_between_users","art":"mcp-art"}`)))
	var mintedPayload struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(minted), &mintedPayload); err != nil {
		t.Fatalf("decode mint_collectible: %v (%s)", err, minted)
	}

	listed := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `3`, "sharecrop.list_collectibles", `{}`)))
	if !strings.Contains(listed, mintedPayload.ID) {
		t.Fatalf("list_collectibles missing the minted collectible: %s", listed)
	}

	transferred := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `4`, "sharecrop.transfer_collectible", `{"collectible_id":"`+mintedPayload.ID+`","recipient_id":"`+recipient.SubjectID+`"}`)))
	if !strings.Contains(transferred, recipient.SubjectID) {
		t.Fatalf("transfer_collectible did not report the new owner: %s", transferred)
	}

	// Fund and refund a collectible reward on a task the owner owns.
	rewardCollectibleMinted := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `5`, "sharecrop.mint_collectible", `{"name":"MCP Reward","kind":"badge","transfer_policy":"transferable_between_users","art":"mcp-reward-art"}`)))
	var rewardCollectible struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(rewardCollectibleMinted), &rewardCollectible); err != nil {
		t.Fatalf("decode reward mint_collectible: %v (%s)", err, rewardCollectibleMinted)
	}
	rewardTask := createPublicUserTask(t, server, owner)

	funded := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `6`, "sharecrop.fund_collectible_reward", `{"task_id":"`+rewardTask.ID+`","collectible_id":"`+rewardCollectible.ID+`"}`)))
	if !strings.Contains(funded, rewardCollectible.ID) {
		t.Fatalf("fund_collectible_reward missing the funded collectible: %s", funded)
	}

	refunded := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `7`, "sharecrop.refund_collectible_reward", `{"task_id":"`+rewardTask.ID+`"}`)))
	if !strings.Contains(refunded, rewardCollectible.ID) {
		t.Fatalf("refund_collectible_reward missing the refunded collectible: %s", refunded)
	}
}

func TestMCPListOrganizationAndTeamCollectibles(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-org-collectible-owner")
	orgResponse := postJSONWithBearer(t, server.URL+"/api/organizations", []byte(`{"name":"MCP Collectible Org"}`), owner.AccessToken)
	defer orgResponse.Body.Close()
	assertStatus(t, orgResponse, http.StatusCreated)
	organization := decodeOrganizationHTTPResponse(t, orgResponse)

	teamResponse := postJSONWithBearer(t, server.URL+"/api/organizations/"+organization.ID+"/teams", []byte(`{"name":"MCP Collectible Team"}`), owner.AccessToken)
	defer teamResponse.Body.Close()
	assertStatus(t, teamResponse, http.StatusCreated)
	var team struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(teamResponse.Body).Decode(&team); err != nil {
		t.Fatalf("decode team response: %v", err)
	}

	ownerAgent := createAgentCredential(t, server, owner.AccessToken, []string{"collectibles_read"})
	session := initializeMCPSession(t, server, ownerAgent)

	orgCollectibles := decodeRPC(t, mcpCall(t, server, ownerAgent, session, `1`, "sharecrop.list_organization_collectibles", `{"organization_id":"`+organization.ID+`"}`))
	toolText(t, orgCollectibles)

	teamCollectibles := decodeRPC(t, mcpCall(t, server, ownerAgent, session, `2`, "sharecrop.list_team_collectibles", `{"team_id":"`+team.ID+`"}`))
	toolText(t, teamCollectibles)
}

package mcp

import (
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/agent"
)

const (
	toolMintCollectible              = "sharecrop.mint_collectible"
	toolCollectibleCatalog           = "sharecrop.collectible_catalog"
	toolTransferCollectible          = "sharecrop.transfer_collectible"
	toolListCollectibles             = "sharecrop.list_collectibles"
	toolFundCollectibleReward        = "sharecrop.fund_collectible_reward"
	toolRefundCollectibleReward      = "sharecrop.refund_collectible_reward"
	toolListOrganizationCollectibles = "sharecrop.list_organization_collectibles"
	toolListTeamCollectibles         = "sharecrop.list_team_collectibles"
)

func collectiblesToolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			Name:        toolMintCollectible,
			Description: "Mint a new collectible owned by the agent's user. kind is badge or item; transfer_policy is non_transferable, transferable_between_users, or transferable_to_organization.",
			Scope:       agent.ScopeCollectiblesManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"kind":{"type":"string"},"transfer_policy":{"type":"string"},"art":{"type":"string"}},"required":["name","kind","transfer_policy"]}`),
		},
		{
			Name:        toolCollectibleCatalog,
			Description: "List the platform's default collectible templates.",
			Scope:       agent.ScopeCollectiblesRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
		{
			Name:        toolTransferCollectible,
			Description: "Transfer an owned, transferable collectible to another user.",
			Scope:       agent.ScopeCollectiblesManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"collectible_id":{"type":"string"},"recipient_id":{"type":"string"}},"required":["collectible_id","recipient_id"]}`),
		},
		{
			Name:        toolListCollectibles,
			Description: "List the agent's user's own collectibles.",
			Scope:       agent.ScopeCollectiblesRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
		{
			Name:        toolFundCollectibleReward,
			Description: "Escrow a collectible reward onto a task the agent's user owns.",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"},"collectible_id":{"type":"string"}},"required":["task_id","collectible_id"]}`),
		},
		{
			Name:        toolRefundCollectibleReward,
			Description: "Refund a task's escrowed collectible reward(s) back to the agent's user.",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"}},"required":["task_id"]}`),
		},
		{
			Name:        toolListOrganizationCollectibles,
			Description: "List collectibles held by an organization.",
			Scope:       agent.ScopeCollectiblesRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"organization_id":{"type":"string"}},"required":["organization_id"]}`),
		},
		{
			Name:        toolListTeamCollectibles,
			Description: "List collectibles held by a team.",
			Scope:       agent.ScopeCollectiblesRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"team_id":{"type":"string"}},"required":["team_id"]}`),
		},
	}
}

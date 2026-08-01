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
	toolAwardCollectible             = "sharecrop.award_collectible"
	toolTransferOrgCollectible       = "sharecrop.transfer_org_collectible"
	toolAddCatalogEntry              = "sharecrop.add_catalog_entry"
	toolWithdrawCatalogEntry         = "sharecrop.withdraw_catalog_entry"
	toolReleaseCatalogEntry          = "sharecrop.release_catalog_entry"
	toolDeleteCatalogEntry           = "sharecrop.delete_catalog_entry"
	toolWithdrawCollectible          = "sharecrop.withdraw_collectible"
	toolReleaseCollectible           = "sharecrop.release_collectible"
	toolDeleteWithdrawnCollectible   = "sharecrop.delete_withdrawn_collectible"
)

func collectiblesToolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			Name:        toolMintCollectible,
			Description: "Mint a new collectible owned by the agent's user. kind is unique, edition, or badge; transfer_policy is non_transferable_except_payout, transferable_between_users, transferable_within_organization, or issuer_controlled.",
			Scope:       agent.ScopeCollectiblesManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"kind":{"type":"string","enum":["unique","edition","badge"]},"transfer_policy":{"type":"string","enum":["non_transferable_except_payout","transferable_between_users","transferable_within_organization","issuer_controlled"]},"art":{"type":"string"}},"required":["name","kind","transfer_policy"]}`),
		},
		{
			Name:        toolCollectibleCatalog,
			Description: "List every platform collectible catalog entry (the templates a platform admin can award), each with its lifecycle state (available or withdrawn), max_editions (1 for uniques, the run size for editions, 0 for uncapped badges), and minted_count (live instances). Withdrawn entries stay listed — their instances remain in circulation — but awarding from one is refused.",
			Scope:       agent.ScopeCollectiblesRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
		{
			Name:        toolTransferCollectible,
			Description: "Transfer an owned, transferable collectible. target_kind \"user\" (the default) moves it to another user; target_kind \"organization\" donates it to an organization's trophy case. recipient_id is the matching user or organization id.",
			Scope:       agent.ScopeCollectiblesManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"collectible_id":{"type":"string"},"target_kind":{"type":"string","enum":["user","organization"]},"recipient_id":{"type":"string"}},"required":["collectible_id","recipient_id"]}`),
		},
		{
			Name:        toolTransferOrgCollectible,
			Description: "Transfer an organization-owned collectible to a user. A personal agent credential acts as its user, who must hold the organization's manage_collectibles permission (verified in the transfer transaction); an organization credential acts for its own organization only, and the recipient must then be an active member of the organization.",
			Scope:       agent.ScopeCollectiblesManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"organization_id":{"type":"string"},"collectible_id":{"type":"string"},"recipient_id":{"type":"string"}},"required":["collectible_id","recipient_id"]}`),
		},
		{
			Name:        toolListCollectibles,
			Description: "List the agent's user's own collectibles.",
			Scope:       agent.ScopeCollectiblesRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}}}`),
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
			InputSchema: json.RawMessage(`{"type":"object","properties":{"organization_id":{"type":"string"},"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}},"required":["organization_id"]}`),
		},
		{
			Name:        toolListTeamCollectibles,
			Description: "List collectibles held by a team.",
			Scope:       agent.ScopeCollectiblesRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"team_id":{"type":"string"},"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}},"required":["team_id"]}`),
		},
		{
			Name:        toolAwardCollectible,
			Description: "Mint a fresh copy of a default catalog collectible (by slug) owned by a recipient user, team, or organization. Awarding from a withdrawn entry, or past a unique/edition cap, is refused. Requires platform admin access.",
			Scope:       agent.ScopePlatformAdmin,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"slug":{"type":"string"},"recipient_kind":{"type":"string"},"recipient_id":{"type":"string"},"organization_id":{"type":"string"}},"required":["slug","recipient_id"]}`),
		},
		{
			Name:        toolAddCatalogEntry,
			Description: "Add a collectible template to the platform catalog so it can be awarded. kind is unique, edition, or badge; max_editions must be 1 for unique, the positive run size for edition, and absent (0) for uncapped badges; art names a sprite from the fixed art registry. Requires platform admin access.",
			Scope:       agent.ScopePlatformAdmin,
			InputSchema: json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"slug":{"type":"string"},"name":{"type":"string"},"kind":{"type":"string","enum":["unique","edition","badge"]},"transfer_policy":{"type":"string","enum":["non_transferable_except_payout","transferable_between_users","transferable_within_organization","issuer_controlled"]},"art":{"type":"string"},"max_editions":{"type":"integer","minimum":0}},"required":["slug","name","kind","transfer_policy"]}`),
		},
		{
			Name:        toolWithdrawCatalogEntry,
			Description: "Withdraw a catalog entry so it can no longer be awarded; existing instances stay in circulation. Requires platform admin access.",
			Scope:       agent.ScopePlatformAdmin,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"slug":{"type":"string"}},"required":["slug"]}`),
		},
		{
			Name:        toolReleaseCatalogEntry,
			Description: "Release a withdrawn catalog entry back to the available state so it can be awarded again; an entry that is not withdrawn is refused with a conflict. Requires platform admin access.",
			Scope:       agent.ScopePlatformAdmin,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"slug":{"type":"string"}},"required":["slug"]}`),
		},
		{
			Name:        toolDeleteCatalogEntry,
			Description: "Delete a withdrawn catalog entry that no instance references — live or withdrawn; a not-yet-withdrawn entry or one with remaining instances is refused with a conflict. Requires platform admin access.",
			Scope:       agent.ScopePlatformAdmin,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"slug":{"type":"string"}},"required":["slug"]}`),
		},
		{
			Name:        toolWithdrawCollectible,
			Description: "Withdraw a catalog-minted collectible instance from its holder; the former holder is notified through the collectible_withdrawn event. Requires platform admin access.",
			Scope:       agent.ScopePlatformAdmin,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"collectible_id":{"type":"string"}},"required":["collectible_id"]}`),
		},
		{
			Name:        toolReleaseCollectible,
			Description: "Release a withdrawn collectible instance back into its holder's inventory (owner unchanged); the holder is notified through the collectible_released event. A non-withdrawn instance, or a unique whose live slot was re-minted while this instance was withdrawn, is refused with a conflict. Requires platform admin access.",
			Scope:       agent.ScopePlatformAdmin,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"collectible_id":{"type":"string"}},"required":["collectible_id"]}`),
		},
		{
			Name:        toolDeleteWithdrawnCollectible,
			Description: "Hard-delete a withdrawn collectible instance; every other state is refused with a conflict. Requires platform admin access.",
			Scope:       agent.ScopePlatformAdmin,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"collectible_id":{"type":"string"}},"required":["collectible_id"]}`),
		},
	}
}

package mcp

import (
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/agent"
)

const (
	toolCreateWebhookSubscription = "sharecrop.create_webhook_subscription"
	toolListWebhookSubscriptions  = "sharecrop.list_webhook_subscriptions"
	toolRevokeWebhookSubscription = "sharecrop.revoke_webhook_subscription"
	toolListWebhookDeliveries     = "sharecrop.list_webhook_deliveries"
)

func webhooksToolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			Name:        toolCreateWebhookSubscription,
			Description: "Create an outbound webhook subscription for the caller (or, with organization_id, for an organization the caller administers). The credential must also hold the read scope matching each subscribed event kind. The response includes the signing secret exactly once.",
			Scope:       agent.ScopeWebhooksManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"url":{"type":"string"},"kinds":{"type":"array","items":{"type":"string"}},"organization_id":{"type":"string"}},"required":["url","kinds"]}`),
		},
		{
			Name:        toolListWebhookSubscriptions,
			Description: "List the caller's webhook subscriptions (or, with organization_id, an organization's).",
			Scope:       agent.ScopeWebhooksRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"organization_id":{"type":"string"}}}`),
		},
		{
			Name:        toolRevokeWebhookSubscription,
			Description: "Revoke a webhook subscription.",
			Scope:       agent.ScopeWebhooksManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"subscription_id":{"type":"string"},"organization_id":{"type":"string"}},"required":["subscription_id"]}`),
		},
		{
			Name:        toolListWebhookDeliveries,
			Description: "List the delivery attempts recorded for a webhook subscription.",
			Scope:       agent.ScopeWebhooksRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"subscription_id":{"type":"string"},"organization_id":{"type":"string"}},"required":["subscription_id"]}`),
		},
	}
}

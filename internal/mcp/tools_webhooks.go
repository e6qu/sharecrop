package mcp

import (
	"encoding/json"
	"strings"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/event"
)

const (
	toolCreateWebhookSubscription = "sharecrop.create_webhook_subscription"
	toolListWebhookSubscriptions  = "sharecrop.list_webhook_subscriptions"
	toolRevokeWebhookSubscription = "sharecrop.revoke_webhook_subscription"
	toolListWebhookDeliveries     = "sharecrop.list_webhook_deliveries"
)

// webhookKindsEnumJSON renders every event kind as a JSON Schema enum, so the
// create tool's input schema names the valid kinds instead of leaving agents
// to guess them.
func webhookKindsEnumJSON() string {
	kinds := event.AllKinds()
	values := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		values = append(values, `"`+kind.String()+`"`)
	}
	return "[" + strings.Join(values, ",") + "]"
}

func webhooksToolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			Name:        toolCreateWebhookSubscription,
			Description: "Create an outbound webhook subscription for the caller (or, with organization_id, for an organization the caller administers). audience is \"recipient\" (the default: events addressed to the owner) or \"marketplace\" (every public open task's task_opened event — the push alternative to polling list_tasks; kinds must then be exactly [\"task_opened\"]). filter_task_type and filter_min_credit_reward narrow a marketplace subscription and are valid only with it. The credential must also hold the read scope matching each subscribed event kind. The response includes the signing secret exactly once.",
			Access:      toolNeedsScope{Value: agent.ScopeWebhooksManage},
			InputSchema: json.RawMessage(`{"type":"object","properties":{"url":{"type":"string"},"kinds":{"type":"array","items":{"type":"string","enum":` + webhookKindsEnumJSON() + `}},"organization_id":{"type":"string"},"audience":{"type":"string","enum":["recipient","marketplace"]},"filter_task_type":{"type":"string","enum":["general","code_review","security_review","product_review","ui_ux_review","qa_testing","document_review","documentation_writing","diagram_writing","planning","research","data_extraction","troubleshooting","code_analysis","architecture_review","threat_analysis"]},"filter_min_credit_reward":{"type":"integer","minimum":1}},"required":["url","kinds"]}`),
		},
		{
			Name:        toolListWebhookSubscriptions,
			Description: "List the caller's webhook subscriptions (or, with organization_id, an organization's). Rows carry audience and the marketplace filter fields.",
			Access:      toolNeedsScope{Value: agent.ScopeWebhooksRead},
			InputSchema: json.RawMessage(`{"type":"object","properties":{"organization_id":{"type":"string"},"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}}}`),
		},
		{
			Name:        toolRevokeWebhookSubscription,
			Description: "Revoke a webhook subscription.",
			Access:      toolNeedsScope{Value: agent.ScopeWebhooksManage},
			InputSchema: json.RawMessage(`{"type":"object","properties":{"subscription_id":{"type":"string"},"organization_id":{"type":"string"}},"required":["subscription_id"]}`),
		},
		{
			Name:        toolListWebhookDeliveries,
			Description: "List the delivery attempts recorded for a webhook subscription.",
			Access:      toolNeedsScope{Value: agent.ScopeWebhooksRead},
			InputSchema: json.RawMessage(`{"type":"object","properties":{"subscription_id":{"type":"string"},"organization_id":{"type":"string"},"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}},"required":["subscription_id"]}`),
		},
	}
}

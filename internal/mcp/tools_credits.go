package mcp

import (
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/agent"
)

const (
	toolGetCreditBalance = "sharecrop.get_credit_balance"
	toolListLedger       = "sharecrop.list_ledger"
	toolGrantCredits     = "sharecrop.grant_credits"
	toolSendCredits      = "sharecrop.send_credits"
)

func creditsToolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			Name:        toolGetCreditBalance,
			Description: "Report the agent's user's credit balance: spendable credits and credits currently allocated (escrowed) onto tasks.",
			Access:      toolNeedsScope{Value: agent.ScopeLedgerRead},
			InputSchema: json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{}}`),
		},
		{
			Name:        toolListLedger,
			Description: "List the agent's user's credit ledger entries, newest first. Entries carry a note when one was recorded (for example the explanation on a platform-admin credit grant). Optional limit/offset page the listing.",
			Access:      toolNeedsScope{Value: agent.ScopeLedgerRead},
			InputSchema: json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}}}`),
		},
		{
			Name:        toolGrantCredits,
			Description: "Grant credits to a user or organization account as a platform-admin manual adjustment, recording the required note on the ledger entry. Replaying the same idempotency_key returns the original entry without double-crediting. Requires platform admin access.",
			Access:      toolNeedsScope{Value: agent.ScopePlatformAdmin},
			InputSchema: json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"target_kind":{"type":"string","enum":["user","organization"]},"target_id":{"type":"string"},"amount":{"type":"integer","minimum":1},"note":{"type":"string"},"idempotency_key":{"type":"string"}},"required":["target_kind","target_id","amount","note","idempotency_key"]}`),
		},
		{
			Name:        toolSendCredits,
			Description: "Send spendable credits to another user or organization, mirroring REST's peer transfer. source_kind is \"self\" (the agent's user's own balance) or \"organization\" (an organization balance the user may spend from, named by source_organization_id); target_kind is \"user\" or \"organization\" with target_id the matching id. amount is in credit base units, note is an optional message recorded on both ledger rows, and replaying the same idempotency_key returns the original transfer without double-sending. Requires the ledger_write scope, so credentials minted before that scope existed must be re-minted to send.",
			Access:      toolNeedsScope{Value: agent.ScopeLedgerWrite},
			InputSchema: json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"source_kind":{"type":"string","enum":["self","organization"]},"source_organization_id":{"type":"string"},"target_kind":{"type":"string","enum":["user","organization"]},"target_id":{"type":"string"},"amount":{"type":"integer","minimum":1},"note":{"type":"string"},"idempotency_key":{"type":"string"}},"required":["source_kind","target_kind","target_id","amount","idempotency_key"]}`),
		},
	}
}

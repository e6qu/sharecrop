package mcp

import (
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/agent"
)

const (
	toolGetCreditBalance = "sharecrop.get_credit_balance"
	toolListLedger       = "sharecrop.list_ledger"
	toolGrantCredits     = "sharecrop.grant_credits"
)

func creditsToolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			Name:        toolGetCreditBalance,
			Description: "Report the agent's user's credit balance: spendable credits and credits currently allocated (escrowed) onto tasks.",
			Scope:       agent.ScopeLedgerRead,
			InputSchema: json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{}}`),
		},
		{
			Name:        toolListLedger,
			Description: "List the agent's user's credit ledger entries, newest first. Entries carry a note when one was recorded (for example the explanation on a platform-admin credit grant). Optional limit/offset page the listing.",
			Scope:       agent.ScopeLedgerRead,
			InputSchema: json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}}}`),
		},
		{
			Name:        toolGrantCredits,
			Description: "Grant credits to a user or organization account as a platform-admin manual adjustment, recording the required note on the ledger entry. Replaying the same idempotency_key returns the original entry without double-crediting. Requires platform admin access.",
			Scope:       agent.ScopePlatformAdmin,
			InputSchema: json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"target_kind":{"type":"string","enum":["user","organization"]},"target_id":{"type":"string"},"amount":{"type":"integer","minimum":1},"note":{"type":"string"},"idempotency_key":{"type":"string"}},"required":["target_kind","target_id","amount","note","idempotency_key"]}`),
		},
	}
}

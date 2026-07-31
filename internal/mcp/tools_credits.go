package mcp

import (
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/agent"
)

const (
	toolGetCreditBalance = "sharecrop.get_credit_balance"
	toolListLedger       = "sharecrop.list_ledger"
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
			Description: "List the agent's user's credit ledger entries, newest first. Optional limit/offset page the listing.",
			Scope:       agent.ScopeLedgerRead,
			InputSchema: json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}}}`),
		},
	}
}

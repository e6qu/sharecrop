package mcp

import (
	"context"
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/ledger"
)

type creditBalancePayload struct {
	SpendableCredits int64 `json:"spendable_credits"`
	AllocatedCredits int64 `json:"allocated_credits"`
}

type ledgerEntrySummary struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Amount int64  `json:"amount"`
	TaskID string `json:"task_id"`
}

type ledgerEntriesPayload struct {
	Entries []ledgerEntrySummary `json:"entries"`
}

func (creditBalancePayload) payloadValue() {}

func (ledgerEntriesPayload) payloadValue() {}

func (server Server) callGetCreditBalance(ctx context.Context, subject auth.UserSubject) toolResult {
	result := server.services.GetCreditBalance(ctx, subject.ID)
	found, matched := result.(ledger.BalanceFound)
	if !matched {
		return toolFailed{code: result.(ledger.BalanceRejected).Reason.Code(), message: result.(ledger.BalanceRejected).Reason.Description()}
	}
	return marshalPayload(creditBalancePayload{
		SpendableCredits: found.Value.Spendable(),
		AllocatedCredits: found.Value.Allocated(),
	})
}

func (server Server) callListLedger(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	page, pageProblem := parseMCPPageArguments(arguments)
	if pageProblem != nil {
		return pageProblem
	}
	result := server.services.ListLedger(ctx, subject.ID, page)
	listed, matched := result.(ledger.EntriesListed)
	if !matched {
		return toolFailed{code: result.(ledger.ListEntriesRejected).Reason.Code(), message: result.(ledger.ListEntriesRejected).Reason.Description()}
	}
	entries := make([]ledgerEntrySummary, 0, len(listed.Values))
	for index := range listed.Values {
		entry := listed.Values[index]
		taskID := ""
		if referenced, taskMatched := entry.TaskRef.(ledger.TaskReferenced); taskMatched {
			taskID = referenced.TaskID.String()
		}
		entries = append(entries, ledgerEntrySummary{
			ID:     entry.ID.String(),
			Kind:   entry.Kind.String(),
			Amount: entry.Amount.Int64(),
			TaskID: taskID,
		})
	}
	return marshalPayload(ledgerEntriesPayload{Entries: entries})
}

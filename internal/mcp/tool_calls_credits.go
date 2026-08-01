package mcp

import (
	"context"
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
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
	Note   string `json:"note"`
}

type ledgerEntriesPayload struct {
	Entries    []ledgerEntrySummary `json:"entries"`
	NextOffset int                  `json:"next_offset"`
	// Total counts every row matching the filter, ignoring limit/offset.
	Total int64 `json:"total"`
}

// creditGrantPayload mirrors REST's creditGrantResponse field names.
type creditGrantPayload struct {
	EntryID string `json:"entry_id"`
	Amount  int64  `json:"amount"`
}

func (creditBalancePayload) payloadValue() {}

func (ledgerEntriesPayload) payloadValue() {}

func (creditGrantPayload) payloadValue() {}

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
	result := server.services.ListLedger(ctx, subject.ID, page.Probe())
	listed, matched := result.(ledger.EntriesListed)
	if !matched {
		return toolFailed{code: result.(ledger.ListEntriesRejected).Reason.Code(), message: result.(ledger.ListEntriesRejected).Reason.Description()}
	}
	visible, nextOffset := core.ProbeListWindow(len(listed.Values), page)
	entries := make([]ledgerEntrySummary, 0, visible)
	for index := range listed.Values[:visible] {
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
			Note:   entry.Note,
		})
	}
	return marshalPayload(ledgerEntriesPayload{Entries: entries, NextOffset: nextOffset, Total: listed.Total})
}

// callGrantCredits is the platform-admin manual credit grant over MCP,
// mirroring REST's POST /api/admin/credits/grants field-for-field. The admin
// re-check happened in dispatch (requireAdminSubjectForTool).
func (server Server) callGrantCredits(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		TargetKind     string `json:"target_kind"`
		TargetID       string `json:"target_id"`
		Amount         int64  `json:"amount"`
		Note           string `json:"note"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}

	target, targetProblem := parseMCPGrantTarget(args.TargetKind, args.TargetID)
	if targetProblem != nil {
		return targetProblem
	}
	amountResult := ledger.NewCreditAmount(args.Amount)
	amount, amountMatched := amountResult.(ledger.CreditAmountAccepted)
	if !amountMatched {
		return toolProtocolError{code: codeInvalidParams, message: amountResult.(ledger.CreditAmountRejected).Reason.Description()}
	}
	noteResult := ledger.NewGrantNote(args.Note)
	note, noteMatched := noteResult.(ledger.GrantNoteAccepted)
	if !noteMatched {
		return toolProtocolError{code: codeInvalidParams, message: noteResult.(ledger.GrantNoteRejected).Reason.Description()}
	}
	keyResult := ledger.NewIdempotencyKey(args.IdempotencyKey)
	key, keyMatched := keyResult.(ledger.IdempotencyKeyAccepted)
	if !keyMatched {
		return toolProtocolError{code: codeInvalidParams, message: keyResult.(ledger.IdempotencyKeyRejected).Reason.Description()}
	}

	result := server.services.GrantCredits(ctx, subject.ID, target, amount.Value, note.Value, key.Value)
	granted, matched := result.(ledger.CreditsGranted)
	if !matched {
		return toolFailed{code: result.(ledger.GrantRejected).Reason.Code(), message: result.(ledger.GrantRejected).Reason.Description()}
	}
	return marshalPayload(creditGrantPayload{EntryID: granted.EntryID.String(), Amount: granted.Amount.Int64()})
}

func parseMCPGrantTarget(rawKind string, rawID string) (ledger.GrantTarget, toolResult) {
	switch rawKind {
	case "user":
		userIDResult := core.ParseUserID(rawID)
		userID, matched := userIDResult.(core.UserIDCreated)
		if !matched {
			return nil, toolProtocolError{code: codeInvalidParams, message: userIDResult.(core.UserIDRejected).Reason.Description()}
		}
		return ledger.GrantToUser{ID: userID.Value}, nil
	case "organization":
		organizationIDResult := core.ParseOrganizationID(rawID)
		organizationID, matched := organizationIDResult.(core.OrganizationIDCreated)
		if !matched {
			return nil, toolProtocolError{code: codeInvalidParams, message: organizationIDResult.(core.OrganizationIDRejected).Reason.Description()}
		}
		return ledger.GrantToOrganization{ID: organizationID.Value}, nil
	default:
		return nil, toolProtocolError{code: codeInvalidParams, message: "target_kind must be user or organization"}
	}
}

package httpserver

import (
	"net/http"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
)

func (server Server) creditsBalance(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	result := server.ledgerService.Balance(r.Context(), actor.subject.ID)
	found, matched := result.(ledger.BalanceFound)
	if !matched {
		rejected := result.(ledger.BalanceRejected)
		writeDomainError(w, rejected.Reason)
		return
	}

	writeJSON(w, http.StatusOK, balanceResponse{Amount: found.Value.Int64()})
}
func (server Server) creditsLedger(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	result := server.ledgerService.ListEntries(r.Context(), actor.subject.ID, parsePage(r))
	listed, matched := result.(ledger.EntriesListed)
	if !matched {
		rejected := result.(ledger.ListEntriesRejected)
		writeDomainError(w, rejected.Reason)
		return
	}

	response := ledgerListResponse{Entries: make([]ledgerEntryResponse, 0, len(listed.Values))}
	for index := range listed.Values {
		response.Entries = append(response.Entries, ledgerEntryToResponse(listed.Values[index]))
	}
	writeJSON(w, http.StatusOK, response)
}
func ledgerEntryToResponse(entry ledger.LedgerEntry) ledgerEntryResponse {
	taskID := ""
	if referenced, matched := entry.TaskRef.(ledger.TaskReferenced); matched {
		taskID = referenced.TaskID.String()
	}
	return ledgerEntryResponse{
		ID:     entry.ID.String(),
		Kind:   entry.Kind.String(),
		Amount: entry.Amount.Int64(),
		TaskID: taskID,
	}
}
func escrowToResponse(escrow ledger.TaskEscrow) taskEscrowResponse {
	return taskEscrowResponse{
		TaskID: escrow.TaskID.String(),
		Amount: escrow.Amount.Int64(),
		State:  escrow.State.String(),
	}
}
func collectibleIDStrings(ids []core.CollectibleID) []string {
	values := make([]string, 0, len(ids))
	for index := range ids {
		values = append(values, ids[index].String())
	}
	return values
}

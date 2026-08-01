package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
)

func (server Server) creditsBalance(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, rejected.reason)
		return
	}

	result := server.ledgerService.Balance(r.Context(), actor.subject.ID)
	found, matched := result.(ledger.BalanceFound)
	if !matched {
		rejected := result.(ledger.BalanceRejected)
		writeDomainError(w, rejected.Reason)
		return
	}

	writeJSON(w, http.StatusOK, balanceResponse{SpendableCredits: found.Value.Spendable(), AllocatedCredits: found.Value.Allocated()})
}
func (server Server) creditsLedger(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, rejected.reason)
		return
	}

	page, pageOK := parsePageOrReject(w, r)
	if !pageOK {
		return
	}
	result := server.ledgerService.ListEntries(r.Context(), actor.subject.ID, page.Probe())
	listed, matched := result.(ledger.EntriesListed)
	if !matched {
		rejected := result.(ledger.ListEntriesRejected)
		writeDomainError(w, rejected.Reason)
		return
	}

	visible, nextOffset := probeListWindow(len(listed.Values), page)
	response := ledgerListResponse{Entries: make([]ledgerEntryResponse, 0, visible), NextOffset: nextOffset, Total: listed.Total}
	for index := range listed.Values[:visible] {
		response.Entries = append(response.Entries, ledgerEntryToResponse(listed.Values[index]))
	}
	writeJSON(w, http.StatusOK, response)
}

// grantCredits is the platform-admin manual credit grant: it credits a user
// or organization account, records the required note, and is idempotent per
// idempotency key.
func (server Server) grantCredits(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.requireAdminSubject(w, r)
	if !ok {
		return
	}
	if !server.allowBySubject(w, actor.subject.ID.String()) {
		return
	}

	var request creditGrantRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
		return
	}

	targetResult := parseGrantTarget(request.TargetKind, request.TargetID)
	target, targetMatched := targetResult.(grantTargetAccepted)
	if !targetMatched {
		writeDomainError(w, targetResult.(grantTargetRejected).reason)
		return
	}

	amountResult := ledger.NewCreditAmount(request.Amount)
	amount, amountMatched := amountResult.(ledger.CreditAmountAccepted)
	if !amountMatched {
		writeDomainError(w, amountResult.(ledger.CreditAmountRejected).Reason)
		return
	}

	noteResult := ledger.NewGrantNote(request.Note)
	note, noteMatched := noteResult.(ledger.GrantNoteAccepted)
	if !noteMatched {
		writeDomainError(w, noteResult.(ledger.GrantNoteRejected).Reason)
		return
	}

	keyResult := ledger.NewIdempotencyKey(request.IdempotencyKey)
	key, keyMatched := keyResult.(ledger.IdempotencyKeyAccepted)
	if !keyMatched {
		writeDomainError(w, keyResult.(ledger.IdempotencyKeyRejected).Reason)
		return
	}

	result := server.ledgerService.GrantCredits(r.Context(), actor.subject.ID, target.value, amount.Value, note.Value, key.Value)
	granted, matched := result.(ledger.CreditsGranted)
	if !matched {
		writeDomainError(w, result.(ledger.GrantRejected).Reason)
		return
	}
	writeJSON(w, http.StatusCreated, creditGrantResponse{EntryID: granted.EntryID.String(), Amount: granted.Amount.Int64()})
}

// sendCredits is the peer credit send: it moves credits from the caller's own
// balance (or an organization balance the caller may spend from) to another
// user or organization account, idempotently per (sender account, key).
func (server Server) sendCredits(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, rejected.reason)
		return
	}
	if !server.allowBySubject(w, actor.subject.ID.String()) {
		return
	}

	var request creditTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
		return
	}

	sourceResult := parseTransferSource(request.SourceKind, request.SourceOrganizationID)
	source, sourceMatched := sourceResult.(transferSourceAccepted)
	if !sourceMatched {
		writeDomainError(w, sourceResult.(transferSourceRejected).reason)
		return
	}

	targetResult := parseTransferTarget(request.TargetKind, request.TargetID)
	target, targetMatched := targetResult.(transferTargetAccepted)
	if !targetMatched {
		writeDomainError(w, targetResult.(transferTargetRejected).reason)
		return
	}

	amountResult := ledger.NewCreditAmount(request.Amount)
	amount, amountMatched := amountResult.(ledger.CreditAmountAccepted)
	if !amountMatched {
		writeDomainError(w, amountResult.(ledger.CreditAmountRejected).Reason)
		return
	}

	noteResult := parseTransferNote(request.Note)
	note, noteMatched := noteResult.(transferNoteAccepted)
	if !noteMatched {
		writeDomainError(w, noteResult.(transferNoteRejected).reason)
		return
	}

	keyResult := ledger.NewIdempotencyKey(request.IdempotencyKey)
	key, keyMatched := keyResult.(ledger.IdempotencyKeyAccepted)
	if !keyMatched {
		writeDomainError(w, keyResult.(ledger.IdempotencyKeyRejected).Reason)
		return
	}

	result := server.ledgerService.SendCredits(r.Context(), actor.subject.ID, source.value, target.value, amount.Value, note.value, key.Value)
	sent, matched := result.(ledger.CreditsSent)
	if !matched {
		writeDomainError(w, result.(ledger.SendRejected).Reason)
		return
	}
	writeJSON(w, http.StatusCreated, creditTransferResponse{EntryID: sent.DebitEntryID.String(), Amount: sent.Amount.Int64()})
}

type transferSourceResult interface {
	transferSourceResult()
}

type transferSourceAccepted struct {
	value ledger.TransferSource
}

type transferSourceRejected struct {
	reason core.DomainError
}

func (transferSourceAccepted) transferSourceResult() {}

func (transferSourceRejected) transferSourceResult() {}

func parseTransferSource(rawKind string, rawOrganizationID string) transferSourceResult {
	switch rawKind {
	case "self":
		return transferSourceAccepted{value: ledger.TransferFromSelf{}}
	case "organization":
		organizationIDResult := core.ParseOrganizationID(rawOrganizationID)
		organizationID, matched := organizationIDResult.(core.OrganizationIDCreated)
		if !matched {
			return transferSourceRejected{reason: organizationIDResult.(core.OrganizationIDRejected).Reason}
		}
		return transferSourceAccepted{value: ledger.TransferFromOrganization{ID: organizationID.Value}}
	default:
		return transferSourceRejected{reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "credit transfer source kind must be self or organization")}
	}
}

type transferTargetResult interface {
	transferTargetResult()
}

type transferTargetAccepted struct {
	value ledger.TransferTarget
}

type transferTargetRejected struct {
	reason core.DomainError
}

func (transferTargetAccepted) transferTargetResult() {}

func (transferTargetRejected) transferTargetResult() {}

func parseTransferTarget(rawKind string, rawID string) transferTargetResult {
	switch rawKind {
	case "user":
		userIDResult := core.ParseUserID(rawID)
		userID, matched := userIDResult.(core.UserIDCreated)
		if !matched {
			return transferTargetRejected{reason: userIDResult.(core.UserIDRejected).Reason}
		}
		return transferTargetAccepted{value: ledger.TransferToUser{ID: userID.Value}}
	case "organization":
		organizationIDResult := core.ParseOrganizationID(rawID)
		organizationID, matched := organizationIDResult.(core.OrganizationIDCreated)
		if !matched {
			return transferTargetRejected{reason: organizationIDResult.(core.OrganizationIDRejected).Reason}
		}
		return transferTargetAccepted{value: ledger.TransferToOrganization{ID: organizationID.Value}}
	default:
		return transferTargetRejected{reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "credit transfer target kind must be user or organization")}
	}
}

type transferNoteResult interface {
	transferNoteResult()
}

type transferNoteAccepted struct {
	value ledger.TransferNote
}

type transferNoteRejected struct {
	reason core.DomainError
}

func (transferNoteAccepted) transferNoteResult() {}

func (transferNoteRejected) transferNoteResult() {}

// parseTransferNote maps the optional wire note: absent/blank is explicitly
// no note, anything else must validate as a note.
func parseTransferNote(raw string) transferNoteResult {
	if strings.TrimSpace(raw) == "" {
		return transferNoteAccepted{value: ledger.NoTransferNote{}}
	}
	noteResult := ledger.NewGrantNote(raw)
	note, matched := noteResult.(ledger.GrantNoteAccepted)
	if !matched {
		return transferNoteRejected{reason: noteResult.(ledger.GrantNoteRejected).Reason}
	}
	return transferNoteAccepted{value: ledger.TransferNoteProvided{Note: note.Value}}
}

type grantTargetResult interface {
	grantTargetResult()
}

type grantTargetAccepted struct {
	value ledger.GrantTarget
}

type grantTargetRejected struct {
	reason core.DomainError
}

func (grantTargetAccepted) grantTargetResult() {}

func (grantTargetRejected) grantTargetResult() {}

func parseGrantTarget(rawKind string, rawID string) grantTargetResult {
	switch rawKind {
	case "user":
		userIDResult := core.ParseUserID(rawID)
		userID, matched := userIDResult.(core.UserIDCreated)
		if !matched {
			return grantTargetRejected{reason: userIDResult.(core.UserIDRejected).Reason}
		}
		return grantTargetAccepted{value: ledger.GrantToUser{ID: userID.Value}}
	case "organization":
		organizationIDResult := core.ParseOrganizationID(rawID)
		organizationID, matched := organizationIDResult.(core.OrganizationIDCreated)
		if !matched {
			return grantTargetRejected{reason: organizationIDResult.(core.OrganizationIDRejected).Reason}
		}
		return grantTargetAccepted{value: ledger.GrantToOrganization{ID: organizationID.Value}}
	default:
		return grantTargetRejected{reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "credit grant target kind is invalid")}
	}
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
		Note:   entry.Note,
	}
}
func fundToResponse(fund ledger.TaskFund) taskFundResponse {
	return taskFundResponse{
		TaskID:       fund.TaskID.String(),
		CreditAmount: fund.CreditAmount.Int64(),
	}
}
func collectibleIDStrings(ids []core.CollectibleID) []string {
	values := make([]string, 0, len(ids))
	for index := range ids {
		values = append(values, ids[index].String())
	}
	return values
}

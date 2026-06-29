package httpserver

import (
	"net/http"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/org"
)

func (server Server) fundTaskFromOrganization(w http.ResponseWriter, r *http.Request, actor auth.UserSubject, taskID core.TaskID, amount ledger.CreditAmount, key ledger.IdempotencyKey, rawOrganizationID string) {
	organizationIDResult := core.ParseOrganizationID(rawOrganizationID)
	organizationID, organizationMatched := organizationIDResult.(core.OrganizationIDCreated)
	if !organizationMatched {
		writeDomainError(w, organizationIDResult.(core.OrganizationIDRejected).Reason)
		return
	}

	check := server.organizationService.CheckOrganizationPermission(r.Context(), organizationID.Value, actor.ID, org.PermissionManageBilling)
	if rejected, denied := check.(org.PermissionDenied); denied {
		writeDomainError(w, rejected.Reason)
		return
	}

	result := server.ledgerService.FundTaskFromOrganization(r.Context(), organizationID.Value, taskID, amount, key)
	funded, matched := result.(ledger.TaskFunded)
	if !matched {
		writeDomainError(w, result.(ledger.FundRejected).Reason)
		return
	}

	writeJSON(w, http.StatusCreated, escrowToResponse(funded.Escrow))
}

func (server Server) organizationCreditsBalance(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := server.requireOrganizationBilling(w, r)
	if !ok {
		return
	}

	result := server.ledgerService.OrganizationBalance(r.Context(), organizationID)
	found, matched := result.(ledger.BalanceFound)
	if !matched {
		writeDomainError(w, result.(ledger.BalanceRejected).Reason)
		return
	}

	writeJSON(w, http.StatusOK, balanceResponse{Amount: found.Value.Int64()})
}

func (server Server) organizationCreditsLedger(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := server.requireOrganizationBilling(w, r)
	if !ok {
		return
	}

	result := server.ledgerService.ListOrganizationEntries(r.Context(), organizationID, parsePage(r))
	listed, matched := result.(ledger.EntriesListed)
	if !matched {
		writeDomainError(w, result.(ledger.ListEntriesRejected).Reason)
		return
	}

	response := ledgerListResponse{Entries: make([]ledgerEntryResponse, 0, len(listed.Values))}
	for index := range listed.Values {
		response.Entries = append(response.Entries, ledgerEntryToResponse(listed.Values[index]))
	}
	writeJSON(w, http.StatusOK, response)
}

func (server Server) requireOrganizationBilling(w http.ResponseWriter, r *http.Request) (core.OrganizationID, bool) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return core.OrganizationID{}, false
	}

	organizationIDResult := parseOrganizationPathValue(r)
	organizationIDAccepted, organizationIDMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationIDMatched {
		writeError(w, http.StatusBadRequest, organizationIDResult.(organizationIDRejected).reason)
		return core.OrganizationID{}, false
	}

	check := server.organizationService.CheckOrganizationPermission(r.Context(), organizationIDAccepted.value, actor.subject.ID, org.PermissionManageBilling)
	if rejected, denied := check.(org.PermissionDenied); denied {
		writeDomainError(w, rejected.Reason)
		return core.OrganizationID{}, false
	}
	return organizationIDAccepted.value, true
}

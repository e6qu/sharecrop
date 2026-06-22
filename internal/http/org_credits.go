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
		writeError(w, http.StatusBadRequest, organizationIDResult.(core.OrganizationIDRejected).Reason.Description())
		return
	}

	check := server.organizationService.CheckOrganizationPermission(r.Context(), organizationID.Value, actor.ID, org.PermissionManageBilling)
	if rejected, denied := check.(org.PermissionDenied); denied {
		writeError(w, http.StatusForbidden, rejected.Reason.Description())
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
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}

	organizationIDResult := parseOrganizationPathValue(r)
	organizationIDAccepted, organizationIDMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationIDMatched {
		writeError(w, http.StatusBadRequest, organizationIDResult.(organizationIDRejected).reason)
		return
	}

	check := server.organizationService.CheckOrganizationPermission(r.Context(), organizationIDAccepted.value, actor.subject.ID, org.PermissionManageBilling)
	if rejected, denied := check.(org.PermissionDenied); denied {
		writeError(w, http.StatusForbidden, rejected.Reason.Description())
		return
	}

	result := server.ledgerService.OrganizationBalance(r.Context(), organizationIDAccepted.value)
	found, matched := result.(ledger.BalanceFound)
	if !matched {
		writeError(w, http.StatusBadRequest, result.(ledger.BalanceRejected).Reason.Description())
		return
	}

	writeJSON(w, http.StatusOK, balanceResponse{Amount: found.Value.Int64()})
}

package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
)

func (server Server) fundTask(w http.ResponseWriter, r *http.Request) {
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

	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, taskIDResult.(taskIDRejected).reason)
		return
	}

	var request fundingRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
		return
	}

	amountResult := ledger.NewCreditAmount(request.Amount)
	amount, amountMatched := amountResult.(ledger.CreditAmountAccepted)
	if !amountMatched {
		writeDomainError(w, amountResult.(ledger.CreditAmountRejected).Reason)
		return
	}

	keyResult := ledger.NewIdempotencyKey(request.IdempotencyKey)
	key, keyMatched := keyResult.(ledger.IdempotencyKeyAccepted)
	if !keyMatched {
		writeDomainError(w, keyResult.(ledger.IdempotencyKeyRejected).Reason)
		return
	}

	if request.OrganizationID != "" {
		server.fundTaskFromOrganization(w, r, actor.subject, taskIDAccepted.value, amount.Value, key.Value, request.OrganizationID)
		return
	}

	result := server.ledgerService.FundTask(r.Context(), actor.subject.ID, taskIDAccepted.value, amount.Value, key.Value, ledger.SpendByUser{})
	funded, matched := result.(ledger.TaskFunded)
	if !matched {
		writeDomainError(w, result.(ledger.FundRejected).Reason)
		return
	}

	server.recordAuditBestEffort(r.Context(), actor.subject.ID, audit.ActionTaskFunded, audit.Subject{Kind: "task", ID: funded.Fund.TaskID.String()}, audit.EmptyMetadata())
	writeJSON(w, http.StatusCreated, fundToResponse(funded.Fund))
}

func (server Server) refundTask(w http.ResponseWriter, r *http.Request) {
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

	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, taskIDResult.(taskIDRejected).reason)
		return
	}

	var request acceptSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
		return
	}

	keyResult := ledger.NewIdempotencyKey(request.IdempotencyKey)
	key, keyMatched := keyResult.(ledger.IdempotencyKeyAccepted)
	if !keyMatched {
		writeDomainError(w, keyResult.(ledger.IdempotencyKeyRejected).Reason)
		return
	}

	result := server.ledgerService.RefundTask(r.Context(), actor.subject.ID, taskIDAccepted.value, key.Value)
	refunded, matched := result.(ledger.TaskRefunded)
	if !matched {
		writeDomainError(w, result.(ledger.RefundRejected).Reason)
		return
	}

	server.recordAuditBestEffort(r.Context(), actor.subject.ID, audit.ActionTaskRefunded, audit.Subject{Kind: "task", ID: refunded.Fund.TaskID.String()}, audit.EmptyMetadata())
	writeJSON(w, http.StatusOK, fundToResponse(refunded.Fund))
}

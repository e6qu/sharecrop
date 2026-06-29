package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/ledger"
)

func (server Server) fundTask(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		writeError(w, http.StatusBadRequest, taskIDResult.(taskIDRejected).reason)
		return
	}

	var request fundingRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}

	amountResult := ledger.NewCreditAmount(request.Amount)
	amount, amountMatched := amountResult.(ledger.CreditAmountAccepted)
	if !amountMatched {
		writeError(w, http.StatusBadRequest, amountResult.(ledger.CreditAmountRejected).Reason.Description())
		return
	}

	keyResult := ledger.NewIdempotencyKey(request.IdempotencyKey)
	key, keyMatched := keyResult.(ledger.IdempotencyKeyAccepted)
	if !keyMatched {
		writeError(w, http.StatusBadRequest, keyResult.(ledger.IdempotencyKeyRejected).Reason.Description())
		return
	}

	if request.OrganizationID != "" {
		server.fundTaskFromOrganization(w, r, actor.subject, taskIDAccepted.value, amount.Value, key.Value, request.OrganizationID)
		return
	}

	result := server.ledgerService.FundTask(r.Context(), actor.subject.ID, taskIDAccepted.value, amount.Value, key.Value)
	funded, matched := result.(ledger.TaskFunded)
	if !matched {
		writeDomainError(w, result.(ledger.FundRejected).Reason)
		return
	}

	writeJSON(w, http.StatusCreated, escrowToResponse(funded.Escrow))
}

func (server Server) refundTask(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		writeError(w, http.StatusBadRequest, taskIDResult.(taskIDRejected).reason)
		return
	}

	var request acceptSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}

	keyResult := ledger.NewIdempotencyKey(request.IdempotencyKey)
	key, keyMatched := keyResult.(ledger.IdempotencyKeyAccepted)
	if !keyMatched {
		writeError(w, http.StatusBadRequest, keyResult.(ledger.IdempotencyKeyRejected).Reason.Description())
		return
	}

	result := server.ledgerService.RefundTask(r.Context(), actor.subject.ID, taskIDAccepted.value, key.Value)
	refunded, matched := result.(ledger.TaskRefunded)
	if !matched {
		writeDomainError(w, result.(ledger.RefundRejected).Reason)
		return
	}

	if !server.recordAudit(w, r.Context(), actor.subject.ID, audit.ActionTaskRefunded, audit.Subject{Kind: "task", ID: refunded.Escrow.TaskID.String()}, audit.EmptyMetadata()) {
		return
	}
	writeJSON(w, http.StatusOK, escrowToResponse(refunded.Escrow))
}

package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/submission"
)

func (server Server) acceptSubmission(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	if !server.allowBySubject(w, actor.subject.ID.String()) {
		return
	}

	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		writeError(w, http.StatusBadRequest, taskIDResult.(taskIDRejected).reason)
		return
	}

	submissionIDResult := core.ParseSubmissionID(r.PathValue("submission_id"))
	submissionIDAccepted, submissionIDMatched := submissionIDResult.(core.SubmissionIDCreated)
	if !submissionIDMatched {
		writeError(w, http.StatusBadRequest, submissionIDResult.(core.SubmissionIDRejected).Reason.Description())
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

	creditSelectionResult := acceptCreditSelection(request.PayoutAmount)
	creditSelection, creditSelectionMatched := creditSelectionResult.(creditSelectionAccepted)
	if !creditSelectionMatched {
		writeError(w, http.StatusBadRequest, creditSelectionResult.(creditSelectionRejected).reason)
		return
	}
	tipSelectionResult := tipSelectionFromAmount(request.TipAmount)
	tipSelection, tipSelectionMatched := tipSelectionResult.(tipSelectionAccepted)
	if !tipSelectionMatched {
		writeError(w, http.StatusBadRequest, tipSelectionResult.(tipSelectionRejected).reason)
		return
	}

	collectibleTip := ledger.CollectibleTipSelection(ledger.NoCollectibleTipSelection{})
	if request.TipCollectibleID != "" {
		collectibleIDResult := core.ParseCollectibleID(request.TipCollectibleID)
		collectibleIDAccepted, collectibleIDMatched := collectibleIDResult.(core.CollectibleIDCreated)
		if !collectibleIDMatched {
			writeError(w, http.StatusBadRequest, collectibleIDResult.(core.CollectibleIDRejected).Reason.Description())
			return
		}
		collectibleTip = ledger.CollectibleTipSelected{ID: collectibleIDAccepted.Value}
	}

	submissionResult := server.submissionService.Get(r.Context(), actor.subject, submissionIDAccepted.Value)
	submissionFound, submissionMatched := submissionResult.(submission.SubmissionGot)
	if !submissionMatched {
		writeDomainError(w, submissionResult.(submission.GetRejected).Reason)
		return
	}

	result := server.ledgerService.ReviewAcceptSubmission(r.Context(), actor.subject.ID, taskIDAccepted.value, submissionIDAccepted.Value, key.Value, creditSelection.value, tipSelection.value, collectibleTip)
	accepted, matched := result.(ledger.SubmissionAccepted)
	if !matched {
		writeDomainError(w, result.(ledger.AcceptRejected).Reason)
		return
	}

	if !server.recordAudit(w, r.Context(), actor.subject.ID, audit.ActionSubmissionAccepted, audit.Subject{Kind: "submission", ID: accepted.SubmissionID.String()}, audit.EmptyMetadata()) {
		return
	}
	if !server.notify(w, r.Context(), submissionFound.Value.SubmitterID, actor.subject.ID, notification.KindSubmissionAccepted, notificationSubjectForSubmission(accepted.SubmissionID), taskNotificationMetadata(accepted.TaskID)) {
		return
	}
	writeJSON(w, http.StatusOK, acceptToResponse(accepted))
}
func (server Server) requestSubmissionChanges(w http.ResponseWriter, r *http.Request) {
	pathResult := server.parseReviewPath(w, r)
	path, pathMatched := pathResult.(reviewPathAccepted)
	if !pathMatched {
		return
	}

	var request requestChangesRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	noteResult := submission.NewRequiredReviewNote(request.ReviewNote)
	note, noteMatched := noteResult.(submission.ReviewNoteAccepted)
	if !noteMatched {
		writeError(w, http.StatusBadRequest, noteResult.(submission.ReviewNoteRejected).Reason.Description())
		return
	}

	submissionResult := server.submissionService.Get(r.Context(), path.actor, path.submissionID)
	submissionFound, submissionMatched := submissionResult.(submission.SubmissionGot)
	if !submissionMatched {
		writeDomainError(w, submissionResult.(submission.GetRejected).Reason)
		return
	}

	result := server.ledgerService.RequestChanges(r.Context(), path.actor.ID, path.taskID, path.submissionID, note.Value)
	changed, matched := result.(ledger.ChangesRequested)
	if !matched {
		writeDomainError(w, result.(ledger.RequestChangesRejected).Reason)
		return
	}
	if !server.recordAudit(w, r.Context(), path.actor.ID, audit.ActionSubmissionChangesRequested, audit.Subject{Kind: "submission", ID: changed.SubmissionID.String()}, audit.EmptyMetadata()) {
		return
	}
	if !server.notify(w, r.Context(), submissionFound.Value.SubmitterID, path.actor.ID, notification.KindSubmissionChangesRequested, notificationSubjectForSubmission(changed.SubmissionID), taskNotificationMetadata(changed.TaskID)) {
		return
	}
	writeJSON(w, http.StatusOK, reviewSubmissionResponse{
		TaskID:       changed.TaskID.String(),
		SubmissionID: changed.SubmissionID.String(),
		State:        "changes_requested",
		ReviewNote:   changed.ReviewNote,
		PayoutKind:   "none",
	})
}
func (server Server) rejectSubmission(w http.ResponseWriter, r *http.Request) {
	pathResult := server.parseReviewPath(w, r)
	path, pathMatched := pathResult.(reviewPathAccepted)
	if !pathMatched {
		return
	}
	if !server.allowBySubject(w, path.actor.ID.String()) {
		return
	}

	var request rejectSubmissionRequest
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
	noteResult := submission.NewRequiredReviewNote(request.ReviewNote)
	note, noteMatched := noteResult.(submission.ReviewNoteAccepted)
	if !noteMatched {
		writeError(w, http.StatusBadRequest, noteResult.(submission.ReviewNoteRejected).Reason.Description())
		return
	}
	creditSelectionResult := rejectCreditSelection(request.PartialCreditAmount)
	creditSelection, creditSelectionMatched := creditSelectionResult.(creditSelectionAccepted)
	if !creditSelectionMatched {
		writeError(w, http.StatusBadRequest, creditSelectionResult.(creditSelectionRejected).reason)
		return
	}
	tipSelectionResult := tipSelectionFromAmount(request.TipAmount)
	tipSelection, tipSelectionMatched := tipSelectionResult.(tipSelectionAccepted)
	if !tipSelectionMatched {
		writeError(w, http.StatusBadRequest, tipSelectionResult.(tipSelectionRejected).reason)
		return
	}
	banSelection := ledger.BanSelection(ledger.NoBanSelection{})
	if request.BanImplementor {
		banSelection = ledger.BanImplementorSelection{}
	}

	submissionResult := server.submissionService.Get(r.Context(), path.actor, path.submissionID)
	submissionFound, submissionMatched := submissionResult.(submission.SubmissionGot)
	if !submissionMatched {
		writeDomainError(w, submissionResult.(submission.GetRejected).Reason)
		return
	}

	result := server.ledgerService.RejectSubmission(r.Context(), path.actor.ID, path.taskID, path.submissionID, key.Value, note.Value, creditSelection.value, tipSelection.value, banSelection)
	rejected, matched := result.(ledger.SubmissionRejected)
	if !matched {
		writeDomainError(w, result.(ledger.RejectRejected).Reason)
		return
	}
	response := reviewOutcomeToResponse(rejected.Payout, rejected.Tip)
	response.TaskID = rejected.TaskID.String()
	response.SubmissionID = rejected.SubmissionID.String()
	response.State = "rejected"
	response.ReviewNote = note.Value.String()
	if !server.recordAudit(w, r.Context(), path.actor.ID, audit.ActionSubmissionRejected, audit.Subject{Kind: "submission", ID: rejected.SubmissionID.String()}, audit.EmptyMetadata()) {
		return
	}
	if !server.notify(w, r.Context(), submissionFound.Value.SubmitterID, path.actor.ID, notification.KindSubmissionRejected, notificationSubjectForSubmission(rejected.SubmissionID), taskNotificationMetadata(rejected.TaskID)) {
		return
	}
	writeJSON(w, http.StatusOK, response)
}
func (server Server) parseReviewPath(w http.ResponseWriter, r *http.Request) reviewPathResult {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return reviewPathRejected{}
	}
	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		writeError(w, http.StatusBadRequest, taskIDResult.(taskIDRejected).reason)
		return reviewPathRejected{}
	}
	submissionIDResult := core.ParseSubmissionID(r.PathValue("submission_id"))
	submissionIDAccepted, submissionIDMatched := submissionIDResult.(core.SubmissionIDCreated)
	if !submissionIDMatched {
		writeError(w, http.StatusBadRequest, submissionIDResult.(core.SubmissionIDRejected).Reason.Description())
		return reviewPathRejected{}
	}
	return reviewPathAccepted{actor: actor.subject, taskID: taskIDAccepted.value, submissionID: submissionIDAccepted.Value}
}
func acceptToResponse(accepted ledger.SubmissionAccepted) acceptSubmissionResponse {
	response := acceptSubmissionResponse{
		TaskID:         accepted.TaskID.String(),
		SubmissionID:   accepted.SubmissionID.String(),
		PayoutKind:     "none",
		CollectibleIDs: []string{},
	}
	switch payout := accepted.Payout.(type) {
	case ledger.CreditPayout:
		response.PayoutKind = "credit"
		response.PayoutAmount = payout.Amount.Int64()
		response.WorkerUserID = payout.WorkerUserID.String()
	case ledger.CollectiblePayout:
		response.PayoutKind = "collectible"
		response.CollectibleIDs = collectibleIDStrings(payout.CollectibleIDs)
		response.WorkerUserID = payout.WorkerUserID.String()
	case ledger.BundlePayout:
		response.PayoutKind = "bundle"
		response.PayoutAmount = payout.Amount.Int64()
		response.CollectibleIDs = collectibleIDStrings(payout.CollectibleIDs)
		response.WorkerUserID = payout.WorkerUserID.String()
	}
	switch tip := accepted.Tip.(type) {
	case ledger.CreditTip:
		response.TipAmount = tip.Amount.Int64()
		if response.WorkerUserID == "" {
			response.WorkerUserID = tip.WorkerUserID.String()
		}
	case ledger.CollectibleTip:
		response.CollectibleIDs = append(response.CollectibleIDs, tip.CollectibleID.String())
		if response.WorkerUserID == "" {
			response.WorkerUserID = tip.WorkerUserID.String()
		}
		response.PayoutKind = appendCollectiblePayoutKind(response.PayoutKind)
	case ledger.BundleTip:
		response.TipAmount = tip.Amount.Int64()
		response.CollectibleIDs = append(response.CollectibleIDs, tip.CollectibleID.String())
		if response.WorkerUserID == "" {
			response.WorkerUserID = tip.WorkerUserID.String()
		}
		response.PayoutKind = appendCollectiblePayoutKind(response.PayoutKind)
	}
	return response
}

func appendCollectiblePayoutKind(current string) string {
	if current == "none" {
		return "collectible"
	}
	if current == "credit" {
		return "bundle"
	}
	return current
}

func reviewOutcomeToResponse(payout ledger.PayoutOutcome, tip ledger.TipOutcome) reviewSubmissionResponse {
	response := reviewSubmissionResponse{PayoutKind: "none"}
	if credit, matched := payout.(ledger.CreditPayout); matched {
		response.PayoutKind = "credit"
		response.PayoutAmount = credit.Amount.Int64()
		response.WorkerUserID = credit.WorkerUserID.String()
	}
	if bundle, matched := payout.(ledger.BundlePayout); matched {
		response.PayoutKind = "bundle"
		response.PayoutAmount = bundle.Amount.Int64()
		response.WorkerUserID = bundle.WorkerUserID.String()
	}
	if creditTip, matched := tip.(ledger.CreditTip); matched {
		response.TipAmount = creditTip.Amount.Int64()
		if response.WorkerUserID == "" {
			response.WorkerUserID = creditTip.WorkerUserID.String()
		}
	}
	return response
}
func acceptCreditSelection(amount int64) creditSelectionResult {
	if amount < 0 {
		return creditSelectionRejected{reason: "payout amount cannot be negative"}
	}
	if amount == 0 {
		return creditSelectionAccepted{value: ledger.FullCreditReviewSelection{}}
	}
	creditResult := ledger.NewCreditAmount(amount)
	credit, matched := creditResult.(ledger.CreditAmountAccepted)
	if !matched {
		return creditSelectionRejected{reason: creditResult.(ledger.CreditAmountRejected).Reason.Description()}
	}
	return creditSelectionAccepted{value: ledger.PartialCreditReviewSelection{Amount: credit.Value}}
}
func rejectCreditSelection(amount int64) creditSelectionResult {
	if amount < 0 {
		return creditSelectionRejected{reason: "partial credit amount cannot be negative"}
	}
	if amount == 0 {
		return creditSelectionAccepted{value: ledger.NoCreditReviewSelection{}}
	}
	creditResult := ledger.NewCreditAmount(amount)
	credit, matched := creditResult.(ledger.CreditAmountAccepted)
	if !matched {
		return creditSelectionRejected{reason: creditResult.(ledger.CreditAmountRejected).Reason.Description()}
	}
	return creditSelectionAccepted{value: ledger.PartialCreditReviewSelection{Amount: credit.Value}}
}
func tipSelectionFromAmount(amount int64) tipSelectionResult {
	if amount < 0 {
		return tipSelectionRejected{reason: "tip amount cannot be negative"}
	}
	if amount == 0 {
		return tipSelectionAccepted{value: ledger.NoTipSelection{}}
	}
	creditResult := ledger.NewCreditAmount(amount)
	credit, matched := creditResult.(ledger.CreditAmountAccepted)
	if !matched {
		return tipSelectionRejected{reason: creditResult.(ledger.CreditAmountRejected).Reason.Description()}
	}
	return tipSelectionAccepted{value: ledger.CreditTipSelection{Amount: credit.Value}}
}

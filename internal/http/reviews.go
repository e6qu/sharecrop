package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/submission"
)

// payoutWorker reports the worker credited by an accept payout (every paid
// payout names the worker; a zero-reward accept does not).
func payoutWorker(payout ledger.PayoutOutcome) (core.UserID, bool) {
	switch outcome := payout.(type) {
	case ledger.CreditPayout:
		return outcome.WorkerUserID, true
	case ledger.CollectiblePayout:
		return outcome.WorkerUserID, true
	case ledger.BundlePayout:
		return outcome.WorkerUserID, true
	default:
		var zero core.UserID
		return zero, false
	}
}

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

	// Validate the optional collectible-tip id up front so a malformed request is
	// rejected before settlement happens.
	var tipCollectibleID core.CollectibleID
	tipCollectible := request.TipCollectibleID != ""
	if tipCollectible {
		collectibleIDResult := core.ParseCollectibleID(request.TipCollectibleID)
		collectibleIDAccepted, collectibleIDMatched := collectibleIDResult.(core.CollectibleIDCreated)
		if !collectibleIDMatched {
			writeError(w, http.StatusBadRequest, collectibleIDResult.(core.CollectibleIDRejected).Reason.Description())
			return
		}
		tipCollectibleID = collectibleIDAccepted.Value
	}

	result := server.ledgerService.ReviewAcceptSubmission(r.Context(), actor.subject.ID, taskIDAccepted.value, submissionIDAccepted.Value, key.Value, creditSelection.value, tipSelection.value)
	accepted, matched := result.(ledger.SubmissionAccepted)
	if !matched {
		writeDomainError(w, result.(ledger.AcceptRejected).Reason)
		return
	}

	// A collectible tip is a separate transfer (assets store) sequenced after the
	// credit settle. The worker is the one the payout credited.
	if tipCollectible {
		worker, hasWorker := payoutWorker(accepted.Payout)
		if !hasWorker {
			writeError(w, http.StatusBadRequest, "a collectible tip requires a paid acceptance that identifies the worker")
			return
		}
		giftResult := server.assetService.GiftCollectible(r.Context(), actor.subject.ID, worker, tipCollectibleID)
		if _, gifted := giftResult.(assets.CollectibleGifted); !gifted {
			writeDomainError(w, giftResult.(assets.GiftRejected).Reason)
			return
		}
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

	result := server.ledgerService.RequestChanges(r.Context(), path.actor.ID, path.taskID, path.submissionID, note.Value)
	changed, matched := result.(ledger.ChangesRequested)
	if !matched {
		writeDomainError(w, result.(ledger.RequestChangesRejected).Reason)
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
	if tip, matched := accepted.Tip.(ledger.CreditTip); matched {
		response.TipAmount = tip.Amount.Int64()
	}
	return response
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

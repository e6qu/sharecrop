package ledger

import (
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
)

// ReviewEventDrafts derives the payout_received / tip_received drafts a
// review produced, mirroring the review draft's subject with the reviewing
// requester as actor. The worker is each variant's audience; the requester is
// the actor, so their inbox stays clean (the fan-out skips the actor) while
// their feed still shows the movement. The store records the returned drafts
// inside the review transaction.
func ReviewEventDrafts(reviewDraft event.Draft, requester core.UserID, taskID core.TaskID, payout PayoutOutcome, tip TipOutcome) ([]event.Draft, *core.DomainError) {
	actor := event.ActorUser{ID: requester}
	drafts := make([]event.Draft, 0, 2)
	appendDraft := func(kind event.Kind, metadata event.Metadata, worker core.UserID) *core.DomainError {
		draftResult := event.NewDraft(kind, actor, reviewDraft.Subject, metadata, event.NewRecipients(worker, requester))
		created, matched := draftResult.(event.DraftCreated)
		if !matched {
			reason := draftResult.(event.DraftRejected).Reason
			return &reason
		}
		drafts = append(drafts, created.Value)
		return nil
	}
	switch typed := payout.(type) {
	case CreditPayout:
		if reason := appendDraft(event.KindPayoutReceived, event.TaskAmountMetadata(taskID, typed.Amount.Int64()), typed.WorkerUserID); reason != nil {
			return nil, reason
		}
	case CollectiblePayout:
		if reason := appendDraft(event.KindPayoutReceived, event.TaskMetadata(taskID), typed.WorkerUserID); reason != nil {
			return nil, reason
		}
	case BundlePayout:
		if reason := appendDraft(event.KindPayoutReceived, event.TaskAmountMetadata(taskID, typed.Amount.Int64()), typed.WorkerUserID); reason != nil {
			return nil, reason
		}
	}
	switch typed := tip.(type) {
	case CreditTip:
		if reason := appendDraft(event.KindTipReceived, event.TaskAmountMetadata(taskID, typed.Amount.Int64()), typed.WorkerUserID); reason != nil {
			return nil, reason
		}
	case CollectibleTip:
		if reason := appendDraft(event.KindTipReceived, event.TaskCollectibleMetadata(taskID, typed.CollectibleID), typed.WorkerUserID); reason != nil {
			return nil, reason
		}
	case BundleTip:
		if reason := appendDraft(event.KindTipReceived, event.TaskAmountMetadata(taskID, typed.Amount.Int64()), typed.WorkerUserID); reason != nil {
			return nil, reason
		}
	}
	return drafts, nil
}

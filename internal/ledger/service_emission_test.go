package ledger

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/event/eventtest"
)

func ledgerRecipientsContain(recipients event.Recipients, user core.UserID) bool {
	for _, value := range recipients.Users {
		if value == user {
			return true
		}
	}
	return false
}

func emittedKinds(events *eventtest.CapturingStore) []string {
	appended := events.Appended()
	kinds := make([]string, 0, len(appended))
	for index := range appended {
		kinds = append(kinds, appended[index].Kind.String())
	}
	return kinds
}

func TestFundTaskEmitsTaskFunded(t *testing.T) {
	events := eventtest.NewCapturingStore()
	store := &memoryStore{events: events}
	service := NewService(store, eventtest.RecorderOver(events), noopAuditRecorder{})
	funder := newTestUserID(t)

	if _, matched := service.FundTask(context.Background(), funder, newTestTaskID(t), newTestAmount(t, 50), newTestKey(t, "fund-emit-1"), SpendByUser{}).(TaskFunded); !matched {
		t.Fatalf("fund rejected")
	}
	appended := events.Appended()
	if len(appended) != 1 || appended[0].Kind != event.KindTaskFunded {
		t.Fatalf("emitted kinds = %v, want [task_funded]", emittedKinds(events))
	}
	if !ledgerRecipientsContain(events.RecipientsAt(0), funder) {
		t.Fatalf("task_funded recipients missed the funder")
	}
}

func TestReviewAcceptEmitsAcceptedPayoutAndTip(t *testing.T) {
	events := eventtest.NewCapturingStore()
	worker := newTestUserID(t)
	store := &memoryStore{
		worker: worker,
		payout: CreditPayout{WorkerUserID: worker, Amount: newTestAmount(t, 40)},
		tip:    CreditTip{WorkerUserID: worker, Amount: newTestAmount(t, 5)},
		events: events,
	}
	service := NewService(store, eventtest.RecorderOver(events), noopAuditRecorder{})
	requester := newTestUserID(t)

	if _, matched := service.ReviewAcceptSubmission(context.Background(), UserReviewer{ID: requester}, newTestTaskID(t), newTestSubmissionID(t), newTestKey(t, "accept-emit-1"), FullCreditReviewSelection{}, CreditTipSelection{Amount: newTestAmount(t, 5)}, NoCollectibleTipSelection{}, SpendByUser{}).(SubmissionAccepted); !matched {
		t.Fatalf("accept rejected")
	}

	appended := events.Appended()
	if len(appended) != 3 {
		t.Fatalf("emitted kinds = %v, want accepted + payout + tip", emittedKinds(events))
	}
	if appended[0].Kind != event.KindSubmissionAccepted || appended[1].Kind != event.KindPayoutReceived || appended[2].Kind != event.KindTipReceived {
		t.Fatalf("emitted kinds = %v", emittedKinds(events))
	}
	for index := range appended {
		recipients := events.RecipientsAt(index)
		if !ledgerRecipientsContain(recipients, worker) {
			t.Fatalf("event %d recipients missed the worker", index)
		}
		if _, matched := appended[index].Subject.Submission.(event.SubmissionSubject); !matched && index == 0 {
			t.Fatalf("submission_accepted event has no submission subject")
		}
	}
	if !ledgerRecipientsContain(events.RecipientsAt(0), requester) {
		t.Fatalf("submission_accepted recipients missed the requester")
	}
}

func TestReviewAcceptWithoutRewardStillEmitsAcceptedToWorker(t *testing.T) {
	events := eventtest.NewCapturingStore()
	worker := newTestUserID(t)
	service := NewService(&memoryStore{worker: worker, events: events}, eventtest.RecorderOver(events), noopAuditRecorder{})

	if _, matched := service.AcceptSubmission(context.Background(), UserReviewer{ID: newTestUserID(t)}, newTestTaskID(t), newTestSubmissionID(t), newTestKey(t, "accept-emit-2")).(SubmissionAccepted); !matched {
		t.Fatalf("accept rejected")
	}
	appended := events.Appended()
	if len(appended) != 1 || appended[0].Kind != event.KindSubmissionAccepted {
		t.Fatalf("emitted kinds = %v, want [submission_accepted]", emittedKinds(events))
	}
	if !ledgerRecipientsContain(events.RecipientsAt(0), worker) {
		t.Fatalf("submission_accepted recipients missed the worker on a no-reward accept")
	}
}

func TestRequestChangesAndRejectEmitReviewEvents(t *testing.T) {
	events := eventtest.NewCapturingStore()
	worker := newTestUserID(t)
	service := NewService(&memoryStore{worker: worker, events: events}, eventtest.RecorderOver(events), noopAuditRecorder{})
	requester := newTestUserID(t)

	if _, matched := service.RequestChanges(context.Background(), UserReviewer{ID: requester}, newTestTaskID(t), newTestSubmissionID(t), newTestKey(t, "changes-emit-1"), submissionNote(t, "needs current data")).(ChangesRequested); !matched {
		t.Fatalf("request changes rejected")
	}
	if _, matched := service.RejectSubmission(context.Background(), UserReviewer{ID: requester}, newTestTaskID(t), newTestSubmissionID(t), newTestKey(t, "reject-emit-1"), submissionNote(t, "stale numbers"), NoCreditReviewSelection{}, NoTipSelection{}, NoBanSelection{}, SpendByUser{}).(SubmissionRejected); !matched {
		t.Fatalf("reject rejected")
	}

	appended := events.Appended()
	if len(appended) != 2 || appended[0].Kind != event.KindSubmissionChangesRequested || appended[1].Kind != event.KindSubmissionRejected {
		t.Fatalf("emitted kinds = %v, want changes_requested then rejected", emittedKinds(events))
	}
	for index := range appended {
		if !ledgerRecipientsContain(events.RecipientsAt(index), worker) {
			t.Fatalf("review event %d recipients missed the worker", index)
		}
	}
}

func TestRefundTaskEmitsTaskCancelledWithRefundCause(t *testing.T) {
	events := eventtest.NewCapturingStore()
	store := &memoryStore{events: events}
	service := NewService(store, eventtest.RecorderOver(events), noopAuditRecorder{})
	requester := newTestUserID(t)
	taskID := newTestTaskID(t)

	if _, matched := service.RefundTask(context.Background(), requester, taskID, newTestKey(t, "refund-emit-1")).(TaskRefunded); !matched {
		t.Fatalf("refund rejected")
	}
	appended := events.Appended()
	if len(appended) != 1 || appended[0].Kind != event.KindTaskCancelled {
		t.Fatalf("emitted kinds = %v, want [task_cancelled]", emittedKinds(events))
	}
	want := `{"task_id":"` + taskID.String() + `","cause":"refund"}`
	if appended[0].Metadata.JSON != want {
		t.Fatalf("refund metadata = %s, want %s", appended[0].Metadata.JSON, want)
	}
	if !ledgerRecipientsContain(events.RecipientsAt(0), requester) {
		t.Fatalf("task_cancelled recipients missed the requester")
	}
}

// capturingAuditRecorder records every audit call for assertions.
type capturingAuditRecorder struct {
	actors  []core.UserID
	actions []string
}

func (recorder *capturingAuditRecorder) Record(_ context.Context, actor core.UserID, action audit.Action, subject audit.Subject, metadata audit.Metadata) audit.RecordResult {
	recorder.actors = append(recorder.actors, actor)
	recorder.actions = append(recorder.actions, action.String())
	return audit.EventRecorded{Value: audit.Event{ActorUserID: actor, Action: action, Subject: subject, Metadata: metadata}}
}

func TestGrantCreditsEmitsCreditGrantedAndAudits(t *testing.T) {
	events := eventtest.NewCapturingStore()
	auditRecorder := &capturingAuditRecorder{}
	service := NewService(&memoryStore{events: events}, eventtest.RecorderOver(events), auditRecorder)
	admin := newTestUserID(t)
	grantee := newTestUserID(t)

	if _, matched := service.GrantCredits(context.Background(), admin, GrantToUser{ID: grantee}, newTestAmount(t, 75), grantNote(t, "manual top-up"), newTestKey(t, "grant-emit-1")).(CreditsGranted); !matched {
		t.Fatalf("grant rejected")
	}

	appended := events.Appended()
	if len(appended) != 1 || appended[0].Kind != event.KindCreditGranted {
		t.Fatalf("emitted kinds = %v, want [credit_granted]", emittedKinds(events))
	}
	if appended[0].Metadata.JSON != `{"amount":75}` {
		t.Fatalf("grant metadata = %s, want amount 75", appended[0].Metadata.JSON)
	}
	if !ledgerRecipientsContain(events.RecipientsAt(0), grantee) {
		t.Fatalf("credit_granted recipients missed the grantee")
	}
	if !ledgerRecipientsContain(events.RecipientsAt(0), admin) {
		t.Fatalf("credit_granted recipients missed the acting admin's own feed")
	}
	if len(auditRecorder.actions) != 1 || auditRecorder.actions[0] != "admin_credit_granted" {
		t.Fatalf("audit actions = %v, want [admin_credit_granted]", auditRecorder.actions)
	}
	if auditRecorder.actors[0] != admin {
		t.Fatalf("audit actor = %s, want the acting admin", auditRecorder.actors[0].String())
	}
}

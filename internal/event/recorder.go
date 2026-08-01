package event

import (
	"context"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/notification"
)

// Recorder is the emission and dispatch point for domain events.
//
// Durable mutations record their event drafts inside the store transaction
// that performs the mutation (the in-transaction outbox); the service then
// calls Dispatch with the recorded drafts to run the post-commit side
// effects inline: inbox fan-out, webhook delivery expansion, and marking the
// event dispatched. A crash between commit and dispatch leaves the event in
// the recorded state, and the lifecycle runner's dispatch sweep replays the
// same idempotent step.
//
// Emit remains for background sweeps whose store mutations do not carry a
// draft: it appends the event (recorded) and dispatches it inline.
type Recorder struct {
	store         Store
	notifications notification.Service
	now           func() time.Time
}

func NewRecorder(store Store, notifications notification.Service) Recorder {
	return Recorder{store: store, notifications: notifications, now: time.Now}
}

// EmitCommand describes one domain event. Recipients are computed by the
// emitting service, which already holds the task/submission/reservation rows;
// the actor should be included so their own feed reflects the action.
type EmitCommand struct {
	Kind       Kind
	Actor      Actor
	Subject    Subject
	Metadata   Metadata
	Recipients Recipients
}

type EmitResult interface {
	emitResult()
}

type EventEmitted struct {
	Value StoredEvent
}

type EmitRejected struct {
	Reason core.DomainError
}

func (EventEmitted) emitResult() {}

func (EmitRejected) emitResult() {}

// Emit appends one event to the stream and dispatches it inline. It is used
// by the lifecycle sweeps (reservation/task expiry), whose store mutations
// commit before the event exists; service mutations record their drafts
// in-transaction instead and call Dispatch.
func (recorder Recorder) Emit(ctx context.Context, command EmitCommand) EmitResult {
	draftResult := NewDraft(command.Kind, command.Actor, command.Subject, command.Metadata, command.Recipients)
	created, matched := draftResult.(DraftCreated)
	if !matched {
		return EmitRejected{Reason: draftResult.(DraftRejected).Reason}
	}
	draft := created.Value

	appendResult := recorder.store.Append(ctx, draft.Event(recorder.now()), draft.Recipients)
	appended, appendMatched := appendResult.(AppendStoreAccepted)
	if !appendMatched {
		return EmitRejected{Reason: appendResult.(AppendStoreRejected).Reason}
	}

	recorder.Dispatch(ctx, draft)
	return EventEmitted{Value: appended.Value}
}

// Dispatch runs the post-commit dispatch step for events already recorded in
// a mutation's transaction: inbox fan-out for the kinds that map to a
// notification, then the store-side effects (webhook expansion, marking
// dispatched). Every part is idempotent per event — notifications are keyed
// by event id, webhook deliveries by (subscription, event), and the state
// flip only touches recorded rows — so an inline dispatch racing the
// recovery sweep produces no duplicates. Failures are deliberately not
// surfaced: the event row is already durable, and the sweep retries.
func (recorder Recorder) Dispatch(ctx context.Context, drafts ...Draft) {
	for _, draft := range drafts {
		recorder.fanOutNotifications(ctx, draft)
		_ = recorder.store.Dispatch(ctx, draft.ID)
	}
}

// fanOutNotifications creates inbox rows for the recipients of kinds that map
// to a notification. notification.Service itself skips the actor, so the
// actor's own feed entry never becomes a self-notification.
func (recorder Recorder) fanOutNotifications(ctx context.Context, draft Draft) {
	rule := NotificationRuleFor(draft.Kind)
	notify, matched := rule.(NotifyAs)
	if !matched {
		return
	}
	actor := ActorUserID(draft.Actor)
	subject := NotificationSubjectFor(draft.Subject)
	metadata := notification.Metadata{JSON: draft.Metadata.JSON}
	for _, recipient := range draft.Recipients.Users {
		_ = recorder.notifications.Notify(ctx, recipient, actor, notify.Kind, subject, metadata, notification.FromEvent{ID: draft.ID})
	}
}

// NotificationRule says whether an event kind produces an inbox notification.
type NotificationRule interface {
	notificationRule()
}

type NoNotification struct{}

type NotifyAs struct {
	Kind notification.Kind
}

func (NoNotification) notificationRule() {}

func (NotifyAs) notificationRule() {}

// NotificationRuleFor is total over AllKinds (a test enforces it). task_opened
// is feed-only: opening your own task notifies nobody, and workers discover
// open tasks through the marketplace, not their inbox. Exported because the
// recorder and tests both consult the mapping.
func NotificationRuleFor(kind Kind) NotificationRule {
	switch kind {
	case KindTaskOpened:
		return NoNotification{}
	case KindTaskFunded:
		return NotifyAs{Kind: notification.KindTaskFunded}
	case KindTaskCancelled:
		return NotifyAs{Kind: notification.KindTaskCancelled}
	case KindTaskExpired:
		return NotifyAs{Kind: notification.KindTaskExpired}
	case KindTaskCommented:
		return NotifyAs{Kind: notification.KindTaskCommented}
	case KindSeriesCommented:
		return NotifyAs{Kind: notification.KindSeriesCommented}
	case KindReservationRequested:
		return NotifyAs{Kind: notification.KindReservationRequested}
	case KindReservationApproved:
		return NotifyAs{Kind: notification.KindReservationApproved}
	case KindReservationDeclined:
		return NotifyAs{Kind: notification.KindReservationDeclined}
	case KindReservationCancelled:
		return NotifyAs{Kind: notification.KindReservationCancelled}
	case KindReservationExpired:
		return NotifyAs{Kind: notification.KindReservationExpired}
	case KindSubmissionCreated:
		return NotifyAs{Kind: notification.KindSubmissionCreated}
	case KindSubmissionAccepted:
		return NotifyAs{Kind: notification.KindSubmissionAccepted}
	case KindSubmissionChangesRequested:
		return NotifyAs{Kind: notification.KindSubmissionChangesRequested}
	case KindSubmissionRejected:
		return NotifyAs{Kind: notification.KindSubmissionRejected}
	case KindSubmissionSuperseded:
		return NotifyAs{Kind: notification.KindSubmissionSuperseded}
	case KindSubmissionCommented:
		return NotifyAs{Kind: notification.KindSubmissionCommented}
	case KindPayoutReceived:
		return NotifyAs{Kind: notification.KindPayoutReceived}
	case KindCreditGranted:
		return NotifyAs{Kind: notification.KindCreditGranted}
	case KindTipReceived:
		return NotifyAs{Kind: notification.KindTipReceived}
	case KindCollectibleAwarded:
		return NotifyAs{Kind: notification.KindCollectibleAwarded}
	default:
		return NoNotification{}
	}
}

// NotificationSubjectFor picks the most specific reference as the inbox
// subject: submission > collectible > task > series > organization. Every
// emitted kind sets at least one reference (the recorder test enforces the
// mapping is meaningful for each kind).
func NotificationSubjectFor(subject Subject) notification.Subject {
	if ref, matched := subject.Submission.(SubmissionSubject); matched {
		return notification.Subject{Kind: "submission", ID: ref.ID.String()}
	}
	if ref, matched := subject.Collectible.(CollectibleSubject); matched {
		return notification.Subject{Kind: "collectible", ID: ref.ID.String()}
	}
	if ref, matched := subject.Task.(TaskSubject); matched {
		return notification.Subject{Kind: "task", ID: ref.ID.String()}
	}
	if ref, matched := subject.Series.(SeriesSubject); matched {
		return notification.Subject{Kind: "series", ID: ref.ID.String()}
	}
	if ref, matched := subject.Organization.(OrganizationSubject); matched {
		return notification.Subject{Kind: "organization", ID: ref.ID.String()}
	}
	return notification.Subject{Kind: "platform", ID: ""}
}

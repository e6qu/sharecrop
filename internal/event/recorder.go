package event

import (
	"context"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/notification"
)

// Recorder is the single emission point for domain events. It appends the
// event with its recipient set and fans out inbox notifications for the kinds
// that warrant one. Services hold a Recorder and call Emit after their
// mutation commits; a failed emission never un-commits the mutation (the same
// exposure the previous handler-level notify calls had — a strict outbox is a
// recorded follow-up).
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

func (recorder Recorder) Emit(ctx context.Context, command EmitCommand) EmitResult {
	idResult := core.NewDomainEventID()
	created, matched := idResult.(core.DomainEventIDCreated)
	if !matched {
		return EmitRejected{Reason: idResult.(core.DomainEventIDRejected).Reason}
	}

	value := Event{
		ID:         created.Value,
		Kind:       command.Kind,
		Actor:      command.Actor,
		Subject:    command.Subject,
		Metadata:   command.Metadata,
		OccurredAt: recorder.now().UTC(),
	}
	appendResult := recorder.store.Append(ctx, value, command.Recipients)
	appended, appendMatched := appendResult.(AppendStoreAccepted)
	if !appendMatched {
		return EmitRejected{Reason: appendResult.(AppendStoreRejected).Reason}
	}

	recorder.fanOutNotifications(ctx, command)
	return EventEmitted{Value: appended.Value}
}

// fanOutNotifications creates inbox rows for the recipients of kinds that map
// to a notification. notification.Service itself skips the actor, so the
// actor's own feed entry never becomes a self-notification. Notification
// failures are deliberately not surfaced: the event is already durable, and
// the inbox is a derived convenience view of it.
func (recorder Recorder) fanOutNotifications(ctx context.Context, command EmitCommand) {
	rule := notificationRuleFor(command.Kind)
	notify, matched := rule.(NotifyAs)
	if !matched {
		return
	}
	actor := ActorUserID(command.Actor)
	subject := notificationSubjectFor(command.Subject)
	metadata := notification.Metadata{JSON: command.Metadata.JSON}
	for _, recipient := range command.Recipients.Users {
		_ = recorder.notifications.Notify(ctx, recipient, actor, notify.Kind, subject, metadata)
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

// notificationRuleFor is total over AllKinds (a test enforces it). task_opened
// is feed-only: opening your own task notifies nobody, and workers discover
// open tasks through the marketplace, not their inbox.
func notificationRuleFor(kind Kind) NotificationRule {
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

// notificationSubjectFor picks the most specific reference as the inbox
// subject: submission > collectible > task > series > organization. Every
// emitted kind sets at least one reference (the recorder test enforces the
// mapping is meaningful for each kind).
func notificationSubjectFor(subject Subject) notification.Subject {
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

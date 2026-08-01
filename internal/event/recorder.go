package event

import (
	"context"

	"github.com/e6qu/sharecrop/internal/notification"
)

// Recorder is the dispatch point for domain events.
//
// Durable mutations — service commands and the lifecycle sweeps alike —
// record their event drafts inside the store transaction that performs the
// mutation (the in-transaction outbox); the caller then passes the recorded
// drafts to Dispatch to run the post-commit side effects inline: inbox
// fan-out, webhook delivery expansion, and marking the event dispatched. A
// crash between commit and dispatch leaves the event in the recorded state,
// and the lifecycle runner's dispatch sweep replays the same idempotent
// step.
type Recorder struct {
	store         Store
	notifications notification.Service
}

func NewRecorder(store Store, notifications notification.Service) Recorder {
	return Recorder{store: store, notifications: notifications}
}

// Dispatch runs the post-commit dispatch step for events already recorded in
// a mutation's transaction: inbox fan-out for the kinds that map to a
// notification, then the store-side effects (webhook expansion, marking
// dispatched). Every part is idempotent per event — notifications are keyed
// by (event, recipient), webhook deliveries by (subscription, event), and
// the state flip only touches recorded rows — so an inline dispatch racing
// the recovery sweep produces no duplicates. The row is flipped to
// dispatched only when every fan-out leg succeeded; on a failed leg it stays
// recorded, so the recovery sweep retries the whole idempotent step instead
// of losing the failed legs forever. Failures are not surfaced to the
// caller: the event row is already durable.
func (recorder Recorder) Dispatch(ctx context.Context, drafts ...Draft) {
	for _, draft := range drafts {
		if !recorder.fanOutNotifications(ctx, draft) {
			continue
		}
		_ = recorder.store.Dispatch(ctx, draft.ID)
	}
}

// fanOutNotifications creates inbox rows for the recipients of kinds that map
// to a notification, reporting whether every leg succeeded.
// notification.Service itself skips the actor, so the actor's own feed entry
// never becomes a self-notification.
func (recorder Recorder) fanOutNotifications(ctx context.Context, draft Draft) bool {
	rule := NotificationRuleFor(draft.Kind)
	notify, matched := rule.(NotifyAs)
	if !matched {
		return true
	}
	actor := ActorUserID(draft.Actor)
	subject := NotificationSubjectFor(draft.Subject)
	metadata := notification.Metadata{JSON: draft.Metadata.JSON}
	completed := true
	for _, recipient := range draft.Recipients.Users {
		result := recorder.notifications.Notify(ctx, recipient, actor, notify.Kind, subject, metadata, notification.FromEvent{ID: draft.ID})
		if _, rejected := result.(notification.NotifyRejected); rejected {
			completed = false
		}
	}
	return completed
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
	case KindCollectibleWithdrawn:
		return NotifyAs{Kind: notification.KindCollectibleWithdrawn}
	case KindCreditsSent:
		return NotifyAs{Kind: notification.KindCreditsReceived}
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

package notification

import (
	"context"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

// Notification also carries two read-time enrichment fields the store
// resolves on reads: the actor's display name, and the subject's title when
// the subject is a task. Both are absent on the create command (fan-out).
type Notification struct {
	ID               core.NotificationID
	RecipientID      core.UserID
	ActorID          core.UserID
	ActorDisplayName auth.DisplayName
	Kind             Kind
	Subject          Subject
	SubjectTitle     SubjectTitleRef
	State            State
	Metadata         Metadata
	// Source names the domain event whose dispatch created this row. The
	// store deduplicates on (event, recipient), so the inline dispatch and
	// the recovery sweep can both fan out the same event without duplicating
	// an inbox row. NoSourceEvent covers rows created before the outbox.
	Source    EventRef
	CreatedAt time.Time
}

// EventRef names the domain event a notification derives from, or its
// explicit absence.
type EventRef interface {
	eventRef()
}

type NoSourceEvent struct{}

type FromEvent struct {
	ID core.DomainEventID
}

func (NoSourceEvent) eventRef() {}

func (FromEvent) eventRef() {}

// SubjectTitleRef names the notification's subject when it is a task, so
// inbox rows can show "on <task title>" without a per-row task fetch.
// NoSubjectTitle covers non-task subjects and create commands.
type SubjectTitleRef interface {
	subjectTitleRef()
}

type NoSubjectTitle struct{}

// TaskSubjectTitle carries a raw display copy of the subject task's title.
type TaskSubjectTitle struct {
	Title string
}

func (NoSubjectTitle) subjectTitleRef() {}

func (TaskSubjectTitle) subjectTitleRef() {}

type Kind struct {
	value string
}

var (
	KindSubmissionCreated          = Kind{value: "submission_created"}
	KindSubmissionAccepted         = Kind{value: "submission_accepted"}
	KindSubmissionChangesRequested = Kind{value: "submission_changes_requested"}
	KindSubmissionRejected         = Kind{value: "submission_rejected"}
	KindSubmissionSuperseded       = Kind{value: "submission_superseded"}
	KindSubmissionCommented        = Kind{value: "submission_commented"}
	KindTaskFunded                 = Kind{value: "task_funded"}
	KindTaskCancelled              = Kind{value: "task_cancelled"}
	KindTaskExpired                = Kind{value: "task_expired"}
	KindTaskCommented              = Kind{value: "task_commented"}
	KindSeriesCommented            = Kind{value: "series_commented"}
	KindReservationRequested       = Kind{value: "reservation_requested"}
	KindReservationApproved        = Kind{value: "reservation_approved"}
	KindReservationDeclined        = Kind{value: "reservation_declined"}
	KindReservationCancelled       = Kind{value: "reservation_cancelled"}
	KindReservationExpired         = Kind{value: "reservation_expired"}
	KindPayoutReceived             = Kind{value: "payout_received"}
	KindCreditGranted              = Kind{value: "credit_granted"}
	KindTipReceived                = Kind{value: "tip_received"}
	KindCollectibleAwarded         = Kind{value: "collectible_awarded"}
	KindCollectibleWithdrawn       = Kind{value: "collectible_withdrawn"}
	KindCreditsReceived            = Kind{value: "credits_received"}
)

// AllKinds returns every notification kind. Totality tests (and the generated
// NotificationKind contract enum) iterate this closed set.
func AllKinds() []Kind {
	return []Kind{
		KindSubmissionCreated,
		KindSubmissionAccepted,
		KindSubmissionChangesRequested,
		KindSubmissionRejected,
		KindSubmissionSuperseded,
		KindSubmissionCommented,
		KindTaskFunded,
		KindTaskCancelled,
		KindTaskExpired,
		KindTaskCommented,
		KindSeriesCommented,
		KindReservationRequested,
		KindReservationApproved,
		KindReservationDeclined,
		KindReservationCancelled,
		KindReservationExpired,
		KindPayoutReceived,
		KindCreditGranted,
		KindTipReceived,
		KindCollectibleAwarded,
		KindCollectibleWithdrawn,
		KindCreditsReceived,
	}
}

func (kind Kind) String() string {
	return kind.value
}

type KindResult interface {
	kindResult()
}

type KindParsed struct {
	Value Kind
}

type KindRejected struct {
	Reason core.DomainError
}

func (KindParsed) kindResult() {}

func (KindRejected) kindResult() {}

// ParseKind converts a raw boundary string into a sealed Kind, rejecting
// strings outside the closed kind set.
func ParseKind(raw string) KindResult {
	for _, kind := range AllKinds() {
		if kind.value == raw {
			return KindParsed{Value: kind}
		}
	}
	return KindRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "unknown notification kind")}
}

type State struct {
	value string
}

var (
	StateUnread = State{value: "unread"}
	StateRead   = State{value: "read"}
)

func (state State) String() string {
	return state.value
}

type StateResult interface {
	stateResult()
}

type StateParsed struct {
	Value State
}

type StateRejected struct {
	Reason core.DomainError
}

func (StateParsed) stateResult() {}

func (StateRejected) stateResult() {}

// ParseState converts a raw boundary string into a sealed State.
func ParseState(raw string) StateResult {
	switch raw {
	case StateUnread.value:
		return StateParsed{Value: StateUnread}
	case StateRead.value:
		return StateParsed{Value: StateRead}
	default:
		return StateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "unknown notification state")}
	}
}

// StateFilter narrows a notification listing by state. AnyState lists the
// whole inbox; UnreadOnly lists only unread rows.
type StateFilter interface {
	stateFilter()
}

type AnyState struct{}

type UnreadOnly struct{}

func (AnyState) stateFilter() {}

func (UnreadOnly) stateFilter() {}

type Subject struct {
	Kind string
	ID   string
}

type Metadata struct {
	JSON string
}

func EmptyMetadata() Metadata {
	return Metadata{JSON: "{}"}
}

type Store interface {
	Create(context.Context, Notification) CreateStoreResult
	List(context.Context, core.UserID, StateFilter, core.Page) ListStoreResult
	CountUnread(context.Context, core.UserID) CountStoreResult
	MarkRead(context.Context, core.UserID, core.NotificationID) MarkReadStoreResult
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) Service {
	return Service{store: store, now: time.Now}
}

type NotifyResult interface {
	notifyResult()
}

type NotificationCreated struct {
	Value Notification
}

type NotificationSkipped struct{}

type NotifyRejected struct {
	Reason core.DomainError
}

func (NotificationCreated) notifyResult() {}

func (NotificationSkipped) notifyResult() {}

func (NotifyRejected) notifyResult() {}

func (service Service) Notify(ctx context.Context, recipient core.UserID, actor core.UserID, kind Kind, subject Subject, metadata Metadata, source EventRef) NotifyResult {
	if recipient == actor {
		return NotificationSkipped{}
	}
	idResult := core.NewNotificationID()
	id, matched := idResult.(core.NotificationIDCreated)
	if !matched {
		return NotifyRejected{Reason: idResult.(core.NotificationIDRejected).Reason}
	}
	value := Notification{
		ID:           id.Value,
		RecipientID:  recipient,
		ActorID:      actor,
		Kind:         kind,
		Subject:      subject,
		SubjectTitle: NoSubjectTitle{},
		State:        StateUnread,
		Metadata:     metadata,
		Source:       source,
		CreatedAt:    service.now().UTC(),
	}
	storeResult := service.store.Create(ctx, value)
	if rejected, rejectedMatched := storeResult.(CreateStoreRejected); rejectedMatched {
		return NotifyRejected{Reason: rejected.Reason}
	}
	return NotificationCreated{Value: value}
}

type ListResult interface {
	listResult()
}

type NotificationsListed struct {
	Values []Notification
	// Total counts every row matching the filter, ignoring limit/offset.
	Total int64
}

type ListRejected struct {
	Reason core.DomainError
}

func (NotificationsListed) listResult() {}

func (ListRejected) listResult() {}

func (service Service) List(ctx context.Context, recipient core.UserID, filter StateFilter, page core.Page) ListResult {
	result := service.store.List(ctx, recipient, filter, page)
	listed, matched := result.(ListStoreAccepted)
	if !matched {
		return ListRejected{Reason: result.(ListStoreRejected).Reason}
	}
	return NotificationsListed{Values: listed.Values, Total: listed.Total}
}

type CountResult interface {
	countResult()
}

type UnreadCounted struct {
	Count int64
}

type CountRejected struct {
	Reason core.DomainError
}

func (UnreadCounted) countResult() {}

func (CountRejected) countResult() {}

// CountUnread reports how many unread notifications the recipient has, for
// the inbox badge.
func (service Service) CountUnread(ctx context.Context, recipient core.UserID) CountResult {
	result := service.store.CountUnread(ctx, recipient)
	counted, matched := result.(CountUnreadCounted)
	if !matched {
		return CountRejected{Reason: result.(CountStoreRejected).Reason}
	}
	return UnreadCounted{Count: counted.Count}
}

type MarkReadResult interface {
	markReadResult()
}

type NotificationRead struct {
	Value Notification
}

type MarkReadRejected struct {
	Reason core.DomainError
}

func (NotificationRead) markReadResult() {}

func (MarkReadRejected) markReadResult() {}

func (service Service) MarkRead(ctx context.Context, recipient core.UserID, id core.NotificationID) MarkReadResult {
	result := service.store.MarkRead(ctx, recipient, id)
	read, matched := result.(MarkReadStoreAccepted)
	if !matched {
		return MarkReadRejected{Reason: result.(MarkReadStoreRejected).Reason}
	}
	return NotificationRead{Value: read.Value}
}

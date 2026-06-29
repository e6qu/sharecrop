package notification

import (
	"context"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
)

type Notification struct {
	ID          core.NotificationID
	RecipientID core.UserID
	ActorID     core.UserID
	Kind        Kind
	Subject     Subject
	State       State
	Metadata    Metadata
	CreatedAt   time.Time
}

type Kind struct {
	value string
}

var (
	KindSubmissionCreated          = Kind{value: "submission_created"}
	KindSubmissionAccepted         = Kind{value: "submission_accepted"}
	KindSubmissionChangesRequested = Kind{value: "submission_changes_requested"}
	KindSubmissionRejected         = Kind{value: "submission_rejected"}
)

func KindFromString(raw string) Kind {
	return Kind{value: raw}
}

func (kind Kind) String() string {
	return kind.value
}

type State struct {
	value string
}

var (
	StateUnread = State{value: "unread"}
	StateRead   = State{value: "read"}
)

func StateFromString(raw string) State {
	return State{value: raw}
}

func (state State) String() string {
	return state.value
}

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
	List(context.Context, core.UserID, core.Page) ListStoreResult
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

func (service Service) Notify(ctx context.Context, recipient core.UserID, actor core.UserID, kind Kind, subject Subject, metadata Metadata) NotifyResult {
	if recipient == actor {
		return NotificationSkipped{}
	}
	idResult := core.NewNotificationID()
	id, matched := idResult.(core.NotificationIDCreated)
	if !matched {
		return NotifyRejected{Reason: idResult.(core.NotificationIDRejected).Reason}
	}
	value := Notification{
		ID:          id.Value,
		RecipientID: recipient,
		ActorID:     actor,
		Kind:        kind,
		Subject:     subject,
		State:       StateUnread,
		Metadata:    metadata,
		CreatedAt:   service.now().UTC(),
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
}

type ListRejected struct {
	Reason core.DomainError
}

func (NotificationsListed) listResult() {}

func (ListRejected) listResult() {}

func (service Service) List(ctx context.Context, recipient core.UserID, page core.Page) ListResult {
	result := service.store.List(ctx, recipient, page)
	listed, matched := result.(ListStoreAccepted)
	if !matched {
		return ListRejected{Reason: result.(ListStoreRejected).Reason}
	}
	return NotificationsListed{Values: listed.Values}
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

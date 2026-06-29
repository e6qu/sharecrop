package audit

import (
	"context"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
)

type Event struct {
	ID          core.AuditEventID
	ActorUserID core.UserID
	Action      Action
	Subject     Subject
	Metadata    Metadata
	CreatedAt   time.Time
}

type Action struct {
	value string
}

var (
	ActionAdminCollectibleAwarded     = Action{value: "admin_collectible_awarded"}
	ActionAccountDeactivated          = Action{value: "account_deactivated"}
	ActionOrganizationMemberProvision = Action{value: "organization_member_provisioned"}
	ActionOrganizationMemberRoles     = Action{value: "organization_member_roles_updated"}
	ActionOrganizationMemberDisabled  = Action{value: "organization_member_deactivated"}
	ActionSubmissionAccepted          = Action{value: "submission_accepted"}
	ActionSubmissionChangesRequested  = Action{value: "submission_changes_requested"}
	ActionSubmissionRejected          = Action{value: "submission_rejected"}
	ActionTaskRefunded                = Action{value: "task_refunded"}
)

func (action Action) String() string {
	return action.value
}

func ActionFromString(raw string) Action {
	return Action{value: raw}
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
	Record(context.Context, Event) RecordResult
	List(context.Context, core.Page) ListResult
}

type Service struct {
	store Store
}

func NewService(store Store) Service {
	return Service{store: store}
}

type RecordResult interface {
	recordResult()
}

type EventRecorded struct{}

type RecordRejected struct {
	Reason core.DomainError
}

func (EventRecorded) recordResult() {}

func (RecordRejected) recordResult() {}

func (service Service) Record(ctx context.Context, actor core.UserID, action Action, subject Subject, metadata Metadata) RecordResult {
	idResult := core.NewAuditEventID()
	idCreated, idMatched := idResult.(core.AuditEventIDCreated)
	if !idMatched {
		return RecordRejected{Reason: idResult.(core.AuditEventIDRejected).Reason}
	}
	return service.store.Record(ctx, Event{
		ID:          idCreated.Value,
		ActorUserID: actor,
		Action:      action,
		Subject:     subject,
		Metadata:    metadata,
		CreatedAt:   time.Now().UTC(),
	})
}

type ListResult interface {
	listResult()
}

type EventsListed struct {
	Values []Event
}

type ListRejected struct {
	Reason core.DomainError
}

func (EventsListed) listResult() {}

func (ListRejected) listResult() {}

func (service Service) List(ctx context.Context, page core.Page) ListResult {
	return service.store.List(ctx, page)
}

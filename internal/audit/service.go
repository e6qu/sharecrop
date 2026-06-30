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
	ActionOrganizationCreated         = Action{value: "organization_created"}
	ActionOrganizationMemberProvision = Action{value: "organization_member_provisioned"}
	ActionOrganizationMemberRoles     = Action{value: "organization_member_roles_updated"}
	ActionOrganizationMemberDisabled  = Action{value: "organization_member_deactivated"}
	ActionSubmissionAccepted          = Action{value: "submission_accepted"}
	ActionSubmissionChangesRequested  = Action{value: "submission_changes_requested"}
	ActionSubmissionRejected          = Action{value: "submission_rejected"}
	ActionTaskRefunded                = Action{value: "task_refunded"}
	ActionPrivacyRequestCreated       = Action{value: "privacy_request_created"}
	ActionModerationReportCreated     = Action{value: "moderation_report_created"}
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
	Get(context.Context, core.AuditEventID) GetResult
	List(context.Context, ListFilters, core.Page) ListResult
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

type EventRecorded struct {
	Value Event
}

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
	event := Event{
		ID:          idCreated.Value,
		ActorUserID: actor,
		Action:      action,
		Subject:     subject,
		Metadata:    metadata,
		CreatedAt:   time.Now().UTC(),
	}
	return service.store.Record(ctx, event)
}

type GetResult interface {
	getResult()
}

type EventFound struct {
	Value Event
}

type GetRejected struct {
	Reason core.DomainError
}

func (EventFound) getResult() {}

func (GetRejected) getResult() {}

func (service Service) Get(ctx context.Context, id core.AuditEventID) GetResult {
	return service.store.Get(ctx, id)
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

type ListFilters struct {
	Action      ActionFilter
	SubjectKind SubjectKindFilter
	SubjectID   SubjectIDFilter
}

func NoListFilters() ListFilters {
	return ListFilters{Action: AnyAction{}, SubjectKind: AnySubjectKind{}, SubjectID: AnySubjectID{}}
}

type ActionFilter interface {
	actionFilter()
}

type AnyAction struct{}

type ActionEquals struct {
	Value Action
}

func (AnyAction) actionFilter() {}

func (ActionEquals) actionFilter() {}

type SubjectKindFilter interface {
	subjectKindFilter()
}

type AnySubjectKind struct{}

type SubjectKindEquals struct {
	Value string
}

func (AnySubjectKind) subjectKindFilter() {}

func (SubjectKindEquals) subjectKindFilter() {}

type SubjectIDFilter interface {
	subjectIDFilter()
}

type AnySubjectID struct{}

type SubjectIDEquals struct {
	Value string
}

func (AnySubjectID) subjectIDFilter() {}

func (SubjectIDEquals) subjectIDFilter() {}

func (service Service) List(ctx context.Context, filters ListFilters, page core.Page) ListResult {
	return service.store.List(ctx, filters, page)
}

package task

import (
	"context"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
)

type Store interface {
	CreateTask(context.Context, core.TaskSeriesID, core.TaskID, CreateCommand) CreateTaskStoreResult
	FindTask(context.Context, core.TaskID) FindTaskStoreResult
	ChangeTaskState(context.Context, core.TaskID, State) ChangeTaskStateStoreResult
	ListTasks(context.Context, ListScope, ListFilters, core.Page) ListTasksStoreResult
	CreateCapabilityToken(context.Context, core.TaskCapabilityTokenID, core.TaskID, CapabilityTokenHash) CreateCapabilityTokenStoreResult
	ListSeries(context.Context, core.UserID, core.Page) ListSeriesStoreResult
	FindSeries(context.Context, core.TaskSeriesID) FindSeriesStoreResult
	CreateSeries(context.Context, Series) SeriesMutationStoreResult
	UpdateSeries(context.Context, core.TaskSeriesID, SeriesTitle, SeriesDescription) SeriesMutationStoreResult
	UpdateSeriesState(context.Context, core.TaskSeriesID, SeriesState) SeriesMutationStoreResult
	AddTaskToSeries(context.Context, core.TaskSeriesID, core.TaskID) SeriesMutationStoreResult
	RemoveTaskFromSeries(context.Context, core.TaskSeriesID, core.TaskID) SeriesMutationStoreResult
	ReorderSeries(context.Context, core.TaskSeriesID, []core.TaskID) SeriesMutationStoreResult
	CreateSeriesComment(context.Context, SeriesComment) CreateSeriesCommentStoreResult
	ListSeriesComments(context.Context, core.TaskSeriesID) ListSeriesCommentsStoreResult
	CreateTaskComment(context.Context, TaskComment) CreateTaskCommentStoreResult
	ListTaskComments(context.Context, core.TaskID) ListTaskCommentsStoreResult
	CreateReservation(context.Context, core.TaskReservationID, ReservationCommand) CreateReservationStoreResult
	ChangeReservationState(context.Context, core.TaskID, core.TaskReservationID, ReservationState) ChangeReservationStateStoreResult
	ListReservations(context.Context, core.TaskID) ListReservationsStoreResult
	CheckSubmissionEligibility(context.Context, core.TaskID, core.UserID) SubmissionEligibilityStoreResult
}

type OrganizationPermissions interface {
	CheckOrganizationPermission(context.Context, core.OrganizationID, core.UserID, org.Permission) org.PermissionCheck
	CheckOrganizationTeamMembership(context.Context, core.OrganizationID, core.TeamID, core.UserID) org.PermissionCheck
	CheckTeamMembership(context.Context, core.TeamID, core.UserID) org.PermissionCheck
}

type Service struct {
	store                   Store
	organizationPermissions OrganizationPermissions
}

func NewService(store Store, organizationPermissions OrganizationPermissions) Service {
	return Service{store: store, organizationPermissions: organizationPermissions}
}

type CreateCommand struct {
	Actor          auth.UserSubject
	Owner          Owner
	Title          Title
	Description    Description
	Type           TaskType
	Reference      ReferenceURL
	Reward         RewardSpec
	Participation  ParticipationPolicy
	AssigneeScope  AssigneeScope
	ReservationTTL ReservationTTL
	Visibility     Visibility
	Placement      SeriesPlacement
	ResponseSchema ResponseSchemaSource
	Payload        DataPayload
}

type CreateResult interface {
	createResult()
}

type TaskCreated struct {
	Value Task
}

type CreateRejected struct {
	Reason core.DomainError
}

func (TaskCreated) createResult() {}

func (CreateRejected) createResult() {}

func (service Service) Create(ctx context.Context, command CreateCommand) CreateResult {
	ownerPermission := service.requireOwnerPermission(ctx, command.Actor, command.Owner)
	if rejected, matched := ownerPermission.(ownerPermissionRejected); matched {
		return CreateRejected{Reason: rejected.reason}
	}

	visibilityPermission := service.requireVisibilityPermission(ctx, command.Actor, command.Owner, command.Visibility)
	if rejected, matched := visibilityPermission.(visibilityPermissionRejected); matched {
		return CreateRejected{Reason: rejected.reason}
	}

	taskIDResult := core.NewTaskID()
	taskIDCreated, taskIDMatched := taskIDResult.(core.TaskIDCreated)
	if !taskIDMatched {
		rejected := taskIDResult.(core.TaskIDRejected)
		return CreateRejected{Reason: rejected.Reason}
	}

	seriesIDResult := core.NewTaskSeriesID()
	seriesIDCreated, seriesIDMatched := seriesIDResult.(core.TaskSeriesIDCreated)
	if !seriesIDMatched {
		rejected := seriesIDResult.(core.TaskSeriesIDRejected)
		return CreateRejected{Reason: rejected.Reason}
	}

	storeResult := service.store.CreateTask(ctx, seriesIDCreated.Value, taskIDCreated.Value, command)
	accepted, matched := storeResult.(CreateTaskStoreAccepted)
	if !matched {
		rejected := storeResult.(CreateTaskStoreRejected)
		return CreateRejected{Reason: rejected.Reason}
	}

	return TaskCreated{Value: accepted.Value}
}

type ChangeStateResult interface {
	changeStateResult()
}

type TaskStateChanged struct {
	Value Task
}

type ChangeStateRejected struct {
	Reason core.DomainError
}

func (TaskStateChanged) changeStateResult() {}

func (ChangeStateRejected) changeStateResult() {}

func (service Service) Open(ctx context.Context, actor auth.UserSubject, taskID core.TaskID) ChangeStateResult {
	return service.changeState(ctx, actor, taskID, OpenState)
}

func (service Service) Cancel(ctx context.Context, actor auth.UserSubject, taskID core.TaskID) ChangeStateResult {
	return service.changeState(ctx, actor, taskID, CancelState)
}

func (service Service) Unpublish(ctx context.Context, actor auth.UserSubject, taskID core.TaskID) ChangeStateResult {
	return service.changeState(ctx, actor, taskID, UnpublishState)
}

type StateTransition func(State) StateTransitionResult

func (service Service) changeState(ctx context.Context, actor auth.UserSubject, taskID core.TaskID, transition StateTransition) ChangeStateResult {
	taskResult := service.store.FindTask(ctx, taskID)
	taskFound, taskMatched := taskResult.(FindTaskStoreAccepted)
	if !taskMatched {
		rejected := taskResult.(FindTaskStoreRejected)
		return ChangeStateRejected{Reason: rejected.Reason}
	}

	ownerPermission := service.requireOwnerPermission(ctx, actor, taskFound.Value.Owner)
	if rejected, matched := ownerPermission.(ownerPermissionRejected); matched {
		return ChangeStateRejected{Reason: rejected.reason}
	}

	transitionResult := transition(taskFound.Value.State)
	transitionAccepted, transitionMatched := transitionResult.(StateTransitionAccepted)
	if !transitionMatched {
		rejected := transitionResult.(StateTransitionRejected)
		return ChangeStateRejected{Reason: rejected.Reason}
	}

	storeResult := service.store.ChangeTaskState(ctx, taskID, transitionAccepted.Value)
	changed, matched := storeResult.(ChangeTaskStateStoreAccepted)
	if !matched {
		rejected := storeResult.(ChangeTaskStateStoreRejected)
		return ChangeStateRejected{Reason: rejected.Reason}
	}

	return TaskStateChanged{Value: changed.Value}
}

type GetResult interface {
	getResult()
}

type TaskGot struct {
	Value Task
}

type GetRejected struct {
	Reason core.DomainError
}

func (TaskGot) getResult() {}

func (GetRejected) getResult() {}

func (service Service) Get(ctx context.Context, actor auth.UserSubject, taskID core.TaskID) GetResult {
	taskResult := service.store.FindTask(ctx, taskID)
	taskFound, taskMatched := taskResult.(FindTaskStoreAccepted)
	if !taskMatched {
		rejected := taskResult.(FindTaskStoreRejected)
		return GetRejected{Reason: rejected.Reason}
	}

	viewPermission := service.requireViewPermission(ctx, actor, taskFound.Value)
	if rejected, matched := viewPermission.(viewPermissionRejected); matched {
		return GetRejected{Reason: rejected.reason}
	}
	return TaskGot{Value: taskFound.Value}
}

type viewPermissionResult interface {
	viewPermissionResult()
}

type viewPermissionAccepted struct{}

type viewPermissionRejected struct {
	reason core.DomainError
}

func (viewPermissionAccepted) viewPermissionResult() {}

func (viewPermissionRejected) viewPermissionResult() {}

func (service Service) requireViewPermission(ctx context.Context, actor auth.UserSubject, value Task) viewPermissionResult {
	if value.CreatedBy == actor.ID {
		return viewPermissionAccepted{}
	}
	switch typed := value.Visibility.(type) {
	case PublicVisibility:
		return viewPermissionAccepted{}
	case UserVisibility:
		if typed.UserID == actor.ID {
			return viewPermissionAccepted{}
		}
		return viewPermissionRejected{reason: core.NewDomainError(core.ErrorCodePermissionDenied, "task view access denied")}
	case OrganizationVisibility:
		return service.requireOrganizationViewPermission(ctx, typed.OrganizationID, actor.ID)
	case OrganizationTeamVisibility:
		return service.requireOrganizationViewPermission(ctx, typed.OrganizationID, actor.ID)
	default:
		return viewPermissionRejected{reason: core.NewDomainError(core.ErrorCodePermissionDenied, "task view access denied")}
	}
}

func (service Service) requireOrganizationViewPermission(ctx context.Context, organizationID core.OrganizationID, userID core.UserID) viewPermissionResult {
	check := service.organizationPermissions.CheckOrganizationPermission(ctx, organizationID, userID, org.PermissionCreateOrganizationTask)
	if rejected, matched := check.(org.PermissionDenied); matched {
		return viewPermissionRejected{reason: rejected.Reason}
	}
	return viewPermissionAccepted{}
}

type ListScope interface {
	listScope()
}

// StateFilter is an optional task-state filter for task listing. AnyStateFilter
// means no state restriction; StateEquals restricts the listing to a single state.
type StateFilter interface {
	stateFilter()
}

type AnyStateFilter struct{}

type StateEquals struct {
	Value State
}

func (AnyStateFilter) stateFilter() {}

func (StateEquals) stateFilter() {}

// ParticipationPolicyFilter is an optional participation-policy filter for task
// listing. AnyParticipationPolicyFilter means no restriction; ParticipationPolicyEquals
// restricts the listing to a single policy.
type ParticipationPolicyFilter interface {
	participationPolicyFilter()
}

type AnyParticipationPolicyFilter struct{}

type ParticipationPolicyEquals struct {
	Value ParticipationPolicy
}

func (AnyParticipationPolicyFilter) participationPolicyFilter() {}

func (ParticipationPolicyEquals) participationPolicyFilter() {}

// SearchFilter is an optional task-list search filter. NoSearchFilter means no
// search restriction; SearchContains restricts the listing to task title or ID
// matches.
type SearchFilter interface {
	searchFilter()
}

type NoSearchFilter struct{}

type SearchContains struct {
	Value SearchText
}

func (NoSearchFilter) searchFilter() {}

func (SearchContains) searchFilter() {}

// TypeFilter is an optional task-type filter for task listing. AnyTypeFilter
// means no restriction; TypeEquals restricts the listing to one task type.
type TypeFilter interface {
	typeFilter()
}

type AnyTypeFilter struct{}

type TypeEquals struct {
	Value TaskType
}

func (AnyTypeFilter) typeFilter() {}

func (TypeEquals) typeFilter() {}

type SortOrder struct {
	value string
}

var (
	SortNewest     = SortOrder{value: "newest"}
	SortOldest     = SortOrder{value: "oldest"}
	SortTitleAsc   = SortOrder{value: "title_asc"}
	SortTitleDesc  = SortOrder{value: "title_desc"}
	SortRewardDesc = SortOrder{value: "reward_desc"}
	SortRewardAsc  = SortOrder{value: "reward_asc"}
)

type SortOrderResult interface {
	sortOrderResult()
}

type SortOrderAccepted struct {
	Value SortOrder
}

type SortOrderRejected struct {
	Reason core.DomainError
}

func (SortOrderAccepted) sortOrderResult() {}

func (SortOrderRejected) sortOrderResult() {}

func ParseSortOrder(raw string) SortOrderResult {
	switch raw {
	case "", SortNewest.value:
		return SortOrderAccepted{Value: SortNewest}
	case SortOldest.value:
		return SortOrderAccepted{Value: SortOldest}
	case SortTitleAsc.value:
		return SortOrderAccepted{Value: SortTitleAsc}
	case SortTitleDesc.value:
		return SortOrderAccepted{Value: SortTitleDesc}
	case SortRewardDesc.value:
		return SortOrderAccepted{Value: SortRewardDesc}
	case SortRewardAsc.value:
		return SortOrderAccepted{Value: SortRewardAsc}
	default:
		return SortOrderRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "task sort is invalid")}
	}
}

func (order SortOrder) String() string {
	return order.value
}

// ListFilters groups the optional discovery/list filters applied to a task listing.
type ListFilters struct {
	State         StateFilter
	Participation ParticipationPolicyFilter
	Search        SearchFilter
	Type          TypeFilter
	Sort          SortOrder
}

func NoListFilters() ListFilters {
	return ListFilters{State: AnyStateFilter{}, Participation: AnyParticipationPolicyFilter{}, Search: NoSearchFilter{}, Type: AnyTypeFilter{}, Sort: SortNewest}
}

type PublicListScope struct {
	ViewerID        core.UserID
	IncludeReserved bool
}

type UserListScope struct {
	UserID          core.UserID
	IncludeReserved bool
}

type OrganizationListScope struct {
	OrganizationID  core.OrganizationID
	UserID          core.UserID
	IncludeReserved bool
}

type TeamListScope struct {
	TeamID          core.TeamID
	IncludeReserved bool
}

// CreatorListScope lists the public tasks created by a user. It exposes only
// publicly visible tasks, so every authenticated viewer may read another user's
// public profile without leaking private or scoped tasks.
type CreatorListScope struct {
	CreatorID core.UserID
}

// AssigneeListScope lists the public tasks a user is actively assigned to (their
// public work). It exposes only publicly visible tasks, so a profile cannot leak
// private or scoped work.
type AssigneeListScope struct {
	AssigneeID core.UserID
}

func (PublicListScope) listScope() {}

func (UserListScope) listScope() {}

func (OrganizationListScope) listScope() {}

func (TeamListScope) listScope() {}

func (CreatorListScope) listScope() {}

func (AssigneeListScope) listScope() {}

type ListResult interface {
	listResult()
}

type TasksListed struct {
	Values []ListItem
}

type ListRejected struct {
	Reason core.DomainError
}

func (TasksListed) listResult() {}

func (ListRejected) listResult() {}

func (service Service) List(ctx context.Context, actor auth.UserSubject, scope ListScope, filters ListFilters, page core.Page) ListResult {
	scopePermission := service.requireListPermission(ctx, actor, scope)
	if rejected, matched := scopePermission.(listPermissionRejected); matched {
		return ListRejected{Reason: rejected.reason}
	}

	storeResult := service.store.ListTasks(ctx, scope, filters, page)
	listed, matched := storeResult.(ListTasksStoreAccepted)
	if !matched {
		rejected := storeResult.(ListTasksStoreRejected)
		return ListRejected{Reason: rejected.Reason}
	}
	return TasksListed{Values: listed.Values}
}

type ReservationCommand struct {
	TaskID      core.TaskID
	Assignee    Assignee
	RequestedBy core.UserID
}

type ReservationResult interface {
	reservationResult()
}

type ReservationCreated struct {
	Value Reservation
}

type ReservationRejected struct {
	Reason core.DomainError
}

func (ReservationCreated) reservationResult() {}

func (ReservationRejected) reservationResult() {}

func (service Service) Reserve(ctx context.Context, actor auth.UserSubject, taskID core.TaskID) ReservationResult {
	return service.reserve(ctx, actor, taskID, UserAssignee{UserID: actor.ID}, AssigneeScopeUser, "this task does not accept user reservations")
}

func (service Service) ReserveForOrganizationTeam(ctx context.Context, actor auth.UserSubject, taskID core.TaskID, organizationID core.OrganizationID, teamID core.TeamID) ReservationResult {
	check := service.organizationPermissions.CheckOrganizationTeamMembership(ctx, organizationID, teamID, actor.ID)
	if rejected, matched := check.(org.PermissionDenied); matched {
		return ReservationRejected{Reason: rejected.Reason}
	}
	return service.reserve(ctx, actor, taskID, OrganizationTeamAssignee{OrganizationID: organizationID, TeamID: teamID}, AssigneeScopeOrganizationTeam, "this task does not accept organization team reservations")
}

func (service Service) ReserveForTeam(ctx context.Context, actor auth.UserSubject, taskID core.TaskID, teamID core.TeamID) ReservationResult {
	check := service.organizationPermissions.CheckTeamMembership(ctx, teamID, actor.ID)
	if rejected, matched := check.(org.PermissionDenied); matched {
		return ReservationRejected{Reason: rejected.Reason}
	}
	return service.reserve(ctx, actor, taskID, TeamAssignee{TeamID: teamID}, AssigneeScopeTeam, "this task does not accept team reservations")
}

func (service Service) reserve(ctx context.Context, actor auth.UserSubject, taskID core.TaskID, assignee Assignee, requiredScope AssigneeScope, wrongScopeMessage string) ReservationResult {
	taskResult := service.store.FindTask(ctx, taskID)
	taskFound, taskMatched := taskResult.(FindTaskStoreAccepted)
	if !taskMatched {
		rejected := taskResult.(FindTaskStoreRejected)
		return ReservationRejected{Reason: rejected.Reason}
	}
	if taskFound.Value.State != StateOpen {
		return ReservationRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only open tasks can be reserved")}
	}
	if taskFound.Value.AssigneeScope != requiredScope {
		return ReservationRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, wrongScopeMessage)}
	}
	if rejected, matched := service.requireViewPermission(ctx, actor, taskFound.Value).(viewPermissionRejected); matched {
		return ReservationRejected{Reason: rejected.reason}
	}
	if taskFound.Value.CreatedBy == actor.ID {
		return ReservationRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "task requester cannot reserve their own task")}
	}

	reservationIDResult := core.NewTaskReservationID()
	reservationIDCreated, reservationIDMatched := reservationIDResult.(core.TaskReservationIDCreated)
	if !reservationIDMatched {
		rejected := reservationIDResult.(core.TaskReservationIDRejected)
		return ReservationRejected{Reason: rejected.Reason}
	}

	storeResult := service.store.CreateReservation(ctx, reservationIDCreated.Value, ReservationCommand{
		TaskID:      taskID,
		Assignee:    assignee,
		RequestedBy: actor.ID,
	})
	created, matched := storeResult.(CreateReservationStoreAccepted)
	if !matched {
		rejected := storeResult.(CreateReservationStoreRejected)
		return ReservationRejected{Reason: rejected.Reason}
	}
	return ReservationCreated{Value: created.Value}
}

type ReservationStateChangeResult interface {
	reservationStateChangeResult()
}

type ReservationStateChanged struct {
	Value Reservation
}

type ReservationStateChangeRejected struct {
	Reason core.DomainError
}

func (ReservationStateChanged) reservationStateChangeResult() {}

func (ReservationStateChangeRejected) reservationStateChangeResult() {}

func (service Service) ApproveReservation(ctx context.Context, actor auth.UserSubject, taskID core.TaskID, reservationID core.TaskReservationID) ReservationStateChangeResult {
	return service.changeReservationByRequester(ctx, actor, taskID, reservationID, ReservationStateActive)
}

func (service Service) DeclineReservation(ctx context.Context, actor auth.UserSubject, taskID core.TaskID, reservationID core.TaskReservationID) ReservationStateChangeResult {
	return service.changeReservationByRequester(ctx, actor, taskID, reservationID, ReservationStateDeclined)
}

func (service Service) CancelReservation(ctx context.Context, actor auth.UserSubject, taskID core.TaskID, reservationID core.TaskReservationID) ReservationStateChangeResult {
	return service.changeReservationByRequester(ctx, actor, taskID, reservationID, ReservationStateCancelledByRequester)
}

func (service Service) changeReservationByRequester(ctx context.Context, actor auth.UserSubject, taskID core.TaskID, reservationID core.TaskReservationID, state ReservationState) ReservationStateChangeResult {
	taskResult := service.store.FindTask(ctx, taskID)
	taskFound, taskMatched := taskResult.(FindTaskStoreAccepted)
	if !taskMatched {
		rejected := taskResult.(FindTaskStoreRejected)
		return ReservationStateChangeRejected{Reason: rejected.Reason}
	}
	ownerPermission := service.requireOwnerPermission(ctx, actor, taskFound.Value.Owner)
	if rejected, matched := ownerPermission.(ownerPermissionRejected); matched {
		return ReservationStateChangeRejected{Reason: rejected.reason}
	}

	// The mutation is bound to taskID in the store query, so a reservation that
	// belongs to a different task is never touched (the post-check below is then
	// only defense-in-depth, not the load-bearing authorization).
	storeResult := service.store.ChangeReservationState(ctx, taskID, reservationID, state)
	changed, matched := storeResult.(ChangeReservationStateStoreAccepted)
	if !matched {
		rejected := storeResult.(ChangeReservationStateStoreRejected)
		return ReservationStateChangeRejected{Reason: rejected.Reason}
	}
	if changed.Value.TaskID != taskID {
		return ReservationStateChangeRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "reservation was not found for the task")}
	}
	return ReservationStateChanged{Value: changed.Value}
}

type ReservationsListResult interface {
	reservationsListResult()
}

type ReservationsListed struct {
	Values []Reservation
}

type ReservationsListRejected struct {
	Reason core.DomainError
}

func (ReservationsListed) reservationsListResult() {}

func (ReservationsListRejected) reservationsListResult() {}

func (service Service) ListReservations(ctx context.Context, actor auth.UserSubject, taskID core.TaskID) ReservationsListResult {
	taskResult := service.store.FindTask(ctx, taskID)
	taskFound, taskMatched := taskResult.(FindTaskStoreAccepted)
	if !taskMatched {
		rejected := taskResult.(FindTaskStoreRejected)
		return ReservationsListRejected{Reason: rejected.Reason}
	}
	ownerPermission := service.requireOwnerPermission(ctx, actor, taskFound.Value.Owner)
	if rejected, matched := ownerPermission.(ownerPermissionRejected); matched {
		return ReservationsListRejected{Reason: rejected.reason}
	}

	storeResult := service.store.ListReservations(ctx, taskID)
	listed, matched := storeResult.(ListReservationsStoreAccepted)
	if !matched {
		rejected := storeResult.(ListReservationsStoreRejected)
		return ReservationsListRejected{Reason: rejected.Reason}
	}
	return ReservationsListed{Values: listed.Values}
}

func (service Service) CheckSubmissionEligibility(ctx context.Context, taskID core.TaskID, submitterID core.UserID) SubmissionEligibilityStoreResult {
	return service.store.CheckSubmissionEligibility(ctx, taskID, submitterID)
}

type CreateCapabilityTokenResult interface {
	createCapabilityTokenResult()
}

type CapabilityTokenCreated struct {
	Value CapabilityToken
	Plain CapabilityTokenPlain
}

type CreateCapabilityTokenRejected struct {
	Reason core.DomainError
}

func (CapabilityTokenCreated) createCapabilityTokenResult() {}

func (CreateCapabilityTokenRejected) createCapabilityTokenResult() {}

func (service Service) CreateCapabilityToken(ctx context.Context, actor auth.UserSubject, taskID core.TaskID) CreateCapabilityTokenResult {
	taskResult := service.store.FindTask(ctx, taskID)
	taskFound, taskMatched := taskResult.(FindTaskStoreAccepted)
	if !taskMatched {
		rejected := taskResult.(FindTaskStoreRejected)
		return CreateCapabilityTokenRejected{Reason: rejected.Reason}
	}

	ownerPermission := service.requireOwnerPermission(ctx, actor, taskFound.Value.Owner)
	if rejected, matched := ownerPermission.(ownerPermissionRejected); matched {
		return CreateCapabilityTokenRejected{Reason: rejected.reason}
	}

	tokenIDResult := core.NewTaskCapabilityTokenID()
	tokenIDCreated, tokenIDMatched := tokenIDResult.(core.TaskCapabilityTokenIDCreated)
	if !tokenIDMatched {
		rejected := tokenIDResult.(core.TaskCapabilityTokenIDRejected)
		return CreateCapabilityTokenRejected{Reason: rejected.Reason}
	}

	plainResult := NewCapabilityTokenPlain()
	plainCreated, plainMatched := plainResult.(CapabilityTokenPlainAccepted)
	if !plainMatched {
		rejected := plainResult.(CapabilityTokenPlainRejected)
		return CreateCapabilityTokenRejected{Reason: rejected.Reason}
	}

	storeResult := service.store.CreateCapabilityToken(ctx, tokenIDCreated.Value, taskID, plainCreated.Value.Hash())
	created, matched := storeResult.(CreateCapabilityTokenStoreAccepted)
	if !matched {
		rejected := storeResult.(CreateCapabilityTokenStoreRejected)
		return CreateCapabilityTokenRejected{Reason: rejected.Reason}
	}

	return CapabilityTokenCreated{Value: created.Value, Plain: plainCreated.Value}
}

type ownerPermissionResult interface {
	ownerPermissionResult()
}

type ownerPermissionAccepted struct{}

type ownerPermissionRejected struct {
	reason core.DomainError
}

func (ownerPermissionAccepted) ownerPermissionResult() {}

func (ownerPermissionRejected) ownerPermissionResult() {}

func (service Service) requireOwnerPermission(ctx context.Context, actor auth.UserSubject, owner Owner) ownerPermissionResult {
	switch typed := owner.(type) {
	case UserOwner:
		if typed.UserID != actor.ID {
			return ownerPermissionRejected{reason: core.NewDomainError(core.ErrorCodePermissionDenied, "task owner access denied")}
		}
		return ownerPermissionAccepted{}
	case OrganizationOwner:
		return service.requireOrganizationPermission(ctx, typed.OrganizationID, actor.ID, org.PermissionCreateOrganizationTask)
	case OrganizationTeamOwner:
		return service.requireOrganizationPermission(ctx, typed.OrganizationID, actor.ID, org.PermissionCreateOrganizationTask)
	case TeamOwner:
		return ownerPermissionRejected{reason: core.NewDomainError(core.ErrorCodePermissionDenied, "team-owned tasks require organization ownership in this release")}
	default:
		return ownerPermissionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task owner is invalid")}
	}
}

func (service Service) requireOrganizationPermission(ctx context.Context, organizationID core.OrganizationID, userID core.UserID, permission org.Permission) ownerPermissionResult {
	check := service.organizationPermissions.CheckOrganizationPermission(ctx, organizationID, userID, permission)
	if rejected, matched := check.(org.PermissionDenied); matched {
		return ownerPermissionRejected{reason: rejected.Reason}
	}
	return ownerPermissionAccepted{}
}

type visibilityPermissionResult interface {
	visibilityPermissionResult()
}

type visibilityPermissionAccepted struct{}

type visibilityPermissionRejected struct {
	reason core.DomainError
}

func (visibilityPermissionAccepted) visibilityPermissionResult() {}

func (visibilityPermissionRejected) visibilityPermissionResult() {}

func (service Service) requireVisibilityPermission(ctx context.Context, actor auth.UserSubject, owner Owner, visibility Visibility) visibilityPermissionResult {
	if _, matched := visibility.(PublicVisibility); matched {
		organizationIDResult := organizationIDForOwner(owner)
		organizationIDFound, organizationIDMatched := organizationIDResult.(organizationIDFound)
		if !organizationIDMatched {
			return visibilityPermissionAccepted{}
		}
		check := service.organizationPermissions.CheckOrganizationPermission(ctx, organizationIDFound.value, actor.ID, org.PermissionPublishPublicTask)
		if rejected, permissionMatched := check.(org.PermissionDenied); permissionMatched {
			return visibilityPermissionRejected{reason: rejected.Reason}
		}
		return visibilityPermissionAccepted{}
	}
	return visibilityPermissionAccepted{}
}

type organizationIDForOwnerResult interface {
	organizationIDForOwnerResult()
}

type organizationIDFound struct {
	value core.OrganizationID
}

type organizationIDMissing struct{}

func (organizationIDFound) organizationIDForOwnerResult() {}

func (organizationIDMissing) organizationIDForOwnerResult() {}

func organizationIDForOwner(owner Owner) organizationIDForOwnerResult {
	switch typed := owner.(type) {
	case OrganizationOwner:
		return organizationIDFound{value: typed.OrganizationID}
	case OrganizationTeamOwner:
		return organizationIDFound{value: typed.OrganizationID}
	default:
		return organizationIDMissing{}
	}
}

type listPermissionResult interface {
	listPermissionResult()
}

type listPermissionAccepted struct{}

type listPermissionRejected struct {
	reason core.DomainError
}

func (listPermissionAccepted) listPermissionResult() {}

func (listPermissionRejected) listPermissionResult() {}

func (service Service) requireListPermission(ctx context.Context, actor auth.UserSubject, scope ListScope) listPermissionResult {
	switch typed := scope.(type) {
	case PublicListScope:
		return listPermissionAccepted{}
	case UserListScope:
		if typed.UserID != actor.ID {
			return listPermissionRejected{reason: core.NewDomainError(core.ErrorCodePermissionDenied, "task list access denied")}
		}
		return listPermissionAccepted{}
	case OrganizationListScope:
		check := service.organizationPermissions.CheckOrganizationPermission(ctx, typed.OrganizationID, actor.ID, org.PermissionCreateOrganizationTask)
		if rejected, matched := check.(org.PermissionDenied); matched {
			return listPermissionRejected{reason: rejected.Reason}
		}
		return listPermissionAccepted{}
	case TeamListScope:
		return listPermissionAccepted{}
	case CreatorListScope:
		return listPermissionAccepted{}
	case AssigneeListScope:
		return listPermissionAccepted{}
	default:
		return listPermissionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task list scope is invalid")}
	}
}

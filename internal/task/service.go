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
	ListTasks(context.Context, ListScope) ListTasksStoreResult
	CreateCapabilityToken(context.Context, core.TaskCapabilityTokenID, core.TaskID, CapabilityTokenHash) CreateCapabilityTokenStoreResult
	ListSeries(context.Context, core.UserID) ListSeriesStoreResult
	FindSeries(context.Context, core.TaskSeriesID) FindSeriesStoreResult
}

type OrganizationPermissions interface {
	CheckOrganizationPermission(context.Context, core.OrganizationID, core.UserID, org.Permission) org.PermissionCheck
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
	Reward         RewardSpec
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

type PublicListScope struct{}

type UserListScope struct {
	UserID core.UserID
}

type OrganizationListScope struct {
	OrganizationID core.OrganizationID
	UserID         core.UserID
}

func (PublicListScope) listScope() {}

func (UserListScope) listScope() {}

func (OrganizationListScope) listScope() {}

type ListResult interface {
	listResult()
}

type TasksListed struct {
	Values []Task
}

type ListRejected struct {
	Reason core.DomainError
}

func (TasksListed) listResult() {}

func (ListRejected) listResult() {}

func (service Service) List(ctx context.Context, actor auth.UserSubject, scope ListScope) ListResult {
	scopePermission := service.requireListPermission(ctx, actor, scope)
	if rejected, matched := scopePermission.(listPermissionRejected); matched {
		return ListRejected{Reason: rejected.reason}
	}

	storeResult := service.store.ListTasks(ctx, scope)
	listed, matched := storeResult.(ListTasksStoreAccepted)
	if !matched {
		rejected := storeResult.(ListTasksStoreRejected)
		return ListRejected{Reason: rejected.Reason}
	}
	return TasksListed{Values: listed.Values}
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
	default:
		return listPermissionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task list scope is invalid")}
	}
}

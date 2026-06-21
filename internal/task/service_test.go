package task

import (
	"context"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
)

func TestCapabilityTokenDoesNotContainTaskIdentifier(t *testing.T) {
	taskID := testTaskID(t)
	result := NewCapabilityTokenPlain()
	created, matched := result.(CapabilityTokenPlainAccepted)
	if !matched {
		t.Fatalf("result = %T, want CapabilityTokenPlainAccepted", result)
	}

	if strings.Contains(created.Value.String(), taskID.String()) {
		t.Fatalf("capability token contained task id")
	}

	if created.Value.Hash().String() == created.Value.String() {
		t.Fatalf("capability token hash equaled plain token")
	}
}

func TestParseCapabilityTokenPlainRejectsInvalidToken(t *testing.T) {
	result := ParseCapabilityTokenPlain("not base64")
	if _, matched := result.(CapabilityTokenPlainRejected); !matched {
		t.Fatalf("result = %T, want CapabilityTokenPlainRejected", result)
	}
}

func TestOpenStateOnlyAcceptsDraft(t *testing.T) {
	result := OpenState(StateDraft)
	if _, matched := result.(StateTransitionAccepted); !matched {
		t.Fatalf("draft transition = %T, want StateTransitionAccepted", result)
	}

	rejectedResult := OpenState(StateOpen)
	if _, matched := rejectedResult.(StateTransitionRejected); !matched {
		t.Fatalf("open transition = %T, want StateTransitionRejected", rejectedResult)
	}
}

func TestParseCapabilityTokenStateRejectsUnknownState(t *testing.T) {
	result := ParseCapabilityTokenState("missing")
	if _, matched := result.(CapabilityTokenStateRejected); !matched {
		t.Fatalf("result = %T, want CapabilityTokenStateRejected", result)
	}
}

func TestServiceRequiresPublicPublisherForOrganizationPublicTask(t *testing.T) {
	store := newTaskMemoryStore()
	permissions := newTaskPermissionStore()
	service := NewService(store, permissions)
	actor := testUserSubject(t)
	organizationID := testOrganizationID(t)
	command := testCreateCommand(t, actor, OrganizationOwner{OrganizationID: organizationID}, PublicVisibility{})

	result := service.Create(context.Background(), command)
	if _, matched := result.(CreateRejected); !matched {
		t.Fatalf("result = %T, want CreateRejected", result)
	}

	permissions.grant(organizationID, actor.ID, org.RolePublicPublisher, org.RoleAdmin)
	acceptedResult := service.Create(context.Background(), command)
	if _, matched := acceptedResult.(TaskCreated); !matched {
		t.Fatalf("result = %T, want TaskCreated", acceptedResult)
	}
}

func TestServiceUsesOrganizationDefaultHiddenVisibility(t *testing.T) {
	organizationID := testOrganizationID(t)
	var visibility Visibility = OrganizationVisibility{OrganizationID: organizationID}
	var owner Owner = OrganizationOwner{OrganizationID: organizationID}

	if _, matched := owner.(OrganizationOwner); !matched {
		t.Fatalf("owner = %T, want OrganizationOwner", owner)
	}
	if _, matched := visibility.(OrganizationVisibility); !matched {
		t.Fatalf("visibility = %T, want OrganizationVisibility", visibility)
	}
}

type taskMemoryStore struct {
	tasks map[string]Task
}

func newTaskMemoryStore() *taskMemoryStore {
	return &taskMemoryStore{tasks: make(map[string]Task)}
}

func (store *taskMemoryStore) CreateTask(_ context.Context, seriesID core.TaskSeriesID, taskID core.TaskID, command CreateCommand) CreateTaskStoreResult {
	taskValue := Task{
		ID:             taskID,
		Owner:          command.Owner,
		Title:          command.Title,
		Description:    command.Description,
		State:          StateDraft,
		Visibility:     command.Visibility,
		Placement:      command.Placement,
		ResponseSchema: command.ResponseSchema,
		Payload:        command.Payload,
		CreatedBy:      command.Actor.ID,
	}
	store.tasks[taskID.String()] = taskValue
	return CreateTaskStoreAccepted{Value: taskValue}
}

func (store *taskMemoryStore) FindTask(_ context.Context, taskID core.TaskID) FindTaskStoreResult {
	value, matched := store.tasks[taskID.String()]
	if !matched {
		return FindTaskStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task missing")}
	}
	return FindTaskStoreAccepted{Value: value}
}

func (store *taskMemoryStore) ChangeTaskState(_ context.Context, taskID core.TaskID, state State) ChangeTaskStateStoreResult {
	value, matched := store.tasks[taskID.String()]
	if !matched {
		return ChangeTaskStateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task missing")}
	}
	value.State = state
	store.tasks[taskID.String()] = value
	return ChangeTaskStateStoreAccepted{Value: value}
}

func (store *taskMemoryStore) ListTasks(context.Context, ListScope) ListTasksStoreResult {
	values := make([]Task, 0, len(store.tasks))
	for taskKey := range store.tasks {
		values = append(values, store.tasks[taskKey])
	}
	return ListTasksStoreAccepted{Values: values}
}

type taskPermissionStore struct {
	grants []taskPermissionGrant
}

type taskPermissionGrant struct {
	organizationID core.OrganizationID
	userID         core.UserID
	roles          []org.Role
}

var taskPermissionSeed = []taskPermissionGrant{}

func newTaskPermissionStore() *taskPermissionStore {
	return &taskPermissionStore{grants: taskPermissionSeed}
}

func (store *taskMemoryStore) CreateCapabilityToken(context.Context, core.TaskCapabilityTokenID, core.TaskID, CapabilityTokenHash) CreateCapabilityTokenStoreResult {
	reason := core.NewDomainError(core.ErrorCodeInvalidState, "not used")
	return CreateCapabilityTokenStoreRejected{Reason: reason}
}

func (store *taskPermissionStore) grant(organizationID core.OrganizationID, userID core.UserID, roles ...org.Role) {
	store.grants = append(store.grants, taskPermissionGrant{organizationID: organizationID, userID: userID, roles: roles})
}

func (store *taskPermissionStore) CheckOrganizationPermission(_ context.Context, organizationID core.OrganizationID, userID core.UserID, permission org.Permission) org.PermissionCheck {
	for grantIndex := range store.grants {
		grant := store.grants[grantIndex]
		if grant.organizationID == organizationID && grant.userID == userID {
			return org.CheckPermission(grant.roles, permission)
		}
	}
	return org.PermissionDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "roles missing")}
}

func testCreateCommand(t *testing.T, actor auth.UserSubject, owner Owner, visibility Visibility) CreateCommand {
	t.Helper()
	title := acceptedTitle(t, "Collect schema examples")
	description := acceptedDescription(t, "Find examples that exercise the local schema parser.")
	schemaSource := acceptedSchemaSource(t, `{"kind":"freeform"}`)
	return CreateCommand{
		Actor:          actor,
		Owner:          owner,
		Title:          title,
		Description:    description,
		Visibility:     visibility,
		Placement:      StandalonePlacement{},
		ResponseSchema: schemaSource,
		Payload:        NoDataPayload{},
	}
}

func acceptedTitle(t *testing.T, raw string) Title {
	t.Helper()
	result := NewTitle(raw)
	accepted, matched := result.(TitleAccepted)
	if !matched {
		t.Fatalf("title = %T, want TitleAccepted", result)
	}
	return accepted.Value
}

func acceptedDescription(t *testing.T, raw string) Description {
	t.Helper()
	result := NewDescription(raw)
	accepted, matched := result.(DescriptionAccepted)
	if !matched {
		t.Fatalf("description = %T, want DescriptionAccepted", result)
	}
	return accepted.Value
}

func acceptedSchemaSource(t *testing.T, raw string) ResponseSchemaSource {
	t.Helper()
	result := NewResponseSchemaSource(raw)
	accepted, matched := result.(ResponseSchemaSourceAccepted)
	if !matched {
		t.Fatalf("schema source = %T, want ResponseSchemaSourceAccepted", result)
	}
	return accepted.Value
}

func testUserSubject(t *testing.T) auth.UserSubject {
	t.Helper()
	return auth.UserSubject{ID: testUserID(t)}
}

func testUserID(t *testing.T) core.UserID {
	t.Helper()
	result := core.NewUserID()
	created, matched := result.(core.UserIDCreated)
	if !matched {
		t.Fatalf("user id = %T, want UserIDCreated", result)
	}
	return created.Value
}

func testTaskID(t *testing.T) core.TaskID {
	t.Helper()
	result := core.NewTaskID()
	created, matched := result.(core.TaskIDCreated)
	if !matched {
		t.Fatalf("task id = %T, want TaskIDCreated", result)
	}
	return created.Value
}

func testOrganizationID(t *testing.T) core.OrganizationID {
	t.Helper()
	result := core.NewOrganizationID()
	created, matched := result.(core.OrganizationIDCreated)
	if !matched {
		t.Fatalf("organization id = %T, want OrganizationIDCreated", result)
	}
	return created.Value
}

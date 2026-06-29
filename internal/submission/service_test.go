package submission

import (
	"context"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/task"
)

func TestSubmissionCreatesReceipt(t *testing.T) {
	store := newSubmissionMemoryStore()
	taskStore := newSubmissionTaskStore(t, task.PublicVisibility{}, `{"kind":"freeform"}`)
	service := NewService(store, taskStore, submissionPermissionStore{})
	command := testSubmitCommand(t, taskStore.value.ID, `{"answer":"done"}`)

	result := service.Submit(context.Background(), command)
	created, matched := result.(SubmissionCreated)
	if !matched {
		t.Fatalf("result = %T, want SubmissionCreated", result)
	}
	if created.ReceiptToken.String() == "" {
		t.Fatalf("receipt token is empty")
	}
}

func TestInvalidSubmissionIsRecordedWithValidationErrors(t *testing.T) {
	store := newSubmissionMemoryStore()
	taskStore := newSubmissionTaskStore(t, task.PublicVisibility{}, `{"kind":"object","fields":[{"name":"answer","presence":"required","schema":{"kind":"string"},"sensitivity":{"category":"","retention":"","redaction":""}}]}`)
	service := NewService(store, taskStore, submissionPermissionStore{})
	command := testSubmitCommand(t, taskStore.value.ID, `{"answer":12}`)

	result := service.Submit(context.Background(), command)
	created, matched := result.(SubmissionCreated)
	if !matched {
		t.Fatalf("result = %T, want SubmissionCreated", result)
	}
	if created.Value.State != StateInvalid {
		t.Fatalf("state = %q, want invalid", created.Value.State.String())
	}
	failed, failedMatched := created.Value.Validation.(ValidationFailed)
	if !failedMatched {
		t.Fatalf("validation = %T, want ValidationFailed", created.Value.Validation)
	}
	if len(failed.Errors) != 1 {
		t.Fatalf("validation errors = %d, want 1", len(failed.Errors))
	}
}

func TestReceiptStatusRedactsSensitiveFields(t *testing.T) {
	store := newSubmissionMemoryStore()
	taskStore := newSubmissionTaskStore(t, task.PublicVisibility{}, `{"kind":"object","fields":[{"name":"email","presence":"required","schema":{"kind":"string"},"sensitivity":{"category":"pii","retention":"delete_on_request","redaction":"replace"}}]}`)
	service := NewService(store, taskStore, submissionPermissionStore{})
	command := testSubmitCommand(t, taskStore.value.ID, `{"email":"person@example.com"}`)

	result := service.Submit(context.Background(), command)
	created := result.(SubmissionCreated)

	statusResult := service.FindByReceipt(context.Background(), created.ReceiptToken)
	statusFound, matched := statusResult.(ReceiptStatusFound)
	if !matched {
		t.Fatalf("status = %T, want ReceiptStatusFound", statusResult)
	}
	if strings.Contains(statusFound.Value.ResponseSource.String(), "person@example.com") {
		t.Fatalf("receipt response was not redacted")
	}
	if !strings.Contains(statusFound.Value.ResponseSource.String(), "[redacted]") {
		t.Fatalf("receipt response did not contain redaction marker")
	}
}

func TestSubmitRejectsClosedTask(t *testing.T) {
	store := newSubmissionMemoryStore()
	taskStore := newSubmissionTaskStore(t, task.PublicVisibility{}, `{"kind":"freeform"}`)
	taskStore.value.State = task.StateClosed
	service := NewService(store, taskStore, submissionPermissionStore{})
	command := testSubmitCommand(t, taskStore.value.ID, `{"answer":"done"}`)

	result := service.Submit(context.Background(), command)
	if _, matched := result.(SubmitRejected); !matched {
		t.Fatalf("result = %T, want SubmitRejected", result)
	}
}

func TestSubmitRejectsIneligibleReservation(t *testing.T) {
	store := newSubmissionMemoryStore()
	taskStore := newSubmissionTaskStore(t, task.PublicVisibility{}, `{"kind":"freeform"}`)
	taskStore.eligible = false
	service := NewService(store, taskStore, submissionPermissionStore{})
	command := testSubmitCommand(t, taskStore.value.ID, `{"answer":"done"}`)

	result := service.Submit(context.Background(), command)
	if _, matched := result.(SubmitRejected); !matched {
		t.Fatalf("result = %T, want SubmitRejected", result)
	}
	if len(store.valuesByID) != 0 {
		t.Fatalf("submissions stored = %d, want 0", len(store.valuesByID))
	}
}

func TestSubmitRejectsTaskHiddenFromSubmitter(t *testing.T) {
	store := newSubmissionMemoryStore()
	ownerID := submissionTestUserID(t)
	taskStore := newSubmissionTaskStore(t, task.UserVisibility{UserID: ownerID}, `{"kind":"freeform"}`)
	taskStore.value.CreatedBy = ownerID
	taskStore.value.Owner = task.UserOwner{UserID: ownerID}
	service := NewService(store, taskStore, submissionPermissionStore{})
	command := testSubmitCommand(t, taskStore.value.ID, `{"answer":"done"}`)

	result := service.Submit(context.Background(), command)
	if _, matched := result.(SubmitRejected); !matched {
		t.Fatalf("result = %T, want SubmitRejected", result)
	}
}

func TestOrganizationReviewerCanListSubmissions(t *testing.T) {
	store := newSubmissionMemoryStore()
	organizationID := submissionTestOrganizationID(t)
	taskStore := newSubmissionTaskStore(t, task.OrganizationVisibility{OrganizationID: organizationID}, `{"kind":"freeform"}`)
	taskStore.value.Owner = task.OrganizationOwner{OrganizationID: organizationID}
	service := NewService(store, taskStore, submissionPermissionStore{organizationID: organizationID, roles: []org.Role{org.RoleReviewer}})
	command := testSubmitCommand(t, taskStore.value.ID, `{"answer":"done"}`)
	if _, matched := service.Submit(context.Background(), command).(SubmissionCreated); !matched {
		t.Fatalf("submit was rejected")
	}

	result := service.ListForTask(context.Background(), testAuthSubject(t, command.SubmitterID), taskStore.value.ID, core.DefaultPage())
	if _, matched := result.(SubmissionsListed); !matched {
		t.Fatalf("result = %T, want SubmissionsListed", result)
	}
}

func TestSubmissionCommentVisibility(t *testing.T) {
	store := newSubmissionMemoryStore()
	taskStore := newSubmissionTaskStore(t, task.PublicVisibility{}, `{"kind":"freeform"}`)
	service := NewService(store, taskStore, submissionPermissionStore{})

	submitterID := submissionTestUserID(t)
	command := SubmitCommand{TaskID: taskStore.value.ID, SubmitterID: submitterID, ResponseSource: acceptedResponseSource(t, `{"answer":"done"}`)}
	created := service.Submit(context.Background(), command).(SubmissionCreated)

	body := acceptedCommentBody(t, "Could you clarify the scope?")

	// The submission's author may comment.
	submitterComment, matched := service.AddSubmissionComment(context.Background(), testAuthSubject(t, submitterID), created.Value.ID, body).(SubmissionCommentAdded)
	if !matched {
		t.Fatalf("submitter comment was not added")
	}
	if submitterComment.TaskID != taskStore.value.ID || submitterComment.SubmitterID != submitterID || submitterComment.TaskCreatorID != taskStore.value.CreatedBy {
		t.Fatalf("comment notification context = task %s submitter %s creator %s", submitterComment.TaskID.String(), submitterComment.SubmitterID.String(), submitterComment.TaskCreatorID.String())
	}

	// The owner of the submission's task (the task creator) may comment.
	ownerID := taskStore.value.CreatedBy
	if _, matched := service.AddSubmissionComment(context.Background(), testAuthSubject(t, ownerID), created.Value.ID, body).(SubmissionCommentAdded); !matched {
		t.Fatalf("task owner comment was not added")
	}

	listed, listMatched := service.ListSubmissionComments(context.Background(), testAuthSubject(t, ownerID), created.Value.ID).(SubmissionCommentsListed)
	if !listMatched {
		t.Fatalf("task owner could not list submission comments")
	}
	if len(listed.Values) != 2 {
		t.Fatalf("submission comments = %d, want 2", len(listed.Values))
	}

	// An unrelated user is denied.
	strangerID := submissionTestUserID(t)
	if _, matched := service.AddSubmissionComment(context.Background(), testAuthSubject(t, strangerID), created.Value.ID, body).(SubmissionCommentRejected); !matched {
		t.Fatalf("stranger comment was not rejected")
	}
	if _, matched := service.ListSubmissionComments(context.Background(), testAuthSubject(t, strangerID), created.Value.ID).(SubmissionCommentsListRejected); !matched {
		t.Fatalf("stranger list was not rejected")
	}
}

func acceptedResponseSource(t *testing.T, raw string) ResponseSource {
	t.Helper()
	result := NewResponseSource(raw)
	accepted := result.(ResponseSourceAccepted)
	return accepted.Value
}

func acceptedCommentBody(t *testing.T, raw string) task.CommentBody {
	t.Helper()
	result := task.NewCommentBody(raw)
	accepted, matched := result.(task.CommentBodyAccepted)
	if !matched {
		t.Fatalf("comment body = %T, want CommentBodyAccepted", result)
	}
	return accepted.Value
}

type submissionMemoryStore struct {
	valuesByID           map[string]Submission
	submissionByHash     map[string]string
	commentsBySubmission map[string][]SubmissionComment
}

func newSubmissionMemoryStore() *submissionMemoryStore {
	return &submissionMemoryStore{valuesByID: make(map[string]Submission), submissionByHash: make(map[string]string), commentsBySubmission: make(map[string][]SubmissionComment)}
}

func (store *submissionMemoryStore) FindSubmission(_ context.Context, submissionID core.SubmissionID) FindSubmissionStoreResult {
	value, matched := store.valuesByID[submissionID.String()]
	if !matched {
		return FindSubmissionStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "submission missing")}
	}
	return FindSubmissionStoreAccepted{Value: value}
}

func (store *submissionMemoryStore) CreateSubmissionComment(_ context.Context, comment SubmissionComment) CreateSubmissionCommentStoreResult {
	store.commentsBySubmission[comment.SubmissionID.String()] = append(store.commentsBySubmission[comment.SubmissionID.String()], comment)
	return CreateSubmissionCommentStoreAccepted{Value: comment}
}

func (store *submissionMemoryStore) ListSubmissionComments(_ context.Context, submissionID core.SubmissionID) ListSubmissionCommentsStoreResult {
	return ListSubmissionCommentsStoreAccepted{Values: store.commentsBySubmission[submissionID.String()]}
}

func (store *submissionMemoryStore) CreateSubmission(_ context.Context, submissionID core.SubmissionID, receiptID core.SubmissionReceiptTokenID, receiptHash ReceiptTokenHash, command SubmitCommand, state State, outcome ValidationOutcome, sensitiveFields []SensitiveField) CreateSubmissionStoreResult {
	value := Submission{ID: submissionID, TaskID: command.TaskID, SubmitterID: command.SubmitterID, State: state, ResponseSource: command.ResponseSource, Validation: outcome}
	store.valuesByID[submissionID.String()] = value
	store.submissionByHash[receiptHash.String()] = submissionID.String()
	accepted := CreateSubmissionStoreAccepted{Value: value}
	return accepted
}

func (store *submissionMemoryStore) FindByReceiptToken(_ context.Context, hash ReceiptTokenHash) FindReceiptStoreResult {
	submissionID, matched := store.submissionByHash[hash.String()]
	if !matched {
		return ReceiptMissing{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "receipt missing")}
	}
	submissionValue := store.valuesByID[submissionID]
	return ReceiptFound{
		Value: submissionValue,
	}
}

type submissionTaskStore struct {
	value    task.Task
	eligible bool
}

func newSubmissionTaskStore(t *testing.T, visibility task.Visibility, schemaSource string) *submissionTaskStore {
	t.Helper()
	return &submissionTaskStore{value: task.Task{
		ID:             submissionTestTaskID(t),
		Owner:          task.UserOwner{UserID: submissionTestUserID(t)},
		Title:          acceptedTaskTitle(t),
		Description:    acceptedTaskDescription(t),
		Reward:         task.NoRewardSpec{},
		Participation:  task.ParticipationPolicyOpen,
		AssigneeScope:  task.AssigneeScopeUser,
		ReservationTTL: task.DefaultReservationTTL(),
		State:          task.StateOpen,
		Visibility:     visibility,
		Placement:      task.StandalonePlacement{},
		ResponseSchema: acceptedTaskSchema(t, schemaSource),
		Payload:        task.NoDataPayload{},
		CreatedBy:      submissionTestUserID(t),
	}, eligible: true}
}

type submissionPermissionStore struct {
	organizationID core.OrganizationID
	roles          []org.Role
}

func (store submissionPermissionStore) CheckOrganizationPermission(_ context.Context, organizationID core.OrganizationID, _ core.UserID, permission org.Permission) org.PermissionCheck {
	if organizationID != store.organizationID {
		return org.PermissionDenied{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "organization permission denied")}
	}
	return org.CheckPermission(store.roles, permission)
}

func (store *submissionTaskStore) FindTask(_ context.Context, taskID core.TaskID) task.FindTaskStoreResult {
	if store.value.ID != taskID {
		return task.FindTaskStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task missing")}
	}
	return task.FindTaskStoreAccepted{Value: store.value}
}

func (store *submissionTaskStore) CheckSubmissionEligibility(context.Context, core.TaskID, core.UserID) task.SubmissionEligibilityStoreResult {
	if !store.eligible {
		return task.SubmissionEligibilityRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "submitter does not hold the active task reservation")}
	}
	return task.SubmissionEligible{}
}

func testSubmitCommand(t *testing.T, taskID core.TaskID, response string) SubmitCommand {
	t.Helper()
	sourceResult := NewResponseSource(response)
	source := sourceResult.(ResponseSourceAccepted)
	return SubmitCommand{TaskID: taskID, SubmitterID: submissionTestUserID(t), ResponseSource: source.Value}
}

func submissionTestUserID(t *testing.T) core.UserID {
	t.Helper()
	result := core.NewUserID()
	created, matched := result.(core.UserIDCreated)
	if !matched {
		t.Fatalf("submission test user id = %T, want UserIDCreated", result)
	}
	return created.Value
}

func submissionTestTaskID(t *testing.T) core.TaskID {
	t.Helper()
	result := core.NewTaskID()
	created, matched := result.(core.TaskIDCreated)
	if !matched {
		t.Fatalf("submission test task id = %T, want TaskIDCreated", result)
	}
	return created.Value
}

func submissionTestOrganizationID(t *testing.T) core.OrganizationID {
	t.Helper()
	result := core.NewOrganizationID()
	created, matched := result.(core.OrganizationIDCreated)
	if !matched {
		t.Fatalf("submission test organization id = %T, want OrganizationIDCreated", result)
	}
	return created.Value
}

func testAuthSubject(t *testing.T, userID core.UserID) auth.UserSubject {
	t.Helper()
	return auth.UserSubject{ID: userID}
}

func acceptedTaskTitle(t *testing.T) task.Title {
	t.Helper()
	result := task.NewTitle("Task")
	accepted, matched := result.(task.TitleAccepted)
	if !matched {
		t.Fatalf("task title = %T, want TitleAccepted", result)
	}
	return accepted.Value
}

func acceptedTaskDescription(t *testing.T) task.Description {
	t.Helper()
	result := task.NewDescription("Task description")
	accepted, matched := result.(task.DescriptionAccepted)
	if !matched {
		t.Fatalf("task description = %T, want DescriptionAccepted", result)
	}
	return accepted.Value
}

func acceptedTaskSchema(t *testing.T, raw string) task.ResponseSchemaSource {
	t.Helper()
	result := task.NewResponseSchemaSource(raw)
	accepted, matched := result.(task.ResponseSchemaSourceAccepted)
	if !matched {
		t.Fatalf("task schema = %T, want ResponseSchemaSourceAccepted", result)
	}
	return accepted.Value
}

func (store *submissionMemoryStore) ListForTask(_ context.Context, taskID core.TaskID, _ core.Page) ListSubmissionsStoreResult {
	values := make([]Submission, 0)
	for key := range store.valuesByID {
		value := store.valuesByID[key]
		if value.TaskID == taskID {
			values = append(values, value)
		}
	}
	return ListSubmissionsStoreAccepted{
		Values: values,
	}
}

func (store *submissionMemoryStore) ListForSubmitter(_ context.Context, submitterID core.UserID) ListSubmissionsStoreResult {
	values := make([]Submission, 0)
	for key := range store.valuesByID {
		value := store.valuesByID[key]
		if value.SubmitterID == submitterID {
			values = append(values, value)
		}
	}
	return ListSubmissionsStoreAccepted{Values: values}
}

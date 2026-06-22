package submission

import (
	"context"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
)

func TestSubmissionCreatesReceipt(t *testing.T) {
	store := newSubmissionMemoryStore()
	taskStore := newSubmissionTaskStore(t, task.PublicVisibility{}, `{"kind":"freeform"}`)
	service := NewService(store, taskStore)
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
	service := NewService(store, taskStore)
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
	service := NewService(store, taskStore)
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

type submissionMemoryStore struct {
	valuesByID       map[string]Submission
	submissionByHash map[string]string
}

func newSubmissionMemoryStore() *submissionMemoryStore {
	return &submissionMemoryStore{valuesByID: make(map[string]Submission), submissionByHash: make(map[string]string)}
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
	value task.Task
}

func newSubmissionTaskStore(t *testing.T, visibility task.Visibility, schemaSource string) *submissionTaskStore {
	t.Helper()
	return &submissionTaskStore{value: task.Task{
		ID:             submissionTestTaskID(t),
		Owner:          task.UserOwner{UserID: submissionTestUserID(t)},
		Title:          acceptedTaskTitle(t),
		Description:    acceptedTaskDescription(t),
		State:          task.StateOpen,
		Visibility:     visibility,
		Placement:      task.StandalonePlacement{},
		ResponseSchema: acceptedTaskSchema(t, schemaSource),
		Payload:        task.NoDataPayload{},
		CreatedBy:      submissionTestUserID(t),
	}}
}

func (store *submissionTaskStore) FindTask(_ context.Context, taskID core.TaskID) task.FindTaskStoreResult {
	if store.value.ID != taskID {
		return task.FindTaskStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task missing")}
	}
	return task.FindTaskStoreAccepted{Value: store.value}
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

func (store *submissionMemoryStore) ListForTask(_ context.Context, taskID core.TaskID) ListSubmissionsStoreResult {
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

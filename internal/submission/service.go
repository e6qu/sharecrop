package submission

import (
	"context"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/schema"
	"github.com/e6qu/sharecrop/internal/task"
)

type Store interface {
	CreateSubmission(context.Context, core.SubmissionID, core.SubmissionReceiptTokenID, ReceiptTokenHash, SubmitCommand, State, ValidationOutcome, []SensitiveField) CreateSubmissionStoreResult
	FindByReceiptToken(context.Context, ReceiptTokenHash) FindReceiptStoreResult
	ListForTask(context.Context, core.TaskID) ListSubmissionsStoreResult
}

type TaskFinder interface {
	FindTask(context.Context, core.TaskID) task.FindTaskStoreResult
}

type Service struct {
	store     Store
	taskStore TaskFinder
}

func NewService(store Store, taskStore TaskFinder) Service {
	return Service{store: store, taskStore: taskStore}
}

type SubmitCommand struct {
	TaskID         core.TaskID
	Submitter      Submitter
	ResponseSource ResponseSource
}

type SubmitResult interface {
	submitResult()
}

type SubmissionCreated struct {
	Value        Submission
	ReceiptToken ReceiptTokenPlain
}

type SubmitRejected struct {
	Reason core.DomainError
}

func (SubmissionCreated) submitResult() {}

func (SubmitRejected) submitResult() {}

func (service Service) Submit(ctx context.Context, command SubmitCommand) SubmitResult {
	taskResult := service.taskStore.FindTask(ctx, command.TaskID)
	taskFound, taskMatched := taskResult.(task.FindTaskStoreAccepted)
	if !taskMatched {
		rejected := taskResult.(task.FindTaskStoreRejected)
		return SubmitRejected{Reason: rejected.Reason}
	}

	permission := canSubmitToTask(command.Submitter, taskFound.Value)
	if rejected, matched := permission.(submitPermissionRejected); matched {
		return SubmitRejected{Reason: rejected.reason}
	}

	schemaResult := schema.ParseSchemaJSON([]byte(taskFound.Value.ResponseSchema.String()))
	schemaParsed, schemaMatched := schemaResult.(schema.SchemaParsed)
	if !schemaMatched {
		rejected := schemaResult.(schema.SchemaParseRejected)
		return SubmitRejected{Reason: rejected.Reason}
	}

	valueResult := schema.ParseValueJSON([]byte(command.ResponseSource.String()))
	valueParsed, valueMatched := valueResult.(schema.ValueParsed)
	if !valueMatched {
		rejected := valueResult.(schema.ValueParseRejected)
		return SubmitRejected{Reason: rejected.Reason}
	}

	outcome := validationOutcome(schemaParsed.Value, valueParsed.Value)
	state := StateForValidation(outcome)
	sensitiveFields := sensitiveFieldsFor(schemaParsed.Value, valueParsed.Value)

	submissionIDResult := core.NewSubmissionID()
	submissionIDCreated, submissionIDMatched := submissionIDResult.(core.SubmissionIDCreated)
	if !submissionIDMatched {
		rejected := submissionIDResult.(core.SubmissionIDRejected)
		return SubmitRejected{Reason: rejected.Reason}
	}

	receiptIDResult := core.NewSubmissionReceiptTokenID()
	receiptIDCreated, receiptIDMatched := receiptIDResult.(core.SubmissionReceiptTokenIDCreated)
	if !receiptIDMatched {
		rejected := receiptIDResult.(core.SubmissionReceiptTokenIDRejected)
		return SubmitRejected{Reason: rejected.Reason}
	}

	receiptTokenResult := NewReceiptTokenPlain()
	receiptTokenCreated, receiptTokenMatched := receiptTokenResult.(ReceiptTokenPlainAccepted)
	if !receiptTokenMatched {
		rejected := receiptTokenResult.(ReceiptTokenPlainRejected)
		return SubmitRejected{Reason: rejected.Reason}
	}

	storeResult := service.store.CreateSubmission(ctx, submissionIDCreated.Value, receiptIDCreated.Value, receiptTokenCreated.Value.Hash(), command, state, outcome, sensitiveFields)
	created, createdMatched := storeResult.(CreateSubmissionStoreAccepted)
	if !createdMatched {
		rejected := storeResult.(CreateSubmissionStoreRejected)
		return SubmitRejected{Reason: rejected.Reason}
	}

	return SubmissionCreated{Value: created.Value, ReceiptToken: receiptTokenCreated.Value}
}

type ReceiptStatusResult interface {
	receiptStatusResult()
}

type ReceiptStatusFound struct {
	Value Submission
}

type ReceiptStatusRejected struct {
	Reason core.DomainError
}

func (ReceiptStatusFound) receiptStatusResult() {}

func (ReceiptStatusRejected) receiptStatusResult() {}

func (service Service) FindByReceipt(ctx context.Context, token ReceiptTokenPlain) ReceiptStatusResult {
	result := service.store.FindByReceiptToken(ctx, token.Hash())
	found, matched := result.(ReceiptFound)
	if !matched {
		rejected := result.(ReceiptMissing)
		return ReceiptStatusRejected{Reason: rejected.Reason}
	}

	redactedResult := service.redactedSubmission(ctx, found.Value)
	redacted, redactedMatched := redactedResult.(redactedSubmissionAccepted)
	if !redactedMatched {
		rejected := redactedResult.(redactedSubmissionRejected)
		return ReceiptStatusRejected{Reason: rejected.reason}
	}

	return ReceiptStatusFound{Value: redacted.value}
}

type ListResult interface {
	listResult()
}

type SubmissionsListed struct {
	Values []Submission
}

type ListRejected struct {
	Reason core.DomainError
}

func (SubmissionsListed) listResult() {}

func (ListRejected) listResult() {}

func (service Service) ListForTask(ctx context.Context, actor auth.UserSubject, taskID core.TaskID) ListResult {
	taskResult := service.taskStore.FindTask(ctx, taskID)
	taskFound, taskMatched := taskResult.(task.FindTaskStoreAccepted)
	if !taskMatched {
		rejected := taskResult.(task.FindTaskStoreRejected)
		return ListRejected{Reason: rejected.Reason}
	}
	if taskFound.Value.CreatedBy != actor.ID {
		return ListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "submission list access denied")}
	}

	result := service.store.ListForTask(ctx, taskID)
	listed, matched := result.(ListSubmissionsStoreAccepted)
	if !matched {
		rejected := result.(ListSubmissionsStoreRejected)
		return ListRejected{Reason: rejected.Reason}
	}
	return SubmissionsListed{Values: listed.Values}
}

type submitPermissionResult interface {
	submitPermissionResult()
}

type submitPermissionAccepted struct{}

type submitPermissionRejected struct {
	reason core.DomainError
}

func (submitPermissionAccepted) submitPermissionResult() {}

func (submitPermissionRejected) submitPermissionResult() {}

func canSubmitToTask(submitter Submitter, value task.Task) submitPermissionResult {
	if _, anonymous := submitter.(AnonymousSubmitter); anonymous {
		if _, public := value.Visibility.(task.PublicVisibility); public {
			return submitPermissionAccepted{}
		}
		return submitPermissionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "anonymous submissions require public tasks")}
	}
	return submitPermissionAccepted{}
}

func validationOutcome(schemaValue schema.Schema, value schema.Value) ValidationOutcome {
	result := schema.Validate(schemaValue, value)
	if _, accepted := result.(schema.ValidationAccepted); accepted {
		return ValidationPassed{}
	}
	rejected := result.(schema.ValidationRejected)
	errors := make([]ValidationError, 0, len(rejected.Errors))
	for errorIndex := range rejected.Errors {
		validationError := rejected.Errors[errorIndex]
		errors = append(errors, ValidationError{Path: validationError.Path.String(), Message: validationError.Message})
	}
	return ValidationFailed{Errors: errors}
}

func sensitiveFieldsFor(schemaValue schema.Schema, value schema.Value) []SensitiveField {
	result := schema.BuildSensitiveIndex(schemaValue, value)
	built := result.(schema.SensitiveIndexBuilt)
	fields := make([]SensitiveField, 0, len(built.Fields))
	for fieldIndex := range built.Fields {
		field := built.Fields[fieldIndex]
		fields = append(fields, SensitiveField{
			Path:      field.Path.String(),
			Category:  field.Sensitivity.Category.String(),
			Retention: field.Sensitivity.Retention.String(),
			Redaction: field.Sensitivity.Redaction.String(),
		})
	}
	return fields
}

type redactedSubmissionResult interface {
	redactedSubmissionResult()
}

type redactedSubmissionAccepted struct {
	value Submission
}

type redactedSubmissionRejected struct {
	reason core.DomainError
}

func (redactedSubmissionAccepted) redactedSubmissionResult() {}

func (redactedSubmissionRejected) redactedSubmissionResult() {}

func (service Service) redactedSubmission(ctx context.Context, value Submission) redactedSubmissionResult {
	taskResult := service.taskStore.FindTask(ctx, value.TaskID)
	taskFound, taskMatched := taskResult.(task.FindTaskStoreAccepted)
	if !taskMatched {
		rejected := taskResult.(task.FindTaskStoreRejected)
		return redactedSubmissionRejected{reason: rejected.Reason}
	}

	schemaResult := schema.ParseSchemaJSON([]byte(taskFound.Value.ResponseSchema.String()))
	schemaParsed, schemaMatched := schemaResult.(schema.SchemaParsed)
	if !schemaMatched {
		rejected := schemaResult.(schema.SchemaParseRejected)
		return redactedSubmissionRejected{reason: rejected.Reason}
	}

	valueResult := schema.ParseValueJSON([]byte(value.ResponseSource.String()))
	valueParsed, valueMatched := valueResult.(schema.ValueParsed)
	if !valueMatched {
		rejected := valueResult.(schema.ValueParseRejected)
		return redactedSubmissionRejected{reason: rejected.Reason}
	}

	redactionResult := schema.RedactSensitive(schemaParsed.Value, valueParsed.Value)
	redactedValue := redactionResult.(schema.ValueRedacted)
	encodedResult := schema.EncodeValueJSON(redactedValue.Value)
	encoded, encodedMatched := encodedResult.(schema.ValueEncoded)
	if !encodedMatched {
		rejected := encodedResult.(schema.ValueEncodeRejected)
		return redactedSubmissionRejected{reason: rejected.Reason}
	}

	sourceResult := NewResponseSource(encoded.Source)
	source, sourceMatched := sourceResult.(ResponseSourceAccepted)
	if !sourceMatched {
		rejected := sourceResult.(ResponseSourceRejected)
		return redactedSubmissionRejected{reason: rejected.Reason}
	}

	value.ResponseSource = source.Value
	return redactedSubmissionAccepted{value: value}
}

package submission

import (
	"context"

	"github.com/e6qu/sharecrop/internal/attachment"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/schema"
	"github.com/e6qu/sharecrop/internal/task"
)

type Store interface {
	CreateSubmission(context.Context, core.SubmissionID, core.SubmissionReceiptTokenID, ReceiptTokenHash, SubmitCommand, State, ValidationOutcome, []SensitiveField) CreateSubmissionStoreResult
	FindByReceiptToken(context.Context, ReceiptTokenHash) FindReceiptStoreResult
	FindSubmission(context.Context, core.SubmissionID) FindSubmissionStoreResult
	ListForTask(context.Context, core.TaskID, core.Page) ListSubmissionsStoreResult
	ListForSubmitter(context.Context, core.UserID, core.Page) ListSubmissionsStoreResult
	CreateSubmissionComment(context.Context, SubmissionComment) CreateSubmissionCommentStoreResult
	ListSubmissionComments(context.Context, core.SubmissionID) ListSubmissionCommentsStoreResult
}

type TaskFinder interface {
	FindTask(context.Context, core.TaskID) task.FindTaskStoreResult
	CheckSubmissionEligibility(context.Context, core.TaskID, core.UserID) task.SubmissionEligibilityStoreResult
}

type OrganizationPermissions interface {
	CheckOrganizationPermission(context.Context, core.OrganizationID, core.UserID, org.Permission) org.PermissionCheck
}

type Service struct {
	store                   Store
	taskStore               TaskFinder
	organizationPermissions OrganizationPermissions
}

func NewService(store Store, taskStore TaskFinder, organizationPermissions OrganizationPermissions) Service {
	return Service{store: store, taskStore: taskStore, organizationPermissions: organizationPermissions}
}

type SubmitCommand struct {
	TaskID         core.TaskID
	SubmitterID    core.UserID
	ResponseSource ResponseSource
	Attachments    []attachment.Attachment
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
	if taskFound.Value.State != task.StateOpen {
		return SubmitRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only open tasks accept submissions")}
	}
	if rejected, matched := service.requireViewPermission(ctx, command.SubmitterID, taskFound.Value).(viewPermissionRejected); matched {
		return SubmitRejected{Reason: rejected.reason}
	}
	eligibility := service.taskStore.CheckSubmissionEligibility(ctx, command.TaskID, command.SubmitterID)
	if rejected, matched := eligibility.(task.SubmissionEligibilityRejected); matched {
		return SubmitRejected{Reason: rejected.Reason}
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

type GetResult interface {
	getResult()
}

type SubmissionGot struct {
	Value Submission
}

type GetRejected struct {
	Reason core.DomainError
}

func (SubmissionGot) getResult() {}

func (GetRejected) getResult() {}

func (service Service) Get(ctx context.Context, actor auth.Subject, submissionID core.SubmissionID) GetResult {
	value, problem := service.loadCommentableSubmission(ctx, actor, submissionID)
	if problem != nil {
		return GetRejected{Reason: *problem}
	}
	return SubmissionGot{Value: value}
}

func (service Service) ListForTask(ctx context.Context, actor auth.Subject, taskID core.TaskID, page core.Page) ListResult {
	taskResult := service.taskStore.FindTask(ctx, taskID)
	taskFound, taskMatched := taskResult.(task.FindTaskStoreAccepted)
	if !taskMatched {
		rejected := taskResult.(task.FindTaskStoreRejected)
		return ListRejected{Reason: rejected.Reason}
	}
	if rejected, matched := service.requireReviewPermissionForActor(ctx, actor, taskFound.Value).(reviewPermissionRejected); matched {
		return ListRejected{Reason: rejected.reason}
	}

	result := service.store.ListForTask(ctx, taskID, page)
	listed, matched := result.(ListSubmissionsStoreAccepted)
	if !matched {
		rejected := result.(ListSubmissionsStoreRejected)
		return ListRejected{Reason: rejected.Reason}
	}
	return SubmissionsListed{Values: listed.Values}
}

// ListForSubmitter returns a submitter's own submissions. Only the submitter may
// read their submissions; another user is denied so submissions never leak.
func (service Service) ListForSubmitter(ctx context.Context, actor auth.UserSubject, submitterID core.UserID, page core.Page) ListResult {
	if actor.ID != submitterID {
		return ListRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "submissions are visible only to their submitter")}
	}

	result := service.store.ListForSubmitter(ctx, submitterID, page)
	listed, matched := result.(ListSubmissionsStoreAccepted)
	if !matched {
		rejected := result.(ListSubmissionsStoreRejected)
		return ListRejected{Reason: rejected.Reason}
	}
	return SubmissionsListed{Values: listed.Values}
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

func (service Service) requireViewPermission(ctx context.Context, userID core.UserID, value task.Task) viewPermissionResult {
	if value.CreatedBy == userID {
		return viewPermissionAccepted{}
	}
	switch typed := value.Visibility.(type) {
	case task.PublicVisibility:
		return viewPermissionAccepted{}
	case task.UserVisibility:
		if typed.UserID == userID {
			return viewPermissionAccepted{}
		}
		return viewPermissionRejected{reason: core.NewDomainError(core.ErrorCodePermissionDenied, "task view access denied")}
	case task.OrganizationVisibility:
		return service.requireOrganizationViewPermission(ctx, typed.OrganizationID, userID)
	case task.OrganizationTeamVisibility:
		return service.requireOrganizationViewPermission(ctx, typed.OrganizationID, userID)
	default:
		return viewPermissionRejected{reason: core.NewDomainError(core.ErrorCodePermissionDenied, "task view access denied")}
	}
}

func (service Service) requireOrganizationViewPermission(ctx context.Context, organizationID core.OrganizationID, userID core.UserID) viewPermissionResult {
	check := service.organizationPermissions.CheckOrganizationPermission(ctx, organizationID, userID, org.PermissionReviewSubmissions)
	if _, granted := check.(org.PermissionGranted); granted {
		return viewPermissionAccepted{}
	}
	check = service.organizationPermissions.CheckOrganizationPermission(ctx, organizationID, userID, org.PermissionCreateOrganizationTask)
	if rejected, matched := check.(org.PermissionDenied); matched {
		return viewPermissionRejected{reason: rejected.Reason}
	}
	return viewPermissionAccepted{}
}

type reviewPermissionResult interface {
	reviewPermissionResult()
}

type reviewPermissionAccepted struct{}

type reviewPermissionRejected struct {
	reason core.DomainError
}

func (reviewPermissionAccepted) reviewPermissionResult() {}

func (reviewPermissionRejected) reviewPermissionResult() {}

func (service Service) requireReviewPermission(ctx context.Context, userID core.UserID, value task.Task) reviewPermissionResult {
	if value.CreatedBy == userID {
		return reviewPermissionAccepted{}
	}
	organizationIDResult := organizationIDForTask(value)
	organizationIDFound, matched := organizationIDResult.(organizationIDFound)
	if !matched {
		return reviewPermissionRejected{reason: core.NewDomainError(core.ErrorCodePermissionDenied, "submission list access denied")}
	}
	check := service.organizationPermissions.CheckOrganizationPermission(ctx, organizationIDFound.value, userID, org.PermissionReviewSubmissions)
	if rejected, permissionMatched := check.(org.PermissionDenied); permissionMatched {
		return reviewPermissionRejected{reason: rejected.Reason}
	}
	return reviewPermissionAccepted{}
}

// requireReviewPermissionForActor adds org-token support in front of
// requireReviewPermission: an org token gets unconditional review access to
// submissions on its own org's tasks (full parity with an org-admin
// member), without needing a userID to check against a per-member
// permission table. A UserSubject actor delegates to the existing
// userID-based check unchanged.
func (service Service) requireReviewPermissionForActor(ctx context.Context, actor auth.Subject, value task.Task) reviewPermissionResult {
	if orgActor, isOrg := actor.(auth.OrgSubject); isOrg {
		organizationIDResult := organizationIDForTask(value)
		if found, matched := organizationIDResult.(organizationIDFound); matched && found.value == orgActor.ID {
			return reviewPermissionAccepted{}
		}
		return reviewPermissionRejected{reason: core.NewDomainError(core.ErrorCodePermissionDenied, "submission list access denied")}
	}
	userActor, isUser := actor.(auth.UserSubject)
	if !isUser {
		return reviewPermissionRejected{reason: core.NewDomainError(core.ErrorCodePermissionDenied, "submission list access denied")}
	}
	return service.requireReviewPermission(ctx, userActor.ID, value)
}

type organizationIDForTaskResult interface {
	organizationIDForTaskResult()
}

type organizationIDFound struct {
	value core.OrganizationID
}

type organizationIDMissing struct{}

func (organizationIDFound) organizationIDForTaskResult() {}

func (organizationIDMissing) organizationIDForTaskResult() {}

func organizationIDForTask(value task.Task) organizationIDForTaskResult {
	switch typed := value.Owner.(type) {
	case task.OrganizationOwner:
		return organizationIDFound{value: typed.OrganizationID}
	case task.OrganizationTeamOwner:
		return organizationIDFound{value: typed.OrganizationID}
	default:
		return organizationIDMissing{}
	}
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
			Path:       field.Path.String(),
			Category:   field.Sensitivity.Category.String(),
			Retention:  field.Sensitivity.Retention.String(),
			Redaction:  field.Sensitivity.Redaction.String(),
			State:      "active",
			RedactedAt: "",
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

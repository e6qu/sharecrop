// Package submissionbridge is the WASI bridge for internal/submission's Store:
// hand-written per-type codecs (this file) plus a generated dispatcher and guest
// client (bridge_gen.go). Shared core types (ids, page) are serialized by
// internal/wasibridge/corewire; the domain error by internal/wasibridge/domainwire.
package submissionbridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/wasibridge/attachmentwire"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
)

// ---- opaque value types (string wrappers) ----

func encodeReceiptTokenHash(hash submission.ReceiptTokenHash) string { return hash.String() }

// decodeReceiptTokenHash reconstructs the stored hash without re-hashing.
func decodeReceiptTokenHash(raw string) (submission.ReceiptTokenHash, error) {
	return submission.ReceiptTokenHashFromString(raw), nil
}

func encodeState(state submission.State) string { return state.String() }

func decodeState(raw string) (submission.State, error) {
	parsed, matched := submission.ParseState(raw).(submission.StateParsed)
	if !matched {
		return submission.State{}, fmt.Errorf("invalid submission state %q", raw)
	}
	return parsed.Value, nil
}

func encodeResponseSource(source submission.ResponseSource) string { return source.String() }

func decodeResponseSource(raw string) (submission.ResponseSource, error) {
	accepted, matched := submission.NewResponseSource(raw).(submission.ResponseSourceAccepted)
	if !matched {
		return submission.ResponseSource{}, fmt.Errorf("invalid submission response source")
	}
	return accepted.Value, nil
}

func encodeReviewNote(note submission.ReviewNote) string { return note.String() }

func decodeReviewNote(raw string) (submission.ReviewNote, error) {
	accepted, matched := submission.NewStoredReviewNote(raw).(submission.ReviewNoteAccepted)
	if !matched {
		return submission.EmptyReviewNote(), fmt.Errorf("invalid review note")
	}
	return accepted.Value, nil
}

func encodeCommentBody(body task.CommentBody) string { return body.String() }

func decodeCommentBody(raw string) (task.CommentBody, error) {
	accepted, matched := task.NewCommentBody(raw).(task.CommentBodyAccepted)
	if !matched {
		return task.CommentBody{}, fmt.Errorf("invalid comment body")
	}
	return accepted.Value, nil
}

// ---- submission.ValidationOutcome ----

type validationErrorWire struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type validationOutcomeWire struct {
	Variant string                `json:"variant"`
	Errors  []validationErrorWire `json:"errors,omitempty"`
}

func encodeValidationOutcome(outcome submission.ValidationOutcome) validationOutcomeWire {
	failed, matched := outcome.(submission.ValidationFailed)
	if !matched {
		return validationOutcomeWire{Variant: "passed"}
	}
	errors := make([]validationErrorWire, 0, len(failed.Errors))
	for index := range failed.Errors {
		errors = append(errors, validationErrorWire{Path: failed.Errors[index].Path, Message: failed.Errors[index].Message})
	}
	return validationOutcomeWire{Variant: "failed", Errors: errors}
}

func decodeValidationOutcome(wire validationOutcomeWire) (submission.ValidationOutcome, error) {
	switch wire.Variant {
	case "passed":
		return submission.ValidationPassed{}, nil
	case "failed":
		errors := make([]submission.ValidationError, 0, len(wire.Errors))
		for index := range wire.Errors {
			errors = append(errors, submission.ValidationError{Path: wire.Errors[index].Path, Message: wire.Errors[index].Message})
		}
		return submission.ValidationFailed{Errors: errors}, nil
	default:
		return nil, fmt.Errorf("unknown validation outcome variant %q", wire.Variant)
	}
}

// ---- submission.SensitiveField ----

type sensitiveFieldWire struct {
	Path       string `json:"path"`
	Category   string `json:"category"`
	Retention  string `json:"retention"`
	Redaction  string `json:"redaction"`
	State      string `json:"state"`
	RedactedAt string `json:"redacted_at"`
}

func encodeSensitiveFields(values []submission.SensitiveField) []sensitiveFieldWire {
	encoded := make([]sensitiveFieldWire, 0, len(values))
	for index := range values {
		encoded = append(encoded, sensitiveFieldWire{
			Path:       values[index].Path,
			Category:   values[index].Category,
			Retention:  values[index].Retention,
			Redaction:  values[index].Redaction,
			State:      values[index].State,
			RedactedAt: values[index].RedactedAt,
		})
	}
	return encoded
}

func decodeSensitiveFields(wires []sensitiveFieldWire) ([]submission.SensitiveField, error) {
	values := make([]submission.SensitiveField, 0, len(wires))
	for index := range wires {
		values = append(values, submission.SensitiveField{
			Path:       wires[index].Path,
			Category:   wires[index].Category,
			Retention:  wires[index].Retention,
			Redaction:  wires[index].Redaction,
			State:      wires[index].State,
			RedactedAt: wires[index].RedactedAt,
		})
	}
	return values, nil
}

// ---- submission.SubmitCommand ----

type submitCommandWire struct {
	TaskID         string                `json:"task_id"`
	SubmitterID    string                `json:"submitter_id"`
	ResponseSource string                `json:"response_source"`
	Attachments    []attachmentwire.Wire `json:"attachments,omitempty"`
}

func encodeSubmitCommand(command submission.SubmitCommand) submitCommandWire {
	return submitCommandWire{
		TaskID:         corewire.EncodeTaskID(command.TaskID),
		SubmitterID:    corewire.EncodeUserID(command.SubmitterID),
		ResponseSource: encodeResponseSource(command.ResponseSource),
		Attachments:    attachmentwire.EncodeSlice(command.Attachments),
	}
}

func decodeSubmitCommand(wire submitCommandWire) (submission.SubmitCommand, error) {
	taskID, err := corewire.DecodeTaskID(wire.TaskID)
	if err != nil {
		return submission.SubmitCommand{}, err
	}
	submitterID, err := corewire.DecodeUserID(wire.SubmitterID)
	if err != nil {
		return submission.SubmitCommand{}, err
	}
	responseSource, err := decodeResponseSource(wire.ResponseSource)
	if err != nil {
		return submission.SubmitCommand{}, err
	}
	attachments, err := attachmentwire.DecodeSlice(wire.Attachments)
	if err != nil {
		return submission.SubmitCommand{}, err
	}
	return submission.SubmitCommand{
		TaskID:         taskID,
		SubmitterID:    submitterID,
		ResponseSource: responseSource,
		Attachments:    attachments,
	}, nil
}

// ---- submission.Submission ----

type submissionWire struct {
	ID              string                `json:"id"`
	TaskID          string                `json:"task_id"`
	SubmitterID     string                `json:"submitter_id"`
	State           string                `json:"state"`
	ResponseSource  string                `json:"response_source"`
	Attachments     []attachmentwire.Wire `json:"attachments,omitempty"`
	Validation      validationOutcomeWire `json:"validation"`
	SensitiveFields []sensitiveFieldWire  `json:"sensitive_fields,omitempty"`
	ReviewNote      string                `json:"review_note"`
}

func encodeSubmission(value submission.Submission) submissionWire {
	return submissionWire{
		ID:              corewire.EncodeSubmissionID(value.ID),
		TaskID:          corewire.EncodeTaskID(value.TaskID),
		SubmitterID:     corewire.EncodeUserID(value.SubmitterID),
		State:           encodeState(value.State),
		ResponseSource:  encodeResponseSource(value.ResponseSource),
		Attachments:     attachmentwire.EncodeSlice(value.Attachments),
		Validation:      encodeValidationOutcome(value.Validation),
		SensitiveFields: encodeSensitiveFields(value.SensitiveFields),
		ReviewNote:      encodeReviewNote(value.ReviewNote),
	}
}

func decodeSubmission(wire submissionWire) (submission.Submission, error) {
	id, err := corewire.DecodeSubmissionID(wire.ID)
	if err != nil {
		return submission.Submission{}, err
	}
	taskID, err := corewire.DecodeTaskID(wire.TaskID)
	if err != nil {
		return submission.Submission{}, err
	}
	submitterID, err := corewire.DecodeUserID(wire.SubmitterID)
	if err != nil {
		return submission.Submission{}, err
	}
	state, err := decodeState(wire.State)
	if err != nil {
		return submission.Submission{}, err
	}
	responseSource, err := decodeResponseSource(wire.ResponseSource)
	if err != nil {
		return submission.Submission{}, err
	}
	attachments, err := attachmentwire.DecodeSlice(wire.Attachments)
	if err != nil {
		return submission.Submission{}, err
	}
	validation, err := decodeValidationOutcome(wire.Validation)
	if err != nil {
		return submission.Submission{}, err
	}
	sensitiveFields, err := decodeSensitiveFields(wire.SensitiveFields)
	if err != nil {
		return submission.Submission{}, err
	}
	reviewNote, err := decodeReviewNote(wire.ReviewNote)
	if err != nil {
		return submission.Submission{}, err
	}
	return submission.Submission{
		ID:              id,
		TaskID:          taskID,
		SubmitterID:     submitterID,
		State:           state,
		ResponseSource:  responseSource,
		Attachments:     attachments,
		Validation:      validation,
		SensitiveFields: sensitiveFields,
		ReviewNote:      reviewNote,
	}, nil
}

func encodeSubmissions(values []submission.Submission) []submissionWire {
	encoded := make([]submissionWire, 0, len(values))
	for index := range values {
		encoded = append(encoded, encodeSubmission(values[index]))
	}
	return encoded
}

func decodeSubmissions(wires []submissionWire) ([]submission.Submission, error) {
	values := make([]submission.Submission, 0, len(wires))
	for index := range wires {
		value, err := decodeSubmission(wires[index])
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// ---- submission.SubmissionComment ----

type submissionCommentWire struct {
	ID           string `json:"id"`
	SubmissionID string `json:"submission_id"`
	AuthorID     string `json:"author_id"`
	Body         string `json:"body"`
	CreatedAt    string `json:"created_at"`
}

func encodeSubmissionComment(comment submission.SubmissionComment) submissionCommentWire {
	return submissionCommentWire{
		ID:           corewire.EncodeSubmissionCommentID(comment.ID),
		SubmissionID: corewire.EncodeSubmissionID(comment.SubmissionID),
		AuthorID:     corewire.EncodeUserID(comment.AuthorID),
		Body:         encodeCommentBody(comment.Body),
		CreatedAt:    corewire.EncodeTime(comment.CreatedAt),
	}
}

func decodeSubmissionComment(wire submissionCommentWire) (submission.SubmissionComment, error) {
	id, err := corewire.DecodeSubmissionCommentID(wire.ID)
	if err != nil {
		return submission.SubmissionComment{}, err
	}
	submissionID, err := corewire.DecodeSubmissionID(wire.SubmissionID)
	if err != nil {
		return submission.SubmissionComment{}, err
	}
	authorID, err := corewire.DecodeUserID(wire.AuthorID)
	if err != nil {
		return submission.SubmissionComment{}, err
	}
	body, err := decodeCommentBody(wire.Body)
	if err != nil {
		return submission.SubmissionComment{}, err
	}
	createdAt, err := corewire.DecodeTime(wire.CreatedAt)
	if err != nil {
		return submission.SubmissionComment{}, err
	}
	return submission.SubmissionComment{
		ID:           id,
		SubmissionID: submissionID,
		AuthorID:     authorID,
		Body:         body,
		CreatedAt:    createdAt,
	}, nil
}

func encodeSubmissionComments(values []submission.SubmissionComment) []submissionCommentWire {
	encoded := make([]submissionCommentWire, 0, len(values))
	for index := range values {
		encoded = append(encoded, encodeSubmissionComment(values[index]))
	}
	return encoded
}

func decodeSubmissionComments(wires []submissionCommentWire) ([]submission.SubmissionComment, error) {
	values := make([]submission.SubmissionComment, 0, len(wires))
	for index := range wires {
		value, err := decodeSubmissionComment(wires[index])
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// ---- result unions ----

// submissionResultWire backs the create, find-by-receipt, and find results,
// which each carry a single submission on success.
type submissionResultWire struct {
	Variant    string                  `json:"variant"`
	Submission *submissionWire         `json:"submission,omitempty"`
	Error      *domainwire.DomainError `json:"error,omitempty"`
}

func encodeCreateSubmissionResult(result submission.CreateSubmissionStoreResult) submissionResultWire {
	switch typed := result.(type) {
	case submission.CreateSubmissionStoreAccepted:
		return acceptedSubmissionWire("created", typed.Value)
	case submission.CreateSubmissionStoreRejected:
		return rejectedSubmissionWire(typed.Reason)
	default:
		return submissionResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown submission result %T", result))}
	}
}

func decodeCreateSubmissionResult(wire submissionResultWire) (submission.CreateSubmissionStoreResult, error) {
	switch wire.Variant {
	case "created":
		value, err := decodeSubmissionPayload(wire.Submission)
		if err != nil {
			return nil, err
		}
		return submission.CreateSubmissionStoreAccepted{Value: value}, nil
	case "rejected":
		return submission.CreateSubmissionStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown create submission result variant %q", wire.Variant)
	}
}

func encodeFindReceiptResult(result submission.FindReceiptStoreResult) submissionResultWire {
	switch typed := result.(type) {
	case submission.ReceiptFound:
		return acceptedSubmissionWire("found", typed.Value)
	case submission.ReceiptMissing:
		return rejectedSubmissionWire(typed.Reason)
	default:
		return submissionResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown submission result %T", result))}
	}
}

func decodeFindReceiptResult(wire submissionResultWire) (submission.FindReceiptStoreResult, error) {
	switch wire.Variant {
	case "found":
		value, err := decodeSubmissionPayload(wire.Submission)
		if err != nil {
			return nil, err
		}
		return submission.ReceiptFound{Value: value}, nil
	case "rejected":
		return submission.ReceiptMissing{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown find receipt result variant %q", wire.Variant)
	}
}

func encodeFindSubmissionResult(result submission.FindSubmissionStoreResult) submissionResultWire {
	switch typed := result.(type) {
	case submission.FindSubmissionStoreAccepted:
		return acceptedSubmissionWire("found", typed.Value)
	case submission.FindSubmissionStoreRejected:
		return rejectedSubmissionWire(typed.Reason)
	default:
		return submissionResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown submission result %T", result))}
	}
}

func decodeFindSubmissionResult(wire submissionResultWire) (submission.FindSubmissionStoreResult, error) {
	switch wire.Variant {
	case "found":
		value, err := decodeSubmissionPayload(wire.Submission)
		if err != nil {
			return nil, err
		}
		return submission.FindSubmissionStoreAccepted{Value: value}, nil
	case "rejected":
		return submission.FindSubmissionStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown find submission result variant %q", wire.Variant)
	}
}

type submissionsResultWire struct {
	Variant     string                  `json:"variant"`
	Submissions []submissionWire        `json:"submissions,omitempty"`
	Error       *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListSubmissionsResult(result submission.ListSubmissionsStoreResult) submissionsResultWire {
	switch typed := result.(type) {
	case submission.ListSubmissionsStoreAccepted:
		return submissionsResultWire{Variant: "listed", Submissions: encodeSubmissions(typed.Values)}
	case submission.ListSubmissionsStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return submissionsResultWire{Variant: "rejected", Error: &reason}
	default:
		return submissionsResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown submission result %T", result))}
	}
}

func decodeListSubmissionsResult(wire submissionsResultWire) (submission.ListSubmissionsStoreResult, error) {
	switch wire.Variant {
	case "listed":
		values, err := decodeSubmissions(wire.Submissions)
		if err != nil {
			return nil, err
		}
		return submission.ListSubmissionsStoreAccepted{Values: values}, nil
	case "rejected":
		return submission.ListSubmissionsStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown list submissions result variant %q", wire.Variant)
	}
}

type submissionCommentResultWire struct {
	Variant string                  `json:"variant"`
	Comment *submissionCommentWire  `json:"comment,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeCreateCommentResult(result submission.CreateSubmissionCommentStoreResult) submissionCommentResultWire {
	switch typed := result.(type) {
	case submission.CreateSubmissionCommentStoreAccepted:
		comment := encodeSubmissionComment(typed.Value)
		return submissionCommentResultWire{Variant: "created", Comment: &comment}
	case submission.CreateSubmissionCommentStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return submissionCommentResultWire{Variant: "rejected", Error: &reason}
	default:
		return submissionCommentResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown submission result %T", result))}
	}
}

func decodeCreateCommentResult(wire submissionCommentResultWire) (submission.CreateSubmissionCommentStoreResult, error) {
	switch wire.Variant {
	case "created":
		value, err := decodeCommentPayload(wire.Comment)
		if err != nil {
			return nil, err
		}
		return submission.CreateSubmissionCommentStoreAccepted{Value: value}, nil
	case "rejected":
		return submission.CreateSubmissionCommentStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown create comment result variant %q", wire.Variant)
	}
}

type submissionCommentsResultWire struct {
	Variant  string                  `json:"variant"`
	Comments []submissionCommentWire `json:"comments,omitempty"`
	Error    *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListCommentsResult(result submission.ListSubmissionCommentsStoreResult) submissionCommentsResultWire {
	switch typed := result.(type) {
	case submission.ListSubmissionCommentsStoreAccepted:
		return submissionCommentsResultWire{Variant: "listed", Comments: encodeSubmissionComments(typed.Values)}
	case submission.ListSubmissionCommentsStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return submissionCommentsResultWire{Variant: "rejected", Error: &reason}
	default:
		return submissionCommentsResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown submission result %T", result))}
	}
}

func decodeListCommentsResult(wire submissionCommentsResultWire) (submission.ListSubmissionCommentsStoreResult, error) {
	switch wire.Variant {
	case "listed":
		values, err := decodeSubmissionComments(wire.Comments)
		if err != nil {
			return nil, err
		}
		return submission.ListSubmissionCommentsStoreAccepted{Values: values}, nil
	case "rejected":
		return submission.ListSubmissionCommentsStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown list comments result variant %q", wire.Variant)
	}
}

// ---- shared result helpers ----

// acceptedSubmissionWire builds the success arm shared by the three
// single-submission results; only the variant tag differs between them.
func acceptedSubmissionWire(variant string, value submission.Submission) submissionResultWire {
	encoded := encodeSubmission(value)
	return submissionResultWire{Variant: variant, Submission: &encoded}
}

func rejectedSubmissionWire(reason core.DomainError) submissionResultWire {
	encoded := domainwire.EncodeDomainError(reason)
	return submissionResultWire{Variant: "rejected", Error: &encoded}
}

func decodeSubmissionPayload(wire *submissionWire) (submission.Submission, error) {
	if wire == nil {
		return submission.Submission{}, fmt.Errorf("result is missing its submission")
	}
	return decodeSubmission(*wire)
}

func decodeCommentPayload(wire *submissionCommentWire) (submission.SubmissionComment, error) {
	if wire == nil {
		return submission.SubmissionComment{}, fmt.Errorf("result is missing its comment")
	}
	return decodeSubmissionComment(*wire)
}

func decodeReason(wire *domainwire.DomainError) core.DomainError {
	if wire == nil {
		return core.NewDomainError(core.ErrorCodeInvalidState, "submission bridge: rejected result is missing its error")
	}
	return domainwire.DecodeDomainError(*wire)
}

func rejectionError(message string) *domainwire.DomainError {
	reason := domainwire.EncodeDomainError(core.NewDomainError(core.ErrorCodeInvalidState, message))
	return &reason
}

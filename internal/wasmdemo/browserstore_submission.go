package wasmdemo

import (
	"context"

	"github.com/e6qu/sharecrop/internal/attachment"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

// SubmissionBrowserStore implements submission.Store against
// BrowserStorage. CreateSubmission also marks the submitter's active
// reservation as "submitted", the same cross-cutting write the real
// Postgres store makes inside the same transaction (internal/db's
// SubmissionStore.CreateSubmission) - so it reads/writes the reservation
// records TaskBrowserStore owns (browserstore_task.go).
type SubmissionBrowserStore struct {
	storage BrowserStorage
	ids     InteractionIDSource
}

func NewSubmissionBrowserStore(storage BrowserStorage, ids InteractionIDSource) SubmissionBrowserStore {
	return SubmissionBrowserStore{storage: storage, ids: ids}
}

type storedSubmission struct {
	ID              string                  `json:"id"`
	TaskID          string                  `json:"task_id"`
	SubmitterID     string                  `json:"submitter_id"`
	State           string                  `json:"state"`
	ResponseJSON    string                  `json:"response_json"`
	ReviewNote      string                  `json:"review_note"`
	ReceiptHash     string                  `json:"receipt_hash"`
	ValidationOK    bool                    `json:"validation_ok"`
	ValidationErrs  []storedValidationError `json:"validation_errors,omitempty"`
	SensitiveFields []storedSensitiveField  `json:"sensitive_fields,omitempty"`
}

type storedValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type storedSensitiveField struct {
	Path       string `json:"path"`
	Category   string `json:"category"`
	Retention  string `json:"retention"`
	Redaction  string `json:"redaction"`
	State      string `json:"state"`
	RedactedAt string `json:"redacted_at"`
}

func submissionRecordKey(id string) string { return "submission:record:" + id }
func submissionReceiptKey(hash string) string {
	return "submission:receipt:" + hash
}
func submissionTaskIndexKey(taskID string) string {
	return "submission:task_index:" + taskID
}
func submissionSubmitterIndexKey(submitterID string) string {
	return "submission:submitter_index:" + submitterID
}

func (store SubmissionBrowserStore) loadSubmission(id string) (storedSubmission, bool, *core.DomainError) {
	var record storedSubmission
	found, ok := getTaskJSON(store.storage, submissionRecordKey(id), &record)
	if !ok {
		reason := invalidState("submission lookup failed")
		return storedSubmission{}, false, &reason
	}
	return record, found, nil
}

func parseStoredSubmission(record storedSubmission) (submission.Submission, *core.DomainError) {
	idResult := core.ParseSubmissionID(record.ID)
	id, idMatched := idResult.(core.SubmissionIDCreated)
	if !idMatched {
		reason := idResult.(core.SubmissionIDRejected).Reason
		return submission.Submission{}, &reason
	}
	taskIDResult := core.ParseTaskID(record.TaskID)
	taskID, taskIDMatched := taskIDResult.(core.TaskIDCreated)
	if !taskIDMatched {
		reason := taskIDResult.(core.TaskIDRejected).Reason
		return submission.Submission{}, &reason
	}
	submitterResult := core.ParseUserID(record.SubmitterID)
	submitter, submitterMatched := submitterResult.(core.UserIDCreated)
	if !submitterMatched {
		reason := submitterResult.(core.UserIDRejected).Reason
		return submission.Submission{}, &reason
	}
	stateResult := submission.ParseState(record.State)
	state, stateMatched := stateResult.(submission.StateParsed)
	if !stateMatched {
		reason := stateResult.(submission.StateParseRejected).Reason
		return submission.Submission{}, &reason
	}
	sourceResult := submission.NewResponseSource(record.ResponseJSON)
	source, sourceMatched := sourceResult.(submission.ResponseSourceAccepted)
	if !sourceMatched {
		reason := sourceResult.(submission.ResponseSourceRejected).Reason
		return submission.Submission{}, &reason
	}
	noteResult := submission.NewStoredReviewNote(record.ReviewNote)
	note, noteMatched := noteResult.(submission.ReviewNoteAccepted)
	if !noteMatched {
		reason := noteResult.(submission.ReviewNoteRejected).Reason
		return submission.Submission{}, &reason
	}

	var outcome submission.ValidationOutcome = submission.ValidationPassed{}
	if !record.ValidationOK {
		errs := make([]submission.ValidationError, 0, len(record.ValidationErrs))
		for _, entry := range record.ValidationErrs {
			errs = append(errs, submission.ValidationError{Path: entry.Path, Message: entry.Message})
		}
		outcome = submission.ValidationFailed{Errors: errs}
	}

	sensitiveFields := make([]submission.SensitiveField, 0, len(record.SensitiveFields))
	for _, entry := range record.SensitiveFields {
		sensitiveFields = append(sensitiveFields, submission.SensitiveField{
			Path: entry.Path, Category: entry.Category, Retention: entry.Retention,
			Redaction: entry.Redaction, State: entry.State, RedactedAt: entry.RedactedAt,
		})
	}

	return submission.Submission{
		ID: id.Value, TaskID: taskID.Value, SubmitterID: submitter.Value, State: state.Value,
		ResponseSource: source.Value, Attachments: []attachment.Attachment{}, Validation: outcome,
		SensitiveFields: sensitiveFields, ReviewNote: note.Value,
	}, nil
}

func (store SubmissionBrowserStore) CreateSubmission(_ context.Context, submissionID core.SubmissionID, receiptID core.SubmissionReceiptTokenID, receiptHash submission.ReceiptTokenHash, command submission.SubmitCommand, state submission.State, outcome submission.ValidationOutcome, sensitiveFields []submission.SensitiveField) submission.CreateSubmissionStoreResult {
	record := storedSubmission{
		ID: submissionID.String(), TaskID: command.TaskID.String(), SubmitterID: command.SubmitterID.String(),
		State: state.String(), ResponseJSON: command.ResponseSource.String(), ReceiptHash: receiptHash.String(),
	}
	if failed, matched := outcome.(submission.ValidationFailed); matched {
		for _, entry := range failed.Errors {
			record.ValidationErrs = append(record.ValidationErrs, storedValidationError{Path: entry.Path, Message: entry.Message})
		}
	} else {
		record.ValidationOK = true
	}
	for _, entry := range sensitiveFields {
		record.SensitiveFields = append(record.SensitiveFields, storedSensitiveField{
			Path: entry.Path, Category: entry.Category, Retention: entry.Retention,
			Redaction: entry.Redaction, State: entry.State, RedactedAt: entry.RedactedAt,
		})
	}

	if !putTaskJSON(store.storage, submissionRecordKey(record.ID), record) {
		return submission.CreateSubmissionStoreRejected{Reason: invalidState("insert submission failed")}
	}
	if !putTaskJSON(store.storage, submissionReceiptKey(receiptHash.String()), record.ID) {
		return submission.CreateSubmissionStoreRejected{Reason: invalidState("insert submission receipt token failed")}
	}
	if _, matched := appendStringIndex(store.storage, submissionTaskIndexKey(record.TaskID), record.ID, "submission").(stringIndexStored); !matched {
		return submission.CreateSubmissionStoreRejected{Reason: invalidState("update submission task index failed")}
	}
	if _, matched := appendStringIndex(store.storage, submissionSubmitterIndexKey(record.SubmitterID), record.ID, "submission").(stringIndexStored); !matched {
		return submission.CreateSubmissionStoreRejected{Reason: invalidState("update submission submitter index failed")}
	}

	if err := store.markActiveReservationSubmitted(command.TaskID.String(), command.SubmitterID.String()); err != nil {
		return submission.CreateSubmissionStoreRejected{Reason: *err}
	}

	value, parseErr := parseStoredSubmission(record)
	if parseErr != nil {
		return submission.CreateSubmissionStoreRejected{Reason: *parseErr}
	}
	return submission.CreateSubmissionStoreAccepted{Value: value}
}

// markActiveReservationSubmitted mirrors internal/db's CreateSubmission:
// the submitter's active user-assignee reservation on this task flips to
// "submitted" so a new reservation can be requested if the submission is
// later rejected.
func (store SubmissionBrowserStore) markActiveReservationSubmitted(taskID string, submitterID string) *core.DomainError {
	taskStore := TaskBrowserStore{storage: store.storage, ids: store.ids}
	reservations, err := taskStore.loadReservations(taskID)
	if err != nil {
		return err
	}
	for _, reservation := range reservations {
		if reservation.State != task.ReservationStateActive {
			continue
		}
		userAssignee, matched := reservation.Assignee.(task.UserAssignee)
		if !matched || userAssignee.UserID.String() != submitterID {
			continue
		}
		var record storedReservation
		found, ok := getTaskJSON(store.storage, reservationRecordKey(reservation.ID.String()), &record)
		if !ok || !found {
			reason := invalidState("mark task reservation submitted failed")
			return &reason
		}
		record.State = "submitted"
		if !putTaskJSON(store.storage, reservationRecordKey(reservation.ID.String()), record) {
			reason := invalidState("mark task reservation submitted failed")
			return &reason
		}
	}
	return nil
}

func (store SubmissionBrowserStore) FindByReceiptToken(_ context.Context, hash submission.ReceiptTokenHash) submission.FindReceiptStoreResult {
	var submissionID string
	found, ok := getTaskJSON(store.storage, submissionReceiptKey(hash.String()), &submissionID)
	if !ok {
		return submission.ReceiptMissing{Reason: invalidState("find submission receipt failed")}
	}
	if !found {
		return submission.ReceiptMissing{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "submission receipt was not found")}
	}
	record, recordFound, err := store.loadSubmission(submissionID)
	if err != nil {
		return submission.ReceiptMissing{Reason: *err}
	}
	if !recordFound {
		return submission.ReceiptMissing{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "submission receipt was not found")}
	}
	value, parseErr := parseStoredSubmission(record)
	if parseErr != nil {
		return submission.ReceiptMissing{Reason: *parseErr}
	}
	return submission.ReceiptFound{Value: value}
}

func (store SubmissionBrowserStore) FindSubmission(_ context.Context, submissionID core.SubmissionID) submission.FindSubmissionStoreResult {
	record, found, err := store.loadSubmission(submissionID.String())
	if err != nil {
		return submission.FindSubmissionStoreRejected{Reason: *err}
	}
	if !found {
		return submission.FindSubmissionStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "submission was not found")}
	}
	value, parseErr := parseStoredSubmission(record)
	if parseErr != nil {
		return submission.FindSubmissionStoreRejected{Reason: *parseErr}
	}
	return submission.FindSubmissionStoreAccepted{Value: value}
}

func (store SubmissionBrowserStore) listByIndex(indexKey string, page core.Page) submission.ListSubmissionsStoreResult {
	indexResult := loadStringIndex(store.storage, indexKey, "submission")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		return submission.ListSubmissionsStoreRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
	}
	values := make([]submission.Submission, 0, len(loaded.values))
	for _, id := range loaded.values {
		record, found, err := store.loadSubmission(id)
		if err != nil {
			return submission.ListSubmissionsStoreRejected{Reason: *err}
		}
		if !found {
			continue
		}
		value, parseErr := parseStoredSubmission(record)
		if parseErr != nil {
			return submission.ListSubmissionsStoreRejected{Reason: *parseErr}
		}
		values = append(values, value)
	}
	start := page.Offset()
	if start > len(values) {
		start = len(values)
	}
	end := start + page.Limit()
	if end > len(values) {
		end = len(values)
	}
	return submission.ListSubmissionsStoreAccepted{Values: values[start:end]}
}

func (store SubmissionBrowserStore) ListForTask(_ context.Context, taskID core.TaskID, page core.Page) submission.ListSubmissionsStoreResult {
	return store.listByIndex(submissionTaskIndexKey(taskID.String()), page)
}

func (store SubmissionBrowserStore) ListForSubmitter(_ context.Context, submitterID core.UserID, page core.Page) submission.ListSubmissionsStoreResult {
	return store.listByIndex(submissionSubmitterIndexKey(submitterID.String()), page)
}

type storedSubmissionComment struct {
	ID           string `json:"id"`
	SubmissionID string `json:"submission_id"`
	AuthorID     string `json:"author_id"`
	Body         string `json:"body"`
}

func submissionCommentRecordKey(id string) string { return "submission:comment:" + id }
func submissionCommentIndexKey(submissionID string) string {
	return "submission:comment_index:" + submissionID
}

func (store SubmissionBrowserStore) CreateSubmissionComment(_ context.Context, comment submission.SubmissionComment) submission.CreateSubmissionCommentStoreResult {
	record := storedSubmissionComment{
		ID: comment.ID.String(), SubmissionID: comment.SubmissionID.String(),
		AuthorID: comment.AuthorID.String(), Body: comment.Body.String(),
	}
	if !putTaskJSON(store.storage, submissionCommentRecordKey(record.ID), record) {
		return submission.CreateSubmissionCommentStoreRejected{Reason: invalidState("insert submission comment failed")}
	}
	if _, matched := appendStringIndex(store.storage, submissionCommentIndexKey(record.SubmissionID), record.ID, "submission comment").(stringIndexStored); !matched {
		return submission.CreateSubmissionCommentStoreRejected{Reason: invalidState("update submission comment index failed")}
	}
	value, parseErr := parseStoredSubmissionComment(record)
	if parseErr != nil {
		return submission.CreateSubmissionCommentStoreRejected{Reason: *parseErr}
	}
	return submission.CreateSubmissionCommentStoreAccepted{Value: value}
}

func parseStoredSubmissionComment(record storedSubmissionComment) (submission.SubmissionComment, *core.DomainError) {
	idResult := core.ParseSubmissionCommentID(record.ID)
	id, idMatched := idResult.(core.SubmissionCommentIDCreated)
	if !idMatched {
		reason := idResult.(core.SubmissionCommentIDRejected).Reason
		return submission.SubmissionComment{}, &reason
	}
	submissionIDResult := core.ParseSubmissionID(record.SubmissionID)
	submissionID, submissionIDMatched := submissionIDResult.(core.SubmissionIDCreated)
	if !submissionIDMatched {
		reason := submissionIDResult.(core.SubmissionIDRejected).Reason
		return submission.SubmissionComment{}, &reason
	}
	authorResult := core.ParseUserID(record.AuthorID)
	author, authorMatched := authorResult.(core.UserIDCreated)
	if !authorMatched {
		reason := authorResult.(core.UserIDRejected).Reason
		return submission.SubmissionComment{}, &reason
	}
	bodyResult := task.NewCommentBody(record.Body)
	body, bodyMatched := bodyResult.(task.CommentBodyAccepted)
	if !bodyMatched {
		reason := bodyResult.(task.CommentBodyRejected).Reason
		return submission.SubmissionComment{}, &reason
	}
	return submission.SubmissionComment{ID: id.Value, SubmissionID: submissionID.Value, AuthorID: author.Value, Body: body.Value}, nil
}

func (store SubmissionBrowserStore) ListSubmissionComments(_ context.Context, submissionID core.SubmissionID) submission.ListSubmissionCommentsStoreResult {
	indexResult := loadStringIndex(store.storage, submissionCommentIndexKey(submissionID.String()), "submission comment")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		return submission.ListSubmissionCommentsStoreRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
	}
	values := make([]submission.SubmissionComment, 0, len(loaded.values))
	for _, id := range loaded.values {
		var record storedSubmissionComment
		found, ok := getTaskJSON(store.storage, submissionCommentRecordKey(id), &record)
		if !ok {
			return submission.ListSubmissionCommentsStoreRejected{Reason: invalidState("read submission comment failed")}
		}
		if !found {
			continue
		}
		value, parseErr := parseStoredSubmissionComment(record)
		if parseErr != nil {
			return submission.ListSubmissionCommentsStoreRejected{Reason: *parseErr}
		}
		values = append(values, value)
	}
	return submission.ListSubmissionCommentsStoreAccepted{Values: values}
}

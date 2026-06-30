package db

import (
	"context"

	"github.com/e6qu/sharecrop/internal/attachment"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubmissionStore struct {
	pool *pgxpool.Pool
}

func NewSubmissionStore(pool *pgxpool.Pool) SubmissionStore {
	return SubmissionStore{pool: pool}
}

func (store SubmissionStore) CreateSubmission(ctx context.Context, submissionID core.SubmissionID, receiptID core.SubmissionReceiptTokenID, receiptHash submission.ReceiptTokenHash, command submission.SubmitCommand, state submission.State, outcome submission.ValidationOutcome, sensitiveFields []submission.SensitiveField) submission.CreateSubmissionStoreResult {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return submission.CreateSubmissionStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin create submission transaction failed")}
	}
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		_ = rollbackErr
	}()

	_, err = tx.Exec(ctx, `
		insert into submissions (id, task_id, user_id, state, response_json)
		values ($1, $2, $3, $4, $5::jsonb)
	`, submissionID.String(), command.TaskID.String(), command.SubmitterID.String(), state.String(), command.ResponseSource.String())
	if err != nil {
		return submission.CreateSubmissionStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert submission failed")}
	}

	_, err = tx.Exec(ctx, `
		update task_reservations
		set state = 'submitted', state_recorded_at = now()
		where task_id = $1 and assignee_kind = 'user' and user_id = $2 and state = 'active'
	`, command.TaskID.String(), command.SubmitterID.String())
	if err != nil {
		return submission.CreateSubmissionStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "mark task reservation submitted failed")}
	}

	_, err = tx.Exec(ctx, `
		insert into submission_receipt_tokens (id, submission_id, token_hash)
		values ($1, $2, $3)
	`, receiptID.String(), submissionID.String(), receiptHash.String())
	if err != nil {
		return submission.CreateSubmissionStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert submission receipt token failed")}
	}

	errorResult := insertValidationErrors(ctx, tx, submissionID, outcome)
	if rejected, matched := errorResult.(insertRowsRejected); matched {
		return submission.CreateSubmissionStoreRejected{Reason: rejected.reason}
	}

	sensitiveResult := insertSensitiveFields(ctx, tx, submissionID, sensitiveFields)
	if rejected, matched := sensitiveResult.(insertRowsRejected); matched {
		return submission.CreateSubmissionStoreRejected{Reason: rejected.reason}
	}

	attachmentsResult := insertSubmissionAttachments(ctx, tx, submissionID, command.Attachments)
	if rejected, matched := attachmentsResult.(insertAttachmentsRejected); matched {
		return submission.CreateSubmissionStoreRejected{Reason: rejected.reason}
	}

	if err := tx.Commit(ctx); err != nil {
		return submission.CreateSubmissionStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit create submission transaction failed")}
	}

	return submission.CreateSubmissionStoreAccepted{Value: submission.Submission{
		ID:              submissionID,
		TaskID:          command.TaskID,
		SubmitterID:     command.SubmitterID,
		State:           state,
		ResponseSource:  command.ResponseSource,
		Attachments:     command.Attachments,
		Validation:      outcome,
		SensitiveFields: sensitiveFields,
		ReviewNote:      submission.EmptyReviewNote(),
	}}
}

func (store SubmissionStore) FindByReceiptToken(ctx context.Context, hash submission.ReceiptTokenHash) submission.FindReceiptStoreResult {
	rows, err := store.pool.Query(ctx, submissionSelectSQL()+`
		join submission_receipt_tokens on submission_receipt_tokens.submission_id = submissions.id
		where submission_receipt_tokens.token_hash = $1
	`, hash.String())
	if err != nil {
		return submission.ReceiptMissing{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "find submission receipt failed")}
	}
	defer rows.Close()

	valuesResult := scanSubmissionRows(rows)
	values, matched := valuesResult.(submissionRowsAccepted)
	if !matched {
		rejected := valuesResult.(submissionRowsRejected)
		return submission.ReceiptMissing{Reason: rejected.reason}
	}
	if len(values.values) != 1 {
		return submission.ReceiptMissing{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "submission receipt was not found")}
	}
	return submission.ReceiptFound{Value: values.values[0]}
}

func (store SubmissionStore) FindSubmission(ctx context.Context, submissionID core.SubmissionID) submission.FindSubmissionStoreResult {
	rows, err := store.pool.Query(ctx, submissionSelectSQL()+`
		where submissions.id = $1
	`, submissionID.String())
	if err != nil {
		return submission.FindSubmissionStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "find submission failed")}
	}
	defer rows.Close()

	valuesResult := scanSubmissionRows(rows)
	values, matched := valuesResult.(submissionRowsAccepted)
	if !matched {
		return submission.FindSubmissionStoreRejected{Reason: valuesResult.(submissionRowsRejected).reason}
	}
	if len(values.values) != 1 {
		return submission.FindSubmissionStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "submission was not found")}
	}
	return submission.FindSubmissionStoreAccepted{Value: values.values[0]}
}

func (store SubmissionStore) ListForSubmitter(ctx context.Context, submitterID core.UserID, page core.Page) submission.ListSubmissionsStoreResult {
	rows, err := store.pool.Query(ctx, submissionSelectSQL()+`
		where submissions.user_id = $1
		order by submissions.created_at
		limit $2 offset $3
	`, submitterID.String(), page.Limit(), page.Offset())
	if err != nil {
		return submission.ListSubmissionsStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list submitter submissions failed")}
	}
	defer rows.Close()

	valuesResult := scanSubmissionRows(rows)
	values, matched := valuesResult.(submissionRowsAccepted)
	if !matched {
		return submission.ListSubmissionsStoreRejected{Reason: valuesResult.(submissionRowsRejected).reason}
	}
	return submission.ListSubmissionsStoreAccepted{Values: values.values}
}

func (store SubmissionStore) ListForTask(ctx context.Context, taskID core.TaskID, page core.Page) submission.ListSubmissionsStoreResult {
	rows, err := store.pool.Query(ctx, submissionSelectSQL()+`
		where submissions.task_id = $1
		order by submissions.created_at
		limit $2 offset $3
	`, taskID.String(), page.Limit(), page.Offset())
	if err != nil {
		return submission.ListSubmissionsStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list submissions failed")}
	}
	defer rows.Close()

	valuesResult := scanSubmissionRows(rows)
	values, matched := valuesResult.(submissionRowsAccepted)
	if !matched {
		rejected := valuesResult.(submissionRowsRejected)
		return submission.ListSubmissionsStoreRejected{Reason: rejected.reason}
	}
	return submission.ListSubmissionsStoreAccepted{Values: values.values}
}

type insertRowsResult interface {
	insertRowsResult()
}

type insertRowsAccepted struct{}

type insertRowsRejected struct {
	reason core.DomainError
}

func (insertRowsAccepted) insertRowsResult() {}

func (insertRowsRejected) insertRowsResult() {}

func insertValidationErrors(ctx context.Context, tx pgx.Tx, submissionID core.SubmissionID, outcome submission.ValidationOutcome) insertRowsResult {
	failed, matched := outcome.(submission.ValidationFailed)
	if !matched {
		return insertRowsAccepted{}
	}
	for errorIndex := range failed.Errors {
		validationError := failed.Errors[errorIndex]
		_, err := tx.Exec(ctx, `
			insert into submission_validation_errors (submission_id, error_index, path, message)
			values ($1, $2, $3, $4)
		`, submissionID.String(), errorIndex, validationError.Path, validationError.Message)
		if err != nil {
			return insertRowsRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert submission validation error failed")}
		}
	}
	return insertRowsAccepted{}
}

func insertSensitiveFields(ctx context.Context, tx pgx.Tx, submissionID core.SubmissionID, fields []submission.SensitiveField) insertRowsResult {
	for fieldIndex := range fields {
		field := fields[fieldIndex]
		_, err := tx.Exec(ctx, `
			insert into submission_sensitive_fields (submission_id, field_index, path, category, retention, redaction)
			values ($1, $2, $3, $4, $5, $6)
		`, submissionID.String(), fieldIndex, field.Path, field.Category, field.Retention, field.Redaction)
		if err != nil {
			return insertRowsRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert submission sensitive field failed")}
		}
	}
	return insertRowsAccepted{}
}

func insertSubmissionAttachments(ctx context.Context, tx pgx.Tx, submissionID core.SubmissionID, attachments []attachment.Attachment) insertAttachmentsResult {
	for index := range attachments {
		value := attachments[index]
		_, err := tx.Exec(ctx, `
			insert into submission_attachments (submission_id, attachment_index, filename, content_type, content)
			values ($1, $2, $3, $4, $5)
		`, submissionID.String(), index, value.Name.String(), value.ContentType.String(), value.Content.Bytes())
		if err != nil {
			return insertAttachmentsRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert submission attachment failed")}
		}
	}
	return insertAttachmentsAccepted{}
}

func submissionSelectSQL() string {
	return `
		select submissions.id::text, submissions.task_id::text, submissions.user_id::text, submissions.state, submissions.response_json::text,
			submissions.review_note,
			coalesce((
				select jsonb_agg(
					jsonb_build_object('path', submission_validation_errors.path, 'message', submission_validation_errors.message)
					order by submission_validation_errors.error_index
				)
				from submission_validation_errors
				where submission_validation_errors.submission_id = submissions.id
			), '[]'::jsonb)::text,
			coalesce((
				select jsonb_agg(
					jsonb_build_object(
						'path', submission_sensitive_fields.path,
						'category', submission_sensitive_fields.category,
						'retention', submission_sensitive_fields.retention,
						'redaction', submission_sensitive_fields.redaction,
						'state', submission_sensitive_fields.state,
						'redacted_at', coalesce(to_char(submission_sensitive_fields.redacted_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
					)
					order by submission_sensitive_fields.field_index
				)
				from submission_sensitive_fields
				where submission_sensitive_fields.submission_id = submissions.id
			), '[]'::jsonb)::text,
			coalesce((
				select jsonb_agg(
					jsonb_build_object(
						'name', submission_attachments.filename,
						'content_type', submission_attachments.content_type,
						'content', encode(submission_attachments.content, 'base64')
					)
					order by submission_attachments.attachment_index
				)
				from submission_attachments
				where submission_attachments.submission_id = submissions.id
			), '[]'::jsonb)::text
		from submissions
	`
}

type submissionRowsResult interface {
	submissionRowsResult()
}

type submissionRowsAccepted struct {
	values []submission.Submission
}

type submissionRowsRejected struct {
	reason core.DomainError
}

func (submissionRowsAccepted) submissionRowsResult() {}

func (submissionRowsRejected) submissionRowsResult() {}

func scanSubmissionRows(rows pgx.Rows) submissionRowsResult {
	values := make([]submission.Submission, 0)
	for rows.Next() {
		parsed := scanSubmissionRow(rows)
		accepted, matched := parsed.(submissionRowAccepted)
		if !matched {
			rejected := parsed.(submissionRowRejected)
			return submissionRowsRejected{reason: rejected.reason}
		}
		values = append(values, accepted.value)
	}
	if err := rows.Err(); err != nil {
		return submissionRowsRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "read submissions failed")}
	}
	return submissionRowsAccepted{values: values}
}

type submissionRowResult interface {
	submissionRowResult()
}

type submissionRowAccepted struct {
	value submission.Submission
}

type submissionRowRejected struct {
	reason core.DomainError
}

func (submissionRowAccepted) submissionRowResult() {}

func (submissionRowRejected) submissionRowResult() {}

func scanSubmissionRow(rows pgx.Rows) submissionRowResult {
	var rawSubmissionID string
	var rawTaskID string
	var rawUserID string
	var rawState string
	var rawResponse string
	var rawReviewNote string
	var rawValidationErrors string
	var rawSensitiveFields string
	var rawAttachments string
	if err := rows.Scan(&rawSubmissionID, &rawTaskID, &rawUserID, &rawState, &rawResponse, &rawReviewNote, &rawValidationErrors, &rawSensitiveFields, &rawAttachments); err != nil {
		return submissionRowRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan submission failed")}
	}
	return parseSubmissionRow(rawSubmissionID, rawTaskID, rawUserID, rawState, rawResponse, rawReviewNote, rawValidationErrors, rawSensitiveFields, rawAttachments)
}

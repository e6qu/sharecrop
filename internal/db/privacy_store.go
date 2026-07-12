package db

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/core/id"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PrivacyStore struct {
	db Beginner
}

func NewPrivacyStore(pool *pgxpool.Pool) PrivacyStore {
	return NewPrivacyStoreFromHandle(NewPGX(pool))
}

func NewPrivacyStoreFromHandle(handle Beginner) PrivacyStore {
	return PrivacyStore{db: handle}
}

func (store PrivacyStore) Create(ctx context.Context, requester core.UserID, kind string) httpserver.PrivacyMutationResult {
	requestIDResult := id.New()
	requestID, requestIDMatched := requestIDResult.(id.IDCreated)
	if !requestIDMatched {
		return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidID, requestIDResult.(id.IDRejected).Description)}
	}
	record := httpserver.PrivacyRequestRecord{ID: requestID.Value.String(), RequestedBy: requester, Kind: kind, State: "queued", CreatedAt: time.Now().UTC()}
	_, err := store.db.Exec(ctx, `
		insert into privacy_requests (id, requested_by_user_id, kind, state, export_json, resolution_note, created_at)
		values ($1, $2, $3, $4, '', '', $5)
	`, record.ID, requester.String(), kind, record.State, record.CreatedAt)
	if err != nil {
		return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "create privacy request failed")}
	}
	return httpserver.PrivacyRequestSaved{Value: record}
}

func (store PrivacyStore) ListForRequester(ctx context.Context, requester core.UserID, page core.Page) httpserver.PrivacyListResult {
	rows, err := store.db.Query(ctx, `
		select id::text, requested_by_user_id::text, kind, state, export_json, resolution_note, created_at, coalesce(resolved_at, '0001-01-01T00:00:00Z'::timestamptz), redacted_field_count
		from privacy_requests
		where requested_by_user_id = $1
		order by created_at desc, id desc
		limit $2 offset $3
	`, requester.String(), page.Limit(), page.Offset())
	if err != nil {
		return httpserver.PrivacyRequestListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list privacy requests failed")}
	}
	return scanPrivacyRequests(rows)
}

func (store PrivacyStore) ListAll(ctx context.Context, page core.Page) httpserver.PrivacyListResult {
	rows, err := store.db.Query(ctx, `
		select id::text, requested_by_user_id::text, kind, state, export_json, resolution_note, created_at, coalesce(resolved_at, '0001-01-01T00:00:00Z'::timestamptz), redacted_field_count
		from privacy_requests
		order by created_at desc, id desc
		limit $1 offset $2
	`, page.Limit(), page.Offset())
	if err != nil {
		return httpserver.PrivacyRequestListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list admin privacy requests failed")}
	}
	return scanPrivacyRequests(rows)
}

func (store PrivacyStore) Resolve(ctx context.Context, requestID string, note string) httpserver.PrivacyMutationResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin privacy request resolution failed")}
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		select id::text, requested_by_user_id::text, kind, state, export_json, resolution_note, created_at, coalesce(resolved_at, '0001-01-01T00:00:00Z'::timestamptz), redacted_field_count
		from privacy_requests
		where id = $1
		for update
	`, requestID)
	var rawID string
	var rawRequesterID string
	var kind string
	var state string
	var exportJSON string
	var existingNote string
	var createdAt time.Time
	var resolvedAt time.Time
	var redactedFieldCount int
	if err := row.Scan(&rawID, &rawRequesterID, &kind, &state, &exportJSON, &existingNote, &createdAt, &resolvedAt, &redactedFieldCount); err != nil {
		if err == ErrNoRows {
			return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "privacy request was not found")}
		}
		return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan privacy request failed")}
	}
	recordResult := parsePrivacyRequestRecord(rawID, rawRequesterID, kind, state, exportJSON, existingNote, createdAt, resolvedAt, redactedFieldCount)
	record, matched := recordResult.(privacyRequestRowAccepted)
	if !matched {
		return httpserver.PrivacyRequestMutationRejected{Reason: recordResult.(privacyRequestRowRejected).reason}
	}
	if record.value.State != "queued" {
		return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "privacy request is already resolved")}
	}

	resolved := record.value
	resolved.State = "resolved"
	resolved.ResolutionNote = strings.TrimSpace(note)
	resolved.ResolvedAt = time.Now().UTC()
	if resolved.Kind == "data_export" {
		exportJSON, err := store.exportPrivacyData(ctx, tx, resolved.RequestedBy)
		if err != nil {
			return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
		}
		resolved.ExportJSON = exportJSON
	}
	if resolved.Kind == "sensitive_field_deletion" {
		rows, err := tx.Query(ctx, `
			update submission_sensitive_fields
			set state = 'redacted', redacted_at = now()
			where state = 'active'
			and retention = 'delete_on_request'
			and submission_id in (select id from submissions where user_id = $1)
			returning submission_id::text, path
		`, resolved.RequestedBy.String())
		if err != nil {
			return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "redact sensitive fields failed")}
		}
		redactedRows, readErr := scanRedactedSensitiveFieldRows(rows)
		if readErr != nil {
			return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, readErr.Error())}
		}
		for rowIndex := range redactedRows {
			eventIDResult := id.New()
			eventID, eventIDMatched := eventIDResult.(id.IDCreated)
			if !eventIDMatched {
				return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidID, eventIDResult.(id.IDRejected).Description)}
			}
			if _, err := tx.Exec(ctx, `
				insert into submission_sensitive_field_events (id, submission_id, actor_user_id, action, field_path)
				values ($1, $2, $3, 'sensitive_field_redacted', $4)
			`, eventID.Value.String(), redactedRows[rowIndex].submissionID, resolved.RequestedBy.String(), redactedRows[rowIndex].path); err != nil {
				return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record sensitive field redaction event failed")}
			}
		}
		resolved.RedactedFieldCount = len(redactedRows)
	}
	if _, err := tx.Exec(ctx, `
		update privacy_requests
		set state = 'resolved', export_json = $2, resolution_note = $3, resolved_at = $4, redacted_field_count = $5
		where id = $1
	`, resolved.ID, resolved.ExportJSON, resolved.ResolutionNote, resolved.ResolvedAt, resolved.RedactedFieldCount); err != nil {
		return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "resolve privacy request failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit privacy request resolution failed")}
	}
	return httpserver.PrivacyRequestSaved{Value: resolved}
}

func (store PrivacyStore) RecordSensitiveFieldAccess(ctx context.Context, actor core.UserID, value submission.Submission) httpserver.PrivacyMutationResult {
	if len(value.SensitiveFields) == 0 {
		return httpserver.PrivacyRequestSaved{Value: httpserver.PrivacyRequestRecord{CreatedAt: time.Now().UTC()}}
	}
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin sensitive-field access recording failed")}
	}
	defer tx.Rollback(ctx)
	for fieldIndex := range value.SensitiveFields {
		field := value.SensitiveFields[fieldIndex]
		eventIDResult := id.New()
		eventID, eventIDMatched := eventIDResult.(id.IDCreated)
		if !eventIDMatched {
			return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidID, eventIDResult.(id.IDRejected).Description)}
		}
		if _, err := tx.Exec(ctx, `
			insert into submission_sensitive_field_events (id, submission_id, actor_user_id, action, field_path)
			values ($1, $2, $3, 'sensitive_field_accessed', $4)
		`, eventID.Value.String(), value.ID.String(), actor.String(), field.Path); err != nil {
			return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record sensitive-field access event failed")}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit sensitive-field access recording failed")}
	}
	return httpserver.PrivacyRequestSaved{Value: httpserver.PrivacyRequestRecord{CreatedAt: time.Now().UTC()}}
}

func (store PrivacyStore) RunRetention(ctx context.Context, actor core.UserID) httpserver.PrivacyRetentionResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return httpserver.PrivacyRetentionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin privacy retention run failed")}
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `
		update submission_sensitive_fields
		set state = 'redacted', redacted_at = now()
		where state = 'active'
		and retention = 'delete_on_request'
		returning submission_id::text, path
	`)
	if err != nil {
		return httpserver.PrivacyRetentionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "run sensitive-field retention failed")}
	}
	redactedRows, readErr := scanRedactedSensitiveFieldRows(rows)
	if readErr != nil {
		return httpserver.PrivacyRetentionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, readErr.Error())}
	}
	for rowIndex := range redactedRows {
		eventIDResult := id.New()
		eventID, eventIDMatched := eventIDResult.(id.IDCreated)
		if !eventIDMatched {
			return httpserver.PrivacyRetentionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidID, eventIDResult.(id.IDRejected).Description)}
		}
		if _, err := tx.Exec(ctx, `
			insert into submission_sensitive_field_events (id, submission_id, actor_user_id, action, field_path)
			values ($1, $2, $3, 'sensitive_field_redacted', $4)
		`, eventID.Value.String(), redactedRows[rowIndex].submissionID, actor.String(), redactedRows[rowIndex].path); err != nil {
			return httpserver.PrivacyRetentionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record retention redaction event failed")}
		}
	}
	runIDResult := id.New()
	runID, runIDMatched := runIDResult.(id.IDCreated)
	if !runIDMatched {
		return httpserver.PrivacyRetentionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidID, runIDResult.(id.IDRejected).Description)}
	}
	if _, err := tx.Exec(ctx, `
		insert into privacy_retention_runs (id, actor_user_id, redacted_field_count)
		values ($1, $2, $3)
	`, runID.Value.String(), actor.String(), len(redactedRows)); err != nil {
		return httpserver.PrivacyRetentionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record privacy retention run failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return httpserver.PrivacyRetentionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit privacy retention run failed")}
	}
	return httpserver.PrivacyRetentionRun{RedactedFieldCount: len(redactedRows)}
}

type privacyExportDocument struct {
	UserID          string                      `json:"user_id"`
	Email           string                      `json:"email"`
	GeneratedAt     string                      `json:"generated_at"`
	Submissions     []privacyExportSubmission   `json:"submissions"`
	SensitiveFields []privacyExportSensitive    `json:"sensitive_fields"`
	Notifications   []privacyExportNotification `json:"notifications"`
	LedgerEntries   []privacyExportLedgerEntry  `json:"ledger_entries"`
	PrivacyRequests []privacyExportRequest      `json:"privacy_requests"`
}

type privacyExportSubmission struct {
	ID           string `json:"id"`
	TaskID       string `json:"task_id"`
	State        string `json:"state"`
	ResponseJSON string `json:"response_json"`
	CreatedAt    string `json:"created_at"`
}

type privacyExportSensitive struct {
	SubmissionID string `json:"submission_id"`
	Path         string `json:"path"`
	Category     string `json:"category"`
	Retention    string `json:"retention"`
	Redaction    string `json:"redaction"`
	State        string `json:"state"`
	RedactedAt   string `json:"redacted_at"`
}

type privacyExportNotification struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	SubjectKind  string `json:"subject_kind"`
	SubjectID    string `json:"subject_id"`
	State        string `json:"state"`
	MetadataJSON string `json:"metadata_json"`
	CreatedAt    string `json:"created_at"`
}

type privacyExportLedgerEntry struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Amount    int64  `json:"amount"`
	TaskID    string `json:"task_id"`
	CreatedAt string `json:"created_at"`
}

type privacyExportRequest struct {
	ID                 string `json:"id"`
	Kind               string `json:"kind"`
	State              string `json:"state"`
	CreatedAt          string `json:"created_at"`
	ResolvedAt         string `json:"resolved_at"`
	RedactedFieldCount int    `json:"redacted_field_count"`
}

func (store PrivacyStore) exportPrivacyData(ctx context.Context, tx Tx, requester core.UserID) (string, error) {
	document := privacyExportDocument{UserID: requester.String(), GeneratedAt: time.Now().UTC().Format(time.RFC3339)}
	if err := tx.QueryRow(ctx, `
		select email
		from users
		where id = $1
	`, requester.String()).Scan(&document.Email); err != nil {
		if err == ErrNoRows {
			return "", errors.New("privacy export user was not found")
		}
		return "", errors.New("privacy export user query failed")
	}
	submissions, err := scanPrivacyExportSubmissions(tx.Query(ctx, `
		select id::text, task_id::text, state, response_json::text, created_at
		from submissions
		where user_id = $1
		order by created_at desc, id desc
	`, requester.String()))
	if err != nil {
		return "", err
	}
	document.Submissions = submissions
	sensitiveFields, err := scanPrivacyExportSensitiveFields(tx.Query(ctx, `
		select submission_id::text, path, category, retention, redaction, state, coalesce(to_char(redacted_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
		from submission_sensitive_fields
		where submission_id in (select id from submissions where user_id = $1)
		order by submission_id, field_index
	`, requester.String()))
	if err != nil {
		return "", err
	}
	document.SensitiveFields = sensitiveFields
	notifications, err := scanPrivacyExportNotifications(tx.Query(ctx, `
		select id::text, kind, subject_kind, subject_id, state, metadata_json::text, created_at
		from notifications
		where recipient_user_id = $1
		order by created_at desc, id desc
	`, requester.String()))
	if err != nil {
		return "", err
	}
	document.Notifications = notifications
	ledgerEntries, err := scanPrivacyExportLedgerEntries(tx.Query(ctx, `
		select ledger_entries.id::text, ledger_entries.kind, ledger_entries.amount, coalesce(ledger_entries.task_id::text, ''), ledger_entries.created_at
		from ledger_entries
		join credit_accounts on credit_accounts.id = ledger_entries.account_id
		where credit_accounts.owner_kind = 'user' and credit_accounts.user_id = $1
		order by ledger_entries.created_at desc, ledger_entries.id desc
	`, requester.String()))
	if err != nil {
		return "", err
	}
	document.LedgerEntries = ledgerEntries
	requests, err := scanPrivacyExportRequests(tx.Query(ctx, `
		select id::text, kind, state, created_at, coalesce(resolved_at, '0001-01-01T00:00:00Z'::timestamptz), redacted_field_count
		from privacy_requests
		where requested_by_user_id = $1
		order by created_at desc, id desc
	`, requester.String()))
	if err != nil {
		return "", err
	}
	document.PrivacyRequests = requests
	encoded, err := json.Marshal(document)
	if err != nil {
		return "", errors.New("privacy export encoding failed")
	}
	return string(encoded), nil
}

type privacyRequestRowResult interface {
	privacyRequestRowResult()
}

type privacyRequestRowAccepted struct {
	value httpserver.PrivacyRequestRecord
}

type privacyRequestRowRejected struct {
	reason core.DomainError
}

func (privacyRequestRowAccepted) privacyRequestRowResult() {}

func (privacyRequestRowRejected) privacyRequestRowResult() {}

func scanPrivacyRequests(rows Rows) httpserver.PrivacyListResult {
	defer rows.Close()
	requests := make([]httpserver.PrivacyRequestRecord, 0)
	for rows.Next() {
		var rawID string
		var rawRequesterID string
		var kind string
		var state string
		var exportJSON string
		var note string
		var createdAt time.Time
		var resolvedAt time.Time
		var redactedFieldCount int
		if err := rows.Scan(&rawID, &rawRequesterID, &kind, &state, &exportJSON, &note, &createdAt, &resolvedAt, &redactedFieldCount); err != nil {
			return httpserver.PrivacyRequestListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan privacy request failed")}
		}
		result := parsePrivacyRequestRecord(rawID, rawRequesterID, kind, state, exportJSON, note, createdAt, resolvedAt, redactedFieldCount)
		accepted, matched := result.(privacyRequestRowAccepted)
		if !matched {
			return httpserver.PrivacyRequestListRejected{Reason: result.(privacyRequestRowRejected).reason}
		}
		requests = append(requests, accepted.value)
	}
	if err := rows.Err(); err != nil {
		return httpserver.PrivacyRequestListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read privacy requests failed")}
	}
	return httpserver.PrivacyRequestsListed{Values: requests}
}

func parsePrivacyRequestRecord(rawID string, rawRequesterID string, kind string, state string, exportJSON string, note string, createdAt time.Time, resolvedAt time.Time, redactedFieldCount int) privacyRequestRowResult {
	requesterResult := core.ParseUserID(rawRequesterID)
	requester, matched := requesterResult.(core.UserIDCreated)
	if !matched {
		return privacyRequestRowRejected{reason: requesterResult.(core.UserIDRejected).Reason}
	}
	return privacyRequestRowAccepted{value: httpserver.PrivacyRequestRecord{
		ID:                 rawID,
		RequestedBy:        requester.Value,
		Kind:               kind,
		State:              state,
		ExportJSON:         exportJSON,
		ResolutionNote:     note,
		CreatedAt:          createdAt,
		ResolvedAt:         resolvedAt,
		RedactedFieldCount: redactedFieldCount,
	}}
}

type redactedSensitiveFieldRow struct {
	submissionID string
	path         string
}

func scanRedactedSensitiveFieldRows(rows Rows) ([]redactedSensitiveFieldRow, error) {
	defer rows.Close()
	values := make([]redactedSensitiveFieldRow, 0)
	for rows.Next() {
		var value redactedSensitiveFieldRow
		if err := rows.Scan(&value.submissionID, &value.path); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func scanPrivacyExportSubmissions(rows Rows, queryErr error) ([]privacyExportSubmission, error) {
	if queryErr != nil {
		return nil, errors.New("privacy export submissions query failed: " + queryErr.Error())
	}
	defer rows.Close()
	values := make([]privacyExportSubmission, 0)
	for rows.Next() {
		var value privacyExportSubmission
		var createdAt time.Time
		if err := rows.Scan(&value.ID, &value.TaskID, &value.State, &value.ResponseJSON, &createdAt); err != nil {
			return nil, errors.New("privacy export submissions scan failed")
		}
		value.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.New("privacy export submissions read failed")
	}
	return values, nil
}

func scanPrivacyExportSensitiveFields(rows Rows, queryErr error) ([]privacyExportSensitive, error) {
	if queryErr != nil {
		return nil, errors.New("privacy export sensitive fields query failed: " + queryErr.Error())
	}
	defer rows.Close()
	values := make([]privacyExportSensitive, 0)
	for rows.Next() {
		var value privacyExportSensitive
		if err := rows.Scan(&value.SubmissionID, &value.Path, &value.Category, &value.Retention, &value.Redaction, &value.State, &value.RedactedAt); err != nil {
			return nil, errors.New("privacy export sensitive fields scan failed")
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.New("privacy export sensitive fields read failed")
	}
	return values, nil
}

func scanPrivacyExportNotifications(rows Rows, queryErr error) ([]privacyExportNotification, error) {
	if queryErr != nil {
		return nil, errors.New("privacy export notifications query failed: " + queryErr.Error())
	}
	defer rows.Close()
	values := make([]privacyExportNotification, 0)
	for rows.Next() {
		var value privacyExportNotification
		var createdAt time.Time
		if err := rows.Scan(&value.ID, &value.Kind, &value.SubjectKind, &value.SubjectID, &value.State, &value.MetadataJSON, &createdAt); err != nil {
			return nil, errors.New("privacy export notifications scan failed")
		}
		value.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.New("privacy export notifications read failed")
	}
	return values, nil
}

func scanPrivacyExportLedgerEntries(rows Rows, queryErr error) ([]privacyExportLedgerEntry, error) {
	if queryErr != nil {
		return nil, errors.New("privacy export ledger query failed: " + queryErr.Error())
	}
	defer rows.Close()
	values := make([]privacyExportLedgerEntry, 0)
	for rows.Next() {
		var value privacyExportLedgerEntry
		var createdAt time.Time
		if err := rows.Scan(&value.ID, &value.Kind, &value.Amount, &value.TaskID, &createdAt); err != nil {
			return nil, errors.New("privacy export ledger scan failed")
		}
		value.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.New("privacy export ledger read failed")
	}
	return values, nil
}

func scanPrivacyExportRequests(rows Rows, queryErr error) ([]privacyExportRequest, error) {
	if queryErr != nil {
		return nil, errors.New("privacy export requests query failed: " + queryErr.Error())
	}
	defer rows.Close()
	values := make([]privacyExportRequest, 0)
	for rows.Next() {
		var value privacyExportRequest
		var createdAt time.Time
		var resolvedAt time.Time
		if err := rows.Scan(&value.ID, &value.Kind, &value.State, &createdAt, &resolvedAt, &value.RedactedFieldCount); err != nil {
			return nil, errors.New("privacy export requests scan failed")
		}
		value.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		if !resolvedAt.IsZero() {
			value.ResolvedAt = resolvedAt.UTC().Format(time.RFC3339)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.New("privacy export requests read failed")
	}
	return values, nil
}

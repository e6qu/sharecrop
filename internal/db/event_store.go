package db

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventStore persists the append-only domain event stream (the
// in-transaction outbox) and serves the per-recipient feed. The lifecycle
// runner's dispatch sweep reads stale recorded rows through the host-only
// struct methods below, never through the bridged event.Store interface.
type EventStore struct {
	db Beginner
}

func NewEventStore(pool *pgxpool.Pool) EventStore {
	return NewEventStoreFromHandle(NewPGX(pool))
}

func NewEventStoreFromHandle(handle Beginner) EventStore {
	return EventStore{db: handle}
}

// eventColumns is qualified with its table so queries that join
// domain_events against tables with overlapping column names (the webhook
// claim join) stay unambiguous.
const eventColumns = `domain_events.seq, domain_events.id::text, domain_events.kind,
	domain_events.actor_kind, domain_events.actor_user_id::text,
	domain_events.task_id::text, domain_events.submission_id::text, domain_events.reservation_id::text,
	domain_events.series_id::text, domain_events.organization_id::text, domain_events.collectible_id::text,
	domain_events.metadata_json::text, domain_events.occurred_at`

func (store EventStore) Append(ctx context.Context, value event.Event, recipients event.Recipients) event.AppendStoreResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return event.AppendStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin event append failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	sequence, err := insertDomainEventInTx(ctx, tx, value, recipients)
	if err != nil {
		return event.AppendStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "append domain event failed")}
	}

	if err := tx.Commit(ctx); err != nil {
		return event.AppendStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit event append failed")}
	}
	return event.AppendStoreAccepted{Value: event.WithoutEnrichment(event.StoredEvent{Event: value, Cursor: event.CursorFromSequence(sequence)})}
}

// insertDomainEventInTx inserts one domain event with its recipient set in
// the recorded dispatch state, inside the caller's transaction.
//
// Before the first insert it locks the single domain_event_fence row, held
// until the caller's transaction commits. This serializes {seq allocation ->
// commit} across every event-recording transaction, so events become visible
// in strictly increasing seq order and the "seq > after" cursor reads (feed
// pages, long-poll head checks, SSE resume) can never skip an event that a
// slower transaction commits later with a lower seq. A per-row
// xid-versus-snapshot-xmin bound was rejected: xid order is not seq order (a
// transaction acquires its xid at its first write, often several statements
// before it allocates the event seq), so a committed row can pass the xmin
// cap while an in-flight transaction still holds a lower seq. Event
// recording is the final step of each mutation transaction, so the fence is
// always the last lock acquired (no deadlock cycles), and the serialized
// window is only the tail of the transaction. On SQLite the FOR UPDATE is
// translated away; its single-writer model already serializes commits.
func insertDomainEventInTx(ctx context.Context, tx Tx, value event.Event, recipients event.Recipients) (int64, error) {
	if _, err := tx.Exec(ctx, "select id from domain_event_fence where id = 1 for update"); err != nil {
		return 0, err
	}

	actorKind := "user"
	if _, matched := value.Actor.(event.ActorSystem); matched {
		actorKind = "system"
	}

	var sequence int64
	err := tx.QueryRow(ctx, `
		insert into domain_events (id, kind, actor_kind, actor_user_id, task_id, submission_id, reservation_id, series_id, organization_id, collectible_id, metadata_json, occurred_at, dispatch_state)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12, $13)
		returning seq
	`,
		value.ID.String(),
		value.Kind.String(),
		actorKind,
		event.ActorUserID(value.Actor).String(),
		taskRefValue(value.Subject.Task),
		submissionRefValue(value.Subject.Submission),
		reservationRefValue(value.Subject.Reservation),
		seriesRefValue(value.Subject.Series),
		organizationRefValue(value.Subject.Organization),
		collectibleRefValue(value.Subject.Collectible),
		value.Metadata.JSON,
		value.OccurredAt,
		event.DispatchStateRecorded.String(),
	).Scan(&sequence)
	if err != nil {
		return 0, err
	}

	for _, recipient := range recipients.Users {
		if _, err := tx.Exec(ctx, `
			insert into domain_event_recipients (event_seq, user_id)
			values ($1, $2)
		`, sequence, recipient.String()); err != nil {
			return 0, err
		}
	}
	return sequence, nil
}

// recordEventDraftInTx records one event draft inside a mutation's
// transaction: the in-transaction outbox write every emitting store mutation
// shares. The occurrence instant is stamped here (the row records when the
// mutation happened, which is this transaction).
func recordEventDraftInTx(ctx context.Context, tx Tx, draft event.Draft) error {
	_, err := insertDomainEventInTx(ctx, tx, draft.Event(time.Now()), draft.Recipients)
	return err
}

// Dispatch performs the store-side dispatch effects for one recorded event:
// it expands the event into pending webhook deliveries and flips the row to
// dispatched, in one transaction. An event that is already dispatched (or
// unknown, e.g. an in-memory-runtime id) completes as a no-op, and the
// delivery expansion is idempotent per (subscription, event), so the inline
// dispatch and the recovery sweep can race safely.
func (store EventStore) Dispatch(ctx context.Context, id core.DomainEventID) event.DispatchStoreResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return event.DispatchStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin event dispatch failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var sequence int64
	var rawState string
	scanErr := tx.QueryRow(ctx, `
		select seq, dispatch_state from domain_events where id = $1 for update
	`, id.String()).Scan(&sequence, &rawState)
	if errors.Is(scanErr, ErrNoRows) {
		return event.DispatchStoreCompleted{}
	}
	if scanErr != nil {
		return event.DispatchStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "lock event for dispatch failed")}
	}
	stateResult, stateMatched := event.ParseDispatchState(rawState).(event.DispatchStateParsed)
	if !stateMatched {
		return event.DispatchStoreRejected{Reason: event.ParseDispatchState(rawState).(event.DispatchStateRejected).Reason}
	}
	if stateResult.Value == event.DispatchStateDispatched || stateResult.Value == event.DispatchStateFailed {
		// Dispatched is done; dispatch_failed is terminal (the sweep retired
		// the row after the attempt cap) — both complete as a no-op.
		return event.DispatchStoreCompleted{}
	}

	if _, expandErr := expandWebhookDeliveries(ctx, tx, sequence); expandErr != nil {
		return event.DispatchStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "expand webhook deliveries failed")}
	}

	if _, err := tx.Exec(ctx, `
		update domain_events
		set dispatch_state = $2, dispatched_at = now()
		where seq = $1 and dispatch_state = $3
	`, sequence, event.DispatchStateDispatched.String(), event.DispatchStateRecorded.String()); err != nil {
		return event.DispatchStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "mark event dispatched failed")}
	}

	if err := tx.Commit(ctx); err != nil {
		return event.DispatchStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit event dispatch failed")}
	}
	return event.DispatchStoreCompleted{}
}

// ---- host-only dispatch sweep reads (the lifecycle runner) ----

// MaxEventDispatchAttempts caps how many recovery-sweep passes may claim one
// recorded event. A row claimed this many times without reaching dispatched
// is moved to the terminal dispatch_failed state on the next pass, so a
// poison row cannot occupy the sweep forever. Inline dispatches are not
// counted; only the sweep increments the counter.
const MaxEventDispatchAttempts = 20

type ListRecordedResult interface {
	listRecordedResult()
}

type RecordedEventsListed struct {
	Values []event.Draft
	// SkippedMalformed counts claimed rows whose stored columns no longer
	// parse into a draft. They are skipped (never blocking later rows) and
	// keep accumulating attempts until the cap retires them.
	SkippedMalformed int
}

type ListRecordedRejected struct {
	Reason core.DomainError
}

func (RecordedEventsListed) listRecordedResult() {}

func (ListRecordedRejected) listRecordedResult() {}

// ListRecordedBefore rebuilds the drafts of events still in the recorded
// dispatch state whose occurrence instant is at or before cutoff — the stale
// rows whose inline dispatch was lost to a crash or a failed fan-out. The
// lifecycle runner's dispatch sweep feeds them back through
// Recorder.Dispatch. Each pass first retires rows whose attempt count
// reached MaxEventDispatchAttempts to dispatch_failed, then claims a batch
// and increments its attempt counters. Rows that fail to parse are skipped
// and reported, never aborting the batch. It is a struct-only method (not
// part of the bridged event.Store), so the WASI guest can never reach it.
func (store EventStore) ListRecordedBefore(ctx context.Context, cutoff time.Time, limit int) ListRecordedResult {
	if _, err := store.db.Exec(ctx, `
		update domain_events
		set dispatch_state = $1
		where dispatch_state = $2 and occurred_at <= $3 and dispatch_attempts >= $4
	`, event.DispatchStateFailed.String(), event.DispatchStateRecorded.String(), cutoff, MaxEventDispatchAttempts); err != nil {
		return ListRecordedRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "retire exhausted recorded events failed")}
	}

	rows, err := store.db.Query(ctx, `
		update domain_events
		set dispatch_attempts = dispatch_attempts + 1
		where seq in (
			select seq from domain_events
			where dispatch_state = $1 and occurred_at <= $2
			order by seq
			limit $3
		)
		returning seq, id::text, kind, actor_kind, actor_user_id::text,
			task_id::text, submission_id::text, reservation_id::text,
			series_id::text, organization_id::text, collectible_id::text,
			metadata_json::text, occurred_at
	`, event.DispatchStateRecorded.String(), cutoff, limit)
	if err != nil {
		return ListRecordedRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list recorded events failed")}
	}
	defer rows.Close()

	stored := make([]event.StoredEvent, 0)
	skipped := 0
	for rows.Next() {
		value, scanErr := scanBareStoredEvent(rows)
		if scanErr != nil {
			// A malformed row must not poison the batch: skip it and let the
			// attempt cap retire it.
			skipped++
			continue
		}
		stored = append(stored, value)
	}
	if rows.Err() != nil {
		return ListRecordedRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "iterate recorded events failed")}
	}
	// RETURNING carries no ordering guarantee; the sweep dispatches in seq
	// order.
	sort.Slice(stored, func(left int, right int) bool {
		return stored[left].Cursor.Sequence() < stored[right].Cursor.Sequence()
	})

	drafts := make([]event.Draft, 0, len(stored))
	for _, value := range stored {
		recipients, recipientsErr := store.eventRecipients(ctx, value.Cursor.Sequence())
		if recipientsErr != nil {
			skipped++
			continue
		}
		drafts = append(drafts, event.Draft{
			ID:         value.Event.ID,
			Kind:       value.Event.Kind,
			Actor:      value.Event.Actor,
			Subject:    value.Event.Subject,
			Metadata:   value.Event.Metadata,
			Recipients: recipients,
		})
	}
	return RecordedEventsListed{Values: drafts, SkippedMalformed: skipped}
}

// FailedEventRow is one event retired to the terminal dispatch_failed state,
// exposed so operators can inspect what the sweep gave up on.
type FailedEventRow struct {
	Sequence   int64
	EventID    string
	Kind       string
	Attempts   int64
	OccurredAt time.Time
}

type ListDispatchFailedResult interface {
	listDispatchFailedResult()
}

type DispatchFailedEventsListed struct {
	Values []FailedEventRow
}

type ListDispatchFailedRejected struct {
	Reason core.DomainError
}

func (DispatchFailedEventsListed) listDispatchFailedResult() {}

func (ListDispatchFailedRejected) listDispatchFailedResult() {}

// ListDispatchFailed lists events in the terminal dispatch_failed state,
// oldest first. Host-only (struct method), for operator inspection.
func (store EventStore) ListDispatchFailed(ctx context.Context, limit int) ListDispatchFailedResult {
	rows, err := store.db.Query(ctx, `
		select seq, id::text, kind, dispatch_attempts, occurred_at
		from domain_events
		where dispatch_state = $1
		order by seq
		limit $2
	`, event.DispatchStateFailed.String(), limit)
	if err != nil {
		return ListDispatchFailedRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list dispatch-failed events failed")}
	}
	defer rows.Close()

	values := make([]FailedEventRow, 0)
	for rows.Next() {
		var row FailedEventRow
		if err := rows.Scan(&row.Sequence, &row.EventID, &row.Kind, &row.Attempts, &row.OccurredAt); err != nil {
			return ListDispatchFailedRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan dispatch-failed event failed")}
		}
		values = append(values, row)
	}
	if rows.Err() != nil {
		return ListDispatchFailedRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "iterate dispatch-failed events failed")}
	}
	return DispatchFailedEventsListed{Values: values}
}

// eventRecipients reads the recipient set of one stored event.
func (store EventStore) eventRecipients(ctx context.Context, sequence int64) (event.Recipients, error) {
	rows, err := store.db.Query(ctx, `
		select user_id::text from domain_event_recipients where event_seq = $1
	`, sequence)
	if err != nil {
		return event.Recipients{}, err
	}
	defer rows.Close()
	users := make([]core.UserID, 0)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return event.Recipients{}, err
		}
		parsed, matched := core.ParseUserID(raw).(core.UserIDCreated)
		if !matched {
			return event.Recipients{}, ErrNoRows
		}
		users = append(users, parsed.Value)
	}
	if rows.Err() != nil {
		return event.Recipients{}, rows.Err()
	}
	return event.NewRecipients(users...), nil
}

// scanBareStoredEvent decodes one row holding exactly the eventColumns.
func scanBareStoredEvent(rows Rows) (event.StoredEvent, error) {
	var sequence int64
	var rawID, rawKind, actorKind, rawActorUser string
	var rawTask, rawSubmission, rawReservation, rawSeries, rawOrganization, rawCollectible *string
	var rawMetadata string
	var occurredAt time.Time
	if err := rows.Scan(&sequence, &rawID, &rawKind, &actorKind, &rawActorUser,
		&rawTask, &rawSubmission, &rawReservation, &rawSeries, &rawOrganization, &rawCollectible,
		&rawMetadata, &occurredAt); err != nil {
		return event.StoredEvent{}, err
	}
	return parseStoredEventColumns(sequence, rawID, rawKind, actorKind, rawActorUser,
		rawTask, rawSubmission, rawReservation, rawSeries, rawOrganization, rawCollectible,
		rawMetadata, occurredAt)
}

func (store EventStore) ListForRecipient(ctx context.Context, recipient core.UserID, filter event.CursorFilter, page core.Page) event.ListStoreResult {
	return store.listEnrichedFeedPage(ctx, `
		join domain_event_recipients on domain_event_recipients.event_seq = domain_events.seq`,
		"domain_event_recipients.user_id = $1", recipient.String(), filter, page)
}

// ListForOrganization serves the cursor feed for an organization: the events
// whose subject organization is the given one — the same visibility rule the
// webhook delivery expansion applies to organization-owned recipient-audience
// subscriptions (expandWebhookDeliveries).
func (store EventStore) ListForOrganization(ctx context.Context, organizationID core.OrganizationID, filter event.CursorFilter, page core.Page) event.ListStoreResult {
	return store.listEnrichedFeedPage(ctx, "", "domain_events.organization_id = $1", organizationID.String(), filter, page)
}

// listEnrichedFeedPage runs one cursor-feed page query: the enriched event
// columns with the given extra join and owner predicate ($1 is the owner
// value), events after the filter cursor, in sequence order.
func (store EventStore) listEnrichedFeedPage(ctx context.Context, ownerJoin string, ownerPredicate string, ownerValue string, filter event.CursorFilter, page core.Page) event.ListStoreResult {
	afterSequence := int64(0)
	if after, matched := filter.(event.After); matched {
		afterSequence = after.Cursor.Sequence()
	}
	rows, err := store.db.Query(ctx, `
		select `+eventColumns+`,
			`+displayNameSQL("actor_user")+`,
			coalesce(subject_task.title, '')
		from domain_events`+ownerJoin+`
		join users as actor_user on actor_user.id = domain_events.actor_user_id
		left join tasks as subject_task on subject_task.id = domain_events.task_id
		where `+ownerPredicate+` and domain_events.seq > $2
		order by domain_events.seq
		limit $3 offset $4
	`, ownerValue, afterSequence, page.Limit(), page.Offset())
	if err != nil {
		return event.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list events failed")}
	}
	defer rows.Close()

	values := make([]event.StoredEvent, 0)
	for rows.Next() {
		stored, scanErr := scanEnrichedFeedEvent(rows)
		if scanErr != nil {
			return event.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan event failed")}
		}
		values = append(values, stored)
	}
	if rows.Err() != nil {
		return event.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "iterate events failed")}
	}
	return event.ListStoreAccepted{Values: values}
}

// scanEnrichedFeedEvent decodes one feed row: the eventColumns plus the
// actor's display name and the referenced task's title (empty when the event
// references no task).
func scanEnrichedFeedEvent(rows Rows) (event.StoredEvent, error) {
	var sequence int64
	var rawID, rawKind, actorKind, rawActorUser string
	var rawTask, rawSubmission, rawReservation, rawSeries, rawOrganization, rawCollectible *string
	var rawMetadata string
	var occurredAt time.Time
	var rawActorName, rawTaskTitle string
	if err := rows.Scan(&sequence, &rawID, &rawKind, &actorKind, &rawActorUser,
		&rawTask, &rawSubmission, &rawReservation, &rawSeries, &rawOrganization, &rawCollectible,
		&rawMetadata, &occurredAt, &rawActorName, &rawTaskTitle); err != nil {
		return event.StoredEvent{}, err
	}
	stored, err := parseStoredEventColumns(sequence, rawID, rawKind, actorKind, rawActorUser,
		rawTask, rawSubmission, rawReservation, rawSeries, rawOrganization, rawCollectible,
		rawMetadata, occurredAt)
	if err != nil {
		return event.StoredEvent{}, err
	}
	nameResult, nameMatched := auth.NewDisplayName(rawActorName).(auth.DisplayNameAccepted)
	if !nameMatched {
		return event.StoredEvent{}, ErrNoRows
	}
	stored.ActorName = event.ActorNamed{DisplayName: nameResult.Value}
	if rawTaskTitle != "" {
		stored.TaskTitle = event.TaskTitled{Title: rawTaskTitle}
	}
	return stored, nil
}

// parseStoredEventColumns rebuilds a stored event from the raw eventColumns
// values. Shared by the feed scanner above and the webhook claim scanner,
// which joins the same columns into a wider row.
func parseStoredEventColumns(sequence int64, rawID string, rawKind string, actorKind string, rawActorUser string,
	rawTask *string, rawSubmission *string, rawReservation *string, rawSeries *string, rawOrganization *string, rawCollectible *string,
	rawMetadata string, occurredAt time.Time) (event.StoredEvent, error) {
	idResult, matched := core.ParseDomainEventID(rawID).(core.DomainEventIDCreated)
	if !matched {
		return event.StoredEvent{}, ErrNoRows
	}
	kindResult, kindMatched := event.ParseKind(rawKind).(event.KindParsed)
	if !kindMatched {
		return event.StoredEvent{}, ErrNoRows
	}
	actorResult, actorMatched := core.ParseUserID(rawActorUser).(core.UserIDCreated)
	if !actorMatched {
		return event.StoredEvent{}, ErrNoRows
	}
	var actor event.Actor = event.ActorUser{ID: actorResult.Value}
	if actorKind == "system" {
		actor = event.ActorSystem{}
	}

	subject := event.NoSubjectRefs()
	if rawTask != nil {
		parsed, ok := core.ParseTaskID(*rawTask).(core.TaskIDCreated)
		if !ok {
			return event.StoredEvent{}, ErrNoRows
		}
		subject.Task = event.TaskSubject{ID: parsed.Value}
	}
	if rawSubmission != nil {
		parsed, ok := core.ParseSubmissionID(*rawSubmission).(core.SubmissionIDCreated)
		if !ok {
			return event.StoredEvent{}, ErrNoRows
		}
		subject.Submission = event.SubmissionSubject{ID: parsed.Value}
	}
	if rawReservation != nil {
		parsed, ok := core.ParseTaskReservationID(*rawReservation).(core.TaskReservationIDCreated)
		if !ok {
			return event.StoredEvent{}, ErrNoRows
		}
		subject.Reservation = event.ReservationSubject{ID: parsed.Value}
	}
	if rawSeries != nil {
		parsed, ok := core.ParseTaskSeriesID(*rawSeries).(core.TaskSeriesIDCreated)
		if !ok {
			return event.StoredEvent{}, ErrNoRows
		}
		subject.Series = event.SeriesSubject{ID: parsed.Value}
	}
	if rawOrganization != nil {
		parsed, ok := core.ParseOrganizationID(*rawOrganization).(core.OrganizationIDCreated)
		if !ok {
			return event.StoredEvent{}, ErrNoRows
		}
		subject.Organization = event.OrganizationSubject{ID: parsed.Value}
	}
	if rawCollectible != nil {
		parsed, ok := core.ParseCollectibleID(*rawCollectible).(core.CollectibleIDCreated)
		if !ok {
			return event.StoredEvent{}, ErrNoRows
		}
		subject.Collectible = event.CollectibleSubject{ID: parsed.Value}
	}

	return event.WithoutEnrichment(event.StoredEvent{
		Event: event.Event{
			ID:         idResult.Value,
			Kind:       kindResult.Value,
			Actor:      actor,
			Subject:    subject,
			Metadata:   event.Metadata{JSON: rawMetadata},
			OccurredAt: occurredAt,
		},
		Cursor: event.CursorFromSequence(sequence),
	}), nil
}

// The ref helpers return the id string or nil for the nullable subject
// columns; the driver boundary is the one place absence becomes NULL. Both
// drivers dereference non-nil *string arguments.

func taskRefValue(ref event.TaskRef) *string {
	if subject, matched := ref.(event.TaskSubject); matched {
		value := subject.ID.String()
		return &value
	}
	return nil
}

func submissionRefValue(ref event.SubmissionRef) *string {
	if subject, matched := ref.(event.SubmissionSubject); matched {
		value := subject.ID.String()
		return &value
	}
	return nil
}

func reservationRefValue(ref event.ReservationRef) *string {
	if subject, matched := ref.(event.ReservationSubject); matched {
		value := subject.ID.String()
		return &value
	}
	return nil
}

func seriesRefValue(ref event.SeriesRef) *string {
	if subject, matched := ref.(event.SeriesSubject); matched {
		value := subject.ID.String()
		return &value
	}
	return nil
}

func organizationRefValue(ref event.OrganizationRef) *string {
	if subject, matched := ref.(event.OrganizationSubject); matched {
		value := subject.ID.String()
		return &value
	}
	return nil
}

func collectibleRefValue(ref event.CollectibleRef) *string {
	if subject, matched := ref.(event.CollectibleSubject); matched {
		value := subject.ID.String()
		return &value
	}
	return nil
}

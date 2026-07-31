package db

import (
	"context"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventStore persists the append-only domain event stream and serves the
// per-recipient feed. The host-only webhook pump reads the stream through
// dedicated struct methods (internal/webhookdispatch), not through the bridged
// event.Store interface.
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

	actorKind := "user"
	if _, matched := value.Actor.(event.ActorSystem); matched {
		actorKind = "system"
	}

	var sequence int64
	err = tx.QueryRow(ctx, `
		insert into domain_events (id, kind, actor_kind, actor_user_id, task_id, submission_id, reservation_id, series_id, organization_id, collectible_id, metadata_json, occurred_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12)
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
	).Scan(&sequence)
	if err != nil {
		return event.AppendStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "append domain event failed")}
	}

	for _, recipient := range recipients.Users {
		if _, err := tx.Exec(ctx, `
			insert into domain_event_recipients (event_seq, user_id)
			values ($1, $2)
		`, sequence, recipient.String()); err != nil {
			return event.AppendStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "append event recipient failed")}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return event.AppendStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit event append failed")}
	}
	return event.AppendStoreAccepted{Value: event.WithoutEnrichment(event.StoredEvent{Event: value, Cursor: event.CursorFromSequence(sequence)})}
}

func (store EventStore) ListForRecipient(ctx context.Context, recipient core.UserID, filter event.CursorFilter, page core.Page) event.ListStoreResult {
	afterSequence := int64(0)
	if after, matched := filter.(event.After); matched {
		afterSequence = after.Cursor.Sequence()
	}
	rows, err := store.db.Query(ctx, `
		select `+eventColumns+`,
			`+displayNameSQL("actor_user")+`,
			coalesce(subject_task.title, '')
		from domain_events
		join domain_event_recipients on domain_event_recipients.event_seq = domain_events.seq
		join users as actor_user on actor_user.id = domain_events.actor_user_id
		left join tasks as subject_task on subject_task.id = domain_events.task_id
		where domain_event_recipients.user_id = $1 and domain_events.seq > $2
		order by domain_events.seq
		limit $3 offset $4
	`, recipient.String(), afterSequence, page.Limit(), page.Offset())
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

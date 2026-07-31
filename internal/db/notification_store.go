package db

import (
	"context"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationStore struct {
	db Querier
}

func NewNotificationStore(pool *pgxpool.Pool) NotificationStore {
	return NewNotificationStoreFromHandle(NewPGX(pool))
}

func NewNotificationStoreFromHandle(handle Beginner) NotificationStore {
	return NotificationStore{db: handle}
}

func (store NotificationStore) Create(ctx context.Context, value notification.Notification) notification.CreateStoreResult {
	_, err := store.db.Exec(ctx, `
		insert into notifications (id, recipient_user_id, actor_user_id, kind, subject_kind, subject_id, state, metadata_json, created_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9)
	`, value.ID.String(), value.RecipientID.String(), value.ActorID.String(), value.Kind.String(), value.Subject.Kind, value.Subject.ID, value.State.String(), value.Metadata.JSON, value.CreatedAt)
	if err != nil {
		return notification.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "create notification failed")}
	}
	return notification.CreateStoreAccepted{}
}

func (store NotificationStore) List(ctx context.Context, recipient core.UserID, filter notification.StateFilter, page core.Page) notification.ListStoreResult {
	// The unread branch is served by notifications_recipient_state_idx.
	stateClause := ""
	if _, unreadOnly := filter.(notification.UnreadOnly); unreadOnly {
		stateClause = " and state = 'unread'"
	}
	query := `
		select id::text, recipient_user_id::text, actor_user_id::text, kind, subject_kind, subject_id, state, metadata_json::text, created_at
		from notifications
		where recipient_user_id = $1` + stateClause + `
		order by created_at desc, id desc
		limit $2 offset $3
	`
	rows, err := store.db.Query(ctx, query, recipient.String(), page.Limit(), page.Offset())
	if err != nil {
		return notification.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list notifications failed")}
	}
	defer rows.Close()
	return scanNotificationRows(rows)
}

func (store NotificationStore) CountUnread(ctx context.Context, recipient core.UserID) notification.CountStoreResult {
	var count int64
	err := store.db.QueryRow(ctx,
		"select count(*) from notifications where recipient_user_id = $1 and state = 'unread'",
		recipient.String()).Scan(&count)
	if err != nil {
		return notification.CountStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "count unread notifications failed")}
	}
	return notification.CountUnreadCounted{Count: count}
}

func (store NotificationStore) MarkRead(ctx context.Context, recipient core.UserID, id core.NotificationID) notification.MarkReadStoreResult {
	rows, err := store.db.Query(ctx, `
		update notifications
		set state = $3
		where id = $1 and recipient_user_id = $2
		returning id::text, recipient_user_id::text, actor_user_id::text, kind, subject_kind, subject_id, state, metadata_json::text, created_at
	`, id.String(), recipient.String(), notification.StateRead.String())
	if err != nil {
		return notification.MarkReadStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "mark notification read failed")}
	}
	defer rows.Close()
	result := scanNotificationRows(rows)
	listed, matched := result.(notification.ListStoreAccepted)
	if !matched {
		return notification.MarkReadStoreRejected{Reason: result.(notification.ListStoreRejected).Reason}
	}
	if len(listed.Values) != 1 {
		return notification.MarkReadStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "notification was not found")}
	}
	return notification.MarkReadStoreAccepted{Value: listed.Values[0]}
}

func scanNotificationRows(rows Rows) notification.ListStoreResult {
	values := make([]notification.Notification, 0)
	for rows.Next() {
		parsed := scanNotificationRow(rows)
		accepted, matched := parsed.(notificationRowAccepted)
		if !matched {
			return notification.ListStoreRejected{Reason: parsed.(notificationRowRejected).reason}
		}
		values = append(values, accepted.value)
	}
	if err := rows.Err(); err != nil {
		return notification.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read notifications failed")}
	}
	return notification.ListStoreAccepted{Values: values}
}

type notificationRowResult interface {
	notificationRowResult()
}

type notificationRowAccepted struct {
	value notification.Notification
}

type notificationRowRejected struct {
	reason core.DomainError
}

func (notificationRowAccepted) notificationRowResult() {}

func (notificationRowRejected) notificationRowResult() {}

func scanNotificationRow(rows Rows) notificationRowResult {
	var rawID string
	var rawRecipientID string
	var rawActorID string
	var kind string
	var subjectKind string
	var subjectID string
	var state string
	var metadataJSON string
	var createdAt time.Time
	if err := rows.Scan(&rawID, &rawRecipientID, &rawActorID, &kind, &subjectKind, &subjectID, &state, &metadataJSON, &createdAt); err != nil {
		return notificationRowRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan notification failed")}
	}
	idResult := core.ParseNotificationID(rawID)
	id, idMatched := idResult.(core.NotificationIDCreated)
	if !idMatched {
		return notificationRowRejected{reason: idResult.(core.NotificationIDRejected).Reason}
	}
	recipientResult := core.ParseUserID(rawRecipientID)
	recipient, recipientMatched := recipientResult.(core.UserIDCreated)
	if !recipientMatched {
		return notificationRowRejected{reason: recipientResult.(core.UserIDRejected).Reason}
	}
	actorResult := core.ParseUserID(rawActorID)
	actor, actorMatched := actorResult.(core.UserIDCreated)
	if !actorMatched {
		return notificationRowRejected{reason: actorResult.(core.UserIDRejected).Reason}
	}
	// A row whose kind or state falls outside the sealed enums is corrupt;
	// reject the read like other stores do for corrupt rows.
	kindResult := notification.ParseKind(kind)
	kindParsed, kindMatched := kindResult.(notification.KindParsed)
	if !kindMatched {
		return notificationRowRejected{reason: kindResult.(notification.KindRejected).Reason}
	}
	stateResult := notification.ParseState(state)
	stateParsed, stateMatched := stateResult.(notification.StateParsed)
	if !stateMatched {
		return notificationRowRejected{reason: stateResult.(notification.StateRejected).Reason}
	}
	return notificationRowAccepted{value: notification.Notification{
		ID:          id.Value,
		RecipientID: recipient.Value,
		ActorID:     actor.Value,
		Kind:        kindParsed.Value,
		Subject:     notification.Subject{Kind: subjectKind, ID: subjectID},
		State:       stateParsed.Value,
		Metadata:    notification.Metadata{JSON: metadataJSON},
		CreatedAt:   createdAt,
	}}
}

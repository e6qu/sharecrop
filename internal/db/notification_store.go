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

func (store NotificationStore) List(ctx context.Context, recipient core.UserID, page core.Page) notification.ListStoreResult {
	rows, err := store.db.Query(ctx, `
		select id::text, recipient_user_id::text, actor_user_id::text, kind, subject_kind, subject_id, state, metadata_json::text, created_at
		from notifications
		where recipient_user_id = $1
		order by created_at desc, id desc
		limit $2 offset $3
	`, recipient.String(), page.Limit(), page.Offset())
	if err != nil {
		return notification.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list notifications failed")}
	}
	defer rows.Close()
	return scanNotificationRows(rows)
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
	return notificationRowAccepted{value: notification.Notification{
		ID:          id.Value,
		RecipientID: recipient.Value,
		ActorID:     actor.Value,
		Kind:        notification.KindFromString(kind),
		Subject:     notification.Subject{Kind: subjectKind, ID: subjectID},
		State:       notification.StateFromString(state),
		Metadata:    notification.Metadata{JSON: metadataJSON},
		CreatedAt:   createdAt,
	}}
}

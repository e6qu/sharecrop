package mcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/notification"
)

type notificationSummary struct {
	ID              string `json:"id"`
	RecipientUserID string `json:"recipient_user_id"`
	ActorUserID     string `json:"actor_user_id"`
	Kind            string `json:"kind"`
	SubjectKind     string `json:"subject_kind"`
	SubjectID       string `json:"subject_id"`
	State           string `json:"state"`
	MetadataJSON    string `json:"metadata_json"`
	CreatedAt       string `json:"created_at"`
}

type notificationsPayload struct {
	Notifications []notificationSummary `json:"notifications"`
}

func (notificationSummary) payloadValue() {}

func (notificationsPayload) payloadValue() {}

func notificationToSummary(value notification.Notification) notificationSummary {
	return notificationSummary{
		ID:              value.ID.String(),
		RecipientUserID: value.RecipientID.String(),
		ActorUserID:     value.ActorID.String(),
		Kind:            value.Kind.String(),
		SubjectKind:     value.Subject.Kind,
		SubjectID:       value.Subject.ID,
		State:           value.State.String(),
		MetadataJSON:    value.Metadata.JSON,
		CreatedAt:       value.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (server Server) callListNotifications(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	result := server.services.ListNotifications(ctx, subject.ID, core.DefaultPage())
	listed, matched := result.(notification.NotificationsListed)
	if !matched {
		return toolFailed{message: result.(notification.ListRejected).Reason.Description()}
	}
	summaries := make([]notificationSummary, 0, len(listed.Values))
	for index := range listed.Values {
		summaries = append(summaries, notificationToSummary(listed.Values[index]))
	}
	return marshalPayload(notificationsPayload{Notifications: summaries})
}

func (server Server) callMarkNotificationRead(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		NotificationID string `json:"notification_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	idResult := core.ParseNotificationID(args.NotificationID)
	id, idMatched := idResult.(core.NotificationIDCreated)
	if !idMatched {
		return toolProtocolError{code: codeInvalidParams, message: idResult.(core.NotificationIDRejected).Reason.Description()}
	}
	result := server.services.MarkNotificationRead(ctx, subject.ID, id.Value)
	read, matched := result.(notification.NotificationRead)
	if !matched {
		return toolFailed{message: result.(notification.MarkReadRejected).Reason.Description()}
	}
	return marshalPayload(notificationToSummary(read.Value))
}

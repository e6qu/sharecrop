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
	// ActorDisplayName names the acting user, mirroring REST's notification
	// row enrichment; empty for system actors.
	ActorDisplayName string `json:"actor_display_name"`
	Kind             string `json:"kind"`
	SubjectKind      string `json:"subject_kind"`
	SubjectID        string `json:"subject_id"`
	// SubjectTitle is the referenced task's title when the subject is a
	// task; empty otherwise, mirroring REST.
	SubjectTitle string `json:"subject_title"`
	State        string `json:"state"`
	MetadataJSON string `json:"metadata_json"`
	CreatedAt    string `json:"created_at"`
}

type notificationsPayload struct {
	Notifications []notificationSummary `json:"notifications"`
	NextOffset    int                   `json:"next_offset"`
	// Total counts every row matching the filter, ignoring limit/offset.
	Total int64 `json:"total"`
}

func (notificationSummary) payloadValue() {}

func (notificationsPayload) payloadValue() {}

func notificationToSummary(value notification.Notification) notificationSummary {
	subjectTitle := ""
	if titled, matched := value.SubjectTitle.(notification.TaskSubjectTitle); matched {
		subjectTitle = titled.Title
	}
	return notificationSummary{
		ID:               value.ID.String(),
		RecipientUserID:  value.RecipientID.String(),
		ActorUserID:      value.ActorID.String(),
		ActorDisplayName: value.ActorDisplayName.String(),
		Kind:             value.Kind.String(),
		SubjectKind:      value.Subject.Kind,
		SubjectID:        value.Subject.ID,
		SubjectTitle:     subjectTitle,
		State:            value.State.String(),
		MetadataJSON:     value.Metadata.JSON,
		CreatedAt:        value.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (server Server) callListNotifications(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	filter := notification.StateFilter(notification.AnyState{})
	switch args.State {
	case "":
	case notification.StateUnread.String():
		filter = notification.UnreadOnly{}
	default:
		return toolProtocolError{code: codeInvalidParams, message: "state must be omitted or \"unread\""}
	}
	page, pageProblem := parseMCPPageArguments(arguments)
	if pageProblem != nil {
		return pageProblem
	}
	result := server.services.ListNotifications(ctx, subject.ID, filter, page.Probe())
	listed, matched := result.(notification.NotificationsListed)
	if !matched {
		return toolFailed{code: result.(notification.ListRejected).Reason.Code(), message: result.(notification.ListRejected).Reason.Description()}
	}
	visible, nextOffset := core.ProbeListWindow(len(listed.Values), page)
	summaries := make([]notificationSummary, 0, visible)
	for index := range listed.Values[:visible] {
		summaries = append(summaries, notificationToSummary(listed.Values[index]))
	}
	return marshalPayload(notificationsPayload{Notifications: summaries, NextOffset: nextOffset, Total: listed.Total})
}

type unreadNotificationCountPayload struct {
	UnreadCount int64 `json:"unread_count"`
}

func (unreadNotificationCountPayload) payloadValue() {}

func (server Server) callGetUnreadNotificationCount(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	result := server.services.CountUnreadNotifications(ctx, subject.ID)
	counted, matched := result.(notification.UnreadCounted)
	if !matched {
		return toolFailed{code: result.(notification.CountRejected).Reason.Code(), message: result.(notification.CountRejected).Reason.Description()}
	}
	return marshalPayload(unreadNotificationCountPayload{UnreadCount: counted.Count})
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
		return invalidIDArgument("notification_id")
	}
	result := server.services.MarkNotificationRead(ctx, subject.ID, id.Value)
	read, matched := result.(notification.NotificationRead)
	if !matched {
		return toolFailed{code: result.(notification.MarkReadRejected).Reason.Code(), message: result.(notification.MarkReadRejected).Reason.Description()}
	}
	return marshalPayload(notificationToSummary(read.Value))
}

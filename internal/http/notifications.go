package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/notification"
)

func (notificationsResponse) writableResponse() {}

func (notificationResponse) writableResponse() {}

func (server Server) listNotifications(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}

	page, pageOK := parsePageOrReject(w, r)
	if !pageOK {
		return
	}
	result := server.notificationService.List(r.Context(), actor.subject.ID, page)
	listed, listedMatched := result.(notification.NotificationsListed)
	if !listedMatched {
		writeDomainError(w, result.(notification.ListRejected).Reason)
		return
	}

	response := notificationsResponse{Notifications: make([]notificationResponse, 0, len(listed.Values))}
	for _, value := range listed.Values {
		response.Notifications = append(response.Notifications, notificationToResponse(value))
	}
	writeJSON(w, http.StatusOK, response)
}

func (server Server) markNotificationRead(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}

	idResult := core.ParseNotificationID(r.PathValue("notification_id"))
	idAccepted, idMatched := idResult.(core.NotificationIDCreated)
	if !idMatched {
		writeError(w, http.StatusBadRequest, idResult.(core.NotificationIDRejected).Reason.Description())
		return
	}

	result := server.notificationService.MarkRead(r.Context(), actor.subject.ID, idAccepted.Value)
	read, readMatched := result.(notification.NotificationRead)
	if !readMatched {
		writeDomainError(w, result.(notification.MarkReadRejected).Reason)
		return
	}
	writeJSON(w, http.StatusOK, notificationToResponse(read.Value))
}

func (server Server) notify(w http.ResponseWriter, ctx context.Context, recipient core.UserID, actor core.UserID, kind notification.Kind, subject notification.Subject, metadata notification.Metadata) bool {
	result := server.notificationService.Notify(ctx, recipient, actor, kind, subject, metadata)
	switch typed := result.(type) {
	case notification.NotificationCreated:
		return true
	case notification.NotificationSkipped:
		return true
	case notification.NotifyRejected:
		writeDomainError(w, typed.Reason)
		return false
	default:
		panic("unhandled notification result")
	}
}

func notificationToResponse(value notification.Notification) notificationResponse {
	return notificationResponse{
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

func notificationSubjectForSubmission(submissionID core.SubmissionID) notification.Subject {
	return notification.Subject{Kind: "submission", ID: submissionID.String()}
}

func taskNotificationMetadata(taskID core.TaskID) notification.Metadata {
	encoded, err := json.Marshal(struct {
		TaskID string `json:"task_id"`
	}{TaskID: taskID.String()})
	if err != nil {
		panic("encode notification metadata: " + err.Error())
	}
	return notification.Metadata{JSON: string(encoded)}
}

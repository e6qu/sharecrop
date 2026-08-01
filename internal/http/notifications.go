package httpserver

import (
	"net/http"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/notification"
)

func (notificationsResponse) writableResponse() {}

func (notificationResponse) writableResponse() {}

func (notificationUnreadCountResponse) writableResponse() {}

func (server Server) listNotifications(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}

	filterResult := parseNotificationStateFilter(r.URL.Query().Get("state"))
	filter, filterMatched := filterResult.(stateFilterAccepted)
	if !filterMatched {
		writeDomainError(w, filterResult.(stateFilterRejected).reason)
		return
	}

	page, pageOK := parsePageOrReject(w, r)
	if !pageOK {
		return
	}
	result := server.notificationService.List(r.Context(), actor.subject.ID, filter.value, page.Probe())
	listed, listedMatched := result.(notification.NotificationsListed)
	if !listedMatched {
		writeDomainError(w, result.(notification.ListRejected).Reason)
		return
	}

	visible, nextOffset := probeListWindow(len(listed.Values), page)
	response := notificationsResponse{Notifications: make([]notificationResponse, 0, visible), NextOffset: nextOffset, Total: listed.Total}
	for _, value := range listed.Values[:visible] {
		response.Notifications = append(response.Notifications, notificationToResponse(value))
	}
	writeJSON(w, http.StatusOK, response)
}

func (server Server) unreadNotificationCount(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}

	result := server.notificationService.CountUnread(r.Context(), actor.subject.ID)
	counted, countedMatched := result.(notification.UnreadCounted)
	if !countedMatched {
		writeDomainError(w, result.(notification.CountRejected).Reason)
		return
	}
	writeJSON(w, http.StatusOK, notificationUnreadCountResponse{UnreadCount: counted.Count})
}

func (server Server) markNotificationRead(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}

	idResult := core.ParseNotificationID(r.PathValue("notification_id"))
	idAccepted, idMatched := idResult.(core.NotificationIDCreated)
	if !idMatched {
		writeDomainError(w, idResult.(core.NotificationIDRejected).Reason)
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

type stateFilterParseResult interface {
	stateFilterParseResult()
}

type stateFilterAccepted struct {
	value notification.StateFilter
}

type stateFilterRejected struct {
	reason core.DomainError
}

func (stateFilterAccepted) stateFilterParseResult() {}

func (stateFilterRejected) stateFilterParseResult() {}

// parseNotificationStateFilter maps the optional ?state= query parameter onto
// the listing filter: absent means the whole inbox, "unread" narrows to
// unread rows, and everything else is an enum error.
func parseNotificationStateFilter(raw string) stateFilterParseResult {
	switch raw {
	case "":
		return stateFilterAccepted{value: notification.AnyState{}}
	case notification.StateUnread.String():
		return stateFilterAccepted{value: notification.UnreadOnly{}}
	default:
		return stateFilterRejected{reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "notification state filter is invalid")}
	}
}

func notificationToResponse(value notification.Notification) notificationResponse {
	subjectTitle := ""
	if titled, matched := value.SubjectTitle.(notification.TaskSubjectTitle); matched {
		subjectTitle = titled.Title
	}
	return notificationResponse{
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

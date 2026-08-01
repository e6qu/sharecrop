package webhook

import (
	"encoding/json"
	"time"

	"github.com/e6qu/sharecrop/internal/event"
)

// EventPayload is the wire shape of one domain event as consumers see it —
// the browser live feed (GET /api/events and its SSE stream) and every
// webhook delivery body carry exactly this struct, so the two surfaces
// cannot drift apart. Absent subject references are empty strings; metadata
// is the event's bounded JSON document, passed through as a string.
type EventPayload struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	ActorKind   string `json:"actor_kind"`
	ActorUserID string `json:"actor_user_id"`
	// ActorDisplayName names the acting user when the store resolved it;
	// empty for system actors and unenriched reads.
	ActorDisplayName string `json:"actor_display_name"`
	OccurredAt       string `json:"occurred_at"`
	Cursor           string `json:"cursor"`
	TaskID           string `json:"task_id"`
	// TaskTitle is the referenced task's title when the store resolved it;
	// empty when the event references no task or the read was unenriched.
	TaskTitle      string `json:"task_title"`
	SubmissionID   string `json:"submission_id"`
	ReservationID  string `json:"reservation_id"`
	SeriesID       string `json:"series_id"`
	OrganizationID string `json:"organization_id"`
	CollectibleID  string `json:"collectible_id"`
	MetadataJSON   string `json:"metadata_json"`
}

// EventPayloadFromStored flattens a stored domain event into its wire shape.
func EventPayloadFromStored(stored event.StoredEvent) EventPayload {
	payload := EventPayload{
		ID:           stored.Event.ID.String(),
		Kind:         stored.Event.Kind.String(),
		ActorKind:    "system",
		OccurredAt:   stored.Event.OccurredAt.UTC().Format(time.RFC3339Nano),
		Cursor:       stored.Cursor.String(),
		MetadataJSON: stored.Event.Metadata.JSON,
	}
	if actor, matched := stored.Event.Actor.(event.ActorUser); matched {
		payload.ActorKind = "user"
		payload.ActorUserID = actor.ID.String()
	}
	if named, matched := stored.ActorName.(event.ActorNamed); matched {
		payload.ActorDisplayName = named.DisplayName.String()
	}
	if titled, matched := stored.TaskTitle.(event.TaskTitled); matched {
		payload.TaskTitle = titled.Title
	}
	if ref, matched := stored.Event.Subject.Task.(event.TaskSubject); matched {
		payload.TaskID = ref.ID.String()
	}
	if ref, matched := stored.Event.Subject.Submission.(event.SubmissionSubject); matched {
		payload.SubmissionID = ref.ID.String()
	}
	if ref, matched := stored.Event.Subject.Reservation.(event.ReservationSubject); matched {
		payload.ReservationID = ref.ID.String()
	}
	if ref, matched := stored.Event.Subject.Series.(event.SeriesSubject); matched {
		payload.SeriesID = ref.ID.String()
	}
	if ref, matched := stored.Event.Subject.Organization.(event.OrganizationSubject); matched {
		payload.OrganizationID = ref.ID.String()
	}
	if ref, matched := stored.Event.Subject.Collectible.(event.CollectibleSubject); matched {
		payload.CollectibleID = ref.ID.String()
	}
	return payload
}

// DeliveryBody is the JSON document POSTed to a webhook receiver: the event
// in the shared feed shape plus the receiving subscription's id.
type DeliveryBody struct {
	Event          EventPayload `json:"event"`
	SubscriptionID string       `json:"subscription_id"`
}

// EncodeDeliveryBody renders the delivery body for one (event, subscription)
// pair. It sits at the JSON boundary, so it returns a plain error.
func EncodeDeliveryBody(stored event.StoredEvent, subscriptionID string) ([]byte, error) {
	return json.Marshal(DeliveryBody{Event: EventPayloadFromStored(stored), SubscriptionID: subscriptionID})
}

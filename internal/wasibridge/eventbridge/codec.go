// Package eventbridge is the WASI bridge for internal/event's Store, built the
// same way as notificationbridge: hand-written per-type codecs (this file)
// plus a generated dispatcher and guest client (bridge_gen.go). Shared core
// types (ids, page, time) are serialized by internal/wasibridge/corewire; only
// event-specific types live here.
package eventbridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
)

// ---- event.Event ----

// eventWire flattens the actor union and the subject refs. ActorKind is
// "user" or "system"; an absent subject ref is the empty string.
type eventWire struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	ActorKind    string `json:"actor_kind"`
	ActorUserID  string `json:"actor_user_id,omitempty"`
	Task         string `json:"task,omitempty"`
	Submission   string `json:"submission,omitempty"`
	Reservation  string `json:"reservation,omitempty"`
	Series       string `json:"series,omitempty"`
	Organization string `json:"organization,omitempty"`
	Collectible  string `json:"collectible,omitempty"`
	Metadata     string `json:"metadata"`
	OccurredAt   string `json:"occurred_at"`
}

func encodeEvent(value event.Event) eventWire {
	wire := eventWire{
		ID:         corewire.EncodeDomainEventID(value.ID),
		Kind:       value.Kind.String(),
		ActorKind:  "system",
		Metadata:   value.Metadata.JSON,
		OccurredAt: corewire.EncodeTime(value.OccurredAt),
	}
	if actor, matched := value.Actor.(event.ActorUser); matched {
		wire.ActorKind = "user"
		wire.ActorUserID = corewire.EncodeUserID(actor.ID)
	}
	if ref, matched := value.Subject.Task.(event.TaskSubject); matched {
		wire.Task = corewire.EncodeTaskID(ref.ID)
	}
	if ref, matched := value.Subject.Submission.(event.SubmissionSubject); matched {
		wire.Submission = corewire.EncodeSubmissionID(ref.ID)
	}
	if ref, matched := value.Subject.Reservation.(event.ReservationSubject); matched {
		wire.Reservation = corewire.EncodeTaskReservationID(ref.ID)
	}
	if ref, matched := value.Subject.Series.(event.SeriesSubject); matched {
		wire.Series = corewire.EncodeTaskSeriesID(ref.ID)
	}
	if ref, matched := value.Subject.Organization.(event.OrganizationSubject); matched {
		wire.Organization = corewire.EncodeOrganizationID(ref.ID)
	}
	if ref, matched := value.Subject.Collectible.(event.CollectibleSubject); matched {
		wire.Collectible = corewire.EncodeCollectibleID(ref.ID)
	}
	return wire
}

func decodeEvent(wire eventWire) (event.Event, error) {
	id, err := corewire.DecodeDomainEventID(wire.ID)
	if err != nil {
		return event.Event{}, err
	}
	kindResult, kindMatched := event.ParseKind(wire.Kind).(event.KindParsed)
	if !kindMatched {
		return event.Event{}, fmt.Errorf("invalid event kind %q", wire.Kind)
	}
	occurredAt, err := corewire.DecodeTime(wire.OccurredAt)
	if err != nil {
		return event.Event{}, err
	}

	var actor event.Actor
	switch wire.ActorKind {
	case "system":
		actor = event.ActorSystem{}
	case "user":
		userID, userErr := corewire.DecodeUserID(wire.ActorUserID)
		if userErr != nil {
			return event.Event{}, userErr
		}
		actor = event.ActorUser{ID: userID}
	default:
		return event.Event{}, fmt.Errorf("invalid event actor kind %q", wire.ActorKind)
	}

	subject := event.NoSubjectRefs()
	if wire.Task != "" {
		ref, refErr := corewire.DecodeTaskID(wire.Task)
		if refErr != nil {
			return event.Event{}, refErr
		}
		subject.Task = event.TaskSubject{ID: ref}
	}
	if wire.Submission != "" {
		ref, refErr := corewire.DecodeSubmissionID(wire.Submission)
		if refErr != nil {
			return event.Event{}, refErr
		}
		subject.Submission = event.SubmissionSubject{ID: ref}
	}
	if wire.Reservation != "" {
		ref, refErr := corewire.DecodeTaskReservationID(wire.Reservation)
		if refErr != nil {
			return event.Event{}, refErr
		}
		subject.Reservation = event.ReservationSubject{ID: ref}
	}
	if wire.Series != "" {
		ref, refErr := corewire.DecodeTaskSeriesID(wire.Series)
		if refErr != nil {
			return event.Event{}, refErr
		}
		subject.Series = event.SeriesSubject{ID: ref}
	}
	if wire.Organization != "" {
		ref, refErr := corewire.DecodeOrganizationID(wire.Organization)
		if refErr != nil {
			return event.Event{}, refErr
		}
		subject.Organization = event.OrganizationSubject{ID: ref}
	}
	if wire.Collectible != "" {
		ref, refErr := corewire.DecodeCollectibleID(wire.Collectible)
		if refErr != nil {
			return event.Event{}, refErr
		}
		subject.Collectible = event.CollectibleSubject{ID: ref}
	}

	return event.Event{
		ID:         id,
		Kind:       kindResult.Value,
		Actor:      actor,
		Subject:    subject,
		Metadata:   event.Metadata{JSON: wire.Metadata},
		OccurredAt: occurredAt,
	}, nil
}

// ---- event.Recipients ----

func encodeRecipients(recipients event.Recipients) []string {
	values := make([]string, 0, len(recipients.Users))
	for _, user := range recipients.Users {
		values = append(values, corewire.EncodeUserID(user))
	}
	return values
}

func decodeRecipients(wire []string) (event.Recipients, error) {
	users := make([]core.UserID, 0, len(wire))
	for _, raw := range wire {
		user, err := corewire.DecodeUserID(raw)
		if err != nil {
			return event.Recipients{}, err
		}
		users = append(users, user)
	}
	return event.NewRecipients(users...), nil
}

// ---- event.CursorFilter ----

// cursorFilterWire carries the after-cursor as a decimal string; the empty
// string is FromStart.
type cursorFilterWire struct {
	After string `json:"after,omitempty"`
}

func encodeCursorFilter(filter event.CursorFilter) cursorFilterWire {
	if after, matched := filter.(event.After); matched {
		return cursorFilterWire{After: after.Cursor.String()}
	}
	return cursorFilterWire{}
}

func decodeCursorFilter(wire cursorFilterWire) (event.CursorFilter, error) {
	if wire.After == "" {
		return event.FromStart{}, nil
	}
	parsed, matched := event.ParseCursor(wire.After).(event.CursorParsed)
	if !matched {
		return nil, fmt.Errorf("invalid event cursor %q", wire.After)
	}
	return event.After{Cursor: parsed.Value}, nil
}

// ---- stored events ----

type storedEventWire struct {
	Event  eventWire `json:"event"`
	Cursor string    `json:"cursor"`
	// ActorName and TaskTitle carry the feed read's enrichment; empty means
	// the absent variant (append results, pump reads).
	ActorName string `json:"actor_name,omitempty"`
	TaskTitle string `json:"task_title,omitempty"`
}

func encodeStoredEvent(value event.StoredEvent) storedEventWire {
	actorName := ""
	if named, matched := value.ActorName.(event.ActorNamed); matched {
		actorName = named.DisplayName.String()
	}
	taskTitle := ""
	if titled, matched := value.TaskTitle.(event.TaskTitled); matched {
		taskTitle = titled.Title
	}
	return storedEventWire{Event: encodeEvent(value.Event), Cursor: value.Cursor.String(), ActorName: actorName, TaskTitle: taskTitle}
}

func decodeStoredEvent(wire storedEventWire) (event.StoredEvent, error) {
	value, err := decodeEvent(wire.Event)
	if err != nil {
		return event.StoredEvent{}, err
	}
	parsed, matched := event.ParseCursor(wire.Cursor).(event.CursorParsed)
	if !matched {
		return event.StoredEvent{}, fmt.Errorf("invalid stored event cursor %q", wire.Cursor)
	}
	stored := event.WithoutEnrichment(event.StoredEvent{Event: value, Cursor: parsed.Value})
	if wire.ActorName != "" {
		nameResult, nameMatched := auth.NewDisplayName(wire.ActorName).(auth.DisplayNameAccepted)
		if !nameMatched {
			return event.StoredEvent{}, fmt.Errorf("stored event actor name is invalid")
		}
		stored.ActorName = event.ActorNamed{DisplayName: nameResult.Value}
	}
	if wire.TaskTitle != "" {
		stored.TaskTitle = event.TaskTitled{Title: wire.TaskTitle}
	}
	return stored, nil
}

// ---- result unions ----

type appendResultWire struct {
	Variant string                  `json:"variant"`
	Stored  *storedEventWire        `json:"stored,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeAppendResult(result event.AppendStoreResult) appendResultWire {
	switch typed := result.(type) {
	case event.AppendStoreAccepted:
		stored := encodeStoredEvent(typed.Value)
		return appendResultWire{Variant: "accepted", Stored: &stored}
	case event.AppendStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return appendResultWire{Variant: "rejected", Error: &reason}
	default:
		return appendResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown event result %T", result))}
	}
}

func decodeAppendResult(wire appendResultWire) (event.AppendStoreResult, error) {
	switch wire.Variant {
	case "accepted":
		if wire.Stored == nil {
			return nil, fmt.Errorf("accepted append result is missing its event")
		}
		stored, err := decodeStoredEvent(*wire.Stored)
		if err != nil {
			return nil, err
		}
		return event.AppendStoreAccepted{Value: stored}, nil
	case "rejected":
		if wire.Error == nil {
			return nil, fmt.Errorf("rejected append result is missing its error")
		}
		return event.AppendStoreRejected{Reason: domainwire.DecodeDomainError(*wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown append result variant %q", wire.Variant)
	}
}

type listResultWire struct {
	Variant string                  `json:"variant"`
	Events  []storedEventWire       `json:"events,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListResult(result event.ListStoreResult) listResultWire {
	switch typed := result.(type) {
	case event.ListStoreAccepted:
		values := make([]storedEventWire, 0, len(typed.Values))
		for index := range typed.Values {
			values = append(values, encodeStoredEvent(typed.Values[index]))
		}
		return listResultWire{Variant: "listed", Events: values}
	case event.ListStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return listResultWire{Variant: "rejected", Error: &reason}
	default:
		return listResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown event result %T", result))}
	}
}

func decodeListResult(wire listResultWire) (event.ListStoreResult, error) {
	switch wire.Variant {
	case "listed":
		values := make([]event.StoredEvent, 0, len(wire.Events))
		for index := range wire.Events {
			value, err := decodeStoredEvent(wire.Events[index])
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return event.ListStoreAccepted{Values: values}, nil
	case "rejected":
		if wire.Error == nil {
			return nil, fmt.Errorf("rejected list result is missing its error")
		}
		return event.ListStoreRejected{Reason: domainwire.DecodeDomainError(*wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown list result variant %q", wire.Variant)
	}
}

func rejectionError(message string) *domainwire.DomainError {
	encoded := domainwire.EncodeDomainError(core.NewDomainError(core.ErrorCodeInvalidState, message))
	return &encoded
}

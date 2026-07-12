// Package auditwire holds the shared wire codec for audit.Event, used by both
// the audit store bridge (auditbridge) and the moderation-triage bridge
// (moderationtriagebridge). Both carry a full audit.Event across the WASI
// boundary; sharing the codec keeps them from drifting and satisfies the
// copy-paste gate, which forbids two identical codecs for the same type.
package auditwire

import (
	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
)

// SubjectWire is the wire form of audit.Subject.
type SubjectWire struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

func encodeSubject(subject audit.Subject) SubjectWire {
	return SubjectWire{Kind: subject.Kind, ID: subject.ID}
}

func decodeSubject(wire SubjectWire) audit.Subject {
	return audit.Subject{Kind: wire.Kind, ID: wire.ID}
}

// EventWire is the wire form of audit.Event, carrying every field.
type EventWire struct {
	ID          string      `json:"id"`
	ActorUserID string      `json:"actor_user_id"`
	Action      string      `json:"action"`
	Subject     SubjectWire `json:"subject"`
	Metadata    string      `json:"metadata"`
	CreatedAt   string      `json:"created_at"`
}

// EncodeEvent serializes an audit.Event to its wire form.
func EncodeEvent(event audit.Event) EventWire {
	return EventWire{
		ID:          corewire.EncodeAuditEventID(event.ID),
		ActorUserID: corewire.EncodeUserID(event.ActorUserID),
		Action:      event.Action.String(),
		Subject:     encodeSubject(event.Subject),
		Metadata:    event.Metadata.JSON,
		CreatedAt:   corewire.EncodeTime(event.CreatedAt),
	}
}

// DecodeEvent reconstructs an audit.Event from its wire form.
func DecodeEvent(wire EventWire) (audit.Event, error) {
	id, err := corewire.DecodeAuditEventID(wire.ID)
	if err != nil {
		return audit.Event{}, err
	}
	actor, err := corewire.DecodeUserID(wire.ActorUserID)
	if err != nil {
		return audit.Event{}, err
	}
	createdAt, err := corewire.DecodeTime(wire.CreatedAt)
	if err != nil {
		return audit.Event{}, err
	}
	return audit.Event{
		ID:          id,
		ActorUserID: actor,
		Action:      audit.ActionFromString(wire.Action),
		Subject:     decodeSubject(wire.Subject),
		Metadata:    audit.Metadata{JSON: wire.Metadata},
		CreatedAt:   createdAt,
	}, nil
}

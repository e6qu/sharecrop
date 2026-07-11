// Package notificationbridge is the WASI bridge for internal/notification's
// Store, built the same way as auditbridge: hand-written per-type codecs (this
// file) plus a generated dispatcher and guest client (bridge_gen.go). Shared
// core types (ids, page, time) are serialized by
// internal/wasibridge/corewire; only notification-specific types live here.
package notificationbridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
)

// ---- notification.Kind / notification.State (free-form string wrappers) ----

func encodeKind(kind notification.Kind) string { return kind.String() }

func decodeKind(raw string) notification.Kind { return notification.KindFromString(raw) }

func encodeState(state notification.State) string { return state.String() }

func decodeState(raw string) notification.State { return notification.StateFromString(raw) }

// ---- notification.Subject / notification.Metadata ----

type subjectWire struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

func encodeSubject(subject notification.Subject) subjectWire {
	return subjectWire{Kind: subject.Kind, ID: subject.ID}
}

func decodeSubject(wire subjectWire) notification.Subject {
	return notification.Subject{Kind: wire.Kind, ID: wire.ID}
}

func encodeMetadata(metadata notification.Metadata) string { return metadata.JSON }

func decodeMetadata(raw string) notification.Metadata { return notification.Metadata{JSON: raw} }

// ---- notification.Notification ----

type notificationWire struct {
	ID          string      `json:"id"`
	RecipientID string      `json:"recipient_id"`
	ActorID     string      `json:"actor_id"`
	Kind        string      `json:"kind"`
	Subject     subjectWire `json:"subject"`
	State       string      `json:"state"`
	Metadata    string      `json:"metadata"`
	CreatedAt   string      `json:"created_at"`
}

func encodeNotification(value notification.Notification) notificationWire {
	return notificationWire{
		ID:          corewire.EncodeNotificationID(value.ID),
		RecipientID: corewire.EncodeUserID(value.RecipientID),
		ActorID:     corewire.EncodeUserID(value.ActorID),
		Kind:        encodeKind(value.Kind),
		Subject:     encodeSubject(value.Subject),
		State:       encodeState(value.State),
		Metadata:    encodeMetadata(value.Metadata),
		CreatedAt:   corewire.EncodeTime(value.CreatedAt),
	}
}

func decodeNotification(wire notificationWire) (notification.Notification, error) {
	id, err := corewire.DecodeNotificationID(wire.ID)
	if err != nil {
		return notification.Notification{}, err
	}
	recipient, err := corewire.DecodeUserID(wire.RecipientID)
	if err != nil {
		return notification.Notification{}, err
	}
	actor, err := corewire.DecodeUserID(wire.ActorID)
	if err != nil {
		return notification.Notification{}, err
	}
	createdAt, err := corewire.DecodeTime(wire.CreatedAt)
	if err != nil {
		return notification.Notification{}, err
	}
	return notification.Notification{
		ID:          id,
		RecipientID: recipient,
		ActorID:     actor,
		Kind:        decodeKind(wire.Kind),
		Subject:     decodeSubject(wire.Subject),
		State:       decodeState(wire.State),
		Metadata:    decodeMetadata(wire.Metadata),
		CreatedAt:   createdAt,
	}, nil
}

// ---- result unions ----

type createResultWire struct {
	Variant string                  `json:"variant"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeCreateResult(result notification.CreateStoreResult) createResultWire {
	switch typed := result.(type) {
	case notification.CreateStoreAccepted:
		return createResultWire{Variant: "accepted"}
	case notification.CreateStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return createResultWire{Variant: "rejected", Error: &reason}
	default:
		return createResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown notification result %T", result))}
	}
}

func decodeCreateResult(wire createResultWire) (notification.CreateStoreResult, error) {
	switch wire.Variant {
	case "accepted":
		return notification.CreateStoreAccepted{}, nil
	case "rejected":
		if wire.Error == nil {
			return nil, fmt.Errorf("rejected create result is missing its error")
		}
		return notification.CreateStoreRejected{Reason: domainwire.DecodeDomainError(*wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown create result variant %q", wire.Variant)
	}
}

type listResultWire struct {
	Variant       string                  `json:"variant"`
	Notifications []notificationWire      `json:"notifications,omitempty"`
	Error         *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListResult(result notification.ListStoreResult) listResultWire {
	switch typed := result.(type) {
	case notification.ListStoreAccepted:
		values := make([]notificationWire, 0, len(typed.Values))
		for index := range typed.Values {
			values = append(values, encodeNotification(typed.Values[index]))
		}
		return listResultWire{Variant: "listed", Notifications: values}
	case notification.ListStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return listResultWire{Variant: "rejected", Error: &reason}
	default:
		return listResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown notification result %T", result))}
	}
}

func decodeListResult(wire listResultWire) (notification.ListStoreResult, error) {
	switch wire.Variant {
	case "listed":
		values := make([]notification.Notification, 0, len(wire.Notifications))
		for index := range wire.Notifications {
			value, err := decodeNotification(wire.Notifications[index])
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return notification.ListStoreAccepted{Values: values}, nil
	case "rejected":
		if wire.Error == nil {
			return nil, fmt.Errorf("rejected list result is missing its error")
		}
		return notification.ListStoreRejected{Reason: domainwire.DecodeDomainError(*wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown list result variant %q", wire.Variant)
	}
}

type markReadResultWire struct {
	Variant      string                  `json:"variant"`
	Notification *notificationWire       `json:"notification,omitempty"`
	Error        *domainwire.DomainError `json:"error,omitempty"`
}

func encodeMarkReadResult(result notification.MarkReadStoreResult) markReadResultWire {
	switch typed := result.(type) {
	case notification.MarkReadStoreAccepted:
		value := encodeNotification(typed.Value)
		return markReadResultWire{Variant: "accepted", Notification: &value}
	case notification.MarkReadStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return markReadResultWire{Variant: "rejected", Error: &reason}
	default:
		return markReadResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown notification result %T", result))}
	}
}

func decodeMarkReadResult(wire markReadResultWire) (notification.MarkReadStoreResult, error) {
	switch wire.Variant {
	case "accepted":
		if wire.Notification == nil {
			return nil, fmt.Errorf("accepted mark-read result is missing its notification")
		}
		value, err := decodeNotification(*wire.Notification)
		if err != nil {
			return nil, err
		}
		return notification.MarkReadStoreAccepted{Value: value}, nil
	case "rejected":
		if wire.Error == nil {
			return nil, fmt.Errorf("rejected mark-read result is missing its error")
		}
		return notification.MarkReadStoreRejected{Reason: domainwire.DecodeDomainError(*wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown mark-read result variant %q", wire.Variant)
	}
}

// rejectionError encodes a defensive rejection for a result type outside its
// known union - a case the sealed interface should make unreachable.
func rejectionError(message string) *domainwire.DomainError {
	reason := domainwire.EncodeDomainError(core.NewDomainError(core.ErrorCodeInvalidState, message))
	return &reason
}

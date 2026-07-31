// Package webhookbridge is the WASI bridge for internal/webhook's Store,
// built the same way as eventbridge: hand-written per-type codecs (this file)
// plus a generated dispatcher and guest client (bridge_gen.go). Shared core
// types (ids, page, time) are serialized by internal/wasibridge/corewire;
// only webhook-specific types live here. The signing secret crosses the
// bridge as written — the host store persists it as written too, because the
// dispatcher signs delivery bodies with it.
package webhookbridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
	"github.com/e6qu/sharecrop/internal/webhook"
)

// ---- webhook.Owner ----

// ownerWire flattens the owner union: kind is "user" or "organization" and
// id is the matching identifier.
type ownerWire struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

func encodeOwner(owner webhook.Owner) ownerWire {
	switch typed := owner.(type) {
	case webhook.OwnerUser:
		return ownerWire{Kind: "user", ID: typed.ID.String()}
	case webhook.OwnerOrganization:
		return ownerWire{Kind: "organization", ID: typed.ID.String()}
	default:
		return ownerWire{}
	}
}

func decodeOwner(wire ownerWire) (webhook.Owner, error) {
	switch wire.Kind {
	case "user":
		id, err := corewire.DecodeUserID(wire.ID)
		if err != nil {
			return nil, err
		}
		return webhook.OwnerUser{ID: id}, nil
	case "organization":
		id, err := corewire.DecodeOrganizationID(wire.ID)
		if err != nil {
			return nil, err
		}
		return webhook.OwnerOrganization{ID: id}, nil
	default:
		return nil, fmt.Errorf("invalid webhook owner kind %q", wire.Kind)
	}
}

// ---- webhook.Secret ----

func encodeSecret(secret webhook.Secret) string {
	return secret.String()
}

func decodeSecret(raw string) (webhook.Secret, error) {
	parsed, matched := webhook.ParseSecret(raw).(webhook.SecretAccepted)
	if !matched {
		return webhook.Secret{}, fmt.Errorf("invalid webhook secret")
	}
	return parsed.Value, nil
}

// ---- webhook.Subscription ----

type subscriptionWire struct {
	ID        string    `json:"id"`
	Owner     ownerWire `json:"owner"`
	URL       string    `json:"url"`
	Kinds     []string  `json:"kinds"`
	State     string    `json:"state"`
	CreatedAt string    `json:"created_at"`
}

func encodeSubscription(value webhook.Subscription) subscriptionWire {
	kinds := value.Kinds.Values()
	rawKinds := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		rawKinds = append(rawKinds, kind.String())
	}
	return subscriptionWire{
		ID:        corewire.EncodeWebhookSubscriptionID(value.ID),
		Owner:     encodeOwner(value.Owner),
		URL:       value.URL.String(),
		Kinds:     rawKinds,
		State:     value.State.String(),
		CreatedAt: corewire.EncodeTime(value.CreatedAt),
	}
}

func decodeSubscription(wire subscriptionWire) (webhook.Subscription, error) {
	id, err := corewire.DecodeWebhookSubscriptionID(wire.ID)
	if err != nil {
		return webhook.Subscription{}, err
	}
	owner, err := decodeOwner(wire.Owner)
	if err != nil {
		return webhook.Subscription{}, err
	}
	endpoint, endpointMatched := webhook.NewEndpointURL(wire.URL).(webhook.EndpointURLAccepted)
	if !endpointMatched {
		return webhook.Subscription{}, fmt.Errorf("invalid webhook url %q", wire.URL)
	}
	kinds := make([]event.Kind, 0, len(wire.Kinds))
	for _, rawKind := range wire.Kinds {
		parsed, matched := event.ParseKind(rawKind).(event.KindParsed)
		if !matched {
			return webhook.Subscription{}, fmt.Errorf("invalid webhook event kind %q", rawKind)
		}
		kinds = append(kinds, parsed.Value)
	}
	filter, filterMatched := webhook.NewKindFilter(kinds).(webhook.KindFilterAccepted)
	if !filterMatched {
		return webhook.Subscription{}, fmt.Errorf("webhook kind filter is empty")
	}
	state, stateMatched := webhook.ParseState(wire.State).(webhook.StateAccepted)
	if !stateMatched {
		return webhook.Subscription{}, fmt.Errorf("invalid webhook subscription state %q", wire.State)
	}
	createdAt, err := corewire.DecodeTime(wire.CreatedAt)
	if err != nil {
		return webhook.Subscription{}, err
	}
	return webhook.Subscription{
		ID:        id,
		Owner:     owner,
		URL:       endpoint.Value,
		Kinds:     filter.Value,
		State:     state.Value,
		CreatedAt: createdAt,
	}, nil
}

// ---- webhook.Delivery ----

type deliveryWire struct {
	ID            string `json:"id"`
	EventCursor   string `json:"event_cursor"`
	State         string `json:"state"`
	AttemptCount  int64  `json:"attempt_count"`
	NextAttemptAt string `json:"next_attempt_at"`
	LastStatus    string `json:"last_status"`
}

func encodeDelivery(value webhook.Delivery) deliveryWire {
	return deliveryWire{
		ID:            corewire.EncodeWebhookDeliveryID(value.ID),
		EventCursor:   value.EventCursor.String(),
		State:         value.State.String(),
		AttemptCount:  value.AttemptCount,
		NextAttemptAt: corewire.EncodeTime(value.NextAttemptAt),
		LastStatus:    value.LastStatus,
	}
}

func decodeDelivery(wire deliveryWire) (webhook.Delivery, error) {
	id, err := corewire.DecodeWebhookDeliveryID(wire.ID)
	if err != nil {
		return webhook.Delivery{}, err
	}
	cursor, cursorMatched := event.ParseCursor(wire.EventCursor).(event.CursorParsed)
	if !cursorMatched {
		return webhook.Delivery{}, fmt.Errorf("invalid webhook delivery cursor %q", wire.EventCursor)
	}
	state, stateMatched := webhook.ParseDeliveryState(wire.State).(webhook.DeliveryStateAccepted)
	if !stateMatched {
		return webhook.Delivery{}, fmt.Errorf("invalid webhook delivery state %q", wire.State)
	}
	nextAttemptAt, err := corewire.DecodeTime(wire.NextAttemptAt)
	if err != nil {
		return webhook.Delivery{}, err
	}
	return webhook.Delivery{
		ID:            id,
		EventCursor:   cursor.Value,
		State:         state.Value,
		AttemptCount:  wire.AttemptCount,
		NextAttemptAt: nextAttemptAt,
		LastStatus:    wire.LastStatus,
	}, nil
}

// ---- result unions ----

type createResultWire struct {
	Variant string                  `json:"variant"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeCreateResult(result webhook.CreateStoreResult) createResultWire {
	switch typed := result.(type) {
	case webhook.CreateStoreAccepted:
		return createResultWire{Variant: "accepted"}
	case webhook.CreateStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return createResultWire{Variant: "rejected", Error: &reason}
	default:
		return createResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown webhook result %T", result))}
	}
}

func decodeCreateResult(wire createResultWire) (webhook.CreateStoreResult, error) {
	switch wire.Variant {
	case "accepted":
		return webhook.CreateStoreAccepted{}, nil
	case "rejected":
		if wire.Error == nil {
			return nil, fmt.Errorf("rejected create result is missing its error")
		}
		return webhook.CreateStoreRejected{Reason: domainwire.DecodeDomainError(*wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown create result variant %q", wire.Variant)
	}
}

type listResultWire struct {
	Variant       string                  `json:"variant"`
	Subscriptions []subscriptionWire      `json:"subscriptions,omitempty"`
	Error         *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListResult(result webhook.ListStoreResult) listResultWire {
	switch typed := result.(type) {
	case webhook.ListStoreListed:
		values := make([]subscriptionWire, 0, len(typed.Values))
		for index := range typed.Values {
			values = append(values, encodeSubscription(typed.Values[index]))
		}
		return listResultWire{Variant: "listed", Subscriptions: values}
	case webhook.ListStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return listResultWire{Variant: "rejected", Error: &reason}
	default:
		return listResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown webhook result %T", result))}
	}
}

func decodeListResult(wire listResultWire) (webhook.ListStoreResult, error) {
	switch wire.Variant {
	case "listed":
		values := make([]webhook.Subscription, 0, len(wire.Subscriptions))
		for index := range wire.Subscriptions {
			value, err := decodeSubscription(wire.Subscriptions[index])
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return webhook.ListStoreListed{Values: values}, nil
	case "rejected":
		if wire.Error == nil {
			return nil, fmt.Errorf("rejected list result is missing its error")
		}
		return webhook.ListStoreRejected{Reason: domainwire.DecodeDomainError(*wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown list result variant %q", wire.Variant)
	}
}

type revokeResultWire struct {
	Variant      string                  `json:"variant"`
	Subscription *subscriptionWire       `json:"subscription,omitempty"`
	Error        *domainwire.DomainError `json:"error,omitempty"`
}

func encodeRevokeResult(result webhook.RevokeStoreResult) revokeResultWire {
	switch typed := result.(type) {
	case webhook.RevokeStoreRevoked:
		value := encodeSubscription(typed.Value)
		return revokeResultWire{Variant: "revoked", Subscription: &value}
	case webhook.RevokeStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return revokeResultWire{Variant: "rejected", Error: &reason}
	default:
		return revokeResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown webhook result %T", result))}
	}
}

func decodeRevokeResult(wire revokeResultWire) (webhook.RevokeStoreResult, error) {
	switch wire.Variant {
	case "revoked":
		if wire.Subscription == nil {
			return nil, fmt.Errorf("revoked result is missing its subscription")
		}
		value, err := decodeSubscription(*wire.Subscription)
		if err != nil {
			return nil, err
		}
		return webhook.RevokeStoreRevoked{Value: value}, nil
	case "rejected":
		if wire.Error == nil {
			return nil, fmt.Errorf("rejected revoke result is missing its error")
		}
		return webhook.RevokeStoreRejected{Reason: domainwire.DecodeDomainError(*wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown revoke result variant %q", wire.Variant)
	}
}

type deliveriesResultWire struct {
	Variant    string                  `json:"variant"`
	Deliveries []deliveryWire          `json:"deliveries,omitempty"`
	Error      *domainwire.DomainError `json:"error,omitempty"`
}

func encodeDeliveriesResult(result webhook.ListDeliveriesStoreResult) deliveriesResultWire {
	switch typed := result.(type) {
	case webhook.ListDeliveriesStoreListed:
		values := make([]deliveryWire, 0, len(typed.Values))
		for index := range typed.Values {
			values = append(values, encodeDelivery(typed.Values[index]))
		}
		return deliveriesResultWire{Variant: "listed", Deliveries: values}
	case webhook.ListDeliveriesStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return deliveriesResultWire{Variant: "rejected", Error: &reason}
	default:
		return deliveriesResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown webhook result %T", result))}
	}
}

func decodeDeliveriesResult(wire deliveriesResultWire) (webhook.ListDeliveriesStoreResult, error) {
	switch wire.Variant {
	case "listed":
		values := make([]webhook.Delivery, 0, len(wire.Deliveries))
		for index := range wire.Deliveries {
			value, err := decodeDelivery(wire.Deliveries[index])
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return webhook.ListDeliveriesStoreListed{Values: values}, nil
	case "rejected":
		if wire.Error == nil {
			return nil, fmt.Errorf("rejected deliveries result is missing its error")
		}
		return webhook.ListDeliveriesStoreRejected{Reason: domainwire.DecodeDomainError(*wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown deliveries result variant %q", wire.Variant)
	}
}

func rejectionError(message string) *domainwire.DomainError {
	encoded := domainwire.EncodeDomainError(core.NewDomainError(core.ErrorCodeInvalidState, message))
	return &encoded
}

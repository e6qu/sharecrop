package webhook

import (
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
)

// Subscription is one outbound webhook registration: an owner, a receiver
// URL, the event kinds it listens for, and its lifecycle state. The signing
// secret is not part of the model; it is passed alongside at creation and
// only ever surfaces again inside the host-side dispatcher.
type Subscription struct {
	ID        core.WebhookSubscriptionID
	Owner     Owner
	URL       EndpointURL
	Kinds     KindFilter
	State     State
	CreatedAt time.Time
}

// DeliveryState is the lifecycle of one webhook delivery attempt chain.
type DeliveryState struct {
	value string
}

var (
	DeliveryStatePending   = DeliveryState{value: "pending"}
	DeliveryStateDelivered = DeliveryState{value: "delivered"}
	DeliveryStateDead      = DeliveryState{value: "dead"}
)

type DeliveryStateResult interface {
	deliveryStateResult()
}

type DeliveryStateAccepted struct {
	Value DeliveryState
}

type DeliveryStateRejected struct {
	Reason core.DomainError
}

func (DeliveryStateAccepted) deliveryStateResult() {}

func (DeliveryStateRejected) deliveryStateResult() {}

func ParseDeliveryState(raw string) DeliveryStateResult {
	switch raw {
	case DeliveryStatePending.value:
		return DeliveryStateAccepted{Value: DeliveryStatePending}
	case DeliveryStateDelivered.value:
		return DeliveryStateAccepted{Value: DeliveryStateDelivered}
	case DeliveryStateDead.value:
		return DeliveryStateAccepted{Value: DeliveryStateDead}
	default:
		return DeliveryStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "webhook delivery state is invalid")}
	}
}

func (state DeliveryState) String() string {
	return state.value
}

// Delivery is the owner-facing read model of one (subscription, event)
// delivery: where it sits in the stream, how far the retry walk has gone,
// and what the last attempt reported.
type Delivery struct {
	ID            core.WebhookDeliveryID
	EventCursor   event.Cursor
	State         DeliveryState
	AttemptCount  int64
	NextAttemptAt time.Time
	LastStatus    string
}

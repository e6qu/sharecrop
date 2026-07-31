package webhook

import (
	"context"

	"github.com/e6qu/sharecrop/internal/core"
)

// Store persists webhook subscriptions and serves the owner-facing delivery
// read model. It is bridged into the WASI guest, so it carries only the
// management surface; the host-only pump and dispatcher read the stream
// through dedicated struct methods on the concrete db store.
//
// CreateSubscription receives the signing secret alongside the subscription
// and MUST store it as written (never hashed): the dispatcher computes an
// HMAC-SHA256 over each delivery body with the original secret, which is
// impossible from a hash.
type Store interface {
	CreateSubscription(context.Context, Subscription, Secret) CreateStoreResult
	ListSubscriptions(context.Context, Owner, core.Page) ListStoreResult
	RevokeSubscription(context.Context, Owner, core.WebhookSubscriptionID) RevokeStoreResult
	ListDeliveries(context.Context, Owner, core.WebhookSubscriptionID, core.Page) ListDeliveriesStoreResult
}

type CreateStoreResult interface {
	createStoreResult()
}

type CreateStoreAccepted struct{}

type CreateStoreRejected struct {
	Reason core.DomainError
}

func (CreateStoreAccepted) createStoreResult() {}

func (CreateStoreRejected) createStoreResult() {}

type ListStoreResult interface {
	listStoreResult()
}

type ListStoreListed struct {
	Values []Subscription
}

type ListStoreRejected struct {
	Reason core.DomainError
}

func (ListStoreListed) listStoreResult() {}

func (ListStoreRejected) listStoreResult() {}

type RevokeStoreResult interface {
	revokeStoreResult()
}

type RevokeStoreRevoked struct {
	Value Subscription
}

type RevokeStoreRejected struct {
	Reason core.DomainError
}

func (RevokeStoreRevoked) revokeStoreResult() {}

func (RevokeStoreRejected) revokeStoreResult() {}

type ListDeliveriesStoreResult interface {
	listDeliveriesStoreResult()
}

type ListDeliveriesStoreListed struct {
	Values []Delivery
}

type ListDeliveriesStoreRejected struct {
	Reason core.DomainError
}

func (ListDeliveriesStoreListed) listDeliveriesStoreResult() {}

func (ListDeliveriesStoreRejected) listDeliveriesStoreResult() {}

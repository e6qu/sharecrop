package webhook

import (
	"context"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
)

// Service is the webhook subscription management surface. Ownership
// authorization (who may act for a user or an organization) and the
// credential kind-entitlement rule are enforced at the HTTP and MCP layers,
// which see the caller; the service assumes the owner is already authorized.
type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) Service {
	return Service{store: store, now: time.Now}
}

type CreateResult interface {
	createResult()
}

type SubscriptionCreated struct {
	Value Subscription
	// Secret is the signing secret, returned ONLY here. List results never
	// carry it; the create response is the single chance to record it.
	Secret Secret
}

type CreateRejected struct {
	Reason core.DomainError
}

func (SubscriptionCreated) createResult() {}

func (CreateRejected) createResult() {}

func (service Service) Create(ctx context.Context, owner Owner, endpoint EndpointURL, kinds KindFilter) CreateResult {
	idResult := core.NewWebhookSubscriptionID()
	idCreated, idMatched := idResult.(core.WebhookSubscriptionIDCreated)
	if !idMatched {
		return CreateRejected{Reason: idResult.(core.WebhookSubscriptionIDRejected).Reason}
	}

	secretResult := NewSecret()
	secretCreated, secretMatched := secretResult.(SecretAccepted)
	if !secretMatched {
		return CreateRejected{Reason: secretResult.(SecretRejected).Reason}
	}

	subscription := Subscription{
		ID:        idCreated.Value,
		Owner:     owner,
		URL:       endpoint,
		Kinds:     kinds,
		State:     StateActive,
		CreatedAt: service.now().UTC(),
	}

	storeResult := service.store.CreateSubscription(ctx, subscription, secretCreated.Value)
	if rejected, matched := storeResult.(CreateStoreRejected); matched {
		return CreateRejected{Reason: rejected.Reason}
	}
	return SubscriptionCreated{Value: subscription, Secret: secretCreated.Value}
}

type ListResult interface {
	listResult()
}

type SubscriptionsListed struct {
	Values []Subscription
}

type ListRejected struct {
	Reason core.DomainError
}

func (SubscriptionsListed) listResult() {}

func (ListRejected) listResult() {}

func (service Service) List(ctx context.Context, owner Owner, page core.Page) ListResult {
	storeResult := service.store.ListSubscriptions(ctx, owner, page)
	listed, matched := storeResult.(ListStoreListed)
	if !matched {
		return ListRejected{Reason: storeResult.(ListStoreRejected).Reason}
	}
	return SubscriptionsListed{Values: listed.Values}
}

type RevokeResult interface {
	revokeResult()
}

type SubscriptionRevoked struct {
	Value Subscription
}

type RevokeRejected struct {
	Reason core.DomainError
}

func (SubscriptionRevoked) revokeResult() {}

func (RevokeRejected) revokeResult() {}

func (service Service) Revoke(ctx context.Context, owner Owner, id core.WebhookSubscriptionID) RevokeResult {
	storeResult := service.store.RevokeSubscription(ctx, owner, id)
	revoked, matched := storeResult.(RevokeStoreRevoked)
	if !matched {
		return RevokeRejected{Reason: storeResult.(RevokeStoreRejected).Reason}
	}
	return SubscriptionRevoked{Value: revoked.Value}
}

type ListDeliveriesResult interface {
	listDeliveriesResult()
}

type DeliveriesListed struct {
	Values []Delivery
}

type ListDeliveriesRejected struct {
	Reason core.DomainError
}

func (DeliveriesListed) listDeliveriesResult() {}

func (ListDeliveriesRejected) listDeliveriesResult() {}

func (service Service) ListDeliveries(ctx context.Context, owner Owner, id core.WebhookSubscriptionID, page core.Page) ListDeliveriesResult {
	storeResult := service.store.ListDeliveries(ctx, owner, id, page)
	listed, matched := storeResult.(ListDeliveriesStoreListed)
	if !matched {
		return ListDeliveriesRejected{Reason: storeResult.(ListDeliveriesStoreRejected).Reason}
	}
	return DeliveriesListed{Values: listed.Values}
}

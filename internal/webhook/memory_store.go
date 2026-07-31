package webhook

import (
	"context"
	"sync"

	"github.com/e6qu/sharecrop/internal/core"
)

// MemoryStore is the in-memory webhook store used by the default test/demo
// runtime; production uses the Postgres-backed db.WebhookStore. It holds the
// management surface only — deliveries exist solely in the Postgres store,
// where the host-side pump writes them, so ListDeliveries is always empty
// here.
type MemoryStore struct {
	mu            sync.Mutex
	subscriptions []Subscription
	secrets       map[string]Secret
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{subscriptions: []Subscription{}, secrets: map[string]Secret{}}
}

func ownerKey(owner Owner) string {
	switch typed := owner.(type) {
	case OwnerUser:
		return "user:" + typed.ID.String()
	case OwnerOrganization:
		return "organization:" + typed.ID.String()
	default:
		return ""
	}
}

func (store *MemoryStore) CreateSubscription(_ context.Context, subscription Subscription, secret Secret) CreateStoreResult {
	store.mu.Lock()
	defer store.mu.Unlock()
	for _, existing := range store.subscriptions {
		if existing.ID == subscription.ID {
			return CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "webhook subscription already exists")}
		}
	}
	store.subscriptions = append(store.subscriptions, subscription)
	store.secrets[subscription.ID.String()] = secret
	return CreateStoreAccepted{}
}

func (store *MemoryStore) ListSubscriptions(_ context.Context, owner Owner, page core.Page) ListStoreResult {
	store.mu.Lock()
	defer store.mu.Unlock()
	matching := make([]Subscription, 0)
	for _, subscription := range store.subscriptions {
		if ownerKey(subscription.Owner) == ownerKey(owner) {
			matching = append(matching, subscription)
		}
	}
	start := page.Offset()
	if start > len(matching) {
		start = len(matching)
	}
	end := start + page.Limit()
	if end > len(matching) {
		end = len(matching)
	}
	values := make([]Subscription, end-start)
	copy(values, matching[start:end])
	return ListStoreListed{Values: values}
}

func (store *MemoryStore) RevokeSubscription(_ context.Context, owner Owner, id core.WebhookSubscriptionID) RevokeStoreResult {
	store.mu.Lock()
	defer store.mu.Unlock()
	for index, subscription := range store.subscriptions {
		if subscription.ID != id || ownerKey(subscription.Owner) != ownerKey(owner) {
			continue
		}
		if subscription.State != StateActive {
			return RevokeStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "active webhook subscription was not found")}
		}
		store.subscriptions[index].State = StateRevoked
		return RevokeStoreRevoked{Value: store.subscriptions[index]}
	}
	return RevokeStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "active webhook subscription was not found")}
}

func (store *MemoryStore) ListDeliveries(_ context.Context, owner Owner, id core.WebhookSubscriptionID, _ core.Page) ListDeliveriesStoreResult {
	store.mu.Lock()
	defer store.mu.Unlock()
	for _, subscription := range store.subscriptions {
		if subscription.ID == id && ownerKey(subscription.Owner) == ownerKey(owner) {
			return ListDeliveriesStoreListed{Values: []Delivery{}}
		}
	}
	return ListDeliveriesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "webhook subscription was not found")}
}

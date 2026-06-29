package notification

import (
	"context"
	"sync"

	"github.com/e6qu/sharecrop/internal/core"
)

type MemoryStore struct {
	mu     sync.Mutex
	values []Notification
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{values: []Notification{}}
}

func (store *MemoryStore) Create(_ context.Context, value Notification) CreateStoreResult {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.values = append(store.values, value)
	return CreateStoreAccepted{}
}

func (store *MemoryStore) List(_ context.Context, recipient core.UserID, page core.Page) ListStoreResult {
	store.mu.Lock()
	defer store.mu.Unlock()
	matching := make([]Notification, 0)
	for index := len(store.values) - 1; index >= 0; index-- {
		value := store.values[index]
		if value.RecipientID == recipient {
			matching = append(matching, value)
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
	values := make([]Notification, end-start)
	copy(values, matching[start:end])
	return ListStoreAccepted{Values: values}
}

func (store *MemoryStore) MarkRead(_ context.Context, recipient core.UserID, id core.NotificationID) MarkReadStoreResult {
	store.mu.Lock()
	defer store.mu.Unlock()
	for index := range store.values {
		value := store.values[index]
		if value.ID == id && value.RecipientID == recipient {
			store.values[index].State = StateRead
			return MarkReadStoreAccepted{Value: store.values[index]}
		}
	}
	return MarkReadStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "notification was not found")}
}

package event

import (
	"context"
	"sync"

	"github.com/e6qu/sharecrop/internal/core"
)

// MemoryStore is the in-memory event store used by the default test/demo
// runtime; production uses the Postgres-backed db.EventStore.
type MemoryStore struct {
	mu         sync.Mutex
	sequence   int64
	values     []StoredEvent
	recipients map[int64][]core.UserID
	dispatched map[string]DispatchState
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{values: []StoredEvent{}, recipients: map[int64][]core.UserID{}, dispatched: map[string]DispatchState{}}
}

func (store *MemoryStore) Append(_ context.Context, value Event, recipients Recipients) AppendStoreResult {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.sequence++
	stored := WithoutEnrichment(StoredEvent{Event: value, Cursor: CursorFromSequence(store.sequence)})
	store.values = append(store.values, stored)
	users := make([]core.UserID, len(recipients.Users))
	copy(users, recipients.Users)
	store.recipients[store.sequence] = users
	store.dispatched[value.ID.String()] = DispatchStateRecorded
	return AppendStoreAccepted{Value: stored}
}

// Dispatch marks the event dispatched. The in-memory runtime has no webhook
// delivery engine, so the store-side dispatch effect reduces to the state
// flip; dispatching an unknown or already-dispatched event is a no-op
// success, matching the idempotency the Postgres store guarantees.
func (store *MemoryStore) Dispatch(_ context.Context, id core.DomainEventID) DispatchStoreResult {
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, known := store.dispatched[id.String()]; known {
		store.dispatched[id.String()] = DispatchStateDispatched
	}
	return DispatchStoreCompleted{}
}

func (store *MemoryStore) ListForRecipient(_ context.Context, recipient core.UserID, filter CursorFilter, page core.Page) ListStoreResult {
	store.mu.Lock()
	defer store.mu.Unlock()
	afterSequence := int64(0)
	if after, matched := filter.(After); matched {
		afterSequence = after.Cursor.Sequence()
	}
	matching := make([]StoredEvent, 0)
	for _, stored := range store.values {
		if stored.Cursor.Sequence() <= afterSequence {
			continue
		}
		for _, user := range store.recipients[stored.Cursor.Sequence()] {
			if user == recipient {
				matching = append(matching, stored)
				break
			}
		}
	}
	return pageStoredEvents(matching, page)
}

func (store *MemoryStore) ListForOrganization(_ context.Context, organizationID core.OrganizationID, filter CursorFilter, page core.Page) ListStoreResult {
	store.mu.Lock()
	defer store.mu.Unlock()
	afterSequence := int64(0)
	if after, matched := filter.(After); matched {
		afterSequence = after.Cursor.Sequence()
	}
	matching := make([]StoredEvent, 0)
	for _, stored := range store.values {
		if stored.Cursor.Sequence() <= afterSequence {
			continue
		}
		subject, matched := stored.Event.Subject.Organization.(OrganizationSubject)
		if matched && subject.ID == organizationID {
			matching = append(matching, stored)
		}
	}
	return pageStoredEvents(matching, page)
}

// pageStoredEvents applies the limit/offset window to an already-filtered
// event slice and returns a defensive copy.
func pageStoredEvents(matching []StoredEvent, page core.Page) ListStoreResult {
	start := page.Offset()
	if start > len(matching) {
		start = len(matching)
	}
	end := start + page.Limit()
	if end > len(matching) {
		end = len(matching)
	}
	values := make([]StoredEvent, end-start)
	copy(values, matching[start:end])
	return ListStoreAccepted{Values: values}
}

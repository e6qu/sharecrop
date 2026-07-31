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
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{values: []StoredEvent{}, recipients: map[int64][]core.UserID{}}
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
	return AppendStoreAccepted{Value: stored}
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

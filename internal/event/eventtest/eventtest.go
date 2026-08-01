// Package eventtest provides a capturing event store and a recorder over it,
// so service tests can assert which domain events a mutation emitted (kind,
// recipients, subject) without a real event store or inbox.
package eventtest

import (
	"context"
	"sync"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/notification"
)

// CapturingStore records every appended event and its recipient set.
type CapturingStore struct {
	mu         sync.Mutex
	appended   []event.Event
	recipients []event.Recipients
}

func NewCapturingStore() *CapturingStore {
	return &CapturingStore{appended: []event.Event{}, recipients: []event.Recipients{}}
}

func (store *CapturingStore) Append(_ context.Context, value event.Event, recipients event.Recipients) event.AppendStoreResult {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.appended = append(store.appended, value)
	store.recipients = append(store.recipients, recipients)
	return event.AppendStoreAccepted{Value: event.WithoutEnrichment(event.StoredEvent{Event: value, Cursor: event.CursorFromSequence(int64(len(store.appended)))})}
}

func (store *CapturingStore) ListForRecipient(context.Context, core.UserID, event.CursorFilter, core.Page) event.ListStoreResult {
	return event.ListStoreAccepted{Values: []event.StoredEvent{}}
}

// Appended returns a copy of the captured events in emission order.
func (store *CapturingStore) Appended() []event.Event {
	store.mu.Lock()
	defer store.mu.Unlock()
	values := make([]event.Event, len(store.appended))
	copy(values, store.appended)
	return values
}

// RecipientsAt returns the recipient set captured for the index-th event.
func (store *CapturingStore) RecipientsAt(index int) event.Recipients {
	store.mu.Lock()
	defer store.mu.Unlock()
	if index < 0 || index >= len(store.recipients) {
		return event.NewRecipients()
	}
	return store.recipients[index]
}

// RecorderOver builds a recorder over the capturing store; the notification
// fan-out goes to a throwaway in-memory inbox.
func RecorderOver(store *CapturingStore) event.Recorder {
	return event.NewRecorder(store, notification.NewService(notification.NewMemoryStore()))
}

// NewRecorder builds a recorder over a fresh in-memory event store and inbox,
// for tests that need a working recorder but assert nothing about emissions.
func NewRecorder() event.Recorder {
	return event.NewRecorder(event.NewMemoryStore(), notification.NewService(notification.NewMemoryStore()))
}

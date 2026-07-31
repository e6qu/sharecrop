package event

import (
	"context"

	"github.com/e6qu/sharecrop/internal/core"
)

// Store persists events and serves the per-recipient feed. Host-only
// consumers (the webhook pump) read the stream through dedicated struct
// methods on the concrete db store, not through this bridged interface.
type Store interface {
	Append(context.Context, Event, Recipients) AppendStoreResult
	ListForRecipient(context.Context, core.UserID, CursorFilter, core.Page) ListStoreResult
}

type AppendStoreResult interface {
	appendStoreResult()
}

type AppendStoreAccepted struct {
	Value StoredEvent
}

type AppendStoreRejected struct {
	Reason core.DomainError
}

func (AppendStoreAccepted) appendStoreResult() {}

func (AppendStoreRejected) appendStoreResult() {}

type ListStoreResult interface {
	listStoreResult()
}

type ListStoreAccepted struct {
	Values []StoredEvent
}

type ListStoreRejected struct {
	Reason core.DomainError
}

func (ListStoreAccepted) listStoreResult() {}

func (ListStoreRejected) listStoreResult() {}

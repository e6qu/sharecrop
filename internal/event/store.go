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
	// Dispatch performs the store-side dispatch effects for one recorded
	// event — expanding webhook deliveries and moving the row from recorded
	// to dispatched. Inbox fan-out happens in the Recorder before this call.
	// Dispatching an already-dispatched event is a no-op success, so the
	// inline dispatch and the recovery sweep can race safely.
	Dispatch(context.Context, core.DomainEventID) DispatchStoreResult
	ListForRecipient(context.Context, core.UserID, CursorFilter, core.Page) ListStoreResult
	// ListForOrganization serves the cursor feed for an organization: the
	// events whose subject organization is the given one. This is the same
	// visibility rule an organization-owned recipient-audience webhook
	// subscription uses, so the feed an org credential polls matches what a
	// webhook for the same organization would deliver.
	ListForOrganization(context.Context, core.OrganizationID, CursorFilter, core.Page) ListStoreResult
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

type DispatchStoreResult interface {
	dispatchStoreResult()
}

type DispatchStoreCompleted struct{}

type DispatchStoreRejected struct {
	Reason core.DomainError
}

func (DispatchStoreCompleted) dispatchStoreResult() {}

func (DispatchStoreRejected) dispatchStoreResult() {}

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

package event

import (
	"time"

	"github.com/e6qu/sharecrop/internal/core"
)

// Draft is one domain event prepared before the store mutation that records
// it. The emitting service builds the draft (generating the event id up
// front), hands it to the store command, and the store inserts the event row
// plus its recipient set inside the same transaction as the mutation it
// describes, in dispatch state "recorded". After commit the service passes
// the draft to Recorder.Dispatch (inbox fan-out, webhook expansion, marking
// dispatched); the lifecycle runner's dispatch sweep re-runs the same step
// for recorded rows whose inline dispatch was lost to a crash.
type Draft struct {
	ID         core.DomainEventID
	Kind       Kind
	Actor      Actor
	Subject    Subject
	Metadata   Metadata
	Recipients Recipients
}

type DraftResult interface {
	draftResult()
}

type DraftCreated struct {
	Value Draft
}

type DraftRejected struct {
	Reason core.DomainError
}

func (DraftCreated) draftResult() {}

func (DraftRejected) draftResult() {}

// NewDraft builds a draft with a freshly generated event id.
func NewDraft(kind Kind, actor Actor, subject Subject, metadata Metadata, recipients Recipients) DraftResult {
	idResult := core.NewDomainEventID()
	created, matched := idResult.(core.DomainEventIDCreated)
	if !matched {
		return DraftRejected{Reason: idResult.(core.DomainEventIDRejected).Reason}
	}
	return DraftCreated{Value: Draft{
		ID:         created.Value,
		Kind:       kind,
		Actor:      actor,
		Subject:    subject,
		Metadata:   metadata,
		Recipients: recipients,
	}}
}

// WithRecipients returns a copy of the draft whose recipient set is the
// deduplicated union of the existing recipients and the given users. Stores
// use it to merge recipients only they can resolve inside the mutation
// transaction (the reservation holder, the grant beneficiaries) into a
// service-built draft.
func (draft Draft) WithRecipients(users ...core.UserID) Draft {
	merged := append(append([]core.UserID{}, draft.Recipients.Users...), users...)
	draft.Recipients = NewRecipients(merged...)
	return draft
}

// Event returns the event value the store persists for this draft, stamped
// with the given occurrence instant.
func (draft Draft) Event(occurredAt time.Time) Event {
	return Event{
		ID:         draft.ID,
		Kind:       draft.Kind,
		Actor:      draft.Actor,
		Subject:    draft.Subject,
		Metadata:   draft.Metadata,
		OccurredAt: occurredAt.UTC(),
	}
}

// Plan says whether a store mutation records a domain event inside its
// transaction: NoEvent for transitions that emit nothing (unpublish), Record
// for those that do.
type Plan interface {
	plan()
}

type NoEvent struct{}

type Record struct {
	Draft Draft
}

func (NoEvent) plan() {}

func (Record) plan() {}

// DispatchState is the outbox lifecycle of a stored event: recorded (the
// mutation transaction committed the row; side effects still due) or
// dispatched (inbox fan-out and webhook expansion completed). The
// dispatched_at column records the completion instant as a fact; this enum,
// not the timestamp, carries the state.
type DispatchState struct {
	value string
}

var (
	DispatchStateRecorded   = DispatchState{value: "recorded"}
	DispatchStateDispatched = DispatchState{value: "dispatched"}
)

func (state DispatchState) String() string {
	return state.value
}

type DispatchStateResult interface {
	dispatchStateResult()
}

type DispatchStateParsed struct {
	Value DispatchState
}

type DispatchStateRejected struct {
	Reason core.DomainError
}

func (DispatchStateParsed) dispatchStateResult() {}

func (DispatchStateRejected) dispatchStateResult() {}

// ParseDispatchState converts a raw boundary string into a sealed
// DispatchState.
func ParseDispatchState(raw string) DispatchStateResult {
	switch raw {
	case DispatchStateRecorded.value:
		return DispatchStateParsed{Value: DispatchStateRecorded}
	case DispatchStateDispatched.value:
		return DispatchStateParsed{Value: DispatchStateDispatched}
	default:
		return DispatchStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "unknown event dispatch state")}
	}
}

package event

import (
	"strconv"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

// Actor is who caused the event: a user (resolved from a session, agent
// credential, or organization credential) or the platform itself (background
// sweeps such as reservation expiry).
type Actor interface {
	actor()
}

type ActorUser struct {
	ID core.UserID
}

type ActorSystem struct{}

func (ActorUser) actor() {}

func (ActorSystem) actor() {}

// ActorUserID resolves the acting user identity for stores and notifications:
// the user for ActorUser, the durable system actor for ActorSystem.
func ActorUserID(actor Actor) core.UserID {
	if user, matched := actor.(ActorUser); matched {
		return user.ID
	}
	return core.SystemUserID()
}

// Subject names what the event is about. Each reference is an explicit
// present/absent sum; an event carries the references that apply to it (a
// submission event carries its task and submission, a series comment only its
// series).
type Subject struct {
	Task         TaskRef
	Submission   SubmissionRef
	Reservation  ReservationRef
	Series       SeriesRef
	Organization OrganizationRef
	Collectible  CollectibleRef
}

// NoSubjectRefs returns a Subject with every reference absent; emitters set
// the ones that apply.
func NoSubjectRefs() Subject {
	return Subject{
		Task:         NoTask{},
		Submission:   NoSubmission{},
		Reservation:  NoReservation{},
		Series:       NoSeries{},
		Organization: NoOrganization{},
		Collectible:  NoCollectible{},
	}
}

type TaskRef interface {
	taskRef()
}

type NoTask struct{}

type TaskSubject struct {
	ID core.TaskID
}

func (NoTask) taskRef() {}

func (TaskSubject) taskRef() {}

type SubmissionRef interface {
	submissionRef()
}

type NoSubmission struct{}

type SubmissionSubject struct {
	ID core.SubmissionID
}

func (NoSubmission) submissionRef() {}

func (SubmissionSubject) submissionRef() {}

type ReservationRef interface {
	reservationRef()
}

type NoReservation struct{}

type ReservationSubject struct {
	ID core.TaskReservationID
}

func (NoReservation) reservationRef() {}

func (ReservationSubject) reservationRef() {}

type SeriesRef interface {
	seriesRef()
}

type NoSeries struct{}

type SeriesSubject struct {
	ID core.TaskSeriesID
}

func (NoSeries) seriesRef() {}

func (SeriesSubject) seriesRef() {}

type OrganizationRef interface {
	organizationRef()
}

type NoOrganization struct{}

type OrganizationSubject struct {
	ID core.OrganizationID
}

func (NoOrganization) organizationRef() {}

func (OrganizationSubject) organizationRef() {}

type CollectibleRef interface {
	collectibleRef()
}

type NoCollectible struct{}

type CollectibleSubject struct {
	ID core.CollectibleID
}

func (NoCollectible) collectibleRef() {}

func (CollectibleSubject) collectibleRef() {}

// Metadata is a bounded JSON payload with event-kind-specific details (amounts,
// note excerpts, collectible names). It is display data, never authorization
// data.
type Metadata struct {
	JSON string
}

func EmptyMetadata() Metadata {
	return Metadata{JSON: "{}"}
}

// Event is one emitted domain event before the store assigns its cursor.
type Event struct {
	ID         core.DomainEventID
	Kind       Kind
	Actor      Actor
	Subject    Subject
	Metadata   Metadata
	OccurredAt time.Time
}

// StoredEvent is an event with its store-assigned cursor position, plus two
// read-time enrichment references the feed read (ListForRecipient) resolves:
// the acting user's display name and, when the event references a task, that
// task's title. Append results and host-side pump reads carry the absent
// variants.
type StoredEvent struct {
	Event     Event
	Cursor    Cursor
	ActorName ActorNameRef
	TaskTitle TaskTitleRef
}

// ActorNameRef is the acting user's display name on an enriched feed row.
type ActorNameRef interface {
	actorNameRef()
}

type NoActorName struct{}

type ActorNamed struct {
	DisplayName auth.DisplayName
}

func (NoActorName) actorNameRef() {}

func (ActorNamed) actorNameRef() {}

// TaskTitleRef is the referenced task's title on an enriched feed row.
type TaskTitleRef interface {
	taskTitleRef()
}

type NoTaskTitle struct{}

// TaskTitled carries a raw display copy of the referenced task's title.
type TaskTitled struct {
	Title string
}

func (NoTaskTitle) taskTitleRef() {}

func (TaskTitled) taskTitleRef() {}

// WithoutEnrichment stamps the absent enrichment variants onto a stored
// event, for construction sites that do not resolve names/titles (append
// results, pump reads, in-memory stores).
func WithoutEnrichment(value StoredEvent) StoredEvent {
	value.ActorName = NoActorName{}
	value.TaskTitle = NoTaskTitle{}
	return value
}

// Cursor is an opaque position in the event stream. Consumers treat it as a
// token: list events after a cursor, remember the last cursor seen.
type Cursor struct {
	value int64
}

// CursorFromSequence wraps a store-assigned sequence number. Only stores
// construct cursors from raw integers.
func CursorFromSequence(sequence int64) Cursor {
	return Cursor{value: sequence}
}

// Sequence exposes the underlying position for store queries.
func (cursor Cursor) Sequence() int64 {
	return cursor.value
}

func (cursor Cursor) String() string {
	return strconv.FormatInt(cursor.value, 10)
}

type CursorResult interface {
	cursorResult()
}

type CursorParsed struct {
	Value Cursor
}

type CursorRejected struct {
	Reason core.DomainError
}

func (CursorParsed) cursorResult() {}

func (CursorRejected) cursorResult() {}

// ParseCursor converts a raw boundary string into a cursor.
func ParseCursor(raw string) CursorResult {
	sequence, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || sequence < 0 {
		return CursorRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "event cursor must be a non-negative integer")}
	}
	return CursorParsed{Value: Cursor{value: sequence}}
}

// CursorFilter selects where a listing starts: from the beginning of the
// stream or strictly after a cursor.
type CursorFilter interface {
	cursorFilter()
}

type FromStart struct{}

type After struct {
	Cursor Cursor
}

func (FromStart) cursorFilter() {}

func (After) cursorFilter() {}

// Recipients are the users an event is visible to in their personal feed. The
// emitting service computes them (owner, assignee, submitter...). The actor
// belongs in the list so their own feed reflects their actions.
type Recipients struct {
	Users []core.UserID
}

// NewRecipients deduplicates the given users into a recipient set.
func NewRecipients(users ...core.UserID) Recipients {
	seen := map[string]bool{}
	deduplicated := make([]core.UserID, 0, len(users))
	for _, user := range users {
		if seen[user.String()] {
			continue
		}
		seen[user.String()] = true
		deduplicated = append(deduplicated, user)
	}
	return Recipients{Users: deduplicated}
}

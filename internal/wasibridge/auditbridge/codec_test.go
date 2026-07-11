package auditbridge

import (
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/audit/audittest"
	"github.com/e6qu/sharecrop/internal/core"
)

func sampleEvent(t *testing.T) audit.Event {
	t.Helper()
	id, matched := core.NewAuditEventID().(core.AuditEventIDCreated)
	if !matched {
		t.Fatalf("audit event id rejected")
	}
	actor, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("user id rejected")
	}
	return audit.Event{
		ID:          id.Value,
		ActorUserID: actor.Value,
		Action:      audit.ActionOrganizationCreated,
		Subject:     audit.Subject{Kind: "organization", ID: "org-123"},
		Metadata:    audit.Metadata{JSON: `{"reason":"created"}`},
		CreatedAt:   time.Date(2026, 7, 11, 12, 30, 0, 0, time.UTC),
	}
}

func assertEventEqual(t *testing.T, got audit.Event, want audit.Event) {
	t.Helper()
	if diff := audittest.EventDiff(got, want); diff != "" {
		t.Errorf("event mismatch: %s", diff)
	}
}

func TestEventRoundTrip(t *testing.T) {
	original := sampleEvent(t)
	restored, err := decodeEvent(encodeEvent(original))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	assertEventEqual(t, restored, original)
}

func TestRecordResultRoundTrip(t *testing.T) {
	event := sampleEvent(t)

	recorded, err := decodeRecordResult(encodeRecordResult(audit.EventRecorded{Value: event}))
	if err != nil {
		t.Fatalf("decode recorded: %v", err)
	}
	if typed, matched := recorded.(audit.EventRecorded); !matched {
		t.Fatalf("recorded result = %T", recorded)
	} else {
		assertEventEqual(t, typed.Value, event)
	}

	rejected, err := decodeRecordResult(encodeRecordResult(audit.RecordRejected{
		Reason: core.NewDomainError(core.ErrorCodeConflict, "duplicate"),
	}))
	if err != nil {
		t.Fatalf("decode rejected: %v", err)
	}
	typed, matched := rejected.(audit.RecordRejected)
	if !matched {
		t.Fatalf("rejected result = %T", rejected)
	}
	if typed.Reason.Code() != core.ErrorCodeConflict || typed.Reason.Description() != "duplicate" {
		t.Errorf("domain error not preserved: %s / %q", typed.Reason.Code(), typed.Reason.Description())
	}
}

func TestGetResultRoundTrip(t *testing.T) {
	event := sampleEvent(t)

	found, err := decodeGetResult(encodeGetResult(audit.EventFound{Value: event}))
	if err != nil {
		t.Fatalf("decode found: %v", err)
	}
	if typed, matched := found.(audit.EventFound); !matched {
		t.Fatalf("found result = %T", found)
	} else {
		assertEventEqual(t, typed.Value, event)
	}

	rejected, err := decodeGetResult(encodeGetResult(audit.GetRejected{
		Reason: core.NewDomainError(core.ErrorCodeNotFound, "missing"),
	}))
	if err != nil {
		t.Fatalf("decode rejected: %v", err)
	}
	if typed, matched := rejected.(audit.GetRejected); !matched || typed.Reason.Code() != core.ErrorCodeNotFound {
		t.Errorf("get rejection not preserved: %T", rejected)
	}
}

func TestListResultRoundTrip(t *testing.T) {
	first := sampleEvent(t)
	second := sampleEvent(t)

	listed, err := decodeListResult(encodeListResult(audit.EventsListed{Values: []audit.Event{first, second}}))
	if err != nil {
		t.Fatalf("decode listed: %v", err)
	}
	typed, matched := listed.(audit.EventsListed)
	if !matched {
		t.Fatalf("listed result = %T", listed)
	}
	if len(typed.Values) != 2 {
		t.Fatalf("listed %d events, want 2", len(typed.Values))
	}
	assertEventEqual(t, typed.Values[0], first)
	assertEventEqual(t, typed.Values[1], second)

	empty, err := decodeListResult(encodeListResult(audit.EventsListed{Values: []audit.Event{}}))
	if err != nil {
		t.Fatalf("decode empty listed: %v", err)
	}
	if typed, matched := empty.(audit.EventsListed); !matched || len(typed.Values) != 0 {
		t.Errorf("empty listing did not round-trip: %T", empty)
	}
}

func TestListFiltersRoundTrip(t *testing.T) {
	filters := audit.ListFilters{
		Action:      audit.ActionEquals{Value: audit.ActionTaskFunded},
		SubjectKind: audit.SubjectKindEquals{Value: "task"},
		SubjectID:   audit.AnySubjectID{},
	}
	restored, err := decodeListFilters(encodeListFilters(filters))
	if err != nil {
		t.Fatalf("decode filters: %v", err)
	}
	action, matched := restored.Action.(audit.ActionEquals)
	if !matched || action.Value.String() != audit.ActionTaskFunded.String() {
		t.Errorf("action filter not preserved: %+v", restored.Action)
	}
	kind, matched := restored.SubjectKind.(audit.SubjectKindEquals)
	if !matched || kind.Value != "task" {
		t.Errorf("subject kind filter not preserved: %+v", restored.SubjectKind)
	}
	if _, matched := restored.SubjectID.(audit.AnySubjectID); !matched {
		t.Errorf("subject id filter not preserved: %+v", restored.SubjectID)
	}

	// The no-filter default round-trips to all three "unfiltered" variants.
	none, err := decodeListFilters(encodeListFilters(audit.NoListFilters()))
	if err != nil {
		t.Fatalf("decode no-filters: %v", err)
	}
	_, actionAny := none.Action.(audit.AnyAction)
	_, kindAny := none.SubjectKind.(audit.AnySubjectKind)
	_, idAny := none.SubjectID.(audit.AnySubjectID)
	if !actionAny || !kindAny || !idAny {
		t.Errorf("no-filter default did not round-trip: %+v", none)
	}
}

func TestDecodeFilterError(t *testing.T) {
	if _, err := decodeActionFilter(actionFilterWire{Variant: "bogus"}); err == nil {
		t.Errorf("decodeActionFilter accepted a bad variant")
	}
}

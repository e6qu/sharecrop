package httpserver

import (
	"context"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
)

func newModerationReportEvent(t *testing.T, actor core.UserID, createdAt time.Time) audit.Event {
	t.Helper()
	idResult := core.NewAuditEventID()
	created, matched := idResult.(core.AuditEventIDCreated)
	if !matched {
		t.Fatalf("new audit event id failed: %#v", idResult)
	}
	return audit.Event{
		ID:          created.Value,
		ActorUserID: actor,
		Action:      audit.ActionModerationReportCreated,
		Subject:     audit.Subject{Kind: "task", ID: "task-" + created.Value.String()},
		Metadata:    audit.Metadata{JSON: `{"reason":"policy","details":""}`},
		CreatedAt:   createdAt,
	}
}

func triagePageForTest(t *testing.T, limit int, offset int) core.Page {
	t.Helper()
	result := core.NewPage(limit, offset)
	accepted, matched := result.(core.PageAccepted)
	if !matched {
		t.Fatalf("new page failed: %#v", result)
	}
	return accepted.Value
}

func requireTriageListedForTest(t *testing.T, result ModerationTriageListResult) []ModerationTriageRecord {
	t.Helper()
	listed, matched := result.(ModerationTriageListed)
	if !matched {
		t.Fatalf("list result = %#v, want listed", result)
	}
	return listed.Values
}

// TestModerationTriageListFiltersBeforePagination pins the pushed-down state
// filter: a page of open reports must be full whenever enough open reports
// exist, even when the newest reports (the earlier pages of the unfiltered
// listing) are all resolved. The old handler filtered a fetched page in
// memory, so those pages came back short or empty.
func TestModerationTriageListFiltersBeforePagination(t *testing.T) {
	ctx := context.Background()
	service := newMemoryModerationTriageService()
	actor := testUserIDForPrivacy(t)

	base := time.Date(2026, 1, 2, 3, 0, 0, 0, time.UTC)
	openIDs := map[string]bool{}
	resolvedIDs := map[string]bool{}
	// 20 older reports stay open; the 10 newest are resolved, so the first
	// unfiltered page is dominated by resolved reports.
	for index := 0; index < 30; index++ {
		event := newModerationReportEvent(t, actor, base.Add(time.Duration(index)*time.Minute))
		if _, matched := service.RecordOpen(ctx, event).(ModerationTriageSaved); !matched {
			t.Fatalf("record open failed for report %d", index)
		}
		if index < 20 {
			openIDs[event.ID.String()] = true
			continue
		}
		if _, matched := service.Update(ctx, actor, event.ID, ModerationTriageStateResolved, "handled").(ModerationTriageSaved); !matched {
			t.Fatalf("resolve failed for report %d", index)
		}
		resolvedIDs[event.ID.String()] = true
	}

	firstOpenPage := requireTriageListedForTest(t, service.List(ctx, TriageStateEquals{State: ModerationTriageStateOpen}, triagePageForTest(t, 10, 0)))
	if len(firstOpenPage) != 10 {
		t.Fatalf("first open page = %d records, want a full page of 10", len(firstOpenPage))
	}
	seen := map[string]bool{}
	for _, record := range firstOpenPage {
		if record.State != ModerationTriageStateOpen.String() {
			t.Fatalf("open-filtered page contains state %q", record.State)
		}
		if !openIDs[record.ReportID.String()] {
			t.Fatalf("open-filtered page contains unexpected report %s", record.ReportID)
		}
		seen[record.ReportID.String()] = true
	}

	secondOpenPage := requireTriageListedForTest(t, service.List(ctx, TriageStateEquals{State: ModerationTriageStateOpen}, triagePageForTest(t, 10, 10)))
	if len(secondOpenPage) != 10 {
		t.Fatalf("second open page = %d records, want the remaining 10", len(secondOpenPage))
	}
	for _, record := range secondOpenPage {
		if seen[record.ReportID.String()] {
			t.Fatalf("second open page repeats report %s", record.ReportID)
		}
		if !openIDs[record.ReportID.String()] {
			t.Fatalf("second open page contains unexpected report %s", record.ReportID)
		}
	}

	resolvedPage := requireTriageListedForTest(t, service.List(ctx, TriageStateEquals{State: ModerationTriageStateResolved}, triagePageForTest(t, 20, 0)))
	if len(resolvedPage) != 10 {
		t.Fatalf("resolved page = %d records, want 10", len(resolvedPage))
	}
	for _, record := range resolvedPage {
		if !resolvedIDs[record.ReportID.String()] {
			t.Fatalf("resolved page contains unexpected report %s", record.ReportID)
		}
	}

	everything := requireTriageListedForTest(t, service.List(ctx, AnyTriageState{}, triagePageForTest(t, 200, 0)))
	if len(everything) != 30 {
		t.Fatalf("unfiltered listing = %d records, want 30", len(everything))
	}
	for index := 1; index < len(everything); index++ {
		if everything[index].CreatedAt.After(everything[index-1].CreatedAt) {
			t.Fatalf("unfiltered listing is not newest-first at index %d", index)
		}
	}
}

func TestParseModerationTriageState(t *testing.T) {
	for _, valid := range []string{"open", "resolved", "dismissed"} {
		accepted, matched := ParseModerationTriageState(valid).(ModerationTriageStateAccepted)
		if !matched {
			t.Fatalf("state %q was rejected", valid)
		}
		if accepted.Value.String() != valid {
			t.Fatalf("state %q round-tripped to %q", valid, accepted.Value.String())
		}
	}
	for _, invalid := range []string{"", "deleted", "OPEN", "closed"} {
		if _, matched := ParseModerationTriageState(invalid).(ModerationTriageStateRejected); !matched {
			t.Fatalf("state %q was not rejected", invalid)
		}
	}
}

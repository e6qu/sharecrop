package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
)

// TestListTasksRowsCarryEnrichmentAndTotal pins the stage-3 list_tasks row
// shape: every row carries the creator's display name, the active
// reservation holder's display name, the funded enum, and the
// pending-review count, and the payload carries the filter-wide total.
func TestListTasksRowsCarryEnrichmentAndTotal(t *testing.T) {
	server := NewServer(fakeServices{listTaskCount: 2})
	content := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.list_tasks","arguments":{"scope":"public"}}`)))
	var payload struct {
		Tasks []struct {
			ID                 string `json:"id"`
			CreatorDisplayName string `json:"creator_display_name"`
			HolderDisplayName  string `json:"holder_display_name"`
			Funded             string `json:"funded"`
			PendingReviewCount int64  `json:"pending_review_count"`
		} `json:"tasks"`
		NextOffset int   `json:"next_offset"`
		Total      int64 `json:"total"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("decode tasks payload: %v (%s)", err, content)
	}
	if len(payload.Tasks) != 2 || payload.Total != 2 {
		t.Fatalf("tasks = %d total = %d, want 2 and 2", len(payload.Tasks), payload.Total)
	}
	row := payload.Tasks[0]
	if row.CreatorDisplayName != "mara" || row.HolderDisplayName != "ada" || row.Funded != "reward_funded" || row.PendingReviewCount != 2 {
		t.Fatalf("row enrichment wrong: %+v", row)
	}
}

// TestListTasksFundedFilterThreads pins the funded input filter: a valid
// enum value threads through as task.FundedEquals and garbage is rejected.
func TestListTasksFundedFilterThreads(t *testing.T) {
	services := newCapturingServices()
	server := NewServer(services)
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.list_tasks","arguments":{"scope":"public","funded":"reward_unfunded"}}`))
	if response.Error != nil {
		t.Fatalf("list_tasks funded filter error: %s", response.Error.Message)
	}
	equals, matched := services.listFilters.Funded.(task.FundedEquals)
	if !matched || equals.Value.String() != "reward_unfunded" {
		t.Fatalf("funded filter = %#v, want FundedEquals reward_unfunded", services.listFilters.Funded)
	}

	invalid := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`2`, "tools/call", `{"name":"sharecrop.list_tasks","arguments":{"scope":"public","funded":"maybe"}}`))
	if invalid.Error == nil || invalid.Error.Code != codeInvalidParams {
		t.Fatalf("expected invalid-params for an unknown funded value, got %+v", invalid.Error)
	}
}

// TestListToolsCarryTotal pins the total field on every MCP list tool whose
// REST counterpart carries Total.
func TestListToolsCarryTotal(t *testing.T) {
	server := NewServer(fakeServices{isAdmin: true, listTaskCount: 1})
	userID := testSubject(t).ID.String()
	subscriptionID := core.NewWebhookSubscriptionID().(core.WebhookSubscriptionIDCreated).Value.String()
	for _, call := range []string{
		`{"name":"sharecrop.list_tasks","arguments":{"scope":"public"}}`,
		`{"name":"sharecrop.list_notifications","arguments":{}}`,
		`{"name":"sharecrop.list_task_submissions","arguments":{"task_id":"` + testTaskID(t) + `"}}`,
		`{"name":"sharecrop.get_user_submissions","arguments":{"user_id":"` + userID + `"}}`,
		`{"name":"sharecrop.list_ledger","arguments":{}}`,
		`{"name":"sharecrop.list_webhook_deliveries","arguments":{"subscription_id":"` + subscriptionID + `"}}`,
		`{"name":"sharecrop.list_admin_moderation_reports","arguments":{}}`,
	} {
		content := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: agent.NewScopeSet(agent.AllScopes())}, request(`1`, "tools/call", call)))
		if !strings.Contains(content, `"total"`) {
			t.Fatalf("list result missing total for %s: %s", call, content)
		}
	}
}

// TestCreateModerationReportDisputeArguments pins the dispute filing shape:
// subject_kind submission plus reason dispute pass through to the service
// unchanged, and the result echoes the category for the reporter.
func TestCreateModerationReportDisputeArguments(t *testing.T) {
	server := NewServer(fakeServices{})
	submissionID := "sub_0123456789abcdef0123456789abcdef"
	content := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.create_moderation_report","arguments":{"subject_kind":"submission","subject_id":"`+submissionID+`","reason":"dispute","details":"The rejection cited a requirement the task never stated."}}`)))
	if !strings.Contains(content, `"reason":"dispute"`) || !strings.Contains(content, `"subject_kind":"submission"`) || !strings.Contains(content, submissionID) {
		t.Fatalf("dispute report payload wrong: %s", content)
	}
}

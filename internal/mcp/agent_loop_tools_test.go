package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
)

// TestSubmitResponseReturnsValidationErrorsAndResubmitGuidance pins the
// invalid-submission contract: the tool result carries the validation_errors
// array (path + message) and guidance that the reservation is kept so the
// agent can resubmit immediately.
func TestSubmitResponseReturnsValidationErrorsAndResubmitGuidance(t *testing.T) {
	server := NewServer(fakeServices{invalidSubmission: true})
	content := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.submit_response","arguments":{"task_id":"`+testTaskID(t)+`","response_json":"{}"}}`)))

	var payload struct {
		State            string `json:"state"`
		ValidationErrors []struct {
			Path    string `json:"path"`
			Message string `json:"message"`
		} `json:"validation_errors"`
		Guidance string `json:"guidance"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("decode submit payload: %v", err)
	}
	if payload.State != "invalid" {
		t.Fatalf("state = %q, want invalid", payload.State)
	}
	if len(payload.ValidationErrors) != 1 || payload.ValidationErrors[0].Path != "$.answer" || payload.ValidationErrors[0].Message == "" {
		t.Fatalf("validation errors = %+v", payload.ValidationErrors)
	}
	if !strings.Contains(payload.Guidance, "reservation is kept") || !strings.Contains(payload.Guidance, "immediately") {
		t.Fatalf("guidance = %q", payload.Guidance)
	}
}

// TestSubmitResponseValidStateHasEmptyErrorsAndNoGuidance pins the happy
// path: an empty validation_errors array (present, not absent) and no
// guidance text.
func TestSubmitResponseValidStateHasEmptyErrorsAndNoGuidance(t *testing.T) {
	server := NewServer(fakeServices{})
	content := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.submit_response","arguments":{"task_id":"`+testTaskID(t)+`","response_json":"{\"answer\":\"done\"}"}}`)))
	if !strings.Contains(content, `"validation_errors":[]`) {
		t.Fatalf("valid submit missing empty validation_errors array: %s", content)
	}
	if strings.Contains(content, `"guidance"`) {
		t.Fatalf("valid submit should carry no guidance: %s", content)
	}
}

// TestSubmitResponseForwardsAttachments pins that the attachments argument
// reaches the submit command (the fake echoes them back).
func TestSubmitResponseForwardsAttachments(t *testing.T) {
	server := NewServer(fakeServices{})
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.submit_response","arguments":{"task_id":"`+testTaskID(t)+`","response_json":"{}","attachments":[{"name":"notes.txt","content_type":"text/plain","data_url":"data:text/plain;base64,aGVsbG8="}]}}`))
	if response.Error != nil {
		t.Fatalf("submit with attachments error: %s", response.Error.Message)
	}

	invalid := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`2`, "tools/call", `{"name":"sharecrop.submit_response","arguments":{"task_id":"`+testTaskID(t)+`","response_json":"{}","attachments":[{"name":"notes.txt","content_type":"text/plain","data_url":"not-a-data-url"}]}}`))
	if invalid.Error == nil || invalid.Error.Code != codeInvalidParams {
		t.Fatalf("expected invalid-params for a malformed attachment, got %+v", invalid.Error)
	}
}

// TestGetSubmissionReturnsFullContent pins the reviewer deep read: full
// response_json plus submitter identity, validation errors, attachments,
// and created_at.
func TestGetSubmissionReturnsFullContent(t *testing.T) {
	server := NewServer(fakeServices{})
	submissionID := core.NewSubmissionID().(core.SubmissionIDCreated).Value
	content := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.get_submission","arguments":{"submission_id":"`+submissionID.String()+`"}}`)))

	var payload struct {
		ID                   string          `json:"id"`
		TaskID               string          `json:"task_id"`
		SubmitterID          string          `json:"submitter_id"`
		SubmitterDisplayName string          `json:"submitter_display_name"`
		State                string          `json:"state"`
		ResponseJSON         string          `json:"response_json"`
		ReviewNote           string          `json:"review_note"`
		Attachments          json.RawMessage `json:"attachments"`
		ValidationErrors     json.RawMessage `json:"validation_errors"`
		CreatedAt            string          `json:"created_at"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("decode submission detail: %v", err)
	}
	if payload.ID != submissionID.String() {
		t.Fatalf("id = %q, want %q", payload.ID, submissionID.String())
	}
	if payload.ResponseJSON != `{"answer":"done"}` {
		t.Fatalf("response_json = %q", payload.ResponseJSON)
	}
	if payload.SubmitterDisplayName != "ada" {
		t.Fatalf("submitter_display_name = %q", payload.SubmitterDisplayName)
	}
	if payload.State != "submitted" {
		t.Fatalf("state = %q", payload.State)
	}
	if payload.CreatedAt != "2026-06-29T00:00:00Z" {
		t.Fatalf("created_at = %q", payload.CreatedAt)
	}
	if string(payload.Attachments) != "[]" || string(payload.ValidationErrors) != "[]" {
		t.Fatalf("attachments/validation_errors = %s / %s", payload.Attachments, payload.ValidationErrors)
	}
}

// TestGetSubmissionRequiresSubmissionsReadScope pins the tool's scope gate.
func TestGetSubmissionRequiresSubmissionsReadScope(t *testing.T) {
	server := NewServer(fakeServices{})
	denied := CallerCredential{Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead})}
	submissionID := core.NewSubmissionID().(core.SubmissionIDCreated).Value
	response := server.Handle(context.Background(), testSubject(t), denied, request(`1`, "tools/call", `{"name":"sharecrop.get_submission","arguments":{"submission_id":"`+submissionID.String()+`"}}`))
	if response.Error == nil || response.Error.Code != codeScopeDenied {
		t.Fatalf("expected scope-denied without submissions_read, got %+v", response.Error)
	}
}

// TestGrantCreditsRequiresLivePlatformAdmin pins that the scope alone is not
// enough: the underlying user must be a platform admin right now.
func TestGrantCreditsRequiresLivePlatformAdmin(t *testing.T) {
	scopes := agent.NewScopeSet([]agent.Scope{agent.ScopePlatformAdmin})
	userID := core.NewUserID().(core.UserIDCreated).Value
	arguments := `{"name":"sharecrop.grant_credits","arguments":{"target_kind":"user","target_id":"` + userID.String() + `","amount":40,"note":"Reimbursed the cancelled batch.","idempotency_key":"grant-1"}}`

	admin := NewServer(fakeServices{isAdmin: true})
	content := decodeToolText(t, admin.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: scopes}, request(`1`, "tools/call", arguments)))
	var payload struct {
		EntryID string `json:"entry_id"`
		Amount  int64  `json:"amount"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("decode grant payload: %v", err)
	}
	if payload.EntryID == "" || payload.Amount != 40 {
		t.Fatalf("grant payload = %+v", payload)
	}

	demoted := NewServer(fakeServices{isAdmin: false})
	response := demoted.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: scopes}, request(`2`, "tools/call", arguments))
	var result toolCallResult
	if err := json.Unmarshal(mustResult(t, response), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if !result.IsError || !strings.Contains(result.Content[0].Text, "platform admin access is required") {
		t.Fatalf("expected a live admin re-check failure, got %s", result.Content[0].Text)
	}

	// Invalid target kinds fail with a clear enum message.
	badTarget := admin.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: scopes}, request(`3`, "tools/call", `{"name":"sharecrop.grant_credits","arguments":{"target_kind":"team","target_id":"`+userID.String()+`","amount":40,"note":"n","idempotency_key":"grant-2"}}`))
	if badTarget.Error == nil || !strings.Contains(badTarget.Error.Message, "target_kind must be user or organization") {
		t.Fatalf("expected a target_kind rejection, got %+v", badTarget.Error)
	}
}

// TestCreateWebhookSubscriptionMarketplaceAudience pins the audience and
// filter arguments: they thread through to the webhook service and echo back
// in the created subscription summary.
func TestCreateWebhookSubscriptionMarketplaceAudience(t *testing.T) {
	server := NewServer(fakeServices{})
	scopes := agent.NewScopeSet([]agent.Scope{agent.ScopeWebhooksManage, agent.ScopeWebhooksRead, agent.ScopeTasksRead})
	content := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: scopes}, request(`1`, "tools/call", `{"name":"sharecrop.create_webhook_subscription","arguments":{"url":"https://receiver.example.com/hooks","kinds":["task_opened"],"audience":"marketplace","filter_task_type":"code_review","filter_min_credit_reward":25}}`)))
	var payload struct {
		Subscription struct {
			Audience              string `json:"audience"`
			FilterTaskType        string `json:"filter_task_type"`
			FilterMinCreditReward int64  `json:"filter_min_credit_reward"`
		} `json:"subscription"`
		Secret string `json:"secret"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("decode webhook payload: %v", err)
	}
	if payload.Subscription.Audience != "marketplace" || payload.Subscription.FilterTaskType != "code_review" || payload.Subscription.FilterMinCreditReward != 25 {
		t.Fatalf("subscription = %+v", payload.Subscription)
	}
	if payload.Secret == "" {
		t.Fatalf("create response missing the one-time secret")
	}

	// A marketplace subscription may listen only for task_opened; the
	// service rule surfaces as a tool-level failure.
	wrongKinds := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: agent.NewScopeSet(agent.AllScopes())}, request(`2`, "tools/call", `{"name":"sharecrop.create_webhook_subscription","arguments":{"url":"https://receiver.example.com/hooks","kinds":["submission_created"],"audience":"marketplace"}}`))
	var wrongResult toolCallResult
	if err := json.Unmarshal(mustResult(t, wrongKinds), &wrongResult); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if !wrongResult.IsError || !strings.Contains(wrongResult.Content[0].Text, "task_opened") {
		t.Fatalf("expected the marketplace kinds rule, got %s", wrongResult.Content[0].Text)
	}

	// Filters without the marketplace audience are rejected up front.
	recipientFilters := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: scopes}, request(`3`, "tools/call", `{"name":"sharecrop.create_webhook_subscription","arguments":{"url":"https://receiver.example.com/hooks","kinds":["task_opened"],"filter_task_type":"code_review"}}`))
	if recipientFilters.Error == nil || !strings.Contains(recipientFilters.Error.Message, "marketplace audience") {
		t.Fatalf("expected a filters-require-marketplace rejection, got %+v", recipientFilters.Error)
	}
}

// TestListToolsReportNextOffset pins the shared pagination contract: a full
// window reports next_offset = offset + limit; a short window reports 0.
func TestListToolsReportNextOffset(t *testing.T) {
	server := NewServer(fakeServices{listTaskCount: 5})
	paged := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.list_tasks","arguments":{"scope":"public","limit":2}}`)))
	var payload struct {
		Tasks      []json.RawMessage `json:"tasks"`
		NextOffset int               `json:"next_offset"`
	}
	if err := json.Unmarshal([]byte(paged), &payload); err != nil {
		t.Fatalf("decode tasks payload: %v", err)
	}
	if len(payload.Tasks) != 2 || payload.NextOffset != 2 {
		t.Fatalf("tasks = %d next_offset = %d, want 2 and 2", len(payload.Tasks), payload.NextOffset)
	}

	last := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`2`, "tools/call", `{"name":"sharecrop.list_tasks","arguments":{"scope":"public","limit":2,"offset":4}}`)))
	if err := json.Unmarshal([]byte(last), &payload); err != nil {
		t.Fatalf("decode tasks payload: %v", err)
	}
	if len(payload.Tasks) != 1 || payload.NextOffset != 0 {
		t.Fatalf("tasks = %d next_offset = %d, want 1 and 0", len(payload.Tasks), payload.NextOffset)
	}

	// Every other list payload carries the field too; spot-check a few that
	// had no pagination at all before.
	for _, call := range []string{
		`{"name":"sharecrop.list_task_submissions","arguments":{"task_id":"` + testTaskID(t) + `"}}`,
		`{"name":"sharecrop.list_task_series","arguments":{}}`,
		`{"name":"sharecrop.list_ledger","arguments":{}}`,
	} {
		content := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: agent.NewScopeSet(agent.AllScopes())}, request(`3`, "tools/call", call)))
		if !strings.Contains(content, `"next_offset"`) {
			t.Fatalf("list result missing next_offset: %s", content)
		}
	}
}

// TestListTasksCreatedAfterFilter pins the created_after argument: an
// RFC3339 value threads through as task.CreatedAfter and garbage is
// rejected with a clear message.
func TestListTasksCreatedAfterFilter(t *testing.T) {
	services := newCapturingServices()
	server := NewServer(services)
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.list_tasks","arguments":{"scope":"public","created_after":"2026-07-01T00:00:00Z"}}`))
	if response.Error != nil {
		t.Fatalf("list_tasks created_after error: %s", response.Error.Message)
	}
	created, matched := services.listFilters.Created.(task.CreatedAfter)
	if !matched || created.Instant.Format(time.RFC3339) != "2026-07-01T00:00:00Z" {
		t.Fatalf("created filter = %#v", services.listFilters.Created)
	}

	invalid := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`2`, "tools/call", `{"name":"sharecrop.list_tasks","arguments":{"scope":"public","created_after":"yesterday"}}`))
	if invalid.Error == nil || !strings.Contains(invalid.Error.Message, "RFC3339") {
		t.Fatalf("expected an RFC3339 rejection, got %+v", invalid.Error)
	}
}

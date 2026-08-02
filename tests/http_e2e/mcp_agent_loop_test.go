//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// This file covers the stage-3 MCP surface: the reviewer deep read
// (get_submission), validation errors + kept reservations on invalid
// submissions, the required visibility_kind, marketplace webhook
// subscriptions, platform-admin credit grants, created_after filtering,
// newest-first ledger ordering, next_offset pagination, and the
// scope-filtered tools/list.

// mcpToolError decodes a tool-level failure (isError result) into its code
// and message.
func mcpToolError(t *testing.T, envelope rpcEnvelope) (string, string) {
	t.Helper()
	if envelope.Error != nil {
		t.Fatalf("expected a tool-level failure, got protocol error: %+v", envelope.Error)
	}
	var result struct {
		IsError bool `json:"isError"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(envelope.Result, &result); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if !result.IsError || len(result.Content) != 1 {
		t.Fatalf("expected an isError tool result, got %+v", result)
	}
	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &body); err != nil {
		t.Fatalf("failure text is not JSON: %q", result.Content[0].Text)
	}
	return body.Code, body.Message
}

type mcpSubmissionDetail struct {
	ID                   string `json:"id"`
	TaskID               string `json:"task_id"`
	SubmitterID          string `json:"submitter_id"`
	SubmitterDisplayName string `json:"submitter_display_name"`
	State                string `json:"state"`
	ResponseJSON         string `json:"response_json"`
	ReviewNote           string `json:"review_note"`
	Attachments          []struct {
		Name        string `json:"name"`
		ContentType string `json:"content_type"`
		DataURL     string `json:"data_url"`
	} `json:"attachments"`
	ValidationErrors []struct {
		Path    string `json:"path"`
		Message string `json:"message"`
	} `json:"validation_errors"`
	CreatedAt string `json:"created_at"`
}

// TestMCPReviewerReadsSubmissionContent is the walkthrough's critical
// scenario: the task owner reads a submission's full content over MCP
// instead of reviewing blind.
func TestMCPReviewerReadsSubmissionContent(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-review-owner")
	worker := registerUser(t, server, "mcp-review-worker")
	stranger := registerUser(t, server, "mcp-review-stranger")

	task := createPublicUserTask(t, server, owner)
	openTask(t, server, owner.AccessToken, task.ID)

	workerAgent := createWorkSeekingAgentCredential(t, server, worker, []string{"tasks_read", "submissions_write", "submissions_read"})
	workerSession := initializeMCPSession(t, server, workerAgent)

	submit := toolText(t, decodeRPC(t, mcpCall(t, server, workerAgent, workerSession, `1`, "sharecrop.submit_response",
		`{"task_id":"`+task.ID+`","response_json":"{\"answer\":\"reviewed thoroughly\"}","attachments":[{"name":"findings.txt","content_type":"text/plain","data_url":"data:text/plain;base64,aGVsbG8="}]}`)))
	var submitted struct {
		SubmissionID string `json:"submission_id"`
		State        string `json:"state"`
	}
	if err := json.Unmarshal([]byte(submit), &submitted); err != nil {
		t.Fatalf("decode submit payload: %v (%s)", err, submit)
	}
	if submitted.State != "submitted" {
		t.Fatalf("submission state = %q, want submitted", submitted.State)
	}

	// The owner lists summaries: submitter display name and created_at are
	// present, and the deep content is not (list rows stay summaries).
	ownerAgent := createAgentCredential(t, server, owner.AccessToken, []string{"submissions_read", "submissions_review"})
	ownerSession := initializeMCPSession(t, server, ownerAgent)
	list := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, ownerSession, `2`, "sharecrop.list_task_submissions", `{"task_id":"`+task.ID+`"}`)))
	var listPayload struct {
		Submissions []struct {
			ID                   string `json:"id"`
			SubmitterDisplayName string `json:"submitter_display_name"`
			CreatedAt            string `json:"created_at"`
		} `json:"submissions"`
		NextOffset int `json:"next_offset"`
	}
	if err := json.Unmarshal([]byte(list), &listPayload); err != nil {
		t.Fatalf("decode list payload: %v (%s)", err, list)
	}
	if len(listPayload.Submissions) != 1 {
		t.Fatalf("submission rows = %d, want 1", len(listPayload.Submissions))
	}
	if listPayload.Submissions[0].SubmitterDisplayName == "" || listPayload.Submissions[0].CreatedAt == "" {
		t.Fatalf("summary row missing display name or created_at: %+v", listPayload.Submissions[0])
	}
	if strings.Contains(list, "response_json") {
		t.Fatalf("summary rows should not carry response_json: %s", list)
	}

	// The deep read returns the full content.
	detailText := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, ownerSession, `3`, "sharecrop.get_submission", `{"submission_id":"`+submitted.SubmissionID+`"}`)))
	var detail mcpSubmissionDetail
	if err := json.Unmarshal([]byte(detailText), &detail); err != nil {
		t.Fatalf("decode submission detail: %v (%s)", err, detailText)
	}
	if detail.ID != submitted.SubmissionID || detail.TaskID != task.ID {
		t.Fatalf("detail ids = %+v", detail)
	}
	// Postgres stores the response as jsonb, so formatting may be
	// normalized; the content is what matters.
	if !strings.Contains(detail.ResponseJSON, "reviewed thoroughly") {
		t.Fatalf("response_json = %q", detail.ResponseJSON)
	}
	if detail.SubmitterID != worker.SubjectID || detail.SubmitterDisplayName == "" {
		t.Fatalf("submitter fields = %q / %q", detail.SubmitterID, detail.SubmitterDisplayName)
	}
	if detail.State != "submitted" || detail.CreatedAt == "" {
		t.Fatalf("state/created_at = %q / %q", detail.State, detail.CreatedAt)
	}
	if len(detail.Attachments) != 1 || detail.Attachments[0].Name != "findings.txt" || !strings.HasPrefix(detail.Attachments[0].DataURL, "data:text/plain;base64,") {
		t.Fatalf("attachments = %+v", detail.Attachments)
	}
	if len(detail.ValidationErrors) != 0 {
		t.Fatalf("validation_errors = %+v, want empty", detail.ValidationErrors)
	}

	// The submitter can read their own submission; a stranger cannot.
	own := decodeRPC(t, mcpCall(t, server, workerAgent, workerSession, `4`, "sharecrop.get_submission", `{"submission_id":"`+submitted.SubmissionID+`"}`))
	if own.Error != nil {
		t.Fatalf("submitter self-read failed: %+v", own.Error)
	}
	strangerAgent := createAgentCredential(t, server, stranger.AccessToken, []string{"submissions_read"})
	strangerSession := initializeMCPSession(t, server, strangerAgent)
	code, _ := mcpToolError(t, decodeRPC(t, mcpCall(t, server, strangerAgent, strangerSession, `5`, "sharecrop.get_submission", `{"submission_id":"`+submitted.SubmissionID+`"}`)))
	if code != "permission_denied" {
		t.Fatalf("stranger read code = %q, want permission_denied", code)
	}
}

// TestMCPInvalidSubmissionReturnsErrorsAndKeepsReservation covers the
// walkthrough's blind-invalid trap: the tool result names the validation
// errors, the reservation stays active, and an immediate resubmit works.
func TestMCPInvalidSubmissionReturnsErrorsAndKeepsReservation(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-invalid-owner")
	worker := registerUser(t, server, "mcp-invalid-worker")

	ownerAgent := createAgentCredential(t, server, owner.AccessToken, []string{"tasks_read", "tasks_write", "submissions_read"})
	ownerSession := initializeMCPSession(t, server, ownerAgent)

	created := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, ownerSession, `1`, "sharecrop.create_task",
		`{"title":"Structured review","description":"Answer with the required field.","response_schema_json":"{\"kind\":\"object\",\"fields\":[{\"name\":\"answer\",\"presence\":\"required\",\"schema\":{\"kind\":\"string\"}}]}","visibility_kind":"public","reward_kind":"none","participation_policy":"reservation_required"}`)))
	var createdDetail struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(created), &createdDetail); err != nil {
		t.Fatalf("decode create payload: %v (%s)", err, created)
	}
	opened := decodeRPC(t, mcpCall(t, server, ownerAgent, ownerSession, `2`, "sharecrop.open_task", `{"task_id":"`+createdDetail.ID+`"}`))
	if opened.Error != nil {
		t.Fatalf("open_task error: %+v", opened.Error)
	}

	workerAgent := createWorkSeekingAgentCredential(t, server, worker, []string{"tasks_read", "submissions_write", "submissions_read"})
	workerSession := initializeMCPSession(t, server, workerAgent)
	reserve := decodeRPC(t, mcpCall(t, server, workerAgent, workerSession, `3`, "sharecrop.reserve_task", `{"task_id":"`+createdDetail.ID+`"}`))
	if reserve.Error != nil {
		t.Fatalf("reserve_task error: %+v", reserve.Error)
	}

	// A schema-invalid submission comes back with the errors and guidance.
	invalid := toolText(t, decodeRPC(t, mcpCall(t, server, workerAgent, workerSession, `4`, "sharecrop.submit_response",
		`{"task_id":"`+createdDetail.ID+`","response_json":"{\"wrong\":\"field\"}"}`)))
	var invalidPayload struct {
		State            string `json:"state"`
		ValidationErrors []struct {
			Path    string `json:"path"`
			Message string `json:"message"`
		} `json:"validation_errors"`
		Guidance string `json:"guidance"`
	}
	if err := json.Unmarshal([]byte(invalid), &invalidPayload); err != nil {
		t.Fatalf("decode invalid payload: %v (%s)", err, invalid)
	}
	if invalidPayload.State != "invalid" {
		t.Fatalf("state = %q, want invalid", invalidPayload.State)
	}
	if len(invalidPayload.ValidationErrors) == 0 || invalidPayload.ValidationErrors[0].Message == "" {
		t.Fatalf("validation_errors = %+v, want the schema errors", invalidPayload.ValidationErrors)
	}
	if !strings.Contains(invalidPayload.Guidance, "reservation") {
		t.Fatalf("guidance = %q, want the kept-reservation note", invalidPayload.Guidance)
	}

	// The reservation is still active for the worker.
	reservations := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, ownerSession, `5`, "sharecrop.list_task_reservations", `{"task_id":"`+createdDetail.ID+`"}`)))
	if !strings.Contains(reservations, `"state":"active"`) {
		t.Fatalf("reservation not active after an invalid submission: %s", reservations)
	}

	// An immediate corrected resubmit succeeds without re-reserving.
	valid := toolText(t, decodeRPC(t, mcpCall(t, server, workerAgent, workerSession, `6`, "sharecrop.submit_response",
		`{"task_id":"`+createdDetail.ID+`","response_json":"{\"answer\":\"done\"}"}`)))
	if !strings.Contains(valid, `"state":"submitted"`) || !strings.Contains(valid, `"validation_errors":[]`) {
		t.Fatalf("resubmit payload = %s", valid)
	}
}

// TestMCPCreateTaskRequiresVisibilityKind pins the invisible-task trap
// closed over the live server: omitting visibility_kind errors clearly and
// names the valid values.
func TestMCPCreateTaskRequiresVisibilityKind(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-visibility-owner")
	ownerAgent := createAgentCredential(t, server, owner.AccessToken, []string{"tasks_read", "tasks_write"})
	session := initializeMCPSession(t, server, ownerAgent)

	response := decodeRPC(t, mcpCall(t, server, ownerAgent, session, `1`, "sharecrop.create_task",
		`{"title":"T","description":"D","response_schema_json":"{\"kind\":\"freeform\"}","reward_kind":"none"}`))
	if response.Error == nil {
		t.Fatalf("expected an error creating a task without visibility_kind")
	}
	for _, expected := range []string{"visibility_kind is required", "public", "organization_team"} {
		if !strings.Contains(response.Error.Message, expected) {
			t.Fatalf("error message %q missing %q", response.Error.Message, expected)
		}
	}
}

// TestMCPMarketplaceWebhookSubscription covers the MCP creation path for the
// marketplace audience plus the kind entitlement rule.
func TestMCPMarketplaceWebhookSubscription(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	operator := registerUser(t, server, "mcp-marketplace-hooks")
	operatorAgent := createAgentCredential(t, server, operator.AccessToken, []string{"webhooks_manage", "webhooks_read", "tasks_read"})
	session := initializeMCPSession(t, server, operatorAgent)

	created := toolText(t, decodeRPC(t, mcpCall(t, server, operatorAgent, session, `1`, "sharecrop.create_webhook_subscription",
		`{"url":"https://receiver.example.com/hooks","kinds":["task_opened"],"audience":"marketplace","filter_task_type":"code_review","filter_min_credit_reward":25}`)))
	var createdPayload struct {
		Subscription struct {
			ID                    string `json:"id"`
			Audience              string `json:"audience"`
			FilterTaskType        string `json:"filter_task_type"`
			FilterMinCreditReward int64  `json:"filter_min_credit_reward"`
		} `json:"subscription"`
		Secret string `json:"secret"`
	}
	if err := json.Unmarshal([]byte(created), &createdPayload); err != nil {
		t.Fatalf("decode webhook payload: %v (%s)", err, created)
	}
	if createdPayload.Subscription.Audience != "marketplace" || createdPayload.Subscription.FilterTaskType != "code_review" || createdPayload.Subscription.FilterMinCreditReward != 25 {
		t.Fatalf("subscription = %+v", createdPayload.Subscription)
	}
	if createdPayload.Secret == "" {
		t.Fatalf("create response missing the one-time secret")
	}

	// The listing echoes the new fields and carries next_offset.
	listed := toolText(t, decodeRPC(t, mcpCall(t, server, operatorAgent, session, `2`, "sharecrop.list_webhook_subscriptions", `{}`)))
	if !strings.Contains(listed, `"audience":"marketplace"`) || !strings.Contains(listed, `"filter_task_type":"code_review"`) || !strings.Contains(listed, `"next_offset"`) {
		t.Fatalf("subscription listing missing audience/filter/next_offset: %s", listed)
	}

	// Marketplace subscriptions listen only for task_opened. Both kinds here
	// are tasks_read-entitled, so the failure is the audience/kind rule, not
	// an entitlement gap.
	code, message := mcpToolError(t, decodeRPC(t, mcpCall(t, server, operatorAgent, session, `3`, "sharecrop.create_webhook_subscription",
		`{"url":"https://receiver.example.com/hooks","kinds":["task_opened","task_commented"],"audience":"marketplace"}`)))
	if code != "invalid_argument" || !strings.Contains(message, "task_opened") {
		t.Fatalf("expected the marketplace kinds rule, got %q %q", code, message)
	}

	// Subscribing to task_opened requires the tasks_read entitlement even
	// with webhooks_manage.
	unentitledAgent := createAgentCredential(t, server, operator.AccessToken, []string{"webhooks_manage", "webhooks_read"})
	unentitledSession := initializeMCPSession(t, server, unentitledAgent)
	denied := decodeRPC(t, mcpCall(t, server, unentitledAgent, unentitledSession, `4`, "sharecrop.create_webhook_subscription",
		`{"url":"https://receiver.example.com/hooks","kinds":["task_opened"],"audience":"marketplace"}`))
	if denied.Error == nil || !strings.Contains(denied.Error.Message, "tasks_read") {
		t.Fatalf("expected a tasks_read entitlement rejection, got %+v", denied.Error)
	}
}

// TestMCPGrantCreditsAndLedgerNote covers the grant tool end to end: a
// platform admin grants credits over MCP with a note, and the recipient's
// ledger lists the entry newest-first with the note.
func TestMCPGrantCreditsAndLedgerNote(t *testing.T) {
	bootstrap := newAuthHTTPServer(t, t.Context())
	admin := registerUser(t, bootstrap, "mcp-grant-admin")
	recipient := registerUser(t, bootstrap, "mcp-grant-recipient")
	bootstrap.Close()

	t.Setenv("SHARECROP_ADMIN_USER_IDS", admin.SubjectID)
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	adminAgent := createAgentCredential(t, server, admin.AccessToken, []string{"platform_admin"})
	adminSession := initializeMCPSession(t, server, adminAgent)

	granted := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, adminSession, `1`, "sharecrop.grant_credits",
		`{"target_kind":"user","target_id":"`+recipient.SubjectID+`","amount":40,"note":"Reimbursed the cancelled batch.","idempotency_key":"mcp-grant-1"}`)))
	var grantPayload struct {
		EntryID string `json:"entry_id"`
		Amount  int64  `json:"amount"`
	}
	if err := json.Unmarshal([]byte(granted), &grantPayload); err != nil {
		t.Fatalf("decode grant payload: %v (%s)", err, granted)
	}
	if grantPayload.EntryID == "" || grantPayload.Amount != 40 {
		t.Fatalf("grant payload = %+v", grantPayload)
	}

	// Replaying the same idempotency key returns the original entry.
	replayed := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, adminSession, `2`, "sharecrop.grant_credits",
		`{"target_kind":"user","target_id":"`+recipient.SubjectID+`","amount":40,"note":"Reimbursed the cancelled batch.","idempotency_key":"mcp-grant-1"}`)))
	if !strings.Contains(replayed, grantPayload.EntryID) {
		t.Fatalf("replayed grant = %s, want the original entry %s", replayed, grantPayload.EntryID)
	}
	if balance := getBalance(t, server, recipient.AccessToken); balance.SpendableCredits != 140 {
		t.Fatalf("recipient balance = %d, want 140 (signup grant + one credit grant)", balance.SpendableCredits)
	}

	// The recipient's ledger is newest-first: the grant (with its note)
	// precedes the older signup grant.
	recipientAgent := createAgentCredential(t, server, recipient.AccessToken, []string{"ledger_read"})
	recipientSession := initializeMCPSession(t, server, recipientAgent)
	ledgerText := toolText(t, decodeRPC(t, mcpCall(t, server, recipientAgent, recipientSession, `3`, "sharecrop.list_ledger", `{}`)))
	var ledgerPayload struct {
		Entries []struct {
			Kind string `json:"kind"`
			Note string `json:"note"`
		} `json:"entries"`
		NextOffset int `json:"next_offset"`
	}
	if err := json.Unmarshal([]byte(ledgerText), &ledgerPayload); err != nil {
		t.Fatalf("decode ledger payload: %v (%s)", err, ledgerText)
	}
	if len(ledgerPayload.Entries) != 2 {
		t.Fatalf("ledger entries = %d, want 2", len(ledgerPayload.Entries))
	}
	if ledgerPayload.Entries[0].Kind != "manual_adjustment" || ledgerPayload.Entries[0].Note != "Reimbursed the cancelled batch." {
		t.Fatalf("newest entry = %+v, want the manual adjustment with its note first", ledgerPayload.Entries[0])
	}
	if ledgerPayload.Entries[1].Kind != "signup_grant" {
		t.Fatalf("older entry = %+v, want the signup grant last", ledgerPayload.Entries[1])
	}
	if ledgerPayload.NextOffset != 0 {
		t.Fatalf("next_offset = %d, want 0 on the last page", ledgerPayload.NextOffset)
	}
}

// TestMCPListTasksCreatedAfterAndNextOffset covers the created_after filter
// and the probe-based next_offset over the live server.
func TestMCPListTasksCreatedAfterAndNextOffset(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-created-after")
	early := createPublicUserTask(t, server, owner)
	openTask(t, server, owner.AccessToken, early.ID)

	time.Sleep(50 * time.Millisecond)
	boundary := time.Now().UTC().Format(time.RFC3339Nano)
	time.Sleep(50 * time.Millisecond)

	late := createPublicUserTask(t, server, owner)
	openTask(t, server, owner.AccessToken, late.ID)

	agent := createAgentCredential(t, server, owner.AccessToken, []string{"tasks_read"})
	session := initializeMCPSession(t, server, agent)

	filtered := toolText(t, decodeRPC(t, mcpCall(t, server, agent, session, `1`, "sharecrop.list_tasks", `{"scope":"user","created_after":"`+boundary+`"}`)))
	if !strings.Contains(filtered, late.ID) || strings.Contains(filtered, early.ID) {
		t.Fatalf("created_after listing = %s, want only the late task", filtered)
	}

	// limit 1 over two tasks: a full window reports next_offset 1, and the
	// second page reports 0.
	first := toolText(t, decodeRPC(t, mcpCall(t, server, agent, session, `2`, "sharecrop.list_tasks", `{"scope":"user","limit":1}`)))
	var page struct {
		Tasks      []struct{} `json:"tasks"`
		NextOffset int        `json:"next_offset"`
	}
	if err := json.Unmarshal([]byte(first), &page); err != nil {
		t.Fatalf("decode first page: %v", err)
	}
	if len(page.Tasks) != 1 || page.NextOffset != 1 {
		t.Fatalf("first page = %d tasks next_offset %d, want 1 and 1", len(page.Tasks), page.NextOffset)
	}
	second := toolText(t, decodeRPC(t, mcpCall(t, server, agent, session, `3`, "sharecrop.list_tasks", `{"scope":"user","limit":1,"offset":1}`)))
	if err := json.Unmarshal([]byte(second), &page); err != nil {
		t.Fatalf("decode second page: %v", err)
	}
	if len(page.Tasks) != 1 || page.NextOffset != 0 {
		t.Fatalf("second page = %d tasks next_offset %d, want 1 and 0", len(page.Tasks), page.NextOffset)
	}

	invalid := decodeRPC(t, mcpCall(t, server, agent, session, `4`, "sharecrop.list_tasks", `{"scope":"user","created_after":"not-a-timestamp"}`))
	if invalid.Error == nil || !strings.Contains(invalid.Error.Message, "RFC3339") {
		t.Fatalf("expected an RFC3339 rejection, got %+v", invalid.Error)
	}
}

// TestMCPToolsListFilteredByScopes covers the honesty of the tool surface
// over the live server: a worker credential is not shown admin tools, and
// the initialize result carries the orientation instructions.
func TestMCPToolsListFilteredByScopes(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	user := registerUser(t, server, "mcp-tools-list")
	workerAgent := createAgentCredential(t, server, user.AccessToken, []string{"tasks_read", "submissions_write", "submissions_read"})

	initialize := decodeRPC(t, mcpRequest(t, server, workerAgent, "", `1`, "initialize", `{}`))
	if initialize.Error != nil {
		t.Fatalf("initialize error: %+v", initialize.Error)
	}
	var initResult struct {
		Instructions string `json:"instructions"`
	}
	if err := json.Unmarshal(initialize.Result, &initResult); err != nil {
		t.Fatalf("decode initialize result: %v", err)
	}
	if !strings.Contains(initResult.Instructions, "sharecrop.list_tasks") || !strings.Contains(initResult.Instructions, "NOT JSON Schema") {
		t.Fatalf("instructions missing orientation content:\n%s", initResult.Instructions)
	}

	session := initializeMCPSession(t, server, workerAgent)
	toolsList := decodeRPC(t, mcpRequest(t, server, workerAgent, session, `2`, "tools/list", `{}`))
	listText := string(toolsList.Result)
	for _, expected := range []string{"sharecrop.submit_response", "sharecrop.get_submission", "sharecrop.reserve_task"} {
		if !strings.Contains(listText, `"`+expected+`"`) {
			t.Fatalf("worker tools/list missing %s: %s", expected, listText)
		}
	}
	for _, hidden := range []string{"sharecrop.grant_credits", "sharecrop.grant_platform_admin", "sharecrop.create_task", "sharecrop.list_ledger", "sharecrop.create_webhook_subscription"} {
		if strings.Contains(listText, `"`+hidden+`"`) {
			t.Fatalf("worker tools/list should not include %s: %s", hidden, listText)
		}
	}
}

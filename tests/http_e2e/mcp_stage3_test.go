//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// mcpEventFeedPayload mirrors the sharecrop.list_events tool result.
type mcpEventFeedPayload struct {
	Events []struct {
		ID               string `json:"id"`
		Kind             string `json:"kind"`
		ActorID          string `json:"actor_id"`
		ActorDisplayName string `json:"actor_display_name"`
		SubjectKind      string `json:"subject_kind"`
		SubjectID        string `json:"subject_id"`
		TaskTitle        string `json:"task_title"`
		OccurredAt       string `json:"occurred_at"`
	} `json:"events"`
	NextCursor string `json:"next_cursor"`
}

func decodeMCPEventFeed(t *testing.T, content string) mcpEventFeedPayload {
	t.Helper()
	var payload mcpEventFeedPayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("decode list_events payload: %v (%s)", err, content)
	}
	return payload
}

// TestMCPListEventsCursorFeedForAgentCredential covers the stage-3 event
// tool end to end: a personal agent credential holding notifications_read
// reads its owner's feed over MCP, resumes with the returned cursor, and a
// credential without the scope is denied.
func TestMCPListEventsCursorFeedForAgentCredential(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-events-owner")
	worker := registerUser(t, server, "mcp-events-worker")

	createResponse := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(publicReservationTaskRequestJSON(owner.SubjectID)), owner.AccessToken)
	defer createResponse.Body.Close()
	assertStatus(t, createResponse, http.StatusCreated)
	taskBody := decodeTaskHTTPResponse(t, createResponse)
	openTask(t, server, owner.AccessToken, taskBody.ID)

	reserveResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskBody.ID+"/reservations", []byte(`{}`), worker.AccessToken)
	defer reserveResponse.Body.Close()
	assertStatus(t, reserveResponse, http.StatusCreated)

	feedAgent := createAgentCredential(t, server, owner.AccessToken, []string{"notifications_read"})
	feedSession := initializeMCPSession(t, server, feedAgent)

	payload := decodeMCPEventFeed(t, toolText(t, decodeRPC(t, mcpCall(t, server, feedAgent, feedSession, `1`, "sharecrop.list_events", `{}`))))
	if len(payload.Events) == 0 || payload.NextCursor == "" {
		t.Fatalf("list_events returned no rows or no cursor: %+v", payload)
	}
	foundReservation := false
	for _, row := range payload.Events {
		if row.Kind != "reservation_requested" || row.SubjectID != taskBody.ID {
			continue
		}
		foundReservation = true
		if row.SubjectKind != "task" || row.TaskTitle == "" {
			t.Fatalf("reservation event subject fields wrong: %+v", row)
		}
		if row.ActorID != worker.SubjectID || row.ActorDisplayName == "" {
			t.Fatalf("reservation event actor fields wrong: %+v", row)
		}
		if row.ID == "" || row.OccurredAt == "" {
			t.Fatalf("reservation event id/occurred_at missing: %+v", row)
		}
	}
	if !foundReservation {
		t.Fatalf("list_events missing the reservation_requested event: %+v", payload.Events)
	}

	// Resuming after the returned cursor yields an empty page with an empty
	// next_cursor.
	resumed := decodeMCPEventFeed(t, toolText(t, decodeRPC(t, mcpCall(t, server, feedAgent, feedSession, `2`, "sharecrop.list_events", `{"after":"`+payload.NextCursor+`"}`))))
	if len(resumed.Events) != 0 || resumed.NextCursor != "" {
		t.Fatalf("resumed page = %d rows, next_cursor %q; want empty", len(resumed.Events), resumed.NextCursor)
	}

	// A malformed cursor is rejected as a protocol error.
	malformed := decodeRPC(t, mcpCall(t, server, feedAgent, feedSession, `3`, "sharecrop.list_events", `{"after":"garbage"}`))
	if malformed.Error == nil {
		t.Fatalf("expected a protocol error for a malformed cursor")
	}

	// A credential without notifications_read is scope-denied.
	tasksOnly := createAgentCredential(t, server, owner.AccessToken, []string{"tasks_read"})
	tasksOnlySession := initializeMCPSession(t, server, tasksOnly)
	denied := decodeRPC(t, mcpCall(t, server, tasksOnly, tasksOnlySession, `4`, "sharecrop.list_events", `{}`))
	if denied.Error == nil || !strings.Contains(denied.Error.Message, "notifications_read") {
		t.Fatalf("expected a notifications_read scope denial, got %+v", denied.Error)
	}
}

// TestMCPOrgCredentialListsOrganizationEvents pins the organization side of
// list_events: an organization credential reads the organization's subject
// events (here the task_funded event from funding an org-owned task out of
// the organization balance).
func TestMCPOrgCredentialListsOrganizationEvents(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-org-events-owner")
	organizationID := createOrganization(t, server, owner, "MCP Org Events Labs")
	taskID := createOrganizationTask(t, server, owner, organizationID)

	fundResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskID+"/funding", []byte(`{"amount":30,"idempotency_key":"org-events-fund-`+taskID+`","organization_id":"`+organizationID+`"}`), owner.AccessToken)
	defer fundResponse.Body.Close()
	assertStatus(t, fundResponse, http.StatusCreated)

	credentialResponse := postJSONWithBearer(t, server.URL+"/api/organizations/"+organizationID+"/credentials", []byte(`{"label":"MCP org events token","scopes":["notifications_read"],"expires_at":""}`), owner.AccessToken)
	defer credentialResponse.Body.Close()
	assertStatus(t, credentialResponse, http.StatusCreated)
	orgCredential := decodeOrgCredentialCreatedHTTPResponse(t, credentialResponse)
	orgSession := initializeMCPSession(t, server, orgCredential.Secret)

	payload := decodeMCPEventFeed(t, toolText(t, decodeRPC(t, mcpCall(t, server, orgCredential.Secret, orgSession, `1`, "sharecrop.list_events", `{}`))))
	foundFunded := false
	for _, row := range payload.Events {
		if row.Kind == "task_funded" && row.SubjectKind == "task" && row.SubjectID == taskID {
			foundFunded = true
		}
	}
	if !foundFunded {
		t.Fatalf("org credential list_events missing the org task_funded event: %+v", payload.Events)
	}
	if payload.NextCursor == "" {
		t.Fatalf("org credential list_events returned no cursor: %+v", payload)
	}
}

// TestMCPListTasksFundedFilterEnrichmentAndTotal drives the stage-3
// list_tasks surface over MCP: the funded input filter, the enriched row
// fields, and the filter-wide total.
func TestMCPListTasksFundedFilterEnrichmentAndTotal(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-funded-owner")
	tag := "mcpfunded-" + uniqueTestSuffix(t)

	fundedTaskID := createUserTaskFromJSON(t, server, owner.AccessToken, taggedPublicTaskRequestJSON(owner.SubjectID, tag+" funded", "credit", 10))
	fundTask(t, server, owner.AccessToken, fundedTaskID, 10, "mcp-matrix-fund-"+fundedTaskID)
	openTask(t, server, owner.AccessToken, fundedTaskID)
	unfundedTaskID := createUserTaskFromJSON(t, server, owner.AccessToken, taggedPublicTaskRequestJSON(owner.SubjectID, tag+" unfunded", "credit", 10))

	agentToken := createAgentCredential(t, server, owner.AccessToken, []string{"tasks_read"})
	session := initializeMCPSession(t, server, agentToken)

	type tasksResult struct {
		Tasks []struct {
			ID                 string `json:"id"`
			Funded             string `json:"funded"`
			CreatorDisplayName string `json:"creator_display_name"`
			HolderDisplayName  string `json:"holder_display_name"`
			PendingReviewCount int64  `json:"pending_review_count"`
		} `json:"tasks"`
		NextOffset int   `json:"next_offset"`
		Total      int64 `json:"total"`
	}

	// The funded filter narrows the owner's own listing to the unfunded
	// draft.
	filteredText := toolText(t, decodeRPC(t, mcpCall(t, server, agentToken, session, `1`, "sharecrop.list_tasks", `{"scope":"user","query":"`+tag+`","funded":"reward_unfunded"}`)))
	var filtered tasksResult
	if err := json.Unmarshal([]byte(filteredText), &filtered); err != nil {
		t.Fatalf("decode filtered tasks: %v (%s)", err, filteredText)
	}
	if len(filtered.Tasks) != 1 || filtered.Tasks[0].ID != unfundedTaskID || filtered.Total != 1 {
		t.Fatalf("funded=reward_unfunded = %+v total %d, want exactly the unfunded draft", filtered.Tasks, filtered.Total)
	}
	if filtered.Tasks[0].Funded != "reward_unfunded" {
		t.Fatalf("filtered row funded = %q, want reward_unfunded", filtered.Tasks[0].Funded)
	}

	// The unfiltered listing carries both rows, the enrichment fields, and
	// the filter-wide total.
	allText := toolText(t, decodeRPC(t, mcpCall(t, server, agentToken, session, `2`, "sharecrop.list_tasks", `{"scope":"user","query":"`+tag+`"}`)))
	var all tasksResult
	if err := json.Unmarshal([]byte(allText), &all); err != nil {
		t.Fatalf("decode unfiltered tasks: %v (%s)", err, allText)
	}
	if len(all.Tasks) != 2 || all.Total != 2 {
		t.Fatalf("unfiltered listing = %d rows total %d, want 2 and 2", len(all.Tasks), all.Total)
	}
	for _, row := range all.Tasks {
		if row.CreatorDisplayName == "" {
			t.Fatalf("row missing creator_display_name: %+v", row)
		}
		if row.ID == fundedTaskID && row.Funded != "reward_funded" {
			t.Fatalf("funded task row funded = %q, want reward_funded", row.Funded)
		}
	}

	// An unknown funded value is rejected as a protocol error.
	invalid := decodeRPC(t, mcpCall(t, server, agentToken, session, `3`, "sharecrop.list_tasks", `{"scope":"user","funded":"bogus"}`))
	if invalid.Error == nil {
		t.Fatalf("expected a protocol error for funded=bogus")
	}
}

// TestMCPDisputeFilingAndAdminTriageVisibility is the stage-3 dispute
// reachability flow: a worker whose submission was rejected files a
// structured dispute over MCP (subject_kind submission, reason dispute), and
// a platform admin sees the category in the triage listing with its total.
func TestMCPDisputeFilingAndAdminTriageVisibility(t *testing.T) {
	bootstrap := newAuthHTTPServer(t, t.Context())
	admin := registerUser(t, bootstrap, "mcp-dispute-admin")
	owner := registerUser(t, bootstrap, "mcp-dispute-owner")
	worker := registerUser(t, bootstrap, "mcp-dispute-worker")
	bootstrap.Close()

	t.Setenv("SHARECROP_ADMIN_USER_IDS", admin.SubjectID)
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	task := createPublicCreditUserTask(t, server, owner, 15)
	fundTask(t, server, owner.AccessToken, task.ID, 15, "dispute-fund-"+task.ID)
	openTask(t, server, owner.AccessToken, task.ID)

	workerAgent := createAgentCredential(t, server, worker.AccessToken, []string{"tasks_read", "submissions_write", "submissions_read"})
	workerSession := initializeMCPSession(t, server, workerAgent)
	ownerAgent := createAgentCredential(t, server, owner.AccessToken, []string{"tasks_read", "submissions_read", "submissions_review"})
	ownerSession := initializeMCPSession(t, server, ownerAgent)

	// The worker submits over MCP; the owner rejects the work.
	submit := toolText(t, decodeRPC(t, mcpCall(t, server, workerAgent, workerSession, `1`, "sharecrop.submit_response", `{"task_id":"`+task.ID+`","response_json":"{\"answer\":\"done\"}"}`)))
	var submitted struct {
		SubmissionID string `json:"submission_id"`
	}
	if err := json.Unmarshal([]byte(submit), &submitted); err != nil {
		t.Fatalf("decode submit payload: %v (%s)", err, submit)
	}
	rejected := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, ownerSession, `2`, "sharecrop.reject_submission", `{"task_id":"`+task.ID+`","submission_id":"`+submitted.SubmissionID+`","idempotency_key":"dispute-reject-`+task.ID+`","review_note":"Does not meet the acceptance criteria."}`)))
	if !strings.Contains(rejected, `"state":"rejected"`) {
		t.Fatalf("reject payload missing rejected state: %s", rejected)
	}

	// The owner's submission listing carries the filter-wide total over MCP.
	submissions := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, ownerSession, `3`, "sharecrop.list_task_submissions", `{"task_id":"`+task.ID+`"}`)))
	var submissionsPage struct {
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal([]byte(submissions), &submissionsPage); err != nil {
		t.Fatalf("decode submissions payload: %v (%s)", err, submissions)
	}
	if submissionsPage.Total != 1 {
		t.Fatalf("list_task_submissions total = %d, want 1", submissionsPage.Total)
	}

	// The worker files a structured dispute referencing the submission.
	dispute := toolText(t, decodeRPC(t, mcpCall(t, server, workerAgent, workerSession, `4`, "sharecrop.create_moderation_report", `{"subject_kind":"submission","subject_id":"`+submitted.SubmissionID+`","reason":"dispute","details":"The rejection cites acceptance criteria the task description never stated."}`)))
	var report struct {
		ID          string `json:"id"`
		SubjectKind string `json:"subject_kind"`
		SubjectID   string `json:"subject_id"`
		Reason      string `json:"reason"`
		State       string `json:"state"`
	}
	if err := json.Unmarshal([]byte(dispute), &report); err != nil {
		t.Fatalf("decode dispute payload: %v (%s)", err, dispute)
	}
	if report.SubjectKind != "submission" || report.SubjectID != submitted.SubmissionID || report.Reason != "dispute" || report.State != "open" {
		t.Fatalf("dispute report fields wrong: %+v", report)
	}

	// The platform admin sees the dispute category in the triage listing.
	adminAgent := createAgentCredential(t, server, admin.AccessToken, []string{"moderation_read"})
	adminSession := initializeMCPSession(t, server, adminAgent)
	triage := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, adminSession, `5`, "sharecrop.list_admin_moderation_reports", `{"state":"open"}`)))
	var triagePage struct {
		Reports []struct {
			ID     string `json:"id"`
			Reason string `json:"reason"`
		} `json:"reports"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal([]byte(triage), &triagePage); err != nil {
		t.Fatalf("decode triage payload: %v (%s)", err, triage)
	}
	foundDispute := false
	for _, row := range triagePage.Reports {
		if row.ID == report.ID && row.Reason == "dispute" {
			foundDispute = true
		}
	}
	if !foundDispute {
		t.Fatalf("admin triage listing missing the dispute report %s: %s", report.ID, triage)
	}
	if triagePage.Total < 1 {
		t.Fatalf("admin triage listing total = %d, want at least 1", triagePage.Total)
	}
}

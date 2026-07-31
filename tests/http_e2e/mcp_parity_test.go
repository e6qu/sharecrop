//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
)

// TestMCPToolFailureIsStructuredJSON verifies a failed tool call carries a
// machine-readable error body: one text item of {"code","message"} with
// isError true, here for a not_found rejection.
func TestMCPToolFailureIsStructuredJSON(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-structured-error")
	ownerAgent := createAgentCredential(t, server, owner.AccessToken, []string{"tasks_read", "tasks_write"})
	session := initializeMCPSession(t, server, ownerAgent)

	missingTaskID := core.NewTaskID().(core.TaskIDCreated).Value.String()
	envelope := decodeRPC(t, mcpCall(t, server, ownerAgent, session, `1`, "sharecrop.open_task", `{"task_id":"`+missingTaskID+`"}`))
	if envelope.Error != nil {
		t.Fatalf("expected a tool-level failure, got protocol error: %s", envelope.Error.Message)
	}
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(envelope.Result, &result); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected isError result: %s", string(envelope.Result))
	}
	if len(result.Content) != 1 {
		t.Fatalf("content count = %d, want 1", len(result.Content))
	}
	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &body); err != nil {
		t.Fatalf("failure text is not JSON: %q", result.Content[0].Text)
	}
	if body.Code != "not_found" {
		t.Fatalf("code = %q, want not_found (%s)", body.Code, result.Content[0].Text)
	}
	if body.Message == "" {
		t.Fatalf("message is empty")
	}
}

// TestMCPCreateTaskParityWithREST creates a task over MCP using the parity
// arguments and asserts the REST task detail reports the same semantics.
func TestMCPCreateTaskParityWithREST(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-parity-owner")
	ownerAgent := createAgentCredential(t, server, owner.AccessToken, []string{"tasks_read", "tasks_write"})
	session := initializeMCPSession(t, server, ownerAgent)

	expiresAt := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second).Format(time.RFC3339)
	created := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `1`, "sharecrop.create_task", `{
		"title": "Parity task",
		"description": "Created over MCP with full REST parity.",
		"response_schema_json": "{\"kind\":\"freeform\"}",
		"visibility": "public",
		"reward_kind": "none",
		"participation_policy": "reservation_required",
		"assignee_scope": "user",
		"reservation_expiry_hours": 72,
		"task_type": "qa_testing",
		"payload_json": "{\"batch\":\"A\"}",
		"expires_at": "`+expiresAt+`"
	}`)))
	var createdDetail struct {
		ID        string `json:"id"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := json.Unmarshal([]byte(created), &createdDetail); err != nil {
		t.Fatalf("decode create payload: %v (%s)", err, created)
	}
	if createdDetail.ExpiresAt != expiresAt {
		t.Fatalf("MCP expires_at = %q, want %q", createdDetail.ExpiresAt, expiresAt)
	}

	response := getWithBearer(t, server.URL+"/api/tasks/"+createdDetail.ID, owner.AccessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
	var restDetail struct {
		ParticipationPolicy    string `json:"participation_policy"`
		AssigneeScope          string `json:"assignee_scope"`
		ReservationExpiryHours int    `json:"reservation_expiry_hours"`
		TaskType               string `json:"task_type"`
		PayloadKind            string `json:"payload_kind"`
		PayloadJSON            string `json:"payload_json"`
		VisibilityKind         string `json:"visibility_kind"`
		ExpiresAt              string `json:"expires_at"`
	}
	if err := json.NewDecoder(response.Body).Decode(&restDetail); err != nil {
		t.Fatalf("decode REST task detail: %v", err)
	}
	// The payload is stored as jsonb, so compare its parsed value rather
	// than byte-for-byte formatting.
	var payload struct {
		Batch string `json:"batch"`
	}
	if err := json.Unmarshal([]byte(restDetail.PayloadJSON), &payload); err != nil {
		t.Fatalf("decode stored payload: %v (%q)", err, restDetail.PayloadJSON)
	}
	if restDetail.ParticipationPolicy != "reservation_required" ||
		restDetail.AssigneeScope != "user" ||
		restDetail.ReservationExpiryHours != 72 ||
		restDetail.TaskType != "qa_testing" ||
		restDetail.PayloadKind != "json" ||
		payload.Batch != "A" ||
		restDetail.VisibilityKind != "public" {
		t.Fatalf("REST detail diverges from MCP create: %+v", restDetail)
	}
	if restDetail.ExpiresAt != expiresAt {
		t.Fatalf("REST expires_at = %q, want %q", restDetail.ExpiresAt, expiresAt)
	}
}

// TestMCPAcceptSubmissionPaysCollectibleTip mirrors REST's tip_collectible_id
// on accept: the collectible moves from the requester to the worker.
func TestMCPAcceptSubmissionPaysCollectibleTip(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-tip-owner")
	worker := registerUser(t, server, "mcp-tip-worker")
	collectibleID := mintTransferableCollectible(t, server, owner.AccessToken, "Tip Trophy")

	created := toolTextForNewTask(t, server, owner)
	openTask(t, server, owner.AccessToken, created)
	submission := submitAuthenticated(t, server, worker.AccessToken, created)

	ownerAgent := createAgentCredential(t, server, owner.AccessToken, []string{"tasks_read", "submissions_read", "submissions_review"})
	session := initializeMCPSession(t, server, ownerAgent)
	accept := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `1`, "sharecrop.accept_submission",
		`{"task_id":"`+created+`","submission_id":"`+submission.Submission.ID+`","idempotency_key":"mcp-tip-`+created+`","tip_collectible_id":"`+collectibleID+`"}`)))
	if !strings.Contains(accept, collectibleID) {
		t.Fatalf("accept payload missing tipped collectible: %s", accept)
	}

	workerCollectibles := listCollectibles(t, server, worker.AccessToken)
	found := false
	for _, collectible := range workerCollectibles {
		if collectible.ID == collectibleID {
			found = true
		}
	}
	if !found {
		t.Fatalf("worker does not own the tipped collectible")
	}
}

// mintTransferableCollectible mints a user collectible whose transfer policy
// allows tipping (unlike the payout-only policy the shared helper mints).
func mintTransferableCollectible(t *testing.T, server *httptest.Server, accessToken string, name string) string {
	t.Helper()
	response := postJSONWithBearer(t, server.URL+"/api/collectibles", []byte(`{"name":"`+name+`","kind":"badge","transfer_policy":"transferable_between_users"}`), accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	var collectible collectibleHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&collectible); err != nil {
		t.Fatalf("decode collectible response: %v", err)
	}
	return collectible.ID
}

// toolTextForNewTask creates a plain open-participation, no-reward, public
// task over REST for MCP review flows, returning its id.
func toolTextForNewTask(t *testing.T, server *httptest.Server, owner authHTTPResponse) string {
	t.Helper()
	body := []byte(`{
		"owner": {"kind": "user", "user_id": "` + owner.SubjectID + `"},
		"title": "Tip flow task",
		"description": "Work rewarded with a collectible tip.",
		"reward": {"kind": "none"},
		"visibility": {"kind": "public"},
		"placement": {"kind": "standalone"},
		"response_schema_json": "{\"kind\":\"freeform\"}",
		"payload": {"kind": "none"}
	}`)
	response := postJSONWithBearer(t, server.URL+"/api/tasks", body, owner.AccessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	var task taskHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&task); err != nil {
		t.Fatalf("decode task response: %v", err)
	}
	return task.ID
}

// TestMCPCreditsTools exercises get_credit_balance and list_ledger with the
// ledger_read scope and confirms the scope gate without it.
func TestMCPCreditsTools(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	user := registerUser(t, server, "mcp-credits")
	grantedAgent := createAgentCredential(t, server, user.AccessToken, []string{"ledger_read"})
	session := initializeMCPSession(t, server, grantedAgent)

	balance := toolText(t, decodeRPC(t, mcpCall(t, server, grantedAgent, session, `1`, "sharecrop.get_credit_balance", `{}`)))
	var balancePayload struct {
		SpendableCredits int64 `json:"spendable_credits"`
		AllocatedCredits int64 `json:"allocated_credits"`
	}
	if err := json.Unmarshal([]byte(balance), &balancePayload); err != nil {
		t.Fatalf("decode balance: %v", err)
	}
	if balancePayload.SpendableCredits != 100 || balancePayload.AllocatedCredits != 0 {
		t.Fatalf("balance = %+v, want the signup grant", balancePayload)
	}

	entries := toolText(t, decodeRPC(t, mcpCall(t, server, grantedAgent, session, `2`, "sharecrop.list_ledger", `{"limit":10}`)))
	if !strings.Contains(entries, "signup_grant") {
		t.Fatalf("ledger missing signup grant: %s", entries)
	}

	deniedAgent := createAgentCredential(t, server, user.AccessToken, []string{"tasks_read"})
	deniedSession := initializeMCPSession(t, server, deniedAgent)
	denied := decodeRPC(t, mcpCall(t, server, deniedAgent, deniedSession, `3`, "sharecrop.get_credit_balance", `{}`))
	if denied.Error == nil || !strings.Contains(denied.Error.Message, "ledger_read") {
		t.Fatalf("expected a ledger_read scope rejection, got %+v", denied.Error)
	}
}

// TestMCPListTasksFilters checks the REST-parity list filters over MCP.
func TestMCPListTasksFilters(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-filters")
	ownerAgent := createAgentCredential(t, server, owner.AccessToken, []string{"tasks_read", "tasks_write"})
	session := initializeMCPSession(t, server, ownerAgent)

	task := createPublicCreditUserTask(t, server, owner, 10)
	fundTask(t, server, owner.AccessToken, task.ID, 10, "fund-"+task.ID)
	openTask(t, server, owner.AccessToken, task.ID)

	filtered := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `1`, "sharecrop.list_tasks",
		`{"scope":"public","states":["open"],"query":"`+task.ID+`","sort":"newest","limit":5}`)))
	if !strings.Contains(filtered, task.ID) {
		t.Fatalf("filtered listing missing the task: %s", filtered)
	}
	excluded := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `2`, "sharecrop.list_tasks",
		`{"scope":"public","states":["cancelled","expired"],"query":"`+task.ID+`"}`)))
	if strings.Contains(excluded, task.ID) {
		t.Fatalf("state filter did not exclude the open task: %s", excluded)
	}
}

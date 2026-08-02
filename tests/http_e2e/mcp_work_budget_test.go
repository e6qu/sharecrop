//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// This file covers the MCP side of agent work budgets: the get_my_budget
// self-introspection tool, and the two refusals a budgeted worker loop has to
// understand — work-seeking disabled (a human must act) and an exhausted
// allowance (wait for the reset).

// mcpBudgetPayload mirrors sharecrop.get_my_budget's personal-credential
// result.
type mcpBudgetPayload struct {
	CredentialKind            string   `json:"credential_kind"`
	WorkSeeking               string   `json:"work_seeking"`
	MaxTasksPerDay            int64    `json:"max_tasks_per_day"`
	TasksUsedToday            int64    `json:"tasks_used_today"`
	TasksRemainingToday       int64    `json:"tasks_remaining_today"`
	MaxConcurrentReservations int64    `json:"max_concurrent_reservations"`
	ActiveReservations        int64    `json:"active_reservations"`
	MaxCreditsPerDay          int64    `json:"max_credits_per_day"`
	CreditsSpentToday         int64    `json:"credits_spent_today"`
	TaskTypes                 []string `json:"task_types"`
	MinRewardCredits          int64    `json:"min_reward_credits"`
	TokenBudgetTokens         int64    `json:"token_budget_tokens"`
	TokenBudgetNote           string   `json:"token_budget_note"`
	ResetsAt                  string   `json:"resets_at"`
}

func readMCPBudget(t *testing.T, server *httptest.Server, agentToken string, sessionID string, id string) mcpBudgetPayload {
	t.Helper()
	content := toolText(t, decodeRPC(t, mcpCall(t, server, agentToken, sessionID, id, "sharecrop.get_my_budget", `{}`)))
	var payload mcpBudgetPayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("decode get_my_budget payload: %v (%s)", err, content)
	}
	return payload
}

// mcpToolFailure decodes an isError tool result into its code, message, and
// guidance line.
func mcpToolFailure(t *testing.T, envelope rpcEnvelope) (code string, message string, guidance string) {
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
		Code     string `json:"code"`
		Message  string `json:"message"`
		Guidance string `json:"guidance"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &body); err != nil {
		t.Fatalf("failure text is not JSON: %q", result.Content[0].Text)
	}
	return body.Code, body.Message, body.Guidance
}

// TestMCPGetMyBudgetTracksConsumption walks the budgeted worker's
// introspection loop over MCP: a fresh credential reads as
// work_seeking_disabled with nothing allowed, an enabled one reports its
// allowances, and reserving a task decrements tasks_remaining_today while
// active_reservations rises.
func TestMCPGetMyBudgetTracksConsumption(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "budget-mcp-owner")
	worker := registerUser(t, server, "budget-mcp-worker")
	created := createAgentCredentialResponse(t, server, worker.AccessToken, []string{"tasks_read", "submissions_write"})
	session := initializeMCPSession(t, server, created.Secret)

	fresh := readMCPBudget(t, server, created.Secret, session, `1`)
	if fresh.CredentialKind != "personal_agent_credential" {
		t.Fatalf("credential_kind = %q", fresh.CredentialKind)
	}
	if fresh.WorkSeeking != "work_seeking_disabled" {
		t.Fatalf("fresh work_seeking = %q, want work_seeking_disabled", fresh.WorkSeeking)
	}
	if fresh.MaxTasksPerDay != 0 || fresh.TasksRemainingToday != 0 {
		t.Fatalf("fresh credential reported an allowance: %+v", fresh)
	}
	if _, err := time.Parse(time.RFC3339, fresh.ResetsAt); err != nil {
		t.Fatalf("resets_at %q is not RFC3339: %v", fresh.ResetsAt, err)
	}

	enableWorkPolicy(t, server, worker, created.Credential.ID, `{
		"work_seeking":"work_seeking_enabled",
		"max_tasks_per_day":3,
		"max_concurrent_reservations":2,
		"max_credits_per_day":25,
		"token_budget_tokens":120000,
		"token_budget_note":"Advisory only."
	}`)

	enabled := readMCPBudget(t, server, created.Secret, session, `2`)
	if enabled.WorkSeeking != "work_seeking_enabled" {
		t.Fatalf("enabled work_seeking = %q", enabled.WorkSeeking)
	}
	if enabled.MaxTasksPerDay != 3 || enabled.TasksUsedToday != 0 || enabled.TasksRemainingToday != 3 {
		t.Fatalf("enabled budget = %+v, want 3 allowed and 3 remaining", enabled)
	}
	if enabled.MaxConcurrentReservations != 2 || enabled.MaxCreditsPerDay != 25 || enabled.MinRewardCredits != 0 {
		t.Fatalf("enabled allowances wrong: %+v", enabled)
	}
	if enabled.TokenBudgetTokens != 120000 || enabled.TokenBudgetNote != "Advisory only." {
		t.Fatalf("advisory token budget did not round-trip: %+v", enabled)
	}

	reserved := createReservationRequiredTask(t, server, owner)
	toolText(t, decodeRPC(t, mcpCall(t, server, created.Secret, session, `3`, "sharecrop.reserve_task", `{"task_id":"`+reserved+`"}`)))

	after := readMCPBudget(t, server, created.Secret, session, `4`)
	if after.TasksUsedToday != 1 || after.TasksRemainingToday != 2 {
		t.Fatalf("after one reservation: used %d remaining %d, want 1 and 2", after.TasksUsedToday, after.TasksRemainingToday)
	}
	if after.ActiveReservations != 1 {
		t.Fatalf("active_reservations = %d, want 1", after.ActiveReservations)
	}
}

// TestMCPDisabledCredentialReserveCarriesGuidance pins the default-deny
// refusal over MCP: reserving with a fresh credential fails with
// permission_denied and a guidance line naming get_my_budget and the operator
// who must enable work-seeking.
func TestMCPDisabledCredentialReserveCarriesGuidance(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "guidance-owner")
	worker := registerUser(t, server, "guidance-worker")
	reserved := createReservationRequiredTask(t, server, owner)

	created := createAgentCredentialResponse(t, server, worker.AccessToken, []string{"tasks_read", "submissions_write"})
	session := initializeMCPSession(t, server, created.Secret)

	code, message, guidance := mcpToolFailure(t, decodeRPC(t, mcpCall(t, server, created.Secret, session, `1`, "sharecrop.reserve_task", `{"task_id":"`+reserved+`"}`)))
	if code != "permission_denied" {
		t.Fatalf("code = %q (%s), want permission_denied", code, message)
	}
	if !strings.Contains(message, "work-seeking") {
		t.Fatalf("message %q does not explain work-seeking", message)
	}
	if !strings.Contains(guidance, "your operator has not enabled work-seeking") || !strings.Contains(guidance, "sharecrop.get_my_budget") {
		t.Fatalf("guidance = %q", guidance)
	}
}

// TestMCPExhaustedBudgetSurfacesBudgetExceeded exhausts a one-task daily
// budget over MCP: the second reservation fails with budget_exceeded and the
// human-readable exhaustion message, and get_my_budget then reports nothing
// remaining plus the reset instant the agent should wait for.
func TestMCPExhaustedBudgetSurfacesBudgetExceeded(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "exhaust-owner")
	worker := registerUser(t, server, "exhaust-worker")
	firstTask := createReservationRequiredTask(t, server, owner)
	secondTask := createReservationRequiredTask(t, server, owner)

	created := createAgentCredentialResponse(t, server, worker.AccessToken, []string{"tasks_read", "submissions_write"})
	enableWorkSeeking(t, server, worker, created.Credential.ID, 1)
	session := initializeMCPSession(t, server, created.Secret)

	toolText(t, decodeRPC(t, mcpCall(t, server, created.Secret, session, `1`, "sharecrop.reserve_task", `{"task_id":"`+firstTask+`"}`)))

	code, message, guidance := mcpToolFailure(t, decodeRPC(t, mcpCall(t, server, created.Secret, session, `2`, "sharecrop.reserve_task", `{"task_id":"`+secondTask+`"}`)))
	if code != "budget_exceeded" {
		t.Fatalf("code = %q (%s), want budget_exceeded", code, message)
	}
	if !strings.Contains(message, "daily task budget exhausted") || !strings.Contains(message, "resets at 00:00 UTC") {
		t.Fatalf("message %q does not carry the budget wording", message)
	}
	if guidance != "" {
		t.Fatalf("an exhausted budget is not an operator-configuration problem, guidance = %q", guidance)
	}

	spent := readMCPBudget(t, server, created.Secret, session, `3`)
	if spent.TasksUsedToday != 1 || spent.TasksRemainingToday != 0 {
		t.Fatalf("exhausted budget = %+v, want 1 used and 0 remaining", spent)
	}
	resets, err := time.Parse(time.RFC3339, spent.ResetsAt)
	if err != nil {
		t.Fatalf("resets_at %q is not RFC3339: %v", spent.ResetsAt, err)
	}
	if !resets.After(time.Now().UTC()) {
		t.Fatalf("resets_at %s is not in the future", resets)
	}
}

// TestMCPCappedSpendRefusalSurfacesBudgetExceeded pins the spend dimension
// over MCP: a credential whose human capped its daily credit spend is refused
// with budget_exceeded when funding a task above the cap, and the refused
// charge leaves credits_spent_today untouched.
func TestMCPCappedSpendRefusalSurfacesBudgetExceeded(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	requester := registerUser(t, server, "spend-cap-requester")
	created := createAgentCredentialResponse(t, server, requester.AccessToken, []string{"tasks_read", "tasks_write"})
	enableWorkPolicy(t, server, requester, created.Credential.ID, `{
		"work_seeking":"work_seeking_enabled",
		"max_tasks_per_day":5,
		"max_credits_per_day":10
	}`)
	session := initializeMCPSession(t, server, created.Secret)

	funded := createPublicCreditUserTask(t, server, requester, 30)
	code, message, _ := mcpToolFailure(t, decodeRPC(t, mcpCall(t, server, created.Secret, session, `1`, "sharecrop.fund_task", `{"task_id":"`+funded.ID+`","amount":30,"idempotency_key":"spend-cap-`+funded.ID+`"}`)))
	if code != "budget_exceeded" {
		t.Fatalf("code = %q (%s), want budget_exceeded", code, message)
	}
	if !strings.Contains(message, "daily credit spend budget exhausted") {
		t.Fatalf("message %q does not name the exhausted spend budget", message)
	}

	if budget := readMCPBudget(t, server, created.Secret, session, `2`); budget.CreditsSpentToday != 0 || budget.MaxCreditsPerDay != 10 {
		t.Fatalf("refused spend was recorded: %+v", budget)
	}
}

// TestMCPGetMyBudgetNeedsNoScopeAndRefusesOrgCredentials pins the tool's
// availability: it is listed and callable for a credential holding a single
// unrelated scope, and an organization credential is told plainly that
// organization credentials carry no work policy.
func TestMCPGetMyBudgetNeedsNoScopeAndRefusesOrgCredentials(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	user := registerUser(t, server, "budget-scope-user")
	created := createAgentCredentialResponse(t, server, user.AccessToken, []string{"notifications_read"})
	session := initializeMCPSession(t, server, created.Secret)

	toolsList := decodeRPC(t, mcpRequest(t, server, created.Secret, session, `1`, "tools/list", `{}`))
	if !strings.Contains(string(toolsList.Result), "sharecrop.get_my_budget") {
		t.Fatalf("tools/list missing get_my_budget for a narrowly scoped credential: %s", string(toolsList.Result))
	}
	if budget := readMCPBudget(t, server, created.Secret, session, `2`); budget.WorkSeeking != "work_seeking_disabled" {
		t.Fatalf("work_seeking = %q", budget.WorkSeeking)
	}

	organizationResponse := postJSONWithBearer(t, server.URL+"/api/organizations", []byte(`{"name":"Budget Org"}`), user.AccessToken)
	defer organizationResponse.Body.Close()
	assertStatus(t, organizationResponse, http.StatusCreated)
	organization := decodeOrganizationHTTPResponse(t, organizationResponse)

	credentialResponse := postJSONWithBearer(t, server.URL+"/api/organizations/"+organization.ID+"/credentials", []byte(`{"label":"Budget org automation","scopes":["tasks_read"],"expires_at":""}`), user.AccessToken)
	defer credentialResponse.Body.Close()
	assertStatus(t, credentialResponse, http.StatusCreated)
	orgCredential := decodeOrgCredentialCreatedHTTPResponse(t, credentialResponse)
	orgSession := initializeMCPSession(t, server, orgCredential.Secret)

	code, message, _ := mcpToolFailure(t, decodeRPC(t, mcpCall(t, server, orgCredential.Secret, orgSession, `3`, "sharecrop.get_my_budget", `{}`)))
	if code != "permission_denied" {
		t.Fatalf("code = %q (%s), want permission_denied", code, message)
	}
	if !strings.Contains(message, "organization credentials do not carry work policies") {
		t.Fatalf("message = %q", message)
	}
}

// TestMCPInitializeInstructionsCoverTheBudgetModel pins the cold-start
// orientation: the instructions name the default-disabled state, the budget
// tool, the budget_exceeded stop rule, and the advisory token budget.
func TestMCPInitializeInstructionsCoverTheBudgetModel(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	user := registerUser(t, server, "budget-instructions-user")
	agentToken := createAgentCredential(t, server, user.AccessToken, []string{"tasks_read"})

	response := mcpRequest(t, server, agentToken, "", `1`, "initialize", `{}`)
	envelope := decodeRPC(t, response)
	var result struct {
		Instructions string `json:"instructions"`
	}
	if err := json.Unmarshal(envelope.Result, &result); err != nil {
		t.Fatalf("decode initialize result: %v", err)
	}
	for _, expected := range []string{
		"sharecrop.get_my_budget",
		"work-seeking disabled",
		"budget_exceeded",
		"resets_at",
		"advisory",
	} {
		if !strings.Contains(result.Instructions, expected) {
			t.Fatalf("instructions missing %q:\n%s", expected, result.Instructions)
		}
	}
}

// enableWorkPolicy applies a full work-policy body over REST, for the budget
// dimensions enableWorkSeeking's daily-task-only shorthand does not cover.
func enableWorkPolicy(t *testing.T, server *httptest.Server, user authHTTPResponse, credentialID string, body string) {
	t.Helper()
	response := putWorkPolicyOverHTTP(t, server, user.AccessToken, credentialID, body)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
}

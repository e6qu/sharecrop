package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
)

func putWorkPolicy(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	credentialID := core.NewAgentCredentialID().(core.AgentCredentialIDCreated).Value.String()
	request := httptest.NewRequest(http.MethodPut, "/api/agent-credentials/"+credentialID+"/work-policy", strings.NewReader(body))
	request.SetPathValue("credential_id", credentialID)
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	return response
}

func decodeWorkPolicyError(t *testing.T, response *httptest.ResponseRecorder) errorResponse {
	t.Helper()
	var body errorResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	return body
}

func TestConfigureWorkPolicyEnableRoundTrip(t *testing.T) {
	response := putWorkPolicy(t, `{
		"work_seeking": "work_seeking_enabled",
		"max_tasks_per_day": 20,
		"max_concurrent_reservations": 3,
		"max_credits_per_day": 50,
		"task_types": ["code_review", "research"],
		"min_reward_credits": 5,
		"token_budget_tokens": 200000,
		"token_budget_note": "Advisory only."
	}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (%s)", response.Code, http.StatusOK, response.Body.String())
	}
	var body agentCredentialResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	policy := body.WorkPolicy
	if policy.WorkSeeking != "work_seeking_enabled" || policy.MaxTasksPerDay != 20 || policy.MaxConcurrentReservations != 3 || policy.MaxCreditsPerDay != 50 || policy.MinRewardCredits != 5 || policy.TokenBudgetTokens != 200000 || policy.TokenBudgetNote != "Advisory only." {
		t.Fatalf("policy did not round-trip: %+v", policy)
	}
	if len(policy.TaskTypes) != 2 || policy.TaskTypes[0] != "code_review" || policy.TaskTypes[1] != "research" {
		t.Fatalf("task types = %v, want [code_review research]", policy.TaskTypes)
	}
	if body.TasksUsedToday != 0 || body.CreditsSpentToday != 0 || body.ActiveReservations != 0 {
		t.Fatalf("fresh consumption = %d/%d/%d, want zeros", body.TasksUsedToday, body.CreditsSpentToday, body.ActiveReservations)
	}
}

func TestConfigureWorkPolicyMinimalEnableDefaultsToUnlimited(t *testing.T) {
	response := putWorkPolicy(t, `{"work_seeking":"work_seeking_enabled","max_tasks_per_day":1}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (%s)", response.Code, http.StatusOK, response.Body.String())
	}
	var body agentCredentialResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	policy := body.WorkPolicy
	if policy.MaxTasksPerDay != 1 || policy.MaxConcurrentReservations != 0 || policy.MaxCreditsPerDay != 0 || policy.MinRewardCredits != 0 || policy.TokenBudgetTokens != 0 || policy.TokenBudgetNote != "" {
		t.Fatalf("absent allowances did not default to 0/empty: %+v", policy)
	}
	if len(policy.TaskTypes) != 0 {
		t.Fatalf("task types = %v, want empty (all types)", policy.TaskTypes)
	}
}

func TestConfigureWorkPolicyDisableClearsAllowances(t *testing.T) {
	// Allowance fields sent alongside a disable are ignored; the stored
	// policy is plain disabled.
	response := putWorkPolicy(t, `{"work_seeking":"work_seeking_disabled","max_tasks_per_day":5,"max_credits_per_day":10}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (%s)", response.Code, http.StatusOK, response.Body.String())
	}
	var body agentCredentialResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.WorkPolicy.WorkSeeking != "work_seeking_disabled" || body.WorkPolicy.MaxTasksPerDay != 0 || body.WorkPolicy.MaxCreditsPerDay != 0 {
		t.Fatalf("disable did not clear allowances: %+v", body.WorkPolicy)
	}
}

func TestConfigureWorkPolicyValidationMatrix(t *testing.T) {
	cases := []struct {
		name string
		body string
		code string
	}{
		{name: "missing work_seeking", body: `{"max_tasks_per_day":5}`, code: "invalid_enum"},
		{name: "unknown work_seeking", body: `{"work_seeking":"sometimes"}`, code: "invalid_enum"},
		{name: "enabled without daily budget", body: `{"work_seeking":"work_seeking_enabled"}`, code: "invalid_argument"},
		{name: "daily budget too large", body: `{"work_seeking":"work_seeking_enabled","max_tasks_per_day":10001}`, code: "invalid_argument"},
		{name: "negative concurrency", body: `{"work_seeking":"work_seeking_enabled","max_tasks_per_day":5,"max_concurrent_reservations":-1}`, code: "invalid_argument"},
		{name: "concurrency too large", body: `{"work_seeking":"work_seeking_enabled","max_tasks_per_day":5,"max_concurrent_reservations":1001}`, code: "invalid_argument"},
		{name: "negative spend cap", body: `{"work_seeking":"work_seeking_enabled","max_tasks_per_day":5,"max_credits_per_day":-3}`, code: "invalid_argument"},
		{name: "unknown task type", body: `{"work_seeking":"work_seeking_enabled","max_tasks_per_day":5,"task_types":["poetry"]}`, code: "invalid_enum"},
		{name: "note without tokens", body: `{"work_seeking":"work_seeking_enabled","max_tasks_per_day":5,"token_budget_note":"pace yourself"}`, code: "invalid_argument"},
		{name: "negative token budget", body: `{"work_seeking":"work_seeking_enabled","max_tasks_per_day":5,"token_budget_tokens":-1}`, code: "invalid_argument"},
		{name: "malformed body", body: `{"work_seeking":`, code: "invalid_argument"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			response := putWorkPolicy(t, testCase.body)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d (%s)", response.Code, http.StatusBadRequest, response.Body.String())
			}
			if body := decodeWorkPolicyError(t, response); body.Code != testCase.code {
				t.Fatalf("code = %q, want %q (%s)", body.Code, testCase.code, body.Error)
			}
		})
	}
}

func TestConfigureWorkPolicyRequiresUserSession(t *testing.T) {
	credentialID := core.NewAgentCredentialID().(core.AgentCredentialIDCreated).Value.String()
	request := httptest.NewRequest(http.MethodPut, "/api/agent-credentials/"+credentialID+"/work-policy", strings.NewReader(`{"work_seeking":"work_seeking_disabled"}`))
	request.SetPathValue("credential_id", credentialID)
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestConfigureWorkPolicyRejectsMalformedCredentialID(t *testing.T) {
	request := httptest.NewRequest(http.MethodPut, "/api/agent-credentials/not-an-id/work-policy", strings.NewReader(`{"work_seeking":"work_seeking_disabled"}`))
	request.SetPathValue("credential_id", "not-an-id")
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

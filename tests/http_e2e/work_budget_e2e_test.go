//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestVerificationGatedGrantOverHTTP walks the registration/verification flow
// over the API: registration alone leaves a zero balance, verification lands
// the 100-credit grant exactly once, and re-verifying does not double-grant.
func TestVerificationGatedGrantOverHTTP(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	user := registerUnverifiedUserWithEmail(t, server, "grant-gate-"+uniqueTestSuffix(t)+"@example.com")
	if balance := getBalance(t, server, user.AccessToken); balance.SpendableCredits != 0 {
		t.Fatalf("unverified balance = %d, want 0", balance.SpendableCredits)
	}

	verifyRegisteredUser(t, server, user)
	if balance := getBalance(t, server, user.AccessToken); balance.SpendableCredits != 100 {
		t.Fatalf("verified balance = %d, want 100", balance.SpendableCredits)
	}

	verifyRegisteredUser(t, server, user)
	if balance := getBalance(t, server, user.AccessToken); balance.SpendableCredits != 100 {
		t.Fatalf("re-verified balance = %d, want exactly 100", balance.SpendableCredits)
	}
}

// TestWorkerEndpointsRefuseDisabledCredential pins the default-deny contract
// over REST: a freshly minted credential cannot reserve (403
// permission_denied) even though its scopes would otherwise allow it.
func TestWorkerEndpointsRefuseDisabledCredential(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "deny-owner")
	worker := registerUser(t, server, "deny-worker")
	task := createReservationRequiredTask(t, server, owner)

	fresh := createAgentCredential(t, server, worker.AccessToken, []string{"tasks_read", "submissions_write"})
	response := postJSONWithBearer(t, server.URL+"/api/tasks/"+task+"/reservations", []byte(`{}`), fresh)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusForbidden)
	code, message := decodeErrorCode(t, response)
	if code != "permission_denied" {
		t.Fatalf("code = %q (%s), want permission_denied", code, message)
	}
	if !strings.Contains(message, "work-seeking") {
		t.Fatalf("message %q does not explain work-seeking", message)
	}
}

// TestBudgetRefusalSurfaces429OnWorkerEndpoints exhausts a 1-task daily
// budget: the first reservation succeeds, the second surfaces HTTP 429 with
// the budget_exceeded code, and a direct submission through the same
// credential is likewise refused (the budget window is shared).
func TestBudgetRefusalSurfaces429OnWorkerEndpoints(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "budget-owner")
	worker := registerUser(t, server, "budget-worker")
	firstTask := createReservationRequiredTask(t, server, owner)
	secondTask := createReservationRequiredTask(t, server, owner)

	created := createAgentCredentialResponse(t, server, worker.AccessToken, []string{"tasks_read", "submissions_write"})
	enableWorkSeeking(t, server, worker, created.Credential.ID, 1)

	first := postJSONWithBearer(t, server.URL+"/api/tasks/"+firstTask+"/reservations", []byte(`{}`), created.Secret)
	defer first.Body.Close()
	assertStatus(t, first, http.StatusCreated)

	second := postJSONWithBearer(t, server.URL+"/api/tasks/"+secondTask+"/reservations", []byte(`{}`), created.Secret)
	defer second.Body.Close()
	assertStatus(t, second, http.StatusTooManyRequests)
	code, message := decodeErrorCode(t, second)
	if code != "budget_exceeded" {
		t.Fatalf("code = %q (%s), want budget_exceeded", code, message)
	}
	if !strings.Contains(message, "daily task budget exhausted") || !strings.Contains(message, "resets at 00:00 UTC") {
		t.Fatalf("message %q does not carry the budget wording", message)
	}

	// A direct submission to an open task consumes the same daily window, so
	// it is refused too.
	openTaskID := createOpenPublicTask(t, server, owner)
	submit := postJSONWithBearer(t, server.URL+"/api/tasks/"+openTaskID+"/submissions", []byte(`{"response_json":"{}"}`), created.Secret)
	defer submit.Body.Close()
	assertStatus(t, submit, http.StatusTooManyRequests)
	submitCode, _ := decodeErrorCode(t, submit)
	if submitCode != "budget_exceeded" {
		t.Fatalf("submit code = %q, want budget_exceeded", submitCode)
	}
}

func decodeErrorCode(t *testing.T, response *http.Response) (code string, message string) {
	t.Helper()
	var body struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	return body.Code, body.Error
}

// createReservationRequiredTask creates and opens a public,
// reservation-required task with a freeform schema.
func createReservationRequiredTask(t *testing.T, server *httptest.Server, owner authHTTPResponse) string {
	t.Helper()
	body := `{
		"owner":{"kind":"user","user_id":"` + owner.SubjectID + `","team_id":"","organization_id":""},
		"title":"Reservation required work",
		"description":"Reserve before submitting.",
		"reward":{"kind":"none","credit_amount":0},
		"participation":{"policy":"reservation_required","assignee_scope":"user","reservation_expiry_hours":48},
		"visibility":{"kind":"public","user_id":"","team_id":"","organization_id":""},
		"placement":{"kind":"standalone","series_id":"","series_title":"","series_position":0},
		"response_schema_json":"{\"kind\":\"freeform\"}",
		"payload":{"kind":"none","json":""}
	}`
	response := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(body), owner.AccessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	created := decodeTaskHTTPResponse(t, response)
	openTask(t, server, owner.AccessToken, created.ID)
	return created.ID
}

func createOpenPublicTask(t *testing.T, server *httptest.Server, owner authHTTPResponse) string {
	t.Helper()
	created := createPublicUserTask(t, server, owner)
	openTask(t, server, owner.AccessToken, created.ID)
	return created.ID
}

// agentCredentialFullHTTPResponse mirrors the full credential DTO including
// the stored work policy and today's consumption.
type agentCredentialFullHTTPResponse struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	State      string `json:"state"`
	TaskID     string `json:"task_id"`
	WorkPolicy struct {
		WorkSeeking               string   `json:"work_seeking"`
		MaxTasksPerDay            int64    `json:"max_tasks_per_day"`
		MaxConcurrentReservations int64    `json:"max_concurrent_reservations"`
		MaxCreditsPerDay          int64    `json:"max_credits_per_day"`
		TaskTypes                 []string `json:"task_types"`
		MinRewardCredits          int64    `json:"min_reward_credits"`
		TokenBudgetTokens         int64    `json:"token_budget_tokens"`
		TokenBudgetNote           string   `json:"token_budget_note"`
	} `json:"work_policy"`
	TasksUsedToday     int64 `json:"tasks_used_today"`
	CreditsSpentToday  int64 `json:"credits_spent_today"`
	ActiveReservations int64 `json:"active_reservations"`
}

func putWorkPolicyOverHTTP(t *testing.T, server *httptest.Server, accessToken string, credentialID string, body string) *http.Response {
	t.Helper()
	return putJSONWithBearer(t, server.URL+"/api/agent-credentials/"+credentialID+"/work-policy", []byte(body), accessToken)
}

func decodeCredentialFull(t *testing.T, response *http.Response) agentCredentialFullHTTPResponse {
	t.Helper()
	var body agentCredentialFullHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode credential response: %v", err)
	}
	return body
}

func listCredentialsFull(t *testing.T, server *httptest.Server, accessToken string) []agentCredentialFullHTTPResponse {
	t.Helper()
	response := getWithBearer(t, server.URL+"/api/agent-credentials", accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
	var body struct {
		Credentials []agentCredentialFullHTTPResponse `json:"credentials"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode credentials list: %v", err)
	}
	return body.Credentials
}

func credentialFullByID(t *testing.T, values []agentCredentialFullHTTPResponse, id string) agentCredentialFullHTTPResponse {
	t.Helper()
	for _, value := range values {
		if value.ID == id {
			return value
		}
	}
	t.Fatalf("credential %s missing from listing", id)
	return agentCredentialFullHTTPResponse{}
}

// TestWorkPolicyEndpointRoundTrip covers PUT
// /api/agent-credentials/{id}/work-policy: enabling with every allowance
// returns and stores the policy, the listing repeats it, disabling clears it,
// a non-owner cannot reach the credential, and a task-scoped worker
// credential rejects a policy outright.
func TestWorkPolicyEndpointRoundTrip(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "policy-owner")
	stranger := registerUser(t, server, "policy-stranger")
	created := createAgentCredentialResponse(t, server, owner.AccessToken, []string{"tasks_read", "submissions_write"})

	// The fresh credential lists as work_seeking_disabled with zero usage.
	fresh := credentialFullByID(t, listCredentialsFull(t, server, owner.AccessToken), created.Credential.ID)
	if fresh.WorkPolicy.WorkSeeking != "work_seeking_disabled" {
		t.Fatalf("fresh policy = %q, want work_seeking_disabled", fresh.WorkPolicy.WorkSeeking)
	}
	if fresh.TasksUsedToday != 0 || fresh.CreditsSpentToday != 0 || fresh.ActiveReservations != 0 {
		t.Fatalf("fresh consumption = %d/%d/%d, want zeros", fresh.TasksUsedToday, fresh.CreditsSpentToday, fresh.ActiveReservations)
	}

	enableBody := `{
		"work_seeking": "work_seeking_enabled",
		"max_tasks_per_day": 12,
		"max_concurrent_reservations": 2,
		"max_credits_per_day": 40,
		"task_types": ["code_review", "research"],
		"min_reward_credits": 5,
		"token_budget_tokens": 150000,
		"token_budget_note": "Advisory: stop when the tokens run out."
	}`
	enabledResponse := putWorkPolicyOverHTTP(t, server, owner.AccessToken, created.Credential.ID, enableBody)
	defer enabledResponse.Body.Close()
	assertStatus(t, enabledResponse, http.StatusOK)
	enabled := decodeCredentialFull(t, enabledResponse)
	policy := enabled.WorkPolicy
	if policy.WorkSeeking != "work_seeking_enabled" || policy.MaxTasksPerDay != 12 || policy.MaxConcurrentReservations != 2 || policy.MaxCreditsPerDay != 40 || policy.MinRewardCredits != 5 || policy.TokenBudgetTokens != 150000 || policy.TokenBudgetNote != "Advisory: stop when the tokens run out." {
		t.Fatalf("enabled policy did not round-trip: %+v", policy)
	}
	if len(policy.TaskTypes) != 2 {
		t.Fatalf("task types = %v, want two entries", policy.TaskTypes)
	}

	// The listing repeats the stored policy.
	listed := credentialFullByID(t, listCredentialsFull(t, server, owner.AccessToken), created.Credential.ID)
	if fmt.Sprintf("%+v", listed.WorkPolicy) != fmt.Sprintf("%+v", enabled.WorkPolicy) {
		t.Fatalf("listed policy %+v != configured policy %+v", listed.WorkPolicy, enabled.WorkPolicy)
	}

	// Another user's session cannot reach the credential: 404, so the
	// endpoint does not reveal whether the id exists.
	strangerResponse := putWorkPolicyOverHTTP(t, server, stranger.AccessToken, created.Credential.ID, `{"work_seeking":"work_seeking_disabled"}`)
	defer strangerResponse.Body.Close()
	assertStatus(t, strangerResponse, http.StatusNotFound)
	if code, _ := decodeErrorCode(t, strangerResponse); code != "not_found" {
		t.Fatalf("stranger code = %q, want not_found", code)
	}

	// Disabling clears every stored allowance.
	disabledResponse := putWorkPolicyOverHTTP(t, server, owner.AccessToken, created.Credential.ID, `{"work_seeking":"work_seeking_disabled"}`)
	defer disabledResponse.Body.Close()
	assertStatus(t, disabledResponse, http.StatusOK)
	disabled := decodeCredentialFull(t, disabledResponse)
	if disabled.WorkPolicy.WorkSeeking != "work_seeking_disabled" || disabled.WorkPolicy.MaxTasksPerDay != 0 || disabled.WorkPolicy.MaxCreditsPerDay != 0 || len(disabled.WorkPolicy.TaskTypes) != 0 || disabled.WorkPolicy.TokenBudgetTokens != 0 {
		t.Fatalf("disable did not clear allowances: %+v", disabled.WorkPolicy)
	}

	// A task-scoped worker credential (auto-issued on reservation) can never
	// carry a policy.
	taskID := createReservationRequiredTask(t, server, owner)
	reserve := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskID+"/reservations", []byte(`{}`), stranger.AccessToken)
	defer reserve.Body.Close()
	assertStatus(t, reserve, http.StatusCreated)
	taskScopedID := ""
	for _, value := range listCredentialsFull(t, server, stranger.AccessToken) {
		if value.TaskID == taskID {
			taskScopedID = value.ID
		}
	}
	if taskScopedID == "" {
		t.Fatalf("reservation did not auto-issue a task-scoped credential")
	}
	taskScopedResponse := putWorkPolicyOverHTTP(t, server, stranger.AccessToken, taskScopedID, `{"work_seeking":"work_seeking_enabled","max_tasks_per_day":5}`)
	defer taskScopedResponse.Body.Close()
	assertStatus(t, taskScopedResponse, http.StatusBadRequest)
	if code, message := decodeErrorCode(t, taskScopedResponse); code != "invalid_argument" || !strings.Contains(message, "task-scoped") {
		t.Fatalf("task-scoped rejection = %q (%s)", code, message)
	}
}

// TestWorkPolicyEndpointValidation pins the request validation over HTTP:
// enum and argument failures answer 400 with the matching code.
func TestWorkPolicyEndpointValidation(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()
	owner := registerUser(t, server, "policy-validation")
	created := createAgentCredentialResponse(t, server, owner.AccessToken, []string{"tasks_read"})

	cases := []struct {
		name string
		body string
		code string
	}{
		{name: "unknown state", body: `{"work_seeking":"maybe"}`, code: "invalid_enum"},
		{name: "enabled without budget", body: `{"work_seeking":"work_seeking_enabled"}`, code: "invalid_argument"},
		{name: "budget too large", body: `{"work_seeking":"work_seeking_enabled","max_tasks_per_day":10001}`, code: "invalid_argument"},
		{name: "unknown task type", body: `{"work_seeking":"work_seeking_enabled","max_tasks_per_day":5,"task_types":["poetry"]}`, code: "invalid_enum"},
		{name: "note without tokens", body: `{"work_seeking":"work_seeking_enabled","max_tasks_per_day":5,"token_budget_note":"pace"}`, code: "invalid_argument"},
	}
	for _, testCase := range cases {
		response := putWorkPolicyOverHTTP(t, server, owner.AccessToken, created.Credential.ID, testCase.body)
		assertStatus(t, response, http.StatusBadRequest)
		if code, message := decodeErrorCode(t, response); code != testCase.code {
			t.Fatalf("%s: code = %q (%s), want %q", testCase.name, code, message, testCase.code)
		}
		response.Body.Close()
	}
}

// TestConsumptionMovesOnCredentialListing drives real consumption through a
// work-seeking credential and watches the listing's counters move: a
// reservation bumps tasks_used_today and active_reservations, and a capped
// MCP credit send bumps credits_spent_today.
func TestConsumptionMovesOnCredentialListing(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "consumption-owner")
	worker := registerUser(t, server, "consumption-worker")
	receiver := registerUser(t, server, "consumption-receiver")

	created := createAgentCredentialResponse(t, server, worker.AccessToken, []string{"tasks_read", "submissions_write", "ledger_read", "ledger_write"})
	enableResponse := putWorkPolicyOverHTTP(t, server, worker.AccessToken, created.Credential.ID,
		`{"work_seeking":"work_seeking_enabled","max_tasks_per_day":10,"max_credits_per_day":50}`)
	defer enableResponse.Body.Close()
	assertStatus(t, enableResponse, http.StatusOK)

	// Reserve through the credential: one daily-task unit, one active
	// reservation.
	taskID := createReservationRequiredTask(t, server, owner)
	reserve := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskID+"/reservations", []byte(`{}`), created.Secret)
	defer reserve.Body.Close()
	assertStatus(t, reserve, http.StatusCreated)

	// Spend through the credential over MCP (REST spend routes require a
	// user session): 5 credits against the 50-credit day cap.
	session := initializeMCPSession(t, server, created.Secret)
	arguments := `{"source_kind":"self","target_kind":"user","target_id":"` + receiver.SubjectID + `","amount":5,"note":"","idempotency_key":"consumption-send-1"}`
	toolText(t, decodeRPC(t, mcpCall(t, server, created.Secret, session, `1`, "sharecrop.send_credits", arguments)))

	after := credentialFullByID(t, listCredentialsFull(t, server, worker.AccessToken), created.Credential.ID)
	if after.TasksUsedToday != 1 {
		t.Fatalf("tasks_used_today = %d, want 1", after.TasksUsedToday)
	}
	if after.ActiveReservations != 1 {
		t.Fatalf("active_reservations = %d, want 1", after.ActiveReservations)
	}
	if after.CreditsSpentToday != 5 {
		t.Fatalf("credits_spent_today = %d, want 5", after.CreditsSpentToday)
	}
}

type opsCountersHTTPResponse struct {
	OutboxRecordedBacklog          int64 `json:"outbox_recorded_backlog"`
	OutboxDispatchFailed           int64 `json:"outbox_dispatch_failed"`
	WebhookDeliveriesPending       int64 `json:"webhook_deliveries_pending"`
	WebhookDeliveriesDead          int64 `json:"webhook_deliveries_dead"`
	OldestPendingWebhookAgeSeconds int64 `json:"oldest_pending_webhook_age_seconds"`
	SignupGrantsToday              int64 `json:"signup_grants_today"`
	PeerTransfersToday             int64 `json:"peer_transfers_today"`
	PeerTransferCreditsToday       int64 `json:"peer_transfer_credits_today"`
	BudgetRefusalsToday            int64 `json:"budget_refusals_today"`
}

// TestOperationsCountersEndpointOverHTTP covers GET
// /api/admin/operations/counters end to end: a non-admin is refused, the
// admin reads a snapshot, and the day totals move with real activity (signup
// grants and a budget refusal).
func TestOperationsCountersEndpointOverHTTP(t *testing.T) {
	bootstrap := newAuthHTTPServer(t, t.Context())
	admin := registerUser(t, bootstrap, "ops-admin")
	bootstrap.Close()

	t.Setenv("SHARECROP_ADMIN_USER_IDS", admin.SubjectID)
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	// A non-admin session is refused.
	outsider := registerUser(t, server, "ops-outsider")
	denied := getWithBearer(t, server.URL+"/api/admin/operations/counters", outsider.AccessToken)
	defer denied.Body.Close()
	assertStatus(t, denied, http.StatusForbidden)

	before := readOpsCounters(t, server, admin.AccessToken)

	// One verified registration lands one signup grant; a 1-task budget
	// exhausted by a second reservation lands one budget refusal.
	worker := registerUser(t, server, "ops-worker")
	requester := registerUser(t, server, "ops-requester")
	created := createAgentCredentialResponse(t, server, worker.AccessToken, []string{"tasks_read", "submissions_write"})
	enableWorkSeeking(t, server, worker, created.Credential.ID, 1)
	firstTask := createReservationRequiredTask(t, server, requester)
	secondTask := createReservationRequiredTask(t, server, requester)
	first := postJSONWithBearer(t, server.URL+"/api/tasks/"+firstTask+"/reservations", []byte(`{}`), created.Secret)
	defer first.Body.Close()
	assertStatus(t, first, http.StatusCreated)
	second := postJSONWithBearer(t, server.URL+"/api/tasks/"+secondTask+"/reservations", []byte(`{}`), created.Secret)
	defer second.Body.Close()
	assertStatus(t, second, http.StatusTooManyRequests)

	after := readOpsCounters(t, server, admin.AccessToken)
	// worker, requester, and the outsider registered (verified) since the
	// "before" read... outsider registered before it, so at least 2 grants.
	if after.SignupGrantsToday < before.SignupGrantsToday+2 {
		t.Fatalf("signup_grants_today = %d, want at least %d", after.SignupGrantsToday, before.SignupGrantsToday+2)
	}
	if after.BudgetRefusalsToday < before.BudgetRefusalsToday+1 {
		t.Fatalf("budget_refusals_today = %d, want at least %d", after.BudgetRefusalsToday, before.BudgetRefusalsToday+1)
	}
	if after.OutboxRecordedBacklog < 0 || after.OldestPendingWebhookAgeSeconds < 0 {
		t.Fatalf("negative aggregate: %+v", after)
	}
}

func readOpsCounters(t *testing.T, server *httptest.Server, accessToken string) opsCountersHTTPResponse {
	t.Helper()
	response := getWithBearer(t, server.URL+"/api/admin/operations/counters", accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
	var body opsCountersHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode ops counters: %v", err)
	}
	return body
}

// TestVerificationStateOnSessionAndProfile pins the API support for the
// "verify your email to receive your signup grant" prompt: registration and
// profile report unverified, verification flips the profile (and subsequent
// logins) to verified.
func TestVerificationStateOnSessionAndProfile(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	email := "verify-state-" + uniqueTestSuffix(t) + "@example.com"
	user := registerUnverifiedUserWithEmail(t, server, email)
	if user.EmailVerificationState != "unverified" {
		t.Fatalf("registration email_verification_state = %q, want unverified", user.EmailVerificationState)
	}
	if state := readProfileVerificationState(t, server, user.AccessToken); state != "unverified" {
		t.Fatalf("profile state before verification = %q, want unverified", state)
	}

	verifyRegisteredUser(t, server, user)
	if state := readProfileVerificationState(t, server, user.AccessToken); state != "verified" {
		t.Fatalf("profile state after verification = %q, want verified", state)
	}

	login := postAuthJSON(t, server.URL+"/api/auth/login", authHTTPRequest{Email: email, Password: "correct horse battery staple"}, nil)
	defer login.Body.Close()
	assertStatus(t, login, http.StatusOK)
	if session := decodeAuthHTTPResponse(t, login); session.EmailVerificationState != "verified" {
		t.Fatalf("login email_verification_state = %q, want verified", session.EmailVerificationState)
	}
}

func readProfileVerificationState(t *testing.T, server *httptest.Server, accessToken string) string {
	t.Helper()
	response := getWithBearer(t, server.URL+"/api/account/profile", accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
	var body struct {
		EmailVerificationState string `json:"email_verification_state"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	return body.EmailVerificationState
}

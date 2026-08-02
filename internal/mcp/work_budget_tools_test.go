package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/task"
)

// refusingWorkerServices refuses worker mutations exactly the way the task and
// submission services refuse a credential whose owner never enabled
// work-seeking, so the MCP guidance decoration is observable.
type refusingWorkerServices struct {
	fakeServices
}

func (services refusingWorkerServices) ReserveTask(_ context.Context, _ auth.UserSubject, _ task.WorkerOrigin, _ core.TaskID) task.ReservationResult {
	return task.ReservationRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "agent credential is not enabled for work-seeking; its owner must enable it with a work budget")}
}

func testCredentialID(t *testing.T) core.AgentCredentialID {
	t.Helper()
	created, matched := core.NewAgentCredentialID().(core.AgentCredentialIDCreated)
	if !matched {
		t.Fatalf("agent credential id rejected")
	}
	return created.Value
}

func mustCreditAmount(t *testing.T, value int64) ledger.CreditAmount {
	t.Helper()
	accepted, matched := ledger.NewCreditAmount(value).(ledger.CreditAmountAccepted)
	if !matched {
		t.Fatalf("credit amount %d rejected", value)
	}
	return accepted.Value
}

// fullyEnabledPolicy is an enabled work policy with every allowance set, so
// the budget tool's flattening is exercised in one call.
func fullyEnabledPolicy(t *testing.T) agent.WorkPolicy {
	t.Helper()
	budget, budgetMatched := agent.NewDailyTaskBudget(5).(agent.DailyTaskBudgetAccepted)
	if !budgetMatched {
		t.Fatalf("daily task budget rejected")
	}
	concurrent, concurrentMatched := agent.NewConcurrentReservationCap(2).(agent.ConcurrentReservationCapAccepted)
	if !concurrentMatched {
		t.Fatalf("concurrent reservation cap rejected")
	}
	types, typesMatched := agent.NewTaskTypesLimited([]task.TaskType{task.TaskTypeCodeReview, task.TaskTypeResearch}).(agent.TaskTypesLimitedAccepted)
	if !typesMatched {
		t.Fatalf("task type restriction rejected")
	}
	tokens, tokensMatched := agent.NewTokenBudgetTokens(150000).(agent.TokenBudgetTokensAccepted)
	if !tokensMatched {
		t.Fatalf("advisory token budget rejected")
	}
	note, noteMatched := agent.NewTokenBudgetNote("Stop when the tokens run out.").(agent.TokenBudgetNoteAccepted)
	if !noteMatched {
		t.Fatalf("advisory token budget note rejected")
	}
	return agent.WorkPolicyEnabled{Allowances: agent.WorkAllowances{
		MaxTasksPerDay:         budget.Value,
		ConcurrentReservations: agent.ConcurrentReservationsCapped{Limit: concurrent.Value},
		DailySpend:             agent.DailySpendCapped{Limit: mustCreditAmount(t, 40)},
		TaskTypes:              types.Value,
		RewardFloor:            agent.RewardFloorAtLeast{Minimum: mustCreditAmount(t, 5)},
		TokenBudget:            agent.TokenBudgetAdvised{Tokens: tokens.Value, Note: note.Value},
	}}
}

func getMyBudget(t *testing.T, server Server, credential CallerCredential) myBudgetPayload {
	t.Helper()
	content := decodeToolText(t, server.Handle(context.Background(), testSubject(t), credential, request(`1`, "tools/call", `{"name":"sharecrop.get_my_budget","arguments":{}}`)))
	var payload myBudgetPayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("decode budget payload: %v (%s)", err, content)
	}
	return payload
}

// TestGetMyBudgetReportsEveryAllowanceAndDerivesRemaining pins the tool shape
// for an enabled credential: every configured allowance is reported next to
// what it has consumed today, tasks_remaining_today is derived server-side,
// and resets_at is the next UTC midnight.
func TestGetMyBudgetReportsEveryAllowanceAndDerivesRemaining(t *testing.T) {
	credentialID := testCredentialID(t)
	server := NewServer(fakeServices{workActivity: agent.CredentialWorkActivity{
		CredentialID:       credentialID,
		TasksToday:         2,
		CreditsSpentToday:  12,
		ActiveReservations: 1,
	}})
	credential := CallerCredential{Scopes: allScopes(), Budget: PersonalCredentialBudget{CredentialID: credentialID, Policy: fullyEnabledPolicy(t)}}

	payload := getMyBudget(t, server, credential)
	if payload.CredentialKind != personalCredentialKind {
		t.Fatalf("credential_kind = %q, want %q", payload.CredentialKind, personalCredentialKind)
	}
	if payload.WorkSeeking != agent.WorkSeekingEnabled.String() {
		t.Fatalf("work_seeking = %q", payload.WorkSeeking)
	}
	if payload.MaxTasksPerDay != 5 || payload.TasksUsedToday != 2 || payload.TasksRemainingToday != 3 {
		t.Fatalf("daily task budget = %d used %d remaining %d, want 5/2/3", payload.MaxTasksPerDay, payload.TasksUsedToday, payload.TasksRemainingToday)
	}
	if payload.MaxConcurrentReservations != 2 || payload.ActiveReservations != 1 {
		t.Fatalf("concurrency = %d of %d, want 1 of 2", payload.ActiveReservations, payload.MaxConcurrentReservations)
	}
	if payload.MaxCreditsPerDay != 40 || payload.CreditsSpentToday != 12 {
		t.Fatalf("spend = %d of %d, want 12 of 40", payload.CreditsSpentToday, payload.MaxCreditsPerDay)
	}
	if len(payload.TaskTypes) != 2 || payload.TaskTypes[0] != task.TaskTypeCodeReview.String() {
		t.Fatalf("task_types = %v", payload.TaskTypes)
	}
	if payload.MinRewardCredits != 5 || payload.TokenBudgetTokens != 150000 || payload.TokenBudgetNote == "" {
		t.Fatalf("floor/token budget wrong: %+v", payload)
	}
	resets, err := time.Parse(time.RFC3339, payload.ResetsAt)
	if err != nil {
		t.Fatalf("resets_at %q is not RFC3339: %v", payload.ResetsAt, err)
	}
	if resets.UTC() != nextUTCMidnight(time.Now()) {
		t.Fatalf("resets_at = %s, want the next UTC midnight %s", resets.UTC(), nextUTCMidnight(time.Now()))
	}
}

// TestGetMyBudgetReportsDisabledCredentialAsNoWorkAllowed pins the default
// state: a fresh credential reports work_seeking_disabled with nothing
// allowed, so an agent can see it may take no work at all before it tries.
func TestGetMyBudgetReportsDisabledCredentialAsNoWorkAllowed(t *testing.T) {
	credentialID := testCredentialID(t)
	server := NewServer(fakeServices{})
	credential := CallerCredential{Scopes: allScopes(), Budget: PersonalCredentialBudget{CredentialID: credentialID, Policy: agent.WorkPolicyDisabled{}}}

	payload := getMyBudget(t, server, credential)
	if payload.WorkSeeking != agent.WorkSeekingDisabled.String() {
		t.Fatalf("work_seeking = %q, want work_seeking_disabled", payload.WorkSeeking)
	}
	if payload.MaxTasksPerDay != 0 || payload.TasksRemainingToday != 0 || payload.MaxCreditsPerDay != 0 || payload.MinRewardCredits != 0 || payload.TokenBudgetTokens != 0 {
		t.Fatalf("disabled policy reported allowances: %+v", payload)
	}
	if len(payload.TaskTypes) != 0 {
		t.Fatalf("task_types = %v, want empty", payload.TaskTypes)
	}
}

// TestGetMyBudgetNeverReportsAnotherCredentialsUsage pins the isolation: the
// activity read is keyed by the calling credential, so another credential's
// consumption reads as zeros rather than leaking.
func TestGetMyBudgetNeverReportsAnotherCredentialsUsage(t *testing.T) {
	server := NewServer(fakeServices{workActivity: agent.CredentialWorkActivity{
		CredentialID:       testCredentialID(t),
		TasksToday:         9,
		CreditsSpentToday:  99,
		ActiveReservations: 9,
	}})
	credential := CallerCredential{Scopes: allScopes(), Budget: PersonalCredentialBudget{CredentialID: testCredentialID(t), Policy: fullyEnabledPolicy(t)}}

	payload := getMyBudget(t, server, credential)
	if payload.TasksUsedToday != 0 || payload.CreditsSpentToday != 0 || payload.ActiveReservations != 0 {
		t.Fatalf("another credential's usage leaked: %+v", payload)
	}
	if payload.TasksRemainingToday != 5 {
		t.Fatalf("tasks_remaining_today = %d, want the full budget of 5", payload.TasksRemainingToday)
	}
}

// TestGetMyBudgetRemainingNeverGoesNegative covers a budget lowered below what
// was already consumed: nothing is left, and the tool never reports a debt.
func TestGetMyBudgetRemainingNeverGoesNegative(t *testing.T) {
	credentialID := testCredentialID(t)
	server := NewServer(fakeServices{workActivity: agent.CredentialWorkActivity{CredentialID: credentialID, TasksToday: 12}})
	credential := CallerCredential{Scopes: allScopes(), Budget: PersonalCredentialBudget{CredentialID: credentialID, Policy: fullyEnabledPolicy(t)}}

	if payload := getMyBudget(t, server, credential); payload.TasksRemainingToday != 0 {
		t.Fatalf("tasks_remaining_today = %d, want 0", payload.TasksRemainingToday)
	}
}

// TestGetMyBudgetRefusesOrganizationCredential pins the org rejection: an
// organization credential carries no work policy, and the refusal says so
// rather than reporting an empty budget.
func TestGetMyBudgetRefusesOrganizationCredential(t *testing.T) {
	server := NewServer(fakeServices{})
	credential := CallerCredential{Scopes: allScopes(), Budget: OrganizationCredentialBudget{}}
	response := server.Handle(context.Background(), auth.OrgSubject{}, credential, request(`1`, "tools/call", `{"name":"sharecrop.get_my_budget","arguments":{}}`))

	code, message := toolFailureBody(t, response)
	if code != core.ErrorCodePermissionDenied.String() {
		t.Fatalf("code = %q, want permission_denied", code)
	}
	if !strings.Contains(message, "organization credentials do not carry work policies") {
		t.Fatalf("message = %q", message)
	}
}

// TestGetMyBudgetReportsTaskScopedCredentialNature pins the task-scoped case:
// the credential reports what it is instead of a budget, since it works
// inside the reservation it was issued for.
func TestGetMyBudgetReportsTaskScopedCredentialNature(t *testing.T) {
	taskID, matched := core.NewTaskID().(core.TaskIDCreated)
	if !matched {
		t.Fatalf("task id rejected")
	}
	server := NewServer(fakeServices{})
	credential := CallerCredential{Scopes: allScopes(), TaskID: &taskID.Value, Budget: TaskScopedCredentialBudget{TaskID: taskID.Value}}
	content := decodeToolText(t, server.Handle(context.Background(), testSubject(t), credential, request(`1`, "tools/call", `{"name":"sharecrop.get_my_budget","arguments":{}}`)))

	var payload taskScopedBudgetPayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("decode task-scoped payload: %v (%s)", err, content)
	}
	if payload.CredentialKind != taskScopedCredentialKind || payload.TaskID != taskID.Value.String() {
		t.Fatalf("task-scoped payload wrong: %+v", payload)
	}
	if !strings.Contains(payload.Guidance, "carries no work policy") {
		t.Fatalf("guidance = %q", payload.Guidance)
	}
}

// TestGetMyBudgetSurfacesActivityReadFailure pins that a failed consumption
// read is reported as a tool failure rather than as a budget of zeros.
func TestGetMyBudgetSurfacesActivityReadFailure(t *testing.T) {
	server := NewServer(fakeServices{rejectWorkActivity: true})
	credential := CallerCredential{Scopes: allScopes(), Budget: PersonalCredentialBudget{CredentialID: testCredentialID(t), Policy: fullyEnabledPolicy(t)}}
	response := server.Handle(context.Background(), testSubject(t), credential, request(`1`, "tools/call", `{"name":"sharecrop.get_my_budget","arguments":{}}`))

	if code, _ := toolFailureBody(t, response); code != core.ErrorCodeUnavailable.String() {
		t.Fatalf("code = %q, want unavailable", code)
	}
}

// TestGetMyBudgetIsListedAndCallableWithoutAnyScope pins the scope-free
// introspection rule: a credential holding no scope at all still sees and can
// call get_my_budget, and sees nothing else.
func TestGetMyBudgetIsListedAndCallableWithoutAnyScope(t *testing.T) {
	credentialID := testCredentialID(t)
	server := NewServer(fakeServices{})
	credential := CallerCredential{Scopes: agent.NewScopeSet([]agent.Scope{}), Budget: PersonalCredentialBudget{CredentialID: credentialID, Policy: agent.WorkPolicyDisabled{}}}

	var listing struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(mustResult(t, server.Handle(context.Background(), testSubject(t), credential, request(`1`, "tools/list", `{}`))), &listing); err != nil {
		t.Fatalf("decode tools/list: %v", err)
	}
	if len(listing.Tools) != 1 || listing.Tools[0].Name != toolGetMyBudget {
		t.Fatalf("scope-free listing = %+v, want only %s", listing.Tools, toolGetMyBudget)
	}

	if payload := getMyBudget(t, server, credential); payload.WorkSeeking != agent.WorkSeekingDisabled.String() {
		t.Fatalf("scope-free call reported %q", payload.WorkSeeking)
	}
}

// TestGetMyBudgetIsListedForEveryCredential pins that the tool is present in
// every credential's listing, whatever scopes it holds.
func TestGetMyBudgetIsListedForEveryCredential(t *testing.T) {
	server := NewServer(fakeServices{})
	for _, scopes := range []agent.ScopeSet{
		agent.NewScopeSet(agent.AllScopes()),
		allScopes(),
		agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead}),
	} {
		var listing struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		}
		credential := CallerCredential{Scopes: scopes, Budget: PersonalCredentialBudget{CredentialID: testCredentialID(t), Policy: agent.WorkPolicyDisabled{}}}
		if err := json.Unmarshal(mustResult(t, server.Handle(context.Background(), testSubject(t), credential, request(`1`, "tools/list", `{}`))), &listing); err != nil {
			t.Fatalf("decode tools/list: %v", err)
		}
		found := false
		for _, tool := range listing.Tools {
			if tool.Name == toolGetMyBudget {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s missing from tools/list for scopes %v", toolGetMyBudget, scopes.Values())
		}
	}
}

// TestDisabledWorkSeekingRefusalCarriesGuidance pins the agent-facing
// guidance: a reserve refused because work-seeking is off keeps the domain
// code and message, and adds the line naming get_my_budget and the human who
// must change the setting.
func TestDisabledWorkSeekingRefusalCarriesGuidance(t *testing.T) {
	credentialID := testCredentialID(t)
	server := NewServer(refusingWorkerServices{})
	credential := CallerCredential{Scopes: allScopes(), Worker: task.WorkerViaCredential{CredentialID: credentialID, Policy: task.CredentialWorkDisabled{}}, Budget: PersonalCredentialBudget{CredentialID: credentialID, Policy: agent.WorkPolicyDisabled{}}}
	response := server.Handle(context.Background(), testSubject(t), credential, request(`1`, "tools/call", `{"name":"sharecrop.reserve_task","arguments":{"task_id":"`+testTaskID(t)+`"}}`))

	body := toolFailureJSON(t, response)
	if body.Code != core.ErrorCodePermissionDenied.String() {
		t.Fatalf("code = %q, want permission_denied", body.Code)
	}
	if !strings.Contains(body.Message, "work-seeking") {
		t.Fatalf("message = %q", body.Message)
	}
	if !strings.Contains(body.Guidance, "your operator has not enabled work-seeking") || !strings.Contains(body.Guidance, toolGetMyBudget) {
		t.Fatalf("guidance = %q", body.Guidance)
	}
}

// TestEnabledCredentialRefusalCarriesNoGuidance pins that the guidance is not
// stamped on unrelated refusals: an enabled credential's permission failure
// stays exactly as the domain worded it.
func TestEnabledCredentialRefusalCarriesNoGuidance(t *testing.T) {
	credentialID := testCredentialID(t)
	server := NewServer(refusingWorkerServices{})
	credential := CallerCredential{Scopes: allScopes(), Budget: PersonalCredentialBudget{CredentialID: credentialID, Policy: fullyEnabledPolicy(t)}}
	response := server.Handle(context.Background(), testSubject(t), credential, request(`1`, "tools/call", `{"name":"sharecrop.reserve_task","arguments":{"task_id":"`+testTaskID(t)+`"}}`))

	if body := toolFailureJSON(t, response); body.Guidance != "" {
		t.Fatalf("guidance = %q, want none", body.Guidance)
	}
}

// toolFailureJSON decodes an isError tool result's single text item.
func toolFailureJSON(t *testing.T, response Response) toolErrorBody {
	t.Helper()
	var result toolCallResult
	if err := json.Unmarshal(mustResult(t, response), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if !result.IsError || len(result.Content) != 1 {
		t.Fatalf("expected an isError tool result with one item, got %+v", result)
	}
	var body toolErrorBody
	if err := json.Unmarshal([]byte(result.Content[0].Text), &body); err != nil {
		t.Fatalf("failure text is not JSON: %q", result.Content[0].Text)
	}
	return body
}

func toolFailureBody(t *testing.T, response Response) (code string, message string) {
	t.Helper()
	body := toolFailureJSON(t, response)
	return body.Code, body.Message
}

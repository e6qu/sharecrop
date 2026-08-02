package agentbridge

import (
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/agent/agenttest"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/task"
)

func sampleCredential(t *testing.T, expiresAt *time.Time) agent.Credential {
	t.Helper()
	id, matched := core.NewAgentCredentialID().(core.AgentCredentialIDCreated)
	if !matched {
		t.Fatalf("agent credential id rejected")
	}
	userID, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("user id rejected")
	}
	label, matched := agent.NewLabel("test-label").(agent.LabelAccepted)
	if !matched {
		t.Fatalf("label rejected")
	}
	scope, matched := agent.ParseScope("tasks_read").(agent.ScopeAccepted)
	if !matched {
		t.Fatalf("scope rejected")
	}
	return agent.Credential{
		ID:        id.Value,
		UserID:    userID.Value,
		Label:     label.Value,
		Scopes:    agent.NewScopeSet([]agent.Scope{scope.Value}),
		State:     agent.StateActive,
		ExpiresAt: expiresAt,
		TaskID:    nil,
	}
}

func assertCredentialEqual(t *testing.T, got, want agent.Credential) {
	t.Helper()
	if diff := agenttest.CredentialDiff(got, want); diff != "" {
		t.Errorf("credential mismatch: %s", diff)
	}
}

func TestCredentialRoundTrip(t *testing.T) {
	expires := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	for _, credential := range []agent.Credential{sampleCredential(t, &expires), sampleCredential(t, nil)} {
		restored, err := decodeCredential(encodeCredential(credential))
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		assertCredentialEqual(t, restored, credential)
	}
}

func TestVerifyRevokeListRoundTrip(t *testing.T) {
	credential := sampleCredential(t, nil)

	verified, err := decodeVerifyResult(encodeVerifyResult(agent.VerifyStoreFound{Value: credential}))
	if err != nil {
		t.Fatalf("decode verify: %v", err)
	}
	if typed, matched := verified.(agent.VerifyStoreFound); !matched {
		t.Fatalf("verify result = %T", verified)
	} else {
		assertCredentialEqual(t, typed.Value, credential)
	}

	revoked, err := decodeRevokeResult(encodeRevokeResult(agent.RevokeStoreRevoked{Value: credential}))
	if err != nil {
		t.Fatalf("decode revoke: %v", err)
	}
	if typed, matched := revoked.(agent.RevokeStoreRevoked); !matched {
		t.Fatalf("revoke result = %T", revoked)
	} else {
		assertCredentialEqual(t, typed.Value, credential)
	}

	listed, err := decodeListResult(encodeListResult(agent.ListStoreListed{Values: []agent.Credential{credential}}))
	if err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if typed, matched := listed.(agent.ListStoreListed); !matched || len(typed.Values) != 1 {
		t.Fatalf("list result = %T", listed)
	} else {
		assertCredentialEqual(t, typed.Values[0], credential)
	}
}

// TestCredentialWorkPolicyRoundTrip pins the work-policy wire codec: a fully
// configured enabled policy and the disabled default both survive the bridge.
func TestCredentialWorkPolicyRoundTrip(t *testing.T) {
	budget, matched := agent.NewDailyTaskBudget(5).(agent.DailyTaskBudgetAccepted)
	if !matched {
		t.Fatalf("daily task budget rejected")
	}
	cap, matched := agent.NewConcurrentReservationCap(2).(agent.ConcurrentReservationCapAccepted)
	if !matched {
		t.Fatalf("concurrent cap rejected")
	}
	spendCap, matched := ledger.NewCreditAmount(60).(ledger.CreditAmountAccepted)
	if !matched {
		t.Fatalf("spend cap rejected")
	}
	floor, matched := ledger.NewCreditAmount(10).(ledger.CreditAmountAccepted)
	if !matched {
		t.Fatalf("reward floor rejected")
	}
	tokens, matched := agent.NewTokenBudgetTokens(500_000).(agent.TokenBudgetTokensAccepted)
	if !matched {
		t.Fatalf("token budget rejected")
	}
	note, matched := agent.NewTokenBudgetNote("stay frugal").(agent.TokenBudgetNoteAccepted)
	if !matched {
		t.Fatalf("token budget note rejected")
	}
	limited, matched := agent.NewTaskTypesLimited([]task.TaskType{task.TaskTypeResearch, task.TaskTypeDataExtraction}).(agent.TaskTypesLimitedAccepted)
	if !matched {
		t.Fatalf("task type restriction rejected")
	}

	credential := sampleCredential(t, nil)
	credential.WorkPolicy = agent.WorkPolicyEnabled{Allowances: agent.WorkAllowances{
		MaxTasksPerDay:         budget.Value,
		ConcurrentReservations: agent.ConcurrentReservationsCapped{Limit: cap.Value},
		DailySpend:             agent.DailySpendCapped{Limit: spendCap.Value},
		TaskTypes:              limited.Value,
		RewardFloor:            agent.RewardFloorAtLeast{Minimum: floor.Value},
		TokenBudget:            agent.TokenBudgetAdvised{Tokens: tokens.Value, Note: note.Value},
	}}
	restored, err := decodeCredential(encodeCredential(credential))
	if err != nil {
		t.Fatalf("decode enabled policy: %v", err)
	}
	assertCredentialEqual(t, restored, credential)

	disabled := sampleCredential(t, nil)
	disabled.WorkPolicy = agent.WorkPolicyDisabled{}
	restoredDisabled, err := decodeCredential(encodeCredential(disabled))
	if err != nil {
		t.Fatalf("decode disabled policy: %v", err)
	}
	assertCredentialEqual(t, restoredDisabled, disabled)
}

func TestDecodeWorkPolicyRejectsUnknownState(t *testing.T) {
	if _, err := decodeWorkPolicy(workPolicyWire{State: "sometimes"}); err == nil {
		t.Fatalf("unknown work policy state was decoded")
	}
	if _, err := decodeWorkPolicy(workPolicyWire{State: "work_seeking_enabled"}); err == nil {
		t.Fatalf("enabled policy without a daily task budget was decoded")
	}
}

func TestUpdateWorkPolicyResultRoundTrip(t *testing.T) {
	credential := sampleCredential(t, nil)
	credential.WorkPolicy = agent.WorkPolicyDisabled{}
	updated, err := decodeUpdateWorkPolicyResult(encodeUpdateWorkPolicyResult(agent.UpdateWorkPolicyStoreUpdated{Value: credential}))
	if err != nil {
		t.Fatalf("decode update result: %v", err)
	}
	if typed, matched := updated.(agent.UpdateWorkPolicyStoreUpdated); !matched {
		t.Fatalf("update result = %T", updated)
	} else {
		assertCredentialEqual(t, typed.Value, credential)
	}

	rejected, err := decodeUpdateWorkPolicyResult(encodeUpdateWorkPolicyResult(agent.UpdateWorkPolicyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "boom")}))
	if err != nil {
		t.Fatalf("decode rejected update result: %v", err)
	}
	if typed, matched := rejected.(agent.UpdateWorkPolicyStoreRejected); !matched || typed.Reason.Description() != "boom" {
		t.Fatalf("rejected result = %#v", rejected)
	}
}

func TestWorkActivityResultRoundTrip(t *testing.T) {
	credentialID := core.NewAgentCredentialID().(core.AgentCredentialIDCreated).Value
	listed, err := decodeWorkActivityResult(encodeWorkActivityResult(agent.WorkActivityStoreListed{Values: []agent.CredentialWorkActivity{
		{CredentialID: credentialID, TasksToday: 3, CreditsSpentToday: 25, ActiveReservations: 2},
	}}))
	if err != nil {
		t.Fatalf("decode work activity result: %v", err)
	}
	typed, matched := listed.(agent.WorkActivityStoreListed)
	if !matched || len(typed.Values) != 1 {
		t.Fatalf("work activity result = %#v", listed)
	}
	got := typed.Values[0]
	if got.CredentialID != credentialID || got.TasksToday != 3 || got.CreditsSpentToday != 25 || got.ActiveReservations != 2 {
		t.Fatalf("work activity = %+v", got)
	}

	rejected, err := decodeWorkActivityResult(encodeWorkActivityResult(agent.WorkActivityStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "boom")}))
	if err != nil {
		t.Fatalf("decode rejected work activity result: %v", err)
	}
	if typed, matched := rejected.(agent.WorkActivityStoreRejected); !matched || typed.Reason.Description() != "boom" {
		t.Fatalf("rejected result = %#v", rejected)
	}

	if _, err := decodeWorkActivityResult(workActivityResultWire{Variant: "bogus"}); err == nil {
		t.Fatalf("unknown variant decoded")
	}
}

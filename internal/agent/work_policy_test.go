package agent

import (
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/task"
)

func TestParseWorkSeekingStateRoundTrips(t *testing.T) {
	for _, state := range []WorkSeekingState{WorkSeekingDisabled, WorkSeekingEnabled} {
		result := ParseWorkSeekingState(state.String())
		accepted, matched := result.(WorkSeekingStateAccepted)
		if !matched {
			t.Fatalf("ParseWorkSeekingState rejected %q", state.String())
		}
		if accepted.Value != state {
			t.Fatalf("ParseWorkSeekingState(%q) = %q", state.String(), accepted.Value.String())
		}
	}
}

func TestParseWorkSeekingStateRejectsUnknown(t *testing.T) {
	for _, raw := range []string{"", "enabled", "disabled", "work_seeking"} {
		if _, matched := ParseWorkSeekingState(raw).(WorkSeekingStateRejected); !matched {
			t.Fatalf("ParseWorkSeekingState accepted %q", raw)
		}
	}
}

func TestWorkPolicyStateMatchesVariant(t *testing.T) {
	if WorkPolicyState(WorkPolicyDisabled{}) != WorkSeekingDisabled {
		t.Fatal("disabled policy did not report the disabled state")
	}
	budget, matched := NewDailyTaskBudget(5).(DailyTaskBudgetAccepted)
	if !matched {
		t.Fatal("NewDailyTaskBudget(5) was rejected")
	}
	enabled := WorkPolicyEnabled{Allowances: WorkAllowances{
		MaxTasksPerDay:         budget.Value,
		ConcurrentReservations: ConcurrentReservationsUnlimited{},
		DailySpend:             DailySpendUnlimited{},
		TaskTypes:              AllTaskTypesAllowed{},
		RewardFloor:            NoRewardFloor{},
		TokenBudget:            NoTokenBudgetAdvisory{},
	}}
	if WorkPolicyState(enabled) != WorkSeekingEnabled {
		t.Fatal("enabled policy did not report the enabled state")
	}
}

func TestNewDailyTaskBudgetBounds(t *testing.T) {
	for _, value := range []int64{-1, 0, 10001} {
		if _, matched := NewDailyTaskBudget(value).(DailyTaskBudgetRejected); !matched {
			t.Fatalf("NewDailyTaskBudget accepted %d", value)
		}
	}
	for _, value := range []int64{1, 5, 10000} {
		accepted, matched := NewDailyTaskBudget(value).(DailyTaskBudgetAccepted)
		if !matched {
			t.Fatalf("NewDailyTaskBudget rejected %d", value)
		}
		if accepted.Value.Int64() != value {
			t.Fatalf("NewDailyTaskBudget(%d) round-tripped as %d", value, accepted.Value.Int64())
		}
	}
}

func TestNewConcurrentReservationCapBounds(t *testing.T) {
	for _, value := range []int64{-1, 0, 1001} {
		if _, matched := NewConcurrentReservationCap(value).(ConcurrentReservationCapRejected); !matched {
			t.Fatalf("NewConcurrentReservationCap accepted %d", value)
		}
	}
	accepted, matched := NewConcurrentReservationCap(3).(ConcurrentReservationCapAccepted)
	if !matched {
		t.Fatal("NewConcurrentReservationCap rejected 3")
	}
	if accepted.Value.Int64() != 3 {
		t.Fatalf("NewConcurrentReservationCap(3) round-tripped as %d", accepted.Value.Int64())
	}
}

func TestNewTaskTypesLimitedDeduplicatesAndRejectsEmpty(t *testing.T) {
	if _, matched := NewTaskTypesLimited(nil).(TaskTypesLimitedRejected); !matched {
		t.Fatal("NewTaskTypesLimited accepted an empty restriction")
	}
	result := NewTaskTypesLimited([]task.TaskType{task.TaskTypeResearch, task.TaskTypePlanning, task.TaskTypeResearch})
	accepted, matched := result.(TaskTypesLimitedAccepted)
	if !matched {
		t.Fatal("NewTaskTypesLimited rejected a valid restriction")
	}
	if got := len(accepted.Value.Values()); got != 2 {
		t.Fatalf("expected 2 unique task types, got %d", got)
	}
	if !accepted.Value.Allows(task.TaskTypePlanning) {
		t.Fatal("restriction does not allow a listed type")
	}
	if accepted.Value.Allows(task.TaskTypeQATesting) {
		t.Fatal("restriction allows an unlisted type")
	}
}

func TestNewTokenBudgetTokensRequiresPositive(t *testing.T) {
	if _, matched := NewTokenBudgetTokens(0).(TokenBudgetTokensRejected); !matched {
		t.Fatal("NewTokenBudgetTokens accepted 0")
	}
	accepted, matched := NewTokenBudgetTokens(2_000_000).(TokenBudgetTokensAccepted)
	if !matched {
		t.Fatal("NewTokenBudgetTokens rejected a positive count")
	}
	if accepted.Value.Int64() != 2_000_000 {
		t.Fatalf("token budget round-tripped as %d", accepted.Value.Int64())
	}
}

func TestNewTokenBudgetNoteTrimsAndBounds(t *testing.T) {
	accepted, matched := NewTokenBudgetNote("  spend sparingly  ").(TokenBudgetNoteAccepted)
	if !matched {
		t.Fatal("NewTokenBudgetNote rejected a short note")
	}
	if accepted.Value.String() != "spend sparingly" {
		t.Fatalf("note was not trimmed: %q", accepted.Value.String())
	}
	long := make([]byte, 501)
	for index := range long {
		long[index] = 'a'
	}
	if _, matched := NewTokenBudgetNote(string(long)).(TokenBudgetNoteRejected); !matched {
		t.Fatal("NewTokenBudgetNote accepted an over-long note")
	}
}

func TestTaskWorkerOriginForTaskScopedCredential(t *testing.T) {
	credentialID := newTestCredentialID(t)
	taskID := newTestTaskID(t)
	credential := Credential{ID: credentialID, TaskID: &taskID, WorkPolicy: WorkPolicyDisabled{}}
	origin, matched := credential.TaskWorkerOrigin().(task.WorkerViaTaskCredential)
	if !matched {
		t.Fatalf("task-scoped origin = %T, want WorkerViaTaskCredential", credential.TaskWorkerOrigin())
	}
	if origin.CredentialID != credentialID {
		t.Fatalf("origin credential = %s, want %s", origin.CredentialID.String(), credentialID.String())
	}
}

func TestTaskWorkerOriginSnapshotsEnabledPolicy(t *testing.T) {
	credentialID := newTestCredentialID(t)
	budget := mustDailyTaskBudget(t, 4)
	cap, capMatched := NewConcurrentReservationCap(2).(ConcurrentReservationCapAccepted)
	if !capMatched {
		t.Fatalf("cap rejected")
	}
	floorAmount, floorMatched := ledger.NewCreditAmount(15).(ledger.CreditAmountAccepted)
	if !floorMatched {
		t.Fatalf("floor amount rejected")
	}
	limited, limitedMatched := NewTaskTypesLimited([]task.TaskType{task.TaskTypeResearch}).(TaskTypesLimitedAccepted)
	if !limitedMatched {
		t.Fatalf("task type restriction rejected")
	}
	credential := Credential{ID: credentialID, WorkPolicy: WorkPolicyEnabled{Allowances: WorkAllowances{
		MaxTasksPerDay:         budget,
		ConcurrentReservations: ConcurrentReservationsCapped{Limit: cap.Value},
		DailySpend:             DailySpendUnlimited{},
		TaskTypes:              limited.Value,
		RewardFloor:            RewardFloorAtLeast{Minimum: floorAmount.Value},
		TokenBudget:            NoTokenBudgetAdvisory{},
	}}}

	origin, matched := credential.TaskWorkerOrigin().(task.WorkerViaCredential)
	if !matched {
		t.Fatalf("origin = %T, want WorkerViaCredential", credential.TaskWorkerOrigin())
	}
	snapshot, enabled := origin.Policy.(task.CredentialWorkBudget)
	if !enabled {
		t.Fatalf("policy snapshot = %T, want CredentialWorkBudget", origin.Policy)
	}
	if snapshot.DailyTaskLimit != 4 {
		t.Fatalf("daily limit = %d, want 4", snapshot.DailyTaskLimit)
	}
	if capped, capOK := snapshot.Concurrent.(task.ReservationConcurrencyCapAtMost); !capOK || capped.Limit != 2 {
		t.Fatalf("concurrency = %#v, want at most 2", snapshot.Concurrent)
	}
	if types, typesOK := snapshot.Types.(task.OnlyWorkTaskTypes); !typesOK || !types.Allows(task.TaskTypeResearch) || types.Allows(task.TaskTypeGeneral) {
		t.Fatalf("type restriction did not survive the snapshot: %#v", snapshot.Types)
	}
	if floor, floorOK := snapshot.RewardFloor.(task.WorkRewardFloorAtLeast); !floorOK || floor.MinimumCredits != 15 {
		t.Fatalf("reward floor = %#v, want at least 15", snapshot.RewardFloor)
	}
}

func TestSpendOriginCapsOnlyConfiguredSpendAllowance(t *testing.T) {
	credentialID := newTestCredentialID(t)
	disabled := Credential{ID: credentialID, WorkPolicy: WorkPolicyDisabled{}}
	origin, matched := disabled.SpendOrigin().(ledger.SpendViaWorkCredential)
	if !matched {
		t.Fatalf("spend origin = %T, want SpendViaWorkCredential", disabled.SpendOrigin())
	}
	if _, uncapped := origin.Cap.(ledger.NoSpendDayCap); !uncapped {
		t.Fatalf("disabled policy carries a spend cap: %#v", origin.Cap)
	}

	capAmount, capMatched := ledger.NewCreditAmount(60).(ledger.CreditAmountAccepted)
	if !capMatched {
		t.Fatalf("cap amount rejected")
	}
	capped := Credential{ID: credentialID, WorkPolicy: WorkPolicyEnabled{Allowances: WorkAllowances{
		MaxTasksPerDay:         mustDailyTaskBudget(t, 1),
		ConcurrentReservations: ConcurrentReservationsUnlimited{},
		DailySpend:             DailySpendCapped{Limit: capAmount.Value},
		TaskTypes:              AllTaskTypesAllowed{},
		RewardFloor:            NoRewardFloor{},
		TokenBudget:            NoTokenBudgetAdvisory{},
	}}}
	cappedOrigin := capped.SpendOrigin().(ledger.SpendViaWorkCredential)
	if limit, limitMatched := cappedOrigin.Cap.(ledger.SpendDayCapAtMost); !limitMatched || limit.Limit.Int64() != 60 {
		t.Fatalf("spend cap = %#v, want at most 60", cappedOrigin.Cap)
	}
}

func mustDailyTaskBudget(t *testing.T, value int64) DailyTaskBudget {
	t.Helper()
	accepted, matched := NewDailyTaskBudget(value).(DailyTaskBudgetAccepted)
	if !matched {
		t.Fatalf("daily task budget rejected: %d", value)
	}
	return accepted.Value
}

func newTestCredentialID(t *testing.T) core.AgentCredentialID {
	t.Helper()
	created, matched := core.NewAgentCredentialID().(core.AgentCredentialIDCreated)
	if !matched {
		t.Fatalf("agent credential id rejected")
	}
	return created.Value
}

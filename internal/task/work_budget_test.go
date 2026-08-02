package task

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/event/eventtest"
)

func testCredentialID(t *testing.T) core.AgentCredentialID {
	t.Helper()
	created, matched := core.NewAgentCredentialID().(core.AgentCredentialIDCreated)
	if !matched {
		t.Fatalf("agent credential id rejected")
	}
	return created.Value
}

func testCreditReward(t *testing.T, amount int64) RewardSpec {
	t.Helper()
	accepted, matched := NewCreditRewardAmount(amount).(CreditRewardAmountAccepted)
	if !matched {
		t.Fatalf("credit reward amount rejected")
	}
	return CreditRewardSpec{Amount: accepted.Value}
}

func testWorkBudget(dailyLimit int64) CredentialWorkBudget {
	return CredentialWorkBudget{
		DailyTaskLimit: dailyLimit,
		Concurrent:     NoReservationConcurrencyCap{},
		Types:          AnyWorkTaskType{},
		RewardFloor:    NoWorkRewardFloor{},
	}
}

func TestWorkBudgetForSubmissionUserAndTaskCredentialAreNeverCharged(t *testing.T) {
	target := Task{Type: TaskTypeGeneral, Reward: NoRewardSpec{}, Participation: ParticipationPolicyOpen}
	for _, origin := range []WorkerOrigin{WorkerIsUser{}, WorkerViaTaskCredential{CredentialID: testCredentialID(t)}} {
		charge, problem := WorkBudgetForSubmission(origin, target)
		if problem != nil {
			t.Fatalf("origin %T was refused: %s", origin, problem.Description())
		}
		if _, matched := charge.(NoWorkBudgetCharge); !matched {
			t.Fatalf("origin %T charge = %T, want NoWorkBudgetCharge", origin, charge)
		}
	}
}

func TestWorkBudgetForSubmissionRefusesDisabledCredentialOnOpenTask(t *testing.T) {
	target := Task{Type: TaskTypeGeneral, Reward: NoRewardSpec{}, Participation: ParticipationPolicyOpen}
	origin := WorkerViaCredential{CredentialID: testCredentialID(t), Policy: CredentialWorkDisabled{}}
	_, problem := WorkBudgetForSubmission(origin, target)
	if problem == nil {
		t.Fatalf("disabled credential was allowed to submit directly")
	}
	if problem.Code() != core.ErrorCodePermissionDenied {
		t.Fatalf("code = %s, want permission_denied", problem.Code().String())
	}
}

func TestWorkBudgetForSubmissionChargesEnabledCredentialOnOpenTask(t *testing.T) {
	credentialID := testCredentialID(t)
	target := Task{Type: TaskTypeResearch, Reward: testCreditReward(t, 20), Participation: ParticipationPolicyOpen}
	origin := WorkerViaCredential{CredentialID: credentialID, Policy: testWorkBudget(5)}
	charge, problem := WorkBudgetForSubmission(origin, target)
	if problem != nil {
		t.Fatalf("enabled credential was refused: %s", problem.Description())
	}
	budgeted, matched := charge.(ChargeDailyTaskBudget)
	if !matched {
		t.Fatalf("charge = %T, want ChargeDailyTaskBudget", charge)
	}
	if budgeted.CredentialID != credentialID || budgeted.DailyTaskLimit != 5 {
		t.Fatalf("charge = %+v, want credential %s with limit 5", budgeted, credentialID.String())
	}
}

func TestWorkBudgetForSubmissionSkipsChargeOnReservationRequiredTask(t *testing.T) {
	target := Task{Type: TaskTypeGeneral, Reward: NoRewardSpec{}, Participation: ParticipationPolicyReservationRequired}
	// Even a disabled credential submits within an already-granted
	// reservation: eligibility requires the submitter's active reservation,
	// so the engagement was budgeted when the reservation was established.
	origin := WorkerViaCredential{CredentialID: testCredentialID(t), Policy: CredentialWorkDisabled{}}
	charge, problem := WorkBudgetForSubmission(origin, target)
	if problem != nil {
		t.Fatalf("reservation-backed submission was refused: %s", problem.Description())
	}
	if _, matched := charge.(NoWorkBudgetCharge); !matched {
		t.Fatalf("charge = %T, want NoWorkBudgetCharge", charge)
	}
}

func TestWorkBudgetForSubmissionEnforcesTypeRestrictionAndRewardFloor(t *testing.T) {
	budget := testWorkBudget(5)
	budget.Types = OnlyWorkTaskTypes{Values: []TaskType{TaskTypeResearch, TaskTypePlanning}}
	budget.RewardFloor = WorkRewardFloorAtLeast{MinimumCredits: 25}
	origin := WorkerViaCredential{CredentialID: testCredentialID(t), Policy: budget}

	wrongType := Task{Type: TaskTypeCodeReview, Reward: testCreditReward(t, 50), Participation: ParticipationPolicyOpen}
	if _, problem := WorkBudgetForSubmission(origin, wrongType); problem == nil || problem.Code() != core.ErrorCodePermissionDenied {
		t.Fatalf("disallowed task type was not refused with permission_denied: %#v", problem)
	}

	lowReward := Task{Type: TaskTypeResearch, Reward: testCreditReward(t, 20), Participation: ParticipationPolicyOpen}
	if _, problem := WorkBudgetForSubmission(origin, lowReward); problem == nil || problem.Code() != core.ErrorCodePermissionDenied {
		t.Fatalf("below-floor reward was not refused with permission_denied: %#v", problem)
	}

	// A collectible-only reward carries zero credits, so it is below every floor.
	noCredits := Task{Type: TaskTypeResearch, Reward: NoRewardSpec{}, Participation: ParticipationPolicyOpen}
	if _, problem := WorkBudgetForSubmission(origin, noCredits); problem == nil {
		t.Fatalf("zero-credit reward passed a 25-credit floor")
	}

	allowed := Task{Type: TaskTypePlanning, Reward: testCreditReward(t, 25), Participation: ParticipationPolicyOpen}
	if _, problem := WorkBudgetForSubmission(origin, allowed); problem != nil {
		t.Fatalf("allowed task was refused: %s", problem.Description())
	}
}

func TestServiceReserveRefusesDisabledWorkCredential(t *testing.T) {
	store := newTaskMemoryStore()
	service := NewService(store, newTaskPermissionStore(), nil, eventtest.NewRecorder())
	requester := testUserSubject(t)
	worker := testUserSubject(t)
	command := testCreateCommand(t, requester, UserOwner{UserID: requester.ID}, PublicVisibility{})
	created := service.Create(context.Background(), command).(TaskCreated)
	store.ChangeTaskState(context.Background(), created.Value.ID, StateOpen, event.NoEvent{})

	origin := WorkerViaCredential{CredentialID: testCredentialID(t), Policy: CredentialWorkDisabled{}}
	result := service.Reserve(context.Background(), worker, origin, created.Value.ID)
	rejected, matched := result.(ReservationRejected)
	if !matched {
		t.Fatalf("result = %T, want ReservationRejected", result)
	}
	if rejected.Reason.Code() != core.ErrorCodePermissionDenied {
		t.Fatalf("code = %s, want permission_denied", rejected.Reason.Code().String())
	}
}

func TestServiceReserveRefusesTaskScopedCredential(t *testing.T) {
	store := newTaskMemoryStore()
	service := NewService(store, newTaskPermissionStore(), nil, eventtest.NewRecorder())
	requester := testUserSubject(t)
	worker := testUserSubject(t)
	command := testCreateCommand(t, requester, UserOwner{UserID: requester.ID}, PublicVisibility{})
	created := service.Create(context.Background(), command).(TaskCreated)
	store.ChangeTaskState(context.Background(), created.Value.ID, StateOpen, event.NoEvent{})

	origin := WorkerViaTaskCredential{CredentialID: testCredentialID(t)}
	result := service.Reserve(context.Background(), worker, origin, created.Value.ID)
	rejected, matched := result.(ReservationRejected)
	if !matched {
		t.Fatalf("result = %T, want ReservationRejected", result)
	}
	if rejected.Reason.Code() != core.ErrorCodePermissionDenied {
		t.Fatalf("code = %s, want permission_denied", rejected.Reason.Code().String())
	}
}

func TestServiceReservePassesWorkCredentialBudgetToStore(t *testing.T) {
	store := newTaskMemoryStore()
	service := NewService(store, newTaskPermissionStore(), nil, eventtest.NewRecorder())
	requester := testUserSubject(t)
	worker := testUserSubject(t)
	command := testCreateCommand(t, requester, UserOwner{UserID: requester.ID}, PublicVisibility{})
	created := service.Create(context.Background(), command).(TaskCreated)
	store.ChangeTaskState(context.Background(), created.Value.ID, StateOpen, event.NoEvent{})

	credentialID := testCredentialID(t)
	budget := testWorkBudget(3)
	budget.Concurrent = ReservationConcurrencyCapAtMost{Limit: 2}
	origin := WorkerViaCredential{CredentialID: credentialID, Policy: budget}
	result := service.Reserve(context.Background(), worker, origin, created.Value.ID)
	if _, matched := result.(ReservationCreated); !matched {
		t.Fatalf("result = %T, want ReservationCreated", result)
	}
	via, matched := store.lastReservationOrigin.(ReservedViaWorkCredential)
	if !matched {
		t.Fatalf("store origin = %T, want ReservedViaWorkCredential", store.lastReservationOrigin)
	}
	if via.CredentialID != credentialID || via.DailyTaskLimit != 3 {
		t.Fatalf("store origin = %+v, want credential %s with daily limit 3", via, credentialID.String())
	}
	if capped, capMatched := via.Concurrent.(ReservationConcurrencyCapAtMost); !capMatched || capped.Limit != 2 {
		t.Fatalf("store concurrency cap = %#v, want at most 2", via.Concurrent)
	}
}

// TestAllTaskTypesRoundTripThroughParseTaskType pins the exported task-type
// enumeration (which the contracts, MCP schemas, and CHECK constraints
// mirror) to the parser: every listed type parses back to itself, and the
// catalog holds exactly the sixteen knowledge-work types.
func TestAllTaskTypesRoundTripThroughParseTaskType(t *testing.T) {
	types := AllTaskTypes()
	if len(types) != 16 {
		t.Fatalf("AllTaskTypes() has %d types, want 16", len(types))
	}
	for _, taskType := range types {
		parsed, matched := ParseTaskType(taskType.String()).(TaskTypeAccepted)
		if !matched || parsed.Value != taskType {
			t.Fatalf("task type %q did not round-trip through ParseTaskType", taskType.String())
		}
	}
	if _, matched := ParseTaskType("").(TaskTypeAccepted); !matched {
		t.Fatalf("empty task type did not default to general")
	}
	if _, matched := ParseTaskType("gardening").(TaskTypeRejected); !matched {
		t.Fatalf("unknown task type was accepted")
	}
}

package task

import (
	"github.com/e6qu/sharecrop/internal/core"
)

// WorkerOrigin says how the acting user authenticated for a worker mutation
// (reserving a task, creating a submission). Humans acting through their own
// session are never budget-limited; a personal agent credential carries its
// work-policy snapshot so the service can fail closed against it. The types
// live here (not in internal/agent) because internal/agent imports this
// package for its task-type allowance; internal/agent adapts a credential
// into these values.
type WorkerOrigin interface {
	workerOrigin()
}

// WorkerIsUser is a signed-in person acting directly. No budget applies.
type WorkerIsUser struct{}

// WorkerViaTaskCredential is a task-scoped credential auto-issued when a
// reservation became active. It operates within that already-granted
// reservation - enforcement happened at grant time - so it carries no work
// policy; it also may not establish new reservations.
type WorkerViaTaskCredential struct {
	CredentialID core.AgentCredentialID
}

// WorkerViaCredential is an unscoped personal agent credential together with
// the work-policy snapshot its owner configured.
type WorkerViaCredential struct {
	CredentialID core.AgentCredentialID
	Policy       CredentialWorkPolicy
}

func (WorkerIsUser) workerOrigin() {}

func (WorkerViaTaskCredential) workerOrigin() {}

func (WorkerViaCredential) workerOrigin() {}

// CredentialWorkPolicy is the enforcement snapshot of a credential's work
// policy: disabled (the default for every credential - the agent does not
// seek work), or enabled with the budget dimensions this package enforces.
type CredentialWorkPolicy interface {
	credentialWorkPolicy()
}

type CredentialWorkDisabled struct{}

type CredentialWorkBudget struct {
	// DailyTaskLimit is the always-present tasks-per-day cap (reservations
	// plus direct submissions to tasks not already reserved by the user).
	DailyTaskLimit int64
	// Concurrent caps active reservations established via the credential.
	Concurrent ReservationConcurrencyCap
	// Types restricts which task types the credential may engage.
	Types WorkTaskTypeCheck
	// RewardFloor skips tasks rewarding fewer credits than the floor.
	RewardFloor WorkRewardFloor
}

func (CredentialWorkDisabled) credentialWorkPolicy() {}

func (CredentialWorkBudget) credentialWorkPolicy() {}

// ReservationConcurrencyCap caps concurrently active reservations, or is
// absent.
type ReservationConcurrencyCap interface {
	reservationConcurrencyCap()
}

type NoReservationConcurrencyCap struct{}

type ReservationConcurrencyCapAtMost struct {
	Limit int64
}

func (NoReservationConcurrencyCap) reservationConcurrencyCap() {}

func (ReservationConcurrencyCapAtMost) reservationConcurrencyCap() {}

// WorkTaskTypeCheck restricts which task types a credential may engage, or is
// absent (every type allowed).
type WorkTaskTypeCheck interface {
	workTaskTypeCheck()
}

type AnyWorkTaskType struct{}

type OnlyWorkTaskTypes struct {
	Values []TaskType
}

func (AnyWorkTaskType) workTaskTypeCheck() {}

func (OnlyWorkTaskTypes) workTaskTypeCheck() {}

// Allows reports whether the restriction permits the task type.
func (check OnlyWorkTaskTypes) Allows(taskType TaskType) bool {
	for _, allowed := range check.Values {
		if allowed == taskType {
			return true
		}
	}
	return false
}

// WorkRewardFloor is the minimum credit reward a credential accepts, or is
// absent.
type WorkRewardFloor interface {
	workRewardFloor()
}

type NoWorkRewardFloor struct{}

type WorkRewardFloorAtLeast struct {
	MinimumCredits int64
}

func (NoWorkRewardFloor) workRewardFloor() {}

func (WorkRewardFloorAtLeast) workRewardFloor() {}

// creditRewardOf reports the credit component of a task's reward (0 when the
// reward carries no credits), for the reward-floor check.
func creditRewardOf(reward RewardSpec) int64 {
	switch typed := reward.(type) {
	case CreditRewardSpec:
		return typed.Amount.Int64()
	case BundleRewardSpec:
		return typed.Credit.Int64()
	default:
		return 0
	}
}

// checkWorkPolicyAgainstTask applies the task-facing policy dimensions (type
// restriction, reward floor) to a target task. These are configuration
// mismatches rather than exhausted quotas, so they refuse with
// permission_denied; the day/concurrency budgets (429 budget_exceeded) are
// consumed in the store transaction instead.
func checkWorkPolicyAgainstTask(budget CredentialWorkBudget, value Task) *core.DomainError {
	if restricted, matched := budget.Types.(OnlyWorkTaskTypes); matched {
		if !restricted.Allows(value.Type) {
			reason := core.NewDomainError(core.ErrorCodePermissionDenied, "agent credential is not allowed to work "+value.Type.String()+" tasks")
			return &reason
		}
	}
	if floor, matched := budget.RewardFloor.(WorkRewardFloorAtLeast); matched {
		if creditRewardOf(value.Reward) < floor.MinimumCredits {
			reason := core.NewDomainError(core.ErrorCodePermissionDenied, "task credit reward is below the agent credential's minimum reward floor")
			return &reason
		}
	}
	return nil
}

// ReservationOrigin travels inside the reservation store command: it says
// whether (and against which credential budget) the store transaction must
// attribute the reservation and consume budget atomically with the insert.
type ReservationOrigin interface {
	reservationOrigin()
}

type ReservedByUserSession struct{}

type ReservedViaWorkCredential struct {
	CredentialID core.AgentCredentialID
	// DailyTaskLimit is consumed (1 per reservation) from the credential's
	// UTC-day counter inside the create transaction.
	DailyTaskLimit int64
	// Concurrent is checked against the credential's active reservations
	// inside the same transaction.
	Concurrent ReservationConcurrencyCap
}

func (ReservedByUserSession) reservationOrigin() {}

func (ReservedViaWorkCredential) reservationOrigin() {}

// WorkBudgetCharge travels inside the submission store command: a direct
// submission via a work-seeking credential consumes 1 from the credential's
// daily task budget atomically with the submission insert.
type WorkBudgetCharge interface {
	workBudgetCharge()
}

type NoWorkBudgetCharge struct{}

type ChargeDailyTaskBudget struct {
	CredentialID   core.AgentCredentialID
	DailyTaskLimit int64
}

func (NoWorkBudgetCharge) workBudgetCharge() {}

func (ChargeDailyTaskBudget) workBudgetCharge() {}

// WorkBudgetForSubmission gates a submission attempt against the worker
// origin. A human session and a task-scoped worker credential are never
// charged. On a reservation-required task the submitter must already hold an
// active reservation (submission eligibility enforces that), so the
// engagement was budgeted when the reservation was established and no policy
// check applies: the agent is working already-granted work, not seeking new
// work. A submission to an open-participation task is a direct engagement:
// the credential must be enabled for work-seeking, the task must pass the
// type and reward-floor checks, and one unit of the daily task budget is
// consumed atomically with the submission insert.
func WorkBudgetForSubmission(origin WorkerOrigin, target Task) (WorkBudgetCharge, *core.DomainError) {
	credential, viaCredential := origin.(WorkerViaCredential)
	if !viaCredential {
		return NoWorkBudgetCharge{}, nil
	}
	if target.Participation == ParticipationPolicyReservationRequired {
		return NoWorkBudgetCharge{}, nil
	}
	budget, enabled := credential.Policy.(CredentialWorkBudget)
	if !enabled {
		reason := core.NewDomainError(core.ErrorCodePermissionDenied, "agent credential is not enabled for work-seeking; its owner must enable it with a work budget")
		return nil, &reason
	}
	if problem := checkWorkPolicyAgainstTask(budget, target); problem != nil {
		return nil, problem
	}
	return ChargeDailyTaskBudget{CredentialID: credential.CredentialID, DailyTaskLimit: budget.DailyTaskLimit}, nil
}
